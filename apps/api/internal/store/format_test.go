package store_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

// formatFixture は 4on4 12 チームの大会に「4 プール(3 チーム)→ T4 × 3」を適用した状態。
type formatFixture struct {
	db       *gorm.DB
	division store.Division
	teams    []store.Team // シード順
	actor    domain.Actor
}

func pb4Input(seeds []int64) store.FormatInput {
	return store.FormatInput{
		Template: format.TemplatePoolBands,
		Params: format.PoolBandsParams{Teams: 12, Pools: 4, Bands: []format.Band{
			{Ranks: []int{1}, Shape: format.ShapeT4},
			{Ranks: []int{2}, Shape: format.ShapeT4},
			{Ranks: []int{3}, Shape: format.ShapeT4},
		}},
		Seeds:    seeds,
		Tiebreak: format.TiebreakCurrent,
		Forfeit:  format.DefaultForfeitScore(),
		Schedule: &store.ScheduleSetting{Date: "2026-10-18", Start: "09:00", SlotMinutes: 20,
			Courts: []string{"1", "2", "3", "4"}},
	}
}

func newFormatFixture(t *testing.T) formatFixture {
	t.Helper()
	db := testdb.New(t)
	b := storetest.New(t, db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	f := formatFixture{db: db, division: division}
	user := store.User{Email: "hq@example.com", PasswordHash: "x", Role: domain.RoleAdmin}
	require.NoError(t, db.Create(&user).Error)
	f.actor = domain.Actor{UserID: user.ID, Role: domain.RoleAdmin}
	seeds := make([]int64, 12)
	for i := range seeds {
		name := fmt.Sprintf("T%02d", i+1)
		team := b.Team(division, func(row *store.Team) { row.NameFullJa, row.NameShortJa = name, name })
		f.teams = append(f.teams, team)
		seeds[i] = team.ID
	}
	preview, err := store.ApplyFormat(context.Background(), db, division.ID, pb4Input(seeds))
	require.NoError(t, err)
	require.Len(t, preview.Matches, 24)
	return f
}

func (f formatFixture) match(t *testing.T, code string) store.Match {
	t.Helper()
	var rows []store.Match
	require.NoError(t, f.db.Where("division_id = ? AND code = ?", f.division.ID, code).Find(&rows).Error)
	require.Len(t, rows, 1, code)
	return rows[0]
}

func (f formatFixture) finish(t *testing.T, code string, left, right int) {
	t.Helper()
	_, err := store.ApplyCorrection(context.Background(), f.db, f.actor, f.match(t, code).ID,
		store.CorrectionInput{Score: domain.Score{Left: left, Right: right}}, true)
	require.NoError(t, err, code)
}

func (f formatFixture) groupID(t *testing.T, name string) int64 {
	t.Helper()
	var rows []store.Group
	require.NoError(t, f.db.Where("division_id = ? AND name = ?", f.division.ID, name).Find(&rows).Error)
	require.Len(t, rows, 1, name)
	return rows[0].ID
}

func teamOf(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

// finishPoolsBySlotOrder はプールの上の枠が下の枠に勝つ結果を入れる(順位 = 枠の順)。D は 3 チームとも
// 引き分けにして、全基準で並ぶ未決を作る。
func (f formatFixture) finishPools(t *testing.T) {
	t.Helper()
	for _, letter := range []string{"A", "B", "C"} {
		f.finish(t, letter+"1", 3, 1) // 1-2
		f.finish(t, letter+"2", 4, 1) // 1-3
		f.finish(t, letter+"3", 2, 1) // 2-3
	}
	for _, code := range []string{"D1", "D2", "D3"} {
		f.finish(t, code, 2, 2)
	}
}

func TestFormatScenarioPoolsToT4(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)

	// シードは蛇行で配る: A = 1, 8, 9。プールの試合は確定チームで作られる
	a1 := f.match(t, "A1")
	assert.Equal(t, f.teams[0].ID, teamOf(a1.LeftTeamID))
	assert.Equal(t, f.teams[7].ID, teamOf(a1.RightTeamID))
	require.NotNil(t, a1.StartTime)
	e2 := f.match(t, "E2")
	assert.Nil(t, e2.LeftTeamID, "プールの確定前は 1st placement の 1 回戦は空")

	f.finishPools(t)
	progress, err := store.GetFormatProgress(ctx, f.db, f.division.ID, now)
	require.NoError(t, err)
	kinds := map[store.TodoKind]int{}
	for _, item := range progress.Todo {
		kinds[item.Kind]++
	}
	assert.Equal(t, 3, kinds[store.TodoConfirmGroup])
	assert.Equal(t, 1, kinds[store.TodoTie])

	for _, name := range []string{"Qualifying pool A", "Qualifying pool B", "Qualifying pool C"} {
		require.NoError(t, store.ConfirmGroup(ctx, f.db, f.groupID(t, name), false, now))
	}
	poolD := f.groupID(t, "Qualifying pool D")
	err = store.ConfirmGroup(ctx, f.db, poolD, false, now)
	require.ErrorIs(t, err, domain.ErrConflict, "未決があるうちは確定できない")
	// D = シード 4, 5, 12。ブリックマーク投げで 12 → 4 → 5 の順にする
	d := []int64{f.teams[11].ID, f.teams[3].ID, f.teams[4].ID}
	require.ErrorIs(t, store.DecideRank(ctx, f.db, f.actor, poolD,
		store.RankDecisionInput{Teams: d[:2], Reason: "ブリックマーク投げ"}), domain.ErrBadRequest)
	require.NoError(t, store.DecideRank(ctx, f.db, f.actor, poolD,
		store.RankDecisionInput{Teams: d, Reason: "ブリックマーク投げ"}))
	require.NoError(t, store.ConfirmGroup(ctx, f.db, poolD, false, now))

	// 1st placement(E): A1 対 B1、C1 対 D1
	e2, e3 := f.match(t, "E2"), f.match(t, "E3")
	assert.Equal(t, []int64{f.teams[0].ID, f.teams[1].ID}, []int64{teamOf(e2.LeftTeamID), teamOf(e2.RightTeamID)})
	assert.Equal(t, []int64{f.teams[2].ID, f.teams[11].ID}, []int64{teamOf(e3.LeftTeamID), teamOf(e3.RightTeamID)})
	// 9th placement(G): A3 対 B3、C3 対 D3
	g2 := f.match(t, "G2")
	assert.Equal(t, []int64{f.teams[8].ID, f.teams[9].ID}, []int64{teamOf(g2.LeftTeamID), teamOf(g2.RightTeamID)})

	// 確定後の訂正: A3(A の 2 位と 3 位の試合)を逆の結果にすると 5th と 9th の 1 回戦が変わる
	impact, err := store.PreviewCorrection(ctx, f.db, f.match(t, "A3").ID,
		store.CorrectionInput{Score: domain.Score{Left: 1, Right: 2}})
	require.NoError(t, err)
	assert.False(t, impact.NeedsApproval)
	assert.Contains(t, strings.Join(impact.Lines, "\n"), "Qualifying pool A の 2 位と 3 位が入れ替わります")
	assert.Contains(t, strings.Join(impact.Lines, "\n"), "5th placement tournament の F2")

	// T4 を全部進める(上の枠が勝つ)
	for _, letter := range []string{"E", "F", "G"} {
		for _, code := range []string{"2", "3", "1", "4"} {
			f.finish(t, letter+code, 5, 3)
		}
	}
	var teams []store.Team
	require.NoError(t, f.db.Where("division_id = ?", f.division.ID).Order("final_rank").Find(&teams).Error)
	ranks := map[int]bool{}
	for _, team := range teams {
		require.NotNil(t, team.FinalRank, team.NameShortJa)
		ranks[*team.FinalRank] = true
	}
	assert.Len(t, ranks, 12, "1〜12 位がすべて決まる")
	assert.Equal(t, f.teams[0].ID, teams[0].ID, "シード 1 が優勝")

	// 結果のある F2 に影響する訂正は承認なしでは保存しない
	impact, err = store.PreviewCorrection(ctx, f.db, f.match(t, "A3").ID,
		store.CorrectionInput{Score: domain.Score{Left: 1, Right: 2}})
	require.NoError(t, err)
	assert.True(t, impact.NeedsApproval)
	_, err = store.ApplyCorrection(ctx, f.db, f.actor, f.match(t, "A3").ID,
		store.CorrectionInput{Score: domain.Score{Left: 1, Right: 2}}, false)
	require.ErrorIs(t, err, domain.ErrConflict)
	_, err = store.ApplyCorrection(ctx, f.db, f.actor, f.match(t, "A3").ID,
		store.CorrectionInput{Score: domain.Score{Left: 1, Right: 2}}, true)
	require.NoError(t, err)
	progress, err = store.GetFormatProgress(ctx, f.db, f.division.ID, now)
	require.NoError(t, err)
	var mismatches int
	for _, item := range progress.Todo {
		if item.Kind == store.TodoMismatch {
			mismatches++
		}
	}
	assert.Positive(t, mismatches, "始まった試合は差し替えず、食い違いとして出す")
}

func TestApplyFormatRefusesStartedDivision(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finish(t, "A1", 1, 0)
	seeds := make([]int64, 12)
	for i, team := range f.teams {
		seeds[i] = team.ID
	}
	_, err := store.ApplyFormat(ctx, f.db, f.division.ID, pb4Input(seeds))
	require.ErrorIs(t, err, domain.ErrConflict)
	_, err = store.PreviewFormat(ctx, f.db, f.division.ID, pb4Input(seeds[:11]))
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

func TestFormatScheduleShiftAndSwap(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	before := f.match(t, "A1")
	court := *before.Field
	_, err := store.ShiftFormatSchedule(ctx, f.db, f.division.ID, &court, 10)
	require.NoError(t, err)
	after := f.match(t, "A1")
	start, err := time.Parse("15:04:05", *before.StartTime)
	require.NoError(t, err)
	assert.Equal(t, start.Add(10*time.Minute).Format("15:04:05"), *after.StartTime)

	other := f.match(t, "B1")
	_, err = store.SwapMatchSchedule(ctx, f.db, after.ID, other.ID)
	require.NoError(t, err)
	assert.Equal(t, *other.StartTime, *f.match(t, "A1").StartTime)
	violations, err := store.CheckFormatSchedule(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	for _, v := range violations {
		assert.NotEmpty(t, v.Detail)
	}
}

func TestSecondDivisionContinuesGroupLetters(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	b := storetest.New(t, f.db)
	second := b.Division(f.division.TournamentID)
	seeds := make([]int64, 12)
	for i := range seeds {
		seeds[i] = b.Team(second).ID
	}
	preview, err := store.PreviewFormat(ctx, f.db, second.ID, pb4Input(seeds))
	require.NoError(t, err)
	assert.Equal(t, "H", preview.StartLetter, "A〜G の次から")
	saved, ok, err := store.GetSavedFormat(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Len(t, saved.Seeds, 12)
	list, err := store.ListSavedFormats(ctx, f.db, second.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
}
