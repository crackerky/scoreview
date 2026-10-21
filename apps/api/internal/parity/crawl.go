package parity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/fetch"
)

// ReferenceOrigin は参照サイトの URL の先頭。キャッシュのキーはこの絶対 URL。
const ReferenceOrigin = "https://score.clubjr.com"

// pagePath は比較対象にするページのパス。大会一覧(sctnmt)は全大会を含むので個別の大会の比較から外す。
var pagePath = regexp.MustCompile(`^/(scview/\d+|pscore/scv_[a-z_]+/open/\d+)$`)

// Kind はパスからページ種別(scview、scv_team など)を返す。
func Kind(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "pscore" {
		return parts[1]
	}
	return parts[0]
}

// Discover は大会トップ(jp と en)を起点に、キャッシュにある参照サイトのページをリンクで辿って集める。
// 参照サイトへは一切アクセスしない(キャッシュにないページは辿らない)。戻り値はパスとクエリ。
func Discover(cache *fetch.Cache, tournamentID int64) ([]string, error) {
	start := []string{
		fmt.Sprintf("/scview/%d?lang=jp", tournamentID),
		fmt.Sprintf("/scview/%d?lang=en", tournamentID),
	}
	seen := map[string]bool{}
	queue := append([]string{}, start...)
	var found []string
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if seen[next] {
			continue
		}
		seen[next] = true
		body, err := cache.Get(ReferenceOrigin + next)
		if errors.Is(err, fetch.ErrCacheMiss) || errors.Is(err, fetch.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		found = append(found, next)
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", next, err)
		}
		doc.Find("a").Each(func(_ int, anchor *goquery.Selection) {
			href := linkOf(anchor)
			path, _, _ := strings.Cut(href, "?")
			if pagePath.MatchString(path) && !seen[href] {
				queue = append(queue, href)
			}
		})
	}
	sort.Strings(found)
	return found, nil
}

// LoadReference はキャッシュから参照サイトのページを読む。
func LoadReference(cache *fetch.Cache, path string) (Page, error) {
	body, err := cache.Get(ReferenceOrigin + path)
	if err != nil {
		return Page{}, fmt.Errorf("reference %s: %w", path, err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Page{}, fmt.Errorf("parse reference %s: %w", path, err)
	}
	return Extract(doc), nil
}

// LoadOurs は本サイト(site は http://localhost:3000 など)のページを取得する。
func LoadOurs(ctx context.Context, client *http.Client, site, path string) (Page, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, site+path, nil)
	if err != nil {
		return Page{}, 0, fmt.Errorf("build request %s: %w", path, err)
	}
	res, err := client.Do(req)
	if err != nil {
		return Page{}, 0, fmt.Errorf("get %s: %w", path, err)
	}
	defer func() { _ = res.Body.Close() }() // 読み終えた後の Close の失敗は結果に影響しない
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Page{}, res.StatusCode, fmt.Errorf("read %s: %w", path, err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Page{}, res.StatusCode, fmt.Errorf("parse %s: %w", path, err)
	}
	return Extract(doc), res.StatusCode, nil
}
