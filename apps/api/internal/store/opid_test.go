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

// 操作 ID の冪等判定は試合をまたがない。別の試合で使った操作 ID を送ったら、適用済みとして
// 黙って捨てずに 409 にする(セキュリティ点検 I-1)。
func TestOperationIDIsScopedToMatch(t *testing.T) {
	f := newMatchFixture(t)
	ctx := context.Background()
	other := storetest.New(t, f.db).Match(f.division, &f.left, &f.right,
		func(row *store.Match) { row.Code = "A2" })
	input := goal(domain.SideLeft, &f.l7, nil)
	input.ClientID = strPtr("6f1c2b3a-2222-4c2d-8e9f-000000000001")
	_, err := store.AddMatchEvent(ctx, f.db, staff, f.match.ID, input, t0)
	require.NoError(t, err)

	_, err = store.AddMatchEvent(ctx, f.db, staff, other.ID, goal(domain.SideLeft, nil, nil), t0)
	require.NoError(t, err)
	status := store.StatusInput{Status: domain.MatchFinished, ClientID: input.ClientID}
	_, err = store.ChangeMatchStatus(ctx, f.db, staff, other.ID, status)
	require.ErrorIs(t, err, domain.ErrConflict)
	detail, err := store.GetMatchDetail(ctx, f.db, other.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchLive, detail.Match.Status, "別の試合の操作は適用しない")

	// 同じ試合への送り直しは今までどおり冪等
	detail, err = store.AddMatchEvent(ctx, f.db, staff, f.match.ID, input, t0)
	require.NoError(t, err)
	assert.Len(t, detail.Events, 1)
}
