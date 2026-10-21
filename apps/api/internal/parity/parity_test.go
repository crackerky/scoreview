package parity_test

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/parity"
)

func extract(t *testing.T, html string) parity.Page {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)
	return parity.Extract(doc)
}

// 参照サイト風(onclick、絶対 URL、末尾空白、&nbsp;)と本サイト風(href、相対 URL)の同じ内容は一致する。
const referenceHTML = `<html><head><title>[CJS] 架空カップ -架空チーム-</title></head><body>
<ul id='score_navi'><li style=''><a href="https://score.clubjr.com/pscore/scv_top/open/1?lang=jp">大会トップ</a></li>
<li style="width:172px"><p>&nbsp;</p></li></ul>
<div id="score_view"><span class='SCV_MIDA_M'>架空チーム<br></span>
<div class="TBL"><table><tr><td class='MD1 CM' width= '9%' >試合数</td></tr>
<tr><td class='CM ' >17&nbsp; </td></tr>
<tr><td class='MC_END RM' ><span class='teamLink'><a href="#" onclick="document.location = 'https://score.clubjr.com/pscore/scv_team/open/5?lang=jp'; return false;">架空</a></span></td></tr>
</table></div>
<div class="TBL"><table><tr><td class='CM'>-対戦結果-</td><td>17:41:25 JST</td></tr></table></div>
</div></body></html>`

const oursHTML = `<html><head><title>[SCV] 架空カップ -架空チーム-</title></head><body>
<ul id="score_navi"><li><a href="/pscore/scv_top/open/1?lang=jp">大会トップ</a></li></ul>
<div id="score_view"><span class="SCV_MIDA_M">架空チーム<br/></span>
<div class="TBL"><table><tbody><tr><td class="MD1 CM" width="9%">試合数</td></tr>
<tr><td class="CM">17</td></tr>
<tr><td class="MC_END RM"><span class="teamLink"><a href="/pscore/scv_team/open/5?lang=jp">架空</a></span></td></tr>
</tbody></table></div>
<div class="TBL"><table><tr><td class="CM">-対戦結果-</td><td>09:00:01 JST</td></tr></table></div>
</div></body></html>`

func TestEquivalentPagesMatch(t *testing.T) {
	res := parity.Compare("/x", "scv_team", extract(t, referenceHTML), extract(t, oursHTML))
	assert.True(t, res.FullMatch(), res.Diffs)
	assert.Equal(t, 5, res.CellsTotal)
}

func TestDifferencesAreReported(t *testing.T) {
	changed := strings.Replace(oursHTML, `<td class="CM">17</td>`, `<td class="LM">18</td>`, 1)
	res := parity.Compare("/x", "scv_team", extract(t, referenceHTML), extract(t, changed))
	assert.False(t, res.FullMatch())
	assert.Equal(t, 4, res.CellsText)
	assert.Equal(t, 4, res.CellsClasses)
	assert.Len(t, res.Diffs, 2)
}

func TestNormalizeHref(t *testing.T) {
	cases := map[string]string{
		"https://score.clubjr.com/pscore/scv_team/open/5?lang=en": "/pscore/scv_team/open/5?lang=en",
		"/pscore/scv_team/open/5":                                 "/pscore/scv_team/open/5?lang=jp",
		"/pscore/scv_match/open/9?lang=jp&refpos=1":               "/pscore/scv_match/open/9?lang=jp",
		"#": "",
	}
	for in, want := range cases {
		assert.Equal(t, want, parity.NormalizeHref(in), in)
	}
}

func TestKind(t *testing.T) {
	assert.Equal(t, "scv_team", parity.Kind("/pscore/scv_team/open/5"))
	assert.Equal(t, "scview", parity.Kind("/scview/525"))
}
