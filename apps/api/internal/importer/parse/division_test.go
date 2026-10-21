package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParseDivisionStandings(t *testing.T) {
	d := parseBoth(t, "division_949_jp.html", "division_949_en.html",
		ParseDivisionStandings, MergeDivisionStandingsEN)

	require.Len(t, d.Rows, 4)
	assert.Equal(t, StandingRow{
		TeamID: 17396, ShortName: Text{JA: "マンマユート", EN: "Mamma Aiuto"},
		Games: 5, Wins: 5, Draws: 0, Losses: 0, GoalsFor: 38, GoalsAgainst: 19, GoalDiff: 19,
		WinPct: "100%", Rank: ptr(1),
	}, d.Rows[0])
	assert.Equal(t, -1, d.Rows[3].GoalDiff)

	require.Len(t, d.Leaders, 2)
	assert.Equal(t, ptr(int64(606311)), d.Leaders[0].PlayerID)
	assert.Equal(t, "87", d.Leaders[0].Number)
	assert.Equal(t, 17, d.Leaders[0].Total)
	assert.NotEmpty(t, d.Leaders[0].Player.JA)
	assert.NotEmpty(t, d.Leaders[0].Player.EN)
	require.Len(t, d.BlockLeaders, 2)
	assert.Equal(t, 9, d.BlockLeaders[0].Blocks)
}

// 954: 未開催のディビジョンは全行 0 と --%、順位は空(clone-spec C 1.3)
func TestParseDivisionStandingsWithoutRanks(t *testing.T) {
	d, err := ParseDivisionStandings(fixture(t, "division_954_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.Len(t, d.Rows, 2)
	for _, r := range d.Rows {
		assert.Nil(t, r.Rank)
		assert.Equal(t, "--%", r.WinPct)
	}
	assert.Empty(t, d.Leaders)
}
