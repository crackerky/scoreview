package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParsePlayer(t *testing.T) {
	p := parseBoth(t, "player_604138_jp.html", "player_604138_en.html", ParsePlayer, MergePlayerEN)

	assert.Equal(t, int64(17491), p.TeamID)
	assert.Equal(t, Text{JA: "雨ニモマケズ", EN: "AmenimoMakezu"}, p.TeamShortName)
	assert.Equal(t, ptr(int64(604137)), p.PrevID)
	assert.Equal(t, ptr(int64(604139)), p.NextID)
	assert.False(t, p.IsStaff)
	assert.Equal(t, "86", p.Number)
	assert.NotEmpty(t, p.Name.JA)
	assert.NotEmpty(t, p.Name.EN)
	assert.Equal(t, "ultimate", p.StatsKind)
	assert.Equal(t, &PlayerStats{Games: 4, Assists: 5, Goals: 0, Total: 5, Blocks: 6}, p.Stats)
	require.Len(t, p.Events, 2)
	assert.Equal(t, PlayerEvent{
		Code: "A6", MonthDay: "10/18", OppTeamID: ptr(int64(17369)), ElapsedSec: 60, Kind: "goal",
		ScorerID: ptr(int64(604131)), AssistID: ptr(int64(604138)),
	}, p.Events[0])
}

func TestParsePlayerStaff(t *testing.T) {
	p, err := ParsePlayer(fixture(t, "player_608232_jp.html"), site.LangJP)
	require.NoError(t, err)
	assert.True(t, p.IsStaff)
	assert.Equal(t, Text{JA: "マネージャ"}, p.Role)
	assert.Equal(t, "", p.Number)
	assert.Nil(t, p.Stats, "スタッフページには成績がない")
	assert.Empty(t, p.Events)
}

func TestParsePlayerDiscGolf(t *testing.T) {
	p, err := ParsePlayer(fixture(t, "player_620269_jp.html"), site.LangJP)
	require.NoError(t, err)
	assert.Equal(t, "discgolf", p.StatsKind)
	assert.Nil(t, p.Stats)
	assert.Equal(t, Text{JA: "WTFR"}, p.TeamShortName, "選手ページの短縮名はさらに短い場合がある")
}
