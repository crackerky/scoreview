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
