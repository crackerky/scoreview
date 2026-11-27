package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/store"
	"scoreview/api/internal/testdb"
)

func TestLineStoreFriendsAndEvents(t *testing.T) {
	db := testdb.New(t)
	lines := store.NewLineStore(db)
	ctx := context.Background()
	at := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	first, err := lines.FirstSeen(ctx, "01A", at)
	require.NoError(t, err)
	assert.True(t, first)
	first, err = lines.FirstSeen(ctx, "01A", at)
	require.NoError(t, err)
	assert.False(t, first)
	require.NoError(t, lines.Forget(ctx, "01A"))
	first, err = lines.FirstSeen(ctx, "01A", at)
	require.NoError(t, err)
	assert.True(t, first, "a forgotten event is handled again when LINE redelivers it")

	require.NoError(t, lines.Unfollow(ctx, "U1", at))
	require.NoError(t, lines.Follow(ctx, "U1", at.Add(time.Hour)))
	id, err := lines.LineUser(ctx, "U1", at)
	require.NoError(t, err)
	again, err := lines.LineUser(ctx, "U1", at)
	require.NoError(t, err)
	assert.Equal(t, id, again)
	var row store.LineUser
	require.NoError(t, db.Where("id = ?", id).Take(&row).Error)
	assert.True(t, row.Friend)
	require.NotNil(t, row.UnfollowedAt)
	require.NoError(t, lines.Unfollow(ctx, "U1", at.Add(2*time.Hour)))
	require.NoError(t, db.Where("id = ?", id).Take(&row).Error)
	assert.False(t, row.Friend)
}

// 使い方の案内は同じ人に 1 日 1 回だけ。まだ記録の無い人(Webhook を受ける前からの友だち)も数える。
func TestLineStoreHelpDueOncePerInterval(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	lines := store.NewLineStore(db)
	at := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		at   time.Time
		want bool
	}{
		{at, true},
		{at.Add(time.Hour), false},
		{at.Add(24 * time.Hour), true},
		{at.Add(25 * time.Hour), false},
	} {
		due, err := lines.HelpDue(ctx, "Uhelp", tc.at, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, tc.want, due, tc.at)
	}
	other, err := lines.HelpDue(ctx, "Uother", at.Add(time.Hour), 24*time.Hour)
	require.NoError(t, err)
	assert.True(t, other, "each person is counted on their own")
}
