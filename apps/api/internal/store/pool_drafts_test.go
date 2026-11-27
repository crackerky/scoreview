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

// 組分けの下書きは、部門のチームだけで、同じチームは 1 回だけ。途中(空の組)も残せ、置き換えられる。
func TestPoolDraft(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	other := b.Division(tour.ID)
	a, c, d := b.Team(division).ID, b.Team(division).ID, b.Team(division).ID
	stranger := b.Team(other).ID
	ctx := context.Background()
	at := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)

	_, _, found, err := store.GetPoolDraft(ctx, db, division.ID)
	require.NoError(t, err)
	assert.False(t, found)

	require.NoError(t, store.SavePoolDraft(ctx, db, division.ID, [][]int64{{a, c}, {}}, at))
	require.NoError(t, store.SavePoolDraft(ctx, db, division.ID, [][]int64{{a}, {c, d}}, at.Add(time.Minute)))
	pools, updated, found, err := store.GetPoolDraft(ctx, db, division.ID)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, [][]int64{{a}, {c, d}}, pools)
	assert.Equal(t, at.Add(time.Minute), updated.UTC())

	for name, bad := range map[string][][]int64{
		"同じチームを 2 つの組": {{a, c}, {c}},
		"ほかの部門のチーム":    {{a, stranger}},
		"組が無い":         {},
	} {
		require.ErrorIs(t, store.SavePoolDraft(ctx, db, division.ID, bad, at), domain.ErrBadRequest, name)
	}
	many := make([][]int64, 17)
	require.ErrorIs(t, store.SavePoolDraft(ctx, db, division.ID, many, at), domain.ErrBadRequest)
}
