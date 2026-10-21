package parse

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParseTeam(t *testing.T) {
	tm := parseBoth(t, "team_17461_jp.html", "team_17461_en.html", ParseTeam, MergeTeamEN)

	assert.Equal(t, Text{JA: "くらんぬ", EN: "clan-nu"}, tm.Name)
	assert.Contains(t, tm.Comment.JA, "架空のチーム紹介")
	assert.Contains(t, tm.Comment.EN, "fictional team")

	require.NotNil(t, tm.Record)
	assert.Nil(t, tm.Record.Rank, "未試合のチームは順位が空")
	assert.Equal(t, "--%", tm.Record.WinPct)

	require.Len(t, tm.Matches, 3)
	for _, m := range tm.Matches {
		assert.Equal(t, "scheduled", m.Status)
		assert.Nil(t, m.MatchID, "未試合には試合ページへのリンクがない(clone-spec D 1.1)")
	}
	assert.Equal(t, ptr(int64(17579)), tm.Matches[0].Right.TeamID)

	require.Len(t, tm.Players, 3)
	p := tm.Players[0]
	assert.Equal(t, int64(608233), p.ID)
	assert.Equal(t, "0", p.Number)
	assert.Equal(t, ptr("M"), p.Gender)
	assert.NotEmpty(t, p.Name.JA)
	assert.NotEmpty(t, p.Name.EN)

	require.Len(t, tm.Staff, 1)
	assert.Equal(t, int64(608232), tm.Staff[0].ID)
	assert.Equal(t, Text{JA: "マネージャ", EN: "Manager"}, tm.Staff[0].Role)
	assert.Equal(t, ptr("F"), tm.Staff[0].Gender)
}

// ディスクゴルフ団体のチームページは成績表と対戦一覧がなく、選手一覧は # と選手だけ(clone-spec D 1.2)
// SOTG スコア一覧(clone-spec D 1.1 の 6): CD から始まる 8 列の表があれば SOTG=true。
func TestParseTeamSOTG(t *testing.T) {
	page := func(extra string) string {
		return `<html><body><div id="score_view"><span class='SCV_MIDA_M'>架空<br></span>` + extra +
			`</div></body></html>`
	}
	sotg := `<div class="TBL"><table><tr class=''>
<td class='MD1 CM'>CD</td><td class='MD1 CM'>相手チーム</td><td class='MD1 CM'>※１</td>
<td class='MD1 CM'>※２</td><td class='MD1 CM'>※３</td><td class='MD1 CM'>※４</td>
<td class='MD1 CM'>※５</td><td class='MD1 CM'>合計</td></tr></table></div>`
	with, err := ParseTeam(strings.NewReader(page(sotg)), site.LangJP)
	require.NoError(t, err)
	assert.True(t, with.SOTG)
	without, err := ParseTeam(strings.NewReader(page("")), site.LangJP)
	require.NoError(t, err)
	assert.False(t, without.SOTG)
}

func TestParseTeamDiscGolf(t *testing.T) {
	tm, err := ParseTeam(fixture(t, "team_17459_jp.html"), site.LangJP)
	require.NoError(t, err)
	assert.Nil(t, tm.Record)
	assert.Empty(t, tm.Matches)
	require.Len(t, tm.Players, 3)
	assert.Equal(t, TeamPlayer{ID: 620269, Number: "1", Name: tm.Players[0].Name, Gender: ptr("M")}, tm.Players[0])
}

// 2017 年: 性別なしの playerLink、SOTG スコア一覧(読まない)、終了試合のリンク
func TestParseTeamOldFormat(t *testing.T) {
	tm, err := ParseTeam(fixture(t, "team_4990_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotNil(t, tm.Record)
	assert.Equal(t, ptr(29), tm.Record.Rank)
	assert.Equal(t, "33.33%", tm.Record.WinPct)
	require.Len(t, tm.Matches, 3)
	assert.Equal(t, ptr(int64(14693)), tm.Matches[0].MatchID)
	require.NotEmpty(t, tm.Players)
	assert.Nil(t, tm.Players[0].Gender)
	assert.Equal(t, TeamPlayer{
		ID: 78602, Number: "8", Name: tm.Players[0].Name, Games: 6, Assists: 9, Goals: 13, Total: 22, Blocks: 3,
	}, tm.Players[0])
}

func TestMergeTeamRejectsMissingPlayer(t *testing.T) {
	jp, err := ParseTeam(fixture(t, "team_17461_jp.html"), site.LangJP)
	require.NoError(t, err)
	en, err := ParseTeam(fixture(t, "team_17461_en.html"), site.LangEN)
	require.NoError(t, err)
	en.Players[0].ID = 1
	_, err = MergeTeamEN(jp, en)
	assert.ErrorIs(t, err, ErrMismatch)
}
