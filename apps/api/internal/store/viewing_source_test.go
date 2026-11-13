package store_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/viewing"
)

func TestViewingSourceReadsPublicRecords(t *testing.T) {
	db, _, sc := setup(t)
	board, err := viewing.NewService(store.NewViewingSource(db)).Board(context.Background(), sc.Tournament.ID)
	require.NoError(t, err)

	assert.Equal(t, sc.Tournament.ID, board.Tournament.ID)
	require.NotNil(t, board.Tournament.Banner)
	assert.Equal(t, "/b1.png", board.Tournament.Banner.ImageURL, "the first banner by sort order")
	require.Len(t, board.Divisions, 2)
	assert.Equal(t, []int64{sc.Mixed.ID, sc.Women.ID}, []int64{board.Divisions[0].ID, board.Divisions[1].ID})

	// 案内するのは公開して名前にリンクのある組だけ(非公開の組と、名前だけのトーナメントは出さない)
	require.Len(t, board.Divisions[0].Pools, 1)
	pool := board.Divisions[0].Pools[0]
	assert.Equal(t, sc.Pool.ID, pool.GroupID)
	require.Len(t, pool.Standings, 3)
	assert.Equal(t, []int64{sc.TeamA.ID, sc.TeamB.ID, sc.TeamC.ID},
		[]int64{pool.Standings[0].Team.ID, pool.Standings[1].Team.ID, pool.Standings[2].Team.ID})
	require.NotNil(t, pool.Standings[0].Rank)
	assert.Equal(t, 1, *pool.Standings[0].Rank)

	require.Len(t, board.Matches, 5)
	byID := map[int64]viewing.BoardMatch{}
	for _, m := range board.Matches {
		byID[m.ID] = m
	}
	require.NotNil(t, byID[sc.MatchAB.ID].Group)
	assert.Equal(t, sc.Pool.ID, byID[sc.MatchAB.ID].Group.ID)
	assert.True(t, byID[sc.MatchAB.ID].Group.Pool)
	assert.Nil(t, byID[sc.MatchFinal.ID].Group, "the bracket group is shown by name only")
	assert.Equal(t, sc.Women.ID, byID[sc.MatchWomen.ID].Division.ID)
	require.NotNil(t, byID[sc.MatchFinal.ID].Right.Placeholder)
	assert.Equal(t, "A1", byID[sc.MatchFinal.ID].Right.Placeholder.Code)
	assert.True(t, board.Tables.Shown, "the default table config")
}

func TestViewingSourceNotFound(t *testing.T) {
	db, _, _ := setup(t)
	_, err := store.NewViewingSource(db).Snapshot(context.Background(), 999999)
	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestViewingSourceQueryCountDoesNotGrow(t *testing.T) {
	db, b, sc := setup(t)
	source := store.NewViewingSource(db)
	count := countQueries(t, db)
	ctx := context.Background()

	count.Store(0)
	_, err := source.Snapshot(ctx, sc.Tournament.ID)
	require.NoError(t, err)
	before := count.Load()

	// 組とチームと試合を増やしても問い合わせの数は変わらない
	for i := range 20 {
		group := b.Group(sc.Mixed.ID, domain.GroupPool, func(row *store.Group) { row.SortOrder = 10 + i })
		left, right := b.Team(sc.Mixed), b.Team(sc.Mixed)
		b.GroupTeam(group.ID, left.ID)
		b.GroupTeam(group.ID, right.ID)
		b.Match(sc.Mixed, &left, &right, storetest.Finished(i%5, 2),
			storetest.At(t, "2026-10-19", "08:00", strconv.Itoa(1+i%10)),
			func(row *store.Match) { row.GroupID = &group.ID })
	}
	count.Store(0)
	snapshot, err := source.Snapshot(ctx, sc.Tournament.ID)
	require.NoError(t, err)
	assert.Len(t, snapshot.Matches, 25)
	assert.Equal(t, before, count.Load())
	assert.Equal(t, int64(14), before)
}
