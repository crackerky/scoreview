// Package sanitize は大会トップの自由記述 HTML(top_html_*)を許可リスト方式で無害化する。
//
// 参照サイトの自由記述は管理画面の WYSIWYG 出力で、インライン style(フォント、色、余白、画像の寸法)、
// span/p/strong/br/img/a/table 系の要素、橙バッジのクラス(SCV_MIDA_M 等)を使う(clone-spec A 4)。
// 見た目を保つためにそれらを残し、script、イベントハンドラ属性、javascript: URL、iframe、style 要素を落とす。
// 参照サイトは無害化せずそのまま出しているが、これは意図的な差分(docs/requirements.md)。
package sanitize

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
)

// Sanitizer は自由記述 HTML の無害化ポリシー。構築に時間がかかるので 1 回作って使い回す。
// 並行に呼んでよい(bluemonday.Policy は構築後に変更しなければ並行安全)。
type Sanitizer struct {
	policy *bluemonday.Policy
}

// New は自由記述用のポリシーを作る。
func New() *Sanitizer {
	// font-family の値: 文字、数字、空白、引用符、カンマ、ハイフン、ピリオドだけ。
	// 括弧、セミコロン、コロン、バックスラッシュを通さないので url() や expression() を書けない
	fontFamily := regexp.MustCompile(`^[\p{L}\p{N} ,'"\-_.]+$`)
	// class の値: 英数字、アンダースコア、ハイフン、空白(SCV_MIDA_M など参照サイトのクラス名)
	className := regexp.MustCompile(`^[A-Za-z0-9_\- ]+$`)
	// 寸法などの属性値: 数字と % だけ
	dimension := regexp.MustCompile(`^[0-9]+%?$`)
	align := regexp.MustCompile(`(?i)^(left|right|center|justify|top|middle|bottom|baseline)$`)

	// インライン style で残すプロパティ。値の検査は bluemonday の既定のプロパティ別ハンドラに任せる
	// (expression() や url() は通らない)。font-family だけは既定のハンドラが日本語のフォント名と
	// 二重引用符を通さないので、上の正規表現で検査する。
	styleProperties := []string{
		"font-size", "color", "background-color", "padding", "border-radius",
		"line-height", "letter-spacing", "text-align", "width", "height", "margin",
		"font-weight", "font-style", "text-decoration", "vertical-align",
	}

	policy := bluemonday.NewPolicy()
	policy.AllowElements(
		"span", "p", "div", "br", "hr", "strong", "b", "em", "i", "u", "s", "small", "big",
		"sub", "sup", "font", "center", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li",
		"blockquote", "pre",
	)
	policy.AllowTables()
	policy.AllowImages()

	policy.AllowStandardURLs() // http, https, mailto。相対 URL も許す
	policy.AllowRelativeURLs(true)
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("target").Matching(regexp.MustCompile(`^_blank$`)).OnElements("a")
	policy.AllowAttrs("src", "alt", "title").OnElements("img")
	policy.AllowAttrs("width", "height", "border", "cellpadding", "cellspacing", "colspan", "rowspan").
		Matching(dimension).Globally()
	policy.AllowAttrs("align", "valign").Matching(align).Globally()
	policy.AllowAttrs("color").Matching(regexp.MustCompile(`^#?[A-Za-z0-9]+$`)).OnElements("font")
	policy.AllowAttrs("class").Matching(className).Globally()
	policy.AllowAttrs("title").Globally()

	policy.AllowStyles(styleProperties...).Globally()
	policy.AllowStyles("font-family").Matching(fontFamily).Globally()
	return &Sanitizer{policy: policy}
}

// HTML は自由記述 HTML を無害化して返す。許可しない要素はタグだけを落として中身の文字を残し、
// script と style は中身ごと落とす(bluemonday の既定)。
func (s *Sanitizer) HTML(raw string) string {
	return s.policy.Sanitize(raw)
}
