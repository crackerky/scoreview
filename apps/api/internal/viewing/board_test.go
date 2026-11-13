package viewing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
	"scoreview/api/internal/viewing"
)

func ptr[T any](v T) *T { return &v }

func team(id int64, name string) domain.TeamRef {
	return domain.TeamRef{ID: id, Name: domain.I18nText{Ja: name}}
}

// match は部門 1 の試合。left/right の点が nil なら未試合。
func match(id int64, date, clock, field string, group *int64, left, right domain.TeamRef,
	scores ...int,
) viewing.Match {
	row := domain.MatchRow{ID: id, Date: &date, Time: &clock, Field: &field, Status: domain.MatchScheduled,
		Left: domain.MatchSide{Team: &left}, Right: domain.MatchSide{Team: &right}}
	if len(scores) == 2 {
		row.Status = domain.MatchFinished
		row.Left.Score, row.Right.Score = &scores[0], &scores[1]
	}
	return viewing.Match{ScheduleRow: domain.ScheduleRow{MatchRow: row,
		Division: domain.ScheduleDivision{ID: 1}}, GroupID: group}
}

func TestAssembleOrdersMatchesAndLinksOnlyShownGroups(t *testing.T) {
	alpha, bravo := team(1, "アルファ"), team(2, "ブラボー")
	pool, hidden := int64(10), int64(99)
	board := viewing.Assemble(viewing.Snapshot{
		Divisions: []viewing.Division{{ID: 1}},
		Groups:    []viewing.Group{{ID: pool, DivisionID: 1, Name: domain.I18nText{Ja: "A"}, Kind: domain.GroupPool}},
		Matches: []viewing.Match{
			match(3, "2026-11-02", "09:00", "1", &pool, alpha, bravo),
			match(2, "2026-11-01", "10:00", "B", &hidden, alpha, bravo),
			match(1, "2026-11-01", "10:00", "2", nil, alpha, bravo),
		},
	})

	var ids []int64
	for _, m := range board.Matches {
		ids = append(ids, m.ID)
	}
	// 日付、時刻、コート(数字のコートが先)の順
	assert.Equal(t, []int64{1, 2, 3}, ids)
	assert.Nil(t, board.Matches[1].Group, "a match of a group that is not shown is not linked")
	assert.Equal(t, &viewing.MatchGroup{ID: pool, Name: domain.I18nText{Ja: "A"}, Pool: true}, board.Matches[2].Group)
}

func TestAssembleRanksPoolsInEntryOrder(t *testing.T) {
	alpha, bravo, charlie := team(1, "アルファ"), team(2, "ブラボー"), team(3, "チャーリー")
	pool := int64(10)
	entries := []viewing.Entry{{Team: alpha}, {Team: bravo}, {Team: charlie}}
	first := match(1, "2026-11-01", "09:00", "1", &pool, alpha, bravo, 9, 3)
	first.Stage = "予選 A 組"
	board := viewing.Assemble(viewing.Snapshot{
		Divisions: []viewing.Division{{ID: 1, Name: domain.I18nText{Ja: "ミックス"}}, {ID: 2}},
		Groups: []viewing.Group{{ID: pool, DivisionID: 1, Name: domain.I18nText{Ja: "A", En: "Pool A"},
			Kind: domain.GroupPool, Advance: 2, Entries: entries}},
		Matches: []viewing.Match{
			first,
			match(2, "2026-11-01", "10:00", "1", &pool, charlie, alpha, 5, 5),
			match(3, "2026-11-01", "11:00", "1", &pool, bravo, charlie),
		},
	})

	require.Len(t, board.Divisions, 2, "a division with nothing to show is kept for the schedule")
	require.Len(t, board.Divisions[0].Pools, 1)
	assert.Empty(t, board.Divisions[1].Pools)
	got := board.Divisions[0].Pools[0]
	assert.Equal(t, domain.I18nText{Ja: "予選 A 組", En: "Pool A"}, got.Name, "the operator calls the pool by its match name")
	assert.False(t, got.Done, "one match is still to be played")
	assert.Equal(t, 2, got.Advance)
	require.Len(t, got.Standings, 3)
	// 登録順のまま、順位はアルファ(勝ち、分け)1、チャーリー(分け)2、ブラボー(負け)3
	assert.Equal(t, []int64{1, 2, 3}, []int64{got.Standings[0].Team.ID, got.Standings[1].Team.ID,
		got.Standings[2].Team.ID})
	assert.Equal(t, []*int{ptr(1), ptr(3), ptr(2)}, []*int{got.Standings[0].Rank, got.Standings[1].Rank,
		got.Standings[2].Rank})
	assert.Equal(t, 4, got.Standings[0].Points)
	assert.Equal(t, domain.Tally{Games: 2, Wins: 1, Draws: 1, GoalsFor: 14, GoalsAgainst: 8}, got.Standings[0].Tally)
}

func TestAssemblePoolIsDoneOnlyWithMatches(t *testing.T) {
	alpha, bravo := team(1, "アルファ"), team(2, "ブラボー")
	played, empty := int64(10), int64(11)
	board := viewing.Assemble(viewing.Snapshot{
		Divisions: []viewing.Division{{ID: 1}},
		Groups: []viewing.Group{
			{ID: played, DivisionID: 1, Kind: domain.GroupPool, Entries: []viewing.Entry{{Team: alpha}, {Team: bravo}}},
			{ID: empty, DivisionID: 1, Kind: domain.GroupPool},
		},
		Matches: []viewing.Match{match(1, "2026-11-01", "09:00", "1", &played, alpha, bravo, 1, 0)},
	})
	pools := board.Divisions[0].Pools
	require.Len(t, pools, 2)
	assert.True(t, pools[0].Done)
	assert.False(t, pools[1].Done)
	assert.Empty(t, pools[1].Standings)
}

func TestAssemblePutsBracketsUnderTheirDivision(t *testing.T) {
	semi := viewing.BracketMatch{Code: "E1", Round: 1, Left: viewing.BracketSide{Seed: "予選 A 組 1位"},
		Right: viewing.BracketSide{From: &format.MatchRef{Code: "D1", Outcome: domain.OutcomeWin}}}
	board := viewing.Assemble(viewing.Snapshot{
		Divisions: []viewing.Division{{ID: 1}, {ID: 2}},
		Groups:    []viewing.Group{{ID: 20, DivisionID: 2, Kind: domain.GroupBracket}},
		Brackets: []viewing.Bracket{
			{DivisionID: 2, GroupID: 20, Name: "1〜4 位を決めるトーナメント", Matches: []viewing.BracketMatch{semi}},
			{DivisionID: 7, GroupID: 70},
		},
	})
	assert.Empty(t, board.Divisions[0].Brackets)
	require.Len(t, board.Divisions[1].Brackets, 1, "a bracket of an unknown division is dropped")
	assert.Equal(t, semi, board.Divisions[1].Brackets[0].Matches[0])
	assert.Empty(t, board.Divisions[1].Pools, "a bracket group is not a pool")
}
