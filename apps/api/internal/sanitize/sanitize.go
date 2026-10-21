// Package sanitize は大会トップの自由記述 HTML(top_html_*)を許可リスト方式で無害化する。
//
// 参照サイトの自由記述は管理画面の WYSIWYG 出力で、インライン style(フォント、色、余白、画像の寸法)、
// span/p/strong/br/img/a/table 系の要素、橙バッジのクラス(SCV_MIDA_M 等)を使う(clone-spec A 4)。
// 見た目を保つためにそれらを残し、script、イベントハンドラ属性、javascript: URL、iframe、style 要素を落とす。
// 参照サイトは無害化せずそのまま出しているが、これは意図的な差分(docs/requirements.md)。
package sanitize

import (
	"regexp"
	"strconv"
	"strings"

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
	// class の値: 参照サイトの自由記述で使う見出しのクラスだけ(1 つ)。任意のクラスを許すと、
	// 閲覧ページの部品(サイドバー、ナビ)のクラスを付けて画面を偽装できる(セキュリティ点検 L-5)
	className := regexp.MustCompile(`^(?:` + strings.Join(contentClasses(), "|") + `)$`)
	// href と src: "//host" と "/\host"(ブラウザが他のサイトと読む)、バックスラッシュ、制御文字を通さない。
	// スキームの検査(http、https、mailto だけ)は AllowStandardURLs が別に行う
	linkURL := regexp.MustCompile(`^(?:[^/\\\x00-\x20]|/[^/\\\x00-\x20]|/$)[^\\\x00-\x1f\x7f]*$`)
	// 寸法などの属性値: 数字と % だけ
	// (2000 以下の数か 100% 以下。style の width と height の上限と揃える)
	dimension := regexp.MustCompile(`^(?:[0-9]{1,3}|1[0-9]{3}|2000|[0-9]{1,2}%|100%)$`)
	align := regexp.MustCompile(`(?i)^(left|right|center|justify|top|middle|bottom|baseline)$`)

	// インライン style で残すプロパティ。値の検査は bluemonday の既定のプロパティ別ハンドラに任せる
	// (expression() や url() は通らない)。font-family だけは既定のハンドラが日本語のフォント名と
	// 二重引用符を通さないので、上の正規表現で検査する。寸法のプロパティは lengthLimits で
	// 負の値と大きすぎる値を落とす。
	styleProperties := []string{
		"color", "background-color", "text-align",
		"font-weight", "font-style", "text-decoration", "vertical-align",
	}

	policy := bluemonday.NewPolicy()
	policy.AllowElements(
		"span", "p", "div", "br", "hr", "strong", "b", "em", "i", "u", "s", "small", "big",
		"sub", "sup", "font", "center", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li",
		"blockquote", "pre",
	)
	policy.AllowTables()
	// AllowImages は使わない(img の src と width、height を検査なしで許すため)。同じ指定を下で
	// 検査つきで書く
	policy.AllowAttrs("align").Matching(bluemonday.ImageAlign).OnElements("img")

	policy.AllowStandardURLs() // http, https, mailto。相対 URL も許す
	policy.AllowRelativeURLs(true)
	policy.AllowAttrs("href").Matching(linkURL).OnElements("a")
	policy.AllowAttrs("target").Matching(regexp.MustCompile(`^_blank$`)).OnElements("a")
	policy.AllowAttrs("src").Matching(linkURL).OnElements("img")
	policy.AllowAttrs("alt").Matching(bluemonday.Paragraph).OnElements("img")
	policy.AllowAttrs("title").OnElements("img")
	policy.AllowAttrs("width", "height", "border", "cellpadding", "cellspacing", "colspan", "rowspan").
		Matching(dimension).Globally()
	policy.AllowAttrs("align", "valign").Matching(align).Globally()
	policy.AllowAttrs("color").Matching(regexp.MustCompile(`^#?[A-Za-z0-9]+$`)).OnElements("font")
	policy.AllowAttrs("class").Matching(className).Globally()
	policy.AllowAttrs("title").Globally()

	policy.AllowStyles(styleProperties...).Globally()
	for property, limit := range lengthLimits() {
		policy.AllowStyles(property).MatchingHandler(limit.allows).Globally()
	}
	policy.AllowStyles("font-family").Matching(fontFamily).Globally()
	return &Sanitizer{policy: policy}
}

// HTML は自由記述 HTML を無害化して返す。許可しない要素はタグだけを落として中身の文字を残し、
// script と style は中身ごと落とす(bluemonday の既定)。
func (s *Sanitizer) HTML(raw string) string {
	return s.policy.Sanitize(raw)
}

// contentClasses は自由記述で残すクラス。参照サイトの大会トップで使われているのは見出しの
// SCV_MIDA_M だけで(代表 16 大会)、同じ系統の大小の見出しも許す(clone-spec A 4、A 6 の CSS 一覧)。
func contentClasses() []string {
	return []string{"SCV_MIDA_L", "SCV_MIDA_M", "SCV_MIDA_S"}
}

// lengthLimit は寸法のプロパティの値の上限。値は空白区切りの 1〜4 個の長さ(padding: 4px 10px)で、
// どれも 0 以上、px に換算して maxPx 以下(% は maxPercent 以下、単位のない数は line-height の倍率
// として maxNumber 以下)でなければ、プロパティごと落とす。負の margin や巨大な padding、
// font-size で、閲覧ページの他の部分に重ねたり覆ったりして画面を偽装させない(セキュリティ点検 L-5)。
type lengthLimit struct {
	// pattern は 1 つの長さ(数と単位)。符号は許さない(負の値を書けない)
	pattern    *regexp.Regexp
	maxPx      float64
	maxPercent float64
	maxNumber  float64
	keywords   []string
}

// lengthLimits はプロパティごとの上限。参照サイトの大会トップ(代表 16 大会)で観測した最大値
// (width 650px、height 433px、font-size 17px、line-height 42.57px、padding 10px、margin 10px)を
// 十分に上回る値にする。
func lengthLimits() map[string]lengthLimit {
	box := lengthLimit{maxPx: 100, maxPercent: 100, keywords: []string{"auto"}}
	limits := map[string]lengthLimit{
		"width":          {maxPx: 2000, maxPercent: 100, keywords: []string{"auto"}},
		"height":         {maxPx: 2000, maxPercent: 100, keywords: []string{"auto"}},
		"font-size":      {maxPx: 72, maxPercent: 400, keywords: fontSizeKeywords()},
		"line-height":    {maxPx: 200, maxPercent: 400, maxNumber: 4, keywords: []string{"normal"}},
		"letter-spacing": {maxPx: 20, keywords: []string{"normal"}},
		"padding":        box,
		"margin":         box,
		"border-radius":  {maxPx: 100, maxPercent: 50},
	}
	pattern := regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?|\.[0-9]+)(px|pt|em|rem|%)?$`)
	for property, limit := range limits {
		limit.pattern = pattern
		limits[property] = limit
	}
	return limits
}

func fontSizeKeywords() []string {
	return []string{
		"xx-small", "x-small", "small", "medium", "large", "x-large", "xx-large", "smaller", "larger",
	}
}

// pxPerUnit は px への換算。em と rem は既定の文字の大きさ 16px で数える。
func pxPerUnit(unit string) float64 {
	switch unit {
	case "pt":
		return 4.0 / 3.0
	case "em", "rem":
		return 16
	default:
		return 1
	}
}

func (l lengthLimit) allows(value string) bool {
	parts := strings.Fields(strings.ToLower(value))
	if len(parts) == 0 || len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		if !l.allowsOne(part) {
			return false
		}
	}
	return true
}

func (l lengthLimit) allowsOne(part string) bool {
	for _, keyword := range l.keywords {
		if part == keyword {
			return true
		}
	}
	match := l.pattern.FindStringSubmatch(part)
	if match == nil {
		return false
	}
	number, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return false
	}
	switch unit := match[2]; unit {
	case "%":
		return number <= l.maxPercent
	case "":
		// 単位のない 0 はどのプロパティでも書ける。0 以外は line-height の倍率だけ
		return number == 0 || number <= l.maxNumber
	default:
		return number*pxPerUnit(unit) <= l.maxPx
	}
}
