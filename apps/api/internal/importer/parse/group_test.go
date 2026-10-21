package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParseGroupBracketT8A(t *testing.T) {
	g := parseBoth(t, "group_6623_jp.html", "group_6623_en.html", ParseGroup, MergeGroupEN)

	assert.Equal(t, "bracket", g.Kind)
	assert.Equal(t, "Tounament A", g.Name, "誤綴りを原文のまま保つ")
	assert.Equal(t, Text{JA: "ビーチミックス", EN: "Beach Mixed"}, g.DivisionName)
	require.NotNil(t, g.Bracket)
	assert.Equal(t, "T8A", g.Bracket.Pattern)
	assert.Equal(t, "2026 Atami Line =Tournament A=", g.Bracket.PtnName)
	assert.Equal(t, 4, g.Bracket.Columns)
	assert.Len(t, g.Bracket.Rows, 40)

	assert.Equal(t, BracketCell{Kind: "txt_sharp", Text: Text{JA: "Placement", EN: "Placement"}}, g.Bracket.Rows[0][3])
	assert.Equal(t, BracketCell{
		Bottom: true, Kind: "team", Text: Text{JA: "とびたーず", EN: "Tobitazu"}, TeamID: ptr(int64(17322)),
	}, g.Bracket.Rows[1][0])
	assert.Equal(t, BracketCell{
		Right: true, Top: true, Kind: "game", Text: Text{JA: "2-12", EN: "2-12"}, MatchID: ptr(int64(45280)),
	}, g.Bracket.Rows[2][0])
	assert.Equal(t, BracketCell{Text: Text{}}, g.Bracket.Rows[1][1], "空セルは &nbsp; だけ")

	require.Len(t, g.Matches, 3)
	assert.Equal(t, "A4", g.Matches[0].Code)
	assert.Nil(t, g.Matches[0].Date, "グループの結果一覧には年がない")
	assert.Equal(t, "10/18", g.Matches[0].MonthDay)
	assert.Nil(t, g.Pool)
	assert.Nil(t, g.Image)
}

func TestParseGroupBracketT4E(t *testing.T) {
	g, err := ParseGroup(fixture(t, "group_6627_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotNil(t, g.Bracket)
	assert.Equal(t, "T4E", g.Bracket.Pattern)
	assert.Equal(t, 3, g.Bracket.Columns)
	require.Len(t, g.Bracket.Rows, 15)
	// 勝者は次の列に win で再掲され、罫線は左と下(clone-spec C 2.3)
	win := g.Bracket.Rows[2][1]
	assert.Equal(t, "win", win.Kind)
	assert.True(t, win.Left)
	assert.True(t, win.Bottom)
	assert.Equal(t, ptr(int64(17498)), win.TeamID)
	assert.Equal(t, "1st", g.Bracket.Rows[5][2].Text.JA)
	assert.Equal(t, "lose", g.Bracket.Rows[7][2].Kind)
	assert.Len(t, g.Matches, 4)
}

func TestParseGroupBracketT12H(t *testing.T) {
	g, err := ParseGroup(fixture(t, "group_6630_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotNil(t, g.Bracket)
	assert.Equal(t, "T12H", g.Bracket.Pattern)
	assert.Equal(t, 4, g.Bracket.Columns)
	assert.Equal(t, BracketCell{Kind: "txt_and", Text: Text{JA: "21th - 23th"}}, g.Bracket.Rows[1][2])
}

func TestParseGroupPool(t *testing.T) {
	g := parseBoth(t, "group_5895_jp.html", "group_5895_en.html", ParseGroup, MergeGroupEN)

	assert.Equal(t, "pool", g.Kind)
	assert.Equal(t, "Qualifying pool A", g.Name)
	assert.Equal(t, Text{JA: "４人制部門", EN: "4 on 4"}, g.DivisionName)
	require.NotNil(t, g.Pool)
	require.Len(t, g.Pool.Teams, 4)
	// 行はチームの登録順で、順位順ではない(clone-spec C 2.1(b))
	ranks := []string{}
	for _, tm := range g.Pool.Teams {
		ranks = append(ranks, tm.RankRaw)
	}
	assert.Equal(t, []string{"1", "3", "2", "4"}, ranks)
	assert.Equal(t, PoolTeam{
		TeamID: ptr(int64(15396)), ShortName: Text{JA: "居酒屋　司", EN: "izakaya tsukasa"},
		Games: 3, Wins: 3, GoalsFor: 26, GoalsAgainst: 7, GoalDiff: 19, RankRaw: "1", Rank: ptr(1),
	}, g.Pool.Teams[0])
	require.Len(t, g.Pool.Cells, 6)
	assert.Equal(t, PoolCell{Row: 1, Col: 2, MatchID: ptr(int64(40614)), Score: "4-5"}, g.Pool.Cells[3])
	assert.Len(t, g.Matches, 2)
}

// 2017 年の 5 チーム総当たり。マトリクスの列見出しにリンクがない
func TestParseGroupPoolFiveTeams(t *testing.T) {
	g, err := ParseGroup(fixture(t, "group_2182_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotNil(t, g.Pool)
	require.Len(t, g.Pool.Teams, 5)
	for _, tm := range g.Pool.Teams {
		assert.NotNil(t, tm.TeamID, "リーグ結果の行にはリンクがある")
	}
	assert.Len(t, g.Pool.Cells, 10, "5 チームの上三角")
	assert.Equal(t, "MAリーグ", g.Name)
}

// 2017 年の 2013: マトリクスのセルに試合リンクがない(clone-spec E 6)
func TestParseGroupPoolWithoutMatchLinks(t *testing.T) {
	g, err := ParseGroup(fixture(t, "group_2013_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotNil(t, g.Pool)
	require.NotEmpty(t, g.Pool.Cells)
	for _, c := range g.Pool.Cells {
		assert.Nil(t, c.MatchID)
		assert.NotEmpty(t, c.Score)
	}
}

func TestParseGroupImage(t *testing.T) {
	g, err := ParseGroup(fixture(t, "group_6539_jp.html"), site.LangJP)
	require.NoError(t, err)
	assert.Equal(t, "image", g.Kind)
	assert.Equal(t, "Temporary ranking", g.Name)
	assert.Equal(t, &GroupImage{
		URL:   "https://score.clubjr.com/zfiles/ScoreViewGrp/event_2026greatest_east_open.jpg",
		Width: ptr(500), Height: ptr(695),
	}, g.Image)
	assert.Empty(t, g.Matches)
}
