package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
)

// 閲覧用の API は、非公開のグループと公開前の最終順位を ID の直接指定でも出さない
// (docs/requirements.md「権限モデル」。セキュリティ点検 L-6)。
func TestBrowseHidesUnpublished(t *testing.T) {
	db, b, sc := setup(t)
	ctx := context.Background()

	_, err := store.GetGroup(ctx, db, sc.Hidden.ID)
	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, domain.ResourceGroup, notFound.Resource)

	// 順位を付けたが公開していないディビジョン。行の順からも順位が分からない(成績の順になる)
	division := b.Division(sc.Tournament.ID)
	low := b.Team(division, func(row *store.Team) { row.FinalRank, row.SortOrder = ptr(2), 0 })
	high := b.Team(division, func(row *store.Team) { row.FinalRank, row.SortOrder = ptr(1), 1 })
	b.Match(division, &low, &high, storetest.Finished(5, 0))

	page, err := store.GetDivisionStandings(ctx, db, division.ID)
	require.NoError(t, err)
	require.Len(t, page.Standings, 2)
	assert.Equal(t, low.ID, page.Standings[0].Team.ID, "勝ち数の順")
	for _, row := range page.Standings {
		assert.Nil(t, row.Rank)
	}
	team, err := store.GetTeam(ctx, db, high.ID)
	require.NoError(t, err)
	assert.Nil(t, team.Rank)

	// 公開すると順位が出る
	require.NoError(t, db.Model(&store.Division{}).Where("id = ?", division.ID).
		Update("standings_published", true).Error)
	page, err = store.GetDivisionStandings(ctx, db, division.ID)
	require.NoError(t, err)
	assert.Equal(t, high.ID, page.Standings[0].Team.ID, "順位の順")
	assert.Equal(t, ptr(1), page.Standings[0].Rank)
	team, err = store.GetTeam(ctx, db, high.ID)
	require.NoError(t, err)
	assert.Equal(t, ptr(1), team.Rank)
}
