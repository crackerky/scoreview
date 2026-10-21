// Package site は参照サイト score.clubjr.com のページの種類、URL の組み立て、
// リンクからの ID の取り出しを持つ。取得器(fetch)とパーサ(parse)の両方が使う。
package site

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// BaseURL は参照サイトの起点。ページ内のリンクはすべてこのホストの絶対 URL である(clone-spec A 0)。
const BaseURL = "https://score.clubjr.com"

// Lang は参照サイトの表示言語。URL の ?lang= だけで切り替わる(clone-spec A 0)。
type Lang string

const (
	LangJP Lang = "jp"
	LangEN Lang = "en"
)

// Page は取得するページの種類。値は参照サイトのコントローラ名。
type Page string

const (
	PageTournamentList Page = "sctnmt"
	PageTournamentTop  Page = "scv_top"
	PageTeamList       Page = "scv_team_list"
	PageScheduleAll    Page = "scv_search_tn"
	PageScheduleDiv    Page = "scv_search_div"
	PageDivision       Page = "scv_division"
	PageGroup          Page = "scv_group"
	PageTeam           Page = "scv_team"
	PageMatch          Page = "scv_match"
	PagePlayer         Page = "scv_player"
	PageGallery        Page = "scv_tgallery"
)

// URL はページの URL を返す。大会一覧は id を使わない。
// 大会トップは /scview/{id} を使う(/pscore/scv_top/open/{id} と同一内容。clone-spec A 0)。
func URL(page Page, id int64, lang Lang) string {
	q := "?lang=" + string(lang)
	switch page {
	case PageTournamentList:
		return BaseURL + "/sctnmt" + q
	case PageTournamentTop:
		return fmt.Sprintf("%s/scview/%d%s", BaseURL, id, q)
	default:
		return fmt.Sprintf("%s/pscore/%s/open/%d%s", BaseURL, page, id, q)
	}
}

// linkPattern は href や onclick に入っている参照サイト内リンクの形。
// 例 "document.location = 'https://score.clubjr.com/pscore/scv_team/open/17322?lang=jp'"。
var linkPattern = regexp.MustCompile(`/(?:pscore/(scv_[a-z_]+)/open|(scview))/(\d+)`)

// LinkID は文字列(href や onclick の値)から、指定した種類のページへのリンクの ID を取り出す。
// /scview/{id} は大会トップとして扱う。見つからなければ ok=false。
func LinkID(s string, page Page) (id int64, ok bool) {
	for _, m := range linkPattern.FindAllStringSubmatch(s, -1) {
		kind := Page(m[1])
		if m[2] == "scview" {
			kind = PageTournamentTop
		}
		if kind != page {
			continue
		}
		n, err := strconv.ParseInt(m[3], 10, 64)
		if err != nil {
			// \d+ に一致した桁数が int64 を超えるときだけ起きる。ID として扱えないので無視する
			continue
		}
		return n, true
	}
	return 0, false
}

// LangOf は参照サイト内リンクの ?lang= から、そのページの表示言語を推定する。
// ページ内の全リンクは現在の lang を引き継ぐ(clone-spec A 0)。見つからなければ ok=false。
func LangOf(html string) (lang Lang, ok bool) {
	i := strings.Index(html, "?lang=")
	if i < 0 {
		return "", false
	}
	rest := html[i+len("?lang="):]
	switch {
	case strings.HasPrefix(rest, string(LangEN)):
		return LangEN, true
	case strings.HasPrefix(rest, string(LangJP)):
		return LangJP, true
	}
	return "", false
}

// AbsURL はページ内の画像やリンクの URL を、参照サイトを起点にした絶対 URL に直す。
// 参照サイトの HTML にはエンコードされていない空白や日本語と、%エンコード済みの部分が混在する
// (clone-spec A 6、E 1)ので、どちらも正しくエンコードされた形にそろえる。
func AbsURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("empty url")
	}
	u, err := url.Parse(s)
	if err != nil {
		// "%" の後ろが 16 進でない(ファイル名に生の % がある)場合に起きる。生の % を %25 にして再試行する
		u, err = url.Parse(escapeStrayPercent(s))
		if err != nil {
			return "", fmt.Errorf("parse url %q: %w", raw, err)
		}
	}
	base, err := url.Parse(BaseURL + "/")
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}
	abs := base.ResolveReference(u)
	// 入力のエンコードの仕方(%28 か生の括弧か)によらず同じ文字列になるよう、デコード済みの
	// Path からエンコードし直す
	abs.RawPath = ""
	return abs.String(), nil
}

func escapeStrayPercent(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && (i+2 >= len(s) || !isHex(s[i+1]) || !isHex(s[i+2])) {
			b.WriteString("%25")
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isHex(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}
