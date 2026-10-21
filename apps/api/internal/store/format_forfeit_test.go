package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 不戦勝のスコアは、記録(SetMatchResult)と順位計算が同じ値を使う(規定スコアは大会、無ければ
// 大会の形の設定)。順位計算は記録したスコアで数えるので、閲覧ページの成績表と順位が食い違わない。

func TestForfeitIsRecordedWithTheFormatRegulation(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t) // 大会に規定スコアは無く、形は 15-0
	left := domain.SideLeft
	a1 := f.match(t, "A1").ID
	_, err := store.SetMatchResult(ctx, f.db, f.actor, a1, store.ResultInput{
		Type: domain.ResultForfeit, WinnerSide: &left, ForfeitScore: &domain.ForfeitScore{Winner: 7, Loser: 0}})
	require.ErrorIs(t, err, domain.ErrBadRequest, "形の規定(15-0)と違うスコアでは記録しない")
	detail, err := store.SetMatchResult(ctx, f.db, f.actor, a1,
		store.ResultInput{Type: domain.ResultForfeit, WinnerSide: &left})
	require.NoError(t, err)
	assert.Equal(t, &domain.ForfeitScore{Winner: 15, Loser: 0}, detail.ForfeitScore)
	stored := f.match(t, "A1")
	require.NotNil(t, stored.LeftScore)
	assert.Equal(t, []int{15, 0}, []int{*stored.LeftScore, *stored.RightScore})
}

func TestForfeitTournamentRegulationComesFirst(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	require.NoError(t, f.db.Model(&store.Tournament{}).Where("id = ?", f.division.TournamentID).
		Updates(map[string]any{"forfeit_winner_score": 10, "forfeit_loser_score": 0}).Error)
	right := domain.SideRight
	_, err := store.SetMatchResult(ctx, f.db, f.actor, f.match(t, "A1").ID,
		store.ResultInput{Type: domain.ResultForfeit, WinnerSide: &right})
	require.NoError(t, err)
	stored := f.match(t, "A1")
	assert.Equal(t, []int{0, 10}, []int{*stored.LeftScore, *stored.RightScore})
}

// 規定と違うスコアで記録された不戦勝(規定を後から変えた等)も、順位は記録したスコアで数える。
// 3 チームとも 1 勝で並び、得失点差で決まる: 記録した 7-0 なら シード 1 は -3 で 3 位(規定の 15-0 で
// 数えると +5 で 1 位になり、閲覧ページの得失点差と順位が食い違う)。
func TestForfeitRankUsesTheRecordedScore(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	left := domain.SideLeft
	a1 := f.match(t, "A1").ID
	_, err := store.SetMatchResult(ctx, f.db, f.actor, a1,
		store.ResultInput{Type: domain.ResultForfeit, WinnerSide: &left})
	require.NoError(t, err)
	require.NoError(t, f.db.Model(&store.Match{}).Where("id = ?", a1).
		Updates(map[string]any{"left_score": 7, "right_score": 0}).Error)
	f.finish(t, "A2", 0, 10) // 枠 3 が枠 1 に勝つ
	f.finish(t, "A3", 10, 0) // 枠 2 が枠 3 に勝つ

	page, err := store.GetGroup(ctx, f.db, f.groupID(t, "Qualifying pool A"))
	require.NoError(t, err)
	require.NotNil(t, page.Pool)
	diffs := map[int64]int{}
	ranks := map[int64]int{}
	for _, r := range page.Pool.Results {
		require.NotNil(t, r.Rank)
		diffs[r.Team.ID], ranks[r.Team.ID] = r.Tally.GoalDiff(), *r.Rank
	}
	assert.Equal(t, -3, diffs[f.teams[0].ID])
	assert.Equal(t, 3, ranks[f.teams[0].ID], "得失点差が最も小さいシード 1 は 3 位")
	assert.Equal(t, 1, ranks[f.teams[7].ID])
	assert.Equal(t, 2, ranks[f.teams[8].ID])
}
