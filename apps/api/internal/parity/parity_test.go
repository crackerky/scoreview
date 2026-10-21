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

// 日程の検索フォームと列見出し: 参照サイトは JavaScript(AddHiddenVal、Action)、本サイトは href と
// <noscript> のボタンで同じことをする。選択肢の間の改行の有無も差にしない。
func TestScheduleFormAndSortLinksMatch(t *testing.T) {
	ref := `<div id="score_view"><table><tr>
<td><select name="team">
<option value="">	</option>
<option value="1">架空A</option>
<option value="2">架空 B</option>
</select></td>
<td class="RM"><a id='Refresh_OK' href="#" onclick="Action('search'); return false;">r</a></td>
</tr></table>
<table><tr><td class='MD1 CM'><a href="#" class="MD1"
 onclick="AddHiddenVal('sortkey1', 'ASC_mat.MATCH_DATE') ;  Action(&quot;sort&quot;)  ; return false; ">日付</a></td></tr></table>
</div>`
	ours := `<div id="score_view"><table><tr>
<td><select name="team"><option value=""></option><option value="1">架空A</option><option value="2">架空 B</option></select></td>
<td class="RM"><a id="Refresh_OK" href="/pscore/scv_search_div/open/1?lang=jp">r</a><noscript><input type="submit" value="検索"/></noscript></td>
</tr></table>
<table><tr><td class="MD1 CM"><a class="MD1" href="/pscore/scv_search_div/open/1?lang=jp&amp;sortkey1=ASC_mat.MATCH_DATE">日付</a></td></tr></table>
</div>`
	refPage, ourPage := extract(t, ref), extract(t, ours)
	res := parity.Compare("/x", "scv_search_div", refPage, ourPage)
	assert.True(t, res.FullMatch(), res.Diffs)
	assert.Equal(t, "架空A 架空 B", ourPage.Tables[0].Rows[0][0].Text)
	assert.Equal(t, "sort:ASC_mat.MATCH_DATE", refPage.Tables[1].Rows[0][0].Href)
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
