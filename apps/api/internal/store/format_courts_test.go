package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/format"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
)

// コートは大会で共有する。同じ大会の 2 つ目のディビジョンの時間割は、1 つ目の試合が使うコートと
// 時刻を避けて作り、手で重ねたらどちらのディビジョンの検査にも出る。
func TestScheduleIsSharedAcrossDivisions(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	b := storetest.New(t, f.db)
	second := b.Division(f.division.TournamentID)
	seeds := make([]int64, 12)
	for i := range seeds {
		seeds[i] = b.Team(second).ID
	}
	_, err := store.ApplyFormat(ctx, f.db, f.actor, second.ID, pb4Input(seeds))
	require.NoError(t, err)

	var first, other []store.Match
	require.NoError(t, f.db.Where("division_id = ?", f.division.ID).Order("id").Find(&first).Error)
	require.NoError(t, f.db.Where("division_id = ?", second.ID).Order("id").Find(&other).Error)
	for _, a := range first {
		for _, o := range other {
			same := *a.Field == *o.Field && *a.StartTime == *o.StartTime && a.MatchDate.Equal(*o.MatchDate)
			assert.False(t, same, "%s と %s が同じコートの同じ時刻", a.Code, o.Code)
		}
	}
	for _, id := range []int64{f.division.ID, second.ID} {
		violations, err := store.CheckFormatSchedule(ctx, f.db, id)
		require.NoError(t, err)
		assert.Empty(t, violations)
	}

	// 2 つ目の最初の試合を、1 つ目の A1 と同じコートと時刻に動かす
	a1 := f.match(t, "A1")
	require.NoError(t, f.db.Model(&store.Match{}).Where("id = ?", other[0].ID).Updates(map[string]any{
		"field": *a1.Field, "start_time": *a1.StartTime, "match_date": *a1.MatchDate}).Error)
	for _, id := range []int64{f.division.ID, second.ID} {
		violations, err := store.CheckFormatSchedule(ctx, f.db, id)
		require.NoError(t, err)
		var conflicts []store.ScheduleViolation
		for _, v := range violations {
			if v.Kind == format.ViolationCourtConflict {
				conflicts = append(conflicts, v)
			}
		}
		require.Len(t, conflicts, 1, "division %d", id)
		assert.ElementsMatch(t, []string{"A1", other[0].Code}, conflicts[0].Codes)
	}
}
