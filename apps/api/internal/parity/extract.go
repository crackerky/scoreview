// Package parity は参照サイトのページと本サイトのページを、画面に出る内容の単位で突き合わせる。
// 比べるのは <title>、大会ナビ、サイドバーの SCORE メニュー、本文(#score_view)の見出しと表で、
// ブランドに属する部分(ヘッダー、ロゴ、バナー画像、フッター)は比べない。
package parity

import (
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Page はページから取り出した比較対象。
type Page struct {
	Title    string
	Navi     []Link
	Menu     []Link
	Headings []string
	Tables   []Table
}

// Link は表示文字列とリンク先(パスとクエリだけ)。リンクでない項目は Href が空。
type Link struct {
	Text string
	Href string
}

// Table は本文の表 1 つ。
type Table struct {
	Rows [][]Cell
}

// Cell は表のセル 1 つ。
type Cell struct {
	Text    string
	Classes string // 空白で区切ったクラスを並べ替えて 1 つの空白で連結したもの
	Href    string
}

var (
	spaceRun = regexp.MustCompile(`[\s\x{00a0}]+`)
	// 参照サイトの onclick="document.location = 'https://...'; return false;" からリンク先を取る
	onclickURL = regexp.MustCompile(`document\.location\s*=\s*'([^']+)'`)
	// 試合ページのフッターは表示時刻なので、取得した時刻の違いを差分にしない
	clockJST = regexp.MustCompile(`\d{2}:\d{2}:\d{2} JST`)
	// 大会ページの <title> の接頭辞はブランド(参照サイト [CJS]、本サイトはブランド設定の値)
	titlePrefix = regexp.MustCompile(`^\[[^\]]*\]`)
)

// NormalizeText は連続する空白(全角空白は除く)と &nbsp; を 1 つの半角空白にし、前後を削る。
// 参照サイトは td の末尾に空白を入れる箇所があるが、画面には出ないので比較から外す。
func NormalizeText(s string) string {
	t := spaceRun.ReplaceAllString(s, " ")
	t = clockJST.ReplaceAllString(t, "HH:MM:SS JST")
	return strings.TrimSpace(t)
}

// NormalizeHref はリンク先からスキームとホストを落とし、パスとクエリ(lang など)だけにする。
// 参照サイトは絶対 URL、本サイトは相対 URL なので、そのままでは比べられない。
func NormalizeHref(raw string) string {
	if raw == "" || raw == "#" {
		return ""
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	query := parsed.Query()
	if query.Get("lang") == "" {
		// 参照サイトは lang 省略を jp として扱う(clone-spec A 0)
		query.Set("lang", "jp")
	}
	query.Del("refpos")
	return parsed.Path + "?" + query.Encode()
}

func normalizeClasses(raw string) string {
	fields := strings.Fields(raw)
	sort.Strings(fields)
	return strings.Join(fields, " ")
}

func linkOf(sel *goquery.Selection) string {
	anchor := sel
	if !sel.Is("a") {
		anchor = sel.Find("a").First()
	}
	if anchor.Length() == 0 {
		return ""
	}
	if onclick, ok := anchor.Attr("onclick"); ok {
		if m := onclickURL.FindStringSubmatch(onclick); m != nil {
			return NormalizeHref(m[1])
		}
	}
	href, _ := anchor.Attr("href")
	return NormalizeHref(href)
}

// Extract は HTML から比較対象を取り出す。
func Extract(doc *goquery.Document) Page {
	var page Page
	page.Title = NormalizeText(titlePrefix.ReplaceAllString(NormalizeText(doc.Find("title").Text()), ""))

	doc.Find("#score_navi li").Each(func(_ int, li *goquery.Selection) {
		// ドロップダウンの見出し(p)とリンク(a)を、入れ子の外側から順に並べる
		head := li.ChildrenFiltered("a, p").First()
		if head.Length() == 0 {
			return
		}
		text := NormalizeText(head.Text())
		if text == "" {
			return // 幅調整用の空 li
		}
		page.Navi = append(page.Navi, Link{Text: text, Href: linkOf(head)})
	})

	doc.Find("#sub .menu_list li").Each(func(_ int, li *goquery.Selection) {
		head := li.ChildrenFiltered("a, p").First()
		if head.Length() == 0 {
			return
		}
		page.Menu = append(page.Menu, Link{Text: NormalizeText(head.Text()), Href: linkOf(head)})
	})

	view := doc.Find("#score_view")
	view.Find(".text_title, .TeamListMIDA, .SCV_MIDA_M").Each(func(_ int, sel *goquery.Selection) {
		// 自由記述欄(#SCV_TopPage)の中の橙バッジは本文の文章として扱い、見出しに数えない
		if sel.ParentsFiltered("#SCV_TopPage").Length() > 0 {
			return
		}
		page.Headings = append(page.Headings, NormalizeText(sel.Text()))
	})

	view.Find("table").Each(func(_ int, table *goquery.Selection) {
		// 入れ子の表は外側の表のセルの文字として数えるので、ここでは最も内側の表だけを取る
		if table.Find("table").Length() > 0 {
			return
		}
		if table.ParentsFiltered("#SCV_TopPage").Length() > 0 {
			return
		}
		var t Table
		table.Find("tr").Each(func(_ int, tr *goquery.Selection) {
			var row []Cell
			tr.ChildrenFiltered("td, th").Each(func(_ int, td *goquery.Selection) {
				classes, _ := td.Attr("class")
				row = append(row, Cell{
					Text:    NormalizeText(td.Text()),
					Classes: normalizeClasses(classes),
					Href:    linkOf(td),
				})
			})
			t.Rows = append(t.Rows, row)
		})
		page.Tables = append(page.Tables, t)
	})
	return page
}
