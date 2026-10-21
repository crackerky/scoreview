package sanitize_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/sanitize"
)

func TestHTMLRemovesXSSVectors(t *testing.T) {
	s := sanitize.New()
	cases := []struct {
		name      string
		input     string
		forbidden []string // 出力に残ってはいけない文字列(小文字で比較)
		kept      string   // 残るべき文字列
	}{
		{
			"script element",
			`<p>ok</p><script>alert(1)</script>`,
			[]string{"<script", "alert"},
			"<p>ok</p>",
		},
		{
			"onerror attribute",
			`<img src="x.png" onerror="alert(1)">`,
			[]string{"onerror", "alert"},
			`src="x.png"`,
		},
		{
			"onclick on span",
			`<span class="SCV_MIDA_M" onclick="steal()">badge</span>`,
			[]string{"onclick", "steal"},
			"badge",
		},
		{
			"javascript URL",
			`<a href="javascript:alert(1)">link</a>`,
			[]string{"javascript:", "href"},
			"link",
		},
		{
			"javascript URL with entities",
			`<a href="jav&#x09;ascript:alert(1)">link</a>`,
			[]string{"ascript:", "href"},
			"link",
		},
		{"data URL image", `<img src="data:text/html;base64,PHNjcmlwdD4=">`, []string{"data:"}, ""},
		{
			"style expression",
			`<span style="width: expression(alert(1)); color: red">x</span>`,
			[]string{"expression"},
			"color: red",
		},
		{
			"style url()",
			`<span style="background-color: url(javascript:alert(1))">x</span>`,
			[]string{"url(", "javascript"},
			"x",
		},
		{"font-family with url()", `<span style="font-family: url(x)">x</span>`, []string{"url("}, "x"},
		{
			"svg onload",
			`<svg onload="alert(1)"><circle r="1"/></svg>`,
			[]string{"<svg", "onload", "alert"},
			"",
		},
		{
			"iframe",
			`<iframe src="https://evil.example/"></iframe>after`,
			[]string{"<iframe", "evil"},
			"after",
		},
		{
			"style element",
			`<style>body{display:none}</style><p>x</p>`,
			[]string{"<style", "display"},
			"<p>x</p>",
		},
		{
			"object and embed",
			`<object data="x.swf"></object><embed src="x.swf">`,
			[]string{"<object", "<embed"},
			"",
		},
		{
			"form",
			`<form action="https://evil.example/"><input name="pw"></form>`,
			[]string{"<form", "<input"},
			"",
		},
		{
			"meta refresh",
			`<meta http-equiv="refresh" content="0;url=https://evil.example/">`,
			[]string{"<meta", "refresh"},
			"",
		},
		{
			"class injection",
			`<span class="x&quot; onmouseover=&quot;alert(1)">x</span>`,
			[]string{"onmouseover"},
			"x",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := s.HTML(tc.input)
			lower := strings.ToLower(got)
			for _, bad := range tc.forbidden {
				assert.NotContains(t, lower, bad, "output: %s", got)
			}
			assert.Contains(t, got, tc.kept)
		})
	}
}

// clone-spec A 4 の自由記述(大会 525 の構成)を模した HTML。人名は含めない。
const referenceLikeHTML = `<p><span class="SCV_MIDA_M" style="background-color: rgb(255, 102, 0); ` +
	`font-family: &quot;ヒラギノ角ゴ Pro W3&quot;, &quot;Hiragino Kaku Gothic Pro&quot;, Meiryo, sans-serif; ` +
	`font-size: 14px; color: rgb(255, 255, 255); padding: 4px 10px; border-radius: 4px; ` +
	`line-height: 250%; letter-spacing: 1px;">【Web Site】</span></p>` +
	`<p style="text-align: left; margin: 0px 0px 10px;"><a href="https://example.com/event" target="_blank">` +
	`https://example.com/event</a></p>` +
	`<p><span class="SCV_MIDA_M">【Result】</span><br>` +
	`<strong>■1st place：サンプルチーム</strong><br>` +
	`<img src="/zfiles/ScoreViewTop/result_1st.jpg" style="width: 600px; height: 400px;" alt="1st"></p>` +
	`<table border="1" cellpadding="2" width="100%"><tbody><tr>` +
	`<td align="center" colspan="2" style="background-color: #ffa000;">MVP</td></tr></tbody></table>`

func TestHTMLKeepsReferenceSiteMarkup(t *testing.T) {
	got := sanitize.New().HTML(referenceLikeHTML)
	for _, want := range []string{
		`class="SCV_MIDA_M"`,
		`background-color: rgb(255, 102, 0)`,
		`ヒラギノ角ゴ Pro W3`,
		`font-size: 14px`,
		`color: rgb(255, 255, 255)`,
		`padding: 4px 10px`,
		`border-radius: 4px`,
		`line-height: 250%`,
		`letter-spacing: 1px`,
		`text-align: left`,
		`margin: 0px 0px 10px`,
		`href="https://example.com/event"`,
		`target="_blank" rel="nofollow noopener"`,
		`<strong>■1st place：サンプルチーム</strong>`,
		`<br>`,
		`src="/zfiles/ScoreViewTop/result_1st.jpg"`,
		`width: 600px`,
		`height: 400px`,
		`<table border="1" cellpadding="2" width="100%">`,
		`<td align="center" colspan="2" style="background-color: #ffa000">MVP</td>`,
	} {
		assert.Contains(t, got, want)
	}
}

func TestHTMLEmptyInput(t *testing.T) {
	assert.Empty(t, sanitize.New().HTML(""))
}
