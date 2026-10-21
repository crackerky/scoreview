// Package parse は参照サイトの各ページの HTML を、取込用の中間表現(ページごとの構造体)に変える。
//
// 各 ParseXxx は 1 言語分のページを読み、言語で変わる文字列は Text の lang 側にだけ入れる。
// MergeXxxEN は日本語版の構造に英語版の文字列を突き合わせて埋める。構造が食い違えば
// ErrMismatch を返し、黙って片方を捨てない。
package parse

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	"scoreview/api/internal/importer/site"
)

// ErrErrorPage は参照サイトのエラー画面(存在しない ID への 500、ルート不一致の 404)を表す。
// clone-spec A 7: エラー画面は外枠のない CodeIgniter 標準の画面で、200 以外で返る。
var ErrErrorPage = errors.New("reference site error page")

// ErrUnexpected はページの構造が想定と違うことを表す。
var ErrUnexpected = errors.New("unexpected page structure")

// ErrMismatch は日本語版と英語版の構造が一致しないことを表す。
var ErrMismatch = errors.New("jp and en pages do not match")

// Text は日英の文字列。英語版が未登録なら EN は空(参照サイトは補完しない。clone-spec A 4)。
type Text struct {
	JA string `json:"ja"`
	EN string `json:"en"`
}

func textOf(lang site.Lang, s string) Text {
	if lang == site.LangEN {
		return Text{EN: s}
	}
	return Text{JA: s}
}

// mergeText は jp 側の Text に en 側の英語文字列を入れたものを返す。
func mergeText(jp, en Text) Text {
	return Text{JA: jp.JA, EN: en.EN}
}

func checkLang(lang site.Lang) error {
	if lang != site.LangJP && lang != site.LangEN {
		return fmt.Errorf("unknown lang %q", lang)
	}
	return nil
}

func unexpected(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrUnexpected, fmt.Sprintf(format, args...))
}

func mismatch(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrMismatch, fmt.Sprintf(format, args...))
}

// load は HTML を読み、参照サイトのエラー画面なら ErrErrorPage を返す。
func load(r io.Reader) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	title := strings.TrimSpace(doc.Find("title").First().Text())
	h1 := strings.TrimSpace(doc.Find("h1").First().Text())
	if title == "Error" || title == "404 Page Not Found" || h1 == "An Error Was Encountered" {
		return nil, fmt.Errorf("%w: %s", ErrErrorPage, strings.TrimSpace(doc.Find("p").First().Text()))
	}
	return doc, nil
}

// scoreView はページ本文 #score_view を返す。
func scoreView(doc *goquery.Document) (*goquery.Selection, error) {
	sv := doc.Find("#score_view")
	if sv.Length() != 1 {
		return nil, unexpected("#score_view count = %d", sv.Length())
	}
	return sv, nil
}

// asciiSpace は文字列の前後から取り除く空白。全角空白(U+3000)はチーム名などの原文に
// 含まれるので残す(clone-spec C 7)。&nbsp; は表示上の詰め物なので取り除く。
const asciiSpace = " \t\r\n "

// clean は前後の半角空白と &nbsp; を取り除く。
func clean(s string) string {
	return strings.Trim(s, asciiSpace)
}

// nameText は名前の文字列を、HTML の字下げ(改行とタブ)だけを除いて返す。名前の前後の半角空白は
// 名前の一部として残す(参照サイトの title に "-BUTTERFLY -" "-17th placement tournament -" と出る)。
func nameText(sel *goquery.Selection) string {
	return strings.Trim(sel.Text(), "\r\n\t")
}

// cellText はセルの文字列を前後の空白を除いて返す。
func cellText(sel *goquery.Selection) string {
	return clean(sel.Text())
}

// linkID は sel の中(sel 自身を含む)の a 要素の href / onclick から page への ID を探す。
func linkID(sel *goquery.Selection, page site.Page) *int64 {
	var found *int64
	sel.Find("a").AddSelection(sel.Filter("a")).EachWithBreak(func(_ int, a *goquery.Selection) bool {
		for _, attr := range []string{"onclick", "href"} {
			if v, ok := a.Attr(attr); ok {
				if id, ok := site.LinkID(v, page); ok {
					found = &id
					return false
				}
			}
		}
		return true
	})
	return found
}

func ptr[T any](v T) *T { return &v }

// atoiPtr は数値の文字列を *int にする。空なら nil。
func atoiPtr(s string) (*int, error) {
	t := clean(s)
	if t == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(t)
	if err != nil {
		return nil, unexpected("not a number %q", s)
	}
	return &n, nil
}

// atoi は数値の文字列を int にする。空文字はエラー。
func atoi(s string) (int, error) {
	n, err := atoiPtr(s)
	if err != nil {
		return 0, err
	}
	if n == nil {
		return 0, unexpected("empty number")
	}
	return *n, nil
}

// atoiBlank は空なら 0 を返す。ブロック数の 0 は空セルで表示される(clone-spec D 1.1)。
func atoiBlank(s string) (int, error) {
	n, err := atoiPtr(s)
	if err != nil || n == nil {
		return 0, err
	}
	return *n, nil
}

var elapsedPattern = regexp.MustCompile(`^(\d+):(\d{2})$`)

// elapsedSec は経過 "MM:SS" を秒にする(clone-spec D 2.3)。
func elapsedSec(s string) (int, error) {
	m := elapsedPattern.FindStringSubmatch(clean(s))
	if m == nil {
		return 0, unexpected("elapsed %q is not MM:SS", s)
	}
	minutes, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, unexpected("elapsed %q: %v", s, err)
	}
	seconds, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, unexpected("elapsed %q: %v", s, err)
	}
	return minutes*60 + seconds, nil
}

// headingText は橙の見出し span.SCV_MIDA_M の "< 文字列 >" から中身を取り出す。
// clone-spec C 0: JP は "< X >"、EN は "<X>"。
func headingText(sel *goquery.Selection) string {
	s := clean(sel.Text())
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimSuffix(s, ">")
	return clean(s)
}

// comments は sel の子孫の HTML コメントの中身を順に返す。
func comments(sel *goquery.Selection) []string {
	var out []string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.CommentNode {
				out = append(out, c.Data)
			}
			walk(c)
		}
	}
	for _, n := range sel.Nodes {
		walk(n)
	}
	return out
}

// genderOf は選手リンクの class から性別を返す。clone-spec D 1.1: playerLinkM=男、playerLinkF=女、
// playerLink は性別データのない大会。
func genderOf(a *goquery.Selection) *string {
	switch {
	case a.HasClass("playerLinkM"):
		return ptr("M")
	case a.HasClass("playerLinkF"):
		return ptr("F")
	}
	return nil
}

// titleName は大会トップの <title> "[CJS] {名前} " から名前を取り出す。
// clone-spec A 1.7: 大会トップの title は "[CJS] {name} "(末尾に半角空白 1 個)で、英語名が空なら空。
func titleName(doc *goquery.Document) string {
	t := doc.Find("title").First().Text()
	t = strings.TrimPrefix(t, "[CJS] ")
	return clean(t)
}
