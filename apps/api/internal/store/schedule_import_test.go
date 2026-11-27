package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

// 貼り付けた時間割は、部門の始まっていない試合に、全部を 1 回で書く(一部だけ書かない)。
func TestImportSchedule(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	other := b.Division(tour.ID)
	atami, ito, numazu := b.Team(division), b.Team(division), b.Team(division)
	first := b.Match(division, &atami, &ito)
	second := b.Match(division, &ito, &numazu)
	foreign := b.Match(other, nil, nil)
	ctx := context.Background()
	day := time.Date(2027, 2, 9, 0, 0, 0, 0, time.UTC)

	require.NoError(t, store.ImportSchedule(ctx, db, division.ID, []store.ScheduleItem{
		{MatchID: first.ID, Date: day, Time: "08:30", Field: " 3 "},
		{MatchID: second.ID, Date: day, Time: "09:20", Field: "1"},
	}))
	var got store.Match
	require.NoError(t, db.First(&got, first.ID).Error)
	assert.Equal(t, "08:30:00", *got.StartTime)
	assert.Equal(t, "3", *got.Field)
	assert.Equal(t, day, got.MatchDate.UTC())

	for name, items := range map[string][]store.ScheduleItem{
		"ほかの部門の試合": {{MatchID: foreign.ID, Date: day, Time: "08:30", Field: "1"}},
		"同じ試合を 2 回": {{MatchID: first.ID, Date: day, Time: "08:30", Field: "1"},
			{MatchID: first.ID, Date: day, Time: "09:20", Field: "2"}},
		"時刻の形":  {{MatchID: first.ID, Date: day, Time: "8時半", Field: "1"}},
		"コートが空": {{MatchID: first.ID, Date: day, Time: "08:30", Field: " "}},
	} {
		require.ErrorIs(t, store.ImportSchedule(ctx, db, division.ID, items), domain.ErrBadRequest, name)
	}

	// 始まった試合が 1 つでもあれば、何も書かない
	require.NoError(t, db.Model(&store.Match{}).Where("id = ?", second.ID).Update("status", "live").Error)
	err := store.ImportSchedule(ctx, db, division.ID, []store.ScheduleItem{
		{MatchID: first.ID, Date: day, Time: "10:10", Field: "2"},
		{MatchID: second.ID, Date: day, Time: "11:00", Field: "2"},
	})
	require.ErrorIs(t, err, domain.ErrConflict)
	require.NoError(t, db.First(&got, first.ID).Error)
	assert.Equal(t, "08:30:00", *got.StartTime, "nothing is written")
}
