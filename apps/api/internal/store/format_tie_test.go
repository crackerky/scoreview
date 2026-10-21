package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 未決の同順位(全基準で並び、運営者が決める)は、プールの全試合が終わった後だけ。指定は指定した
// 時点の成績に結び付け、訂正で成績が変われば使わずに未決へ戻す。

func TestTieIsOnlyDecidedAfterAllGames(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	poolD := f.groupID(t, "Qualifying pool D")
	d := []int64{f.teams[11].ID, f.teams[3].ID, f.teams[4].ID}
	assert.Empty(t, f.todo(t, store.TodoTie), "試合前の 0 勝 0 敗は未決ではない")
	err := store.DecideRank(ctx, f.db, f.actor, poolD, store.RankDecisionInput{Teams: d, Reason: "誤操作"})
	require.ErrorIs(t, err, domain.ErrConflict)

	f.finish(t, "D1", 2, 2)
	f.finish(t, "D2", 2, 2)
	assert.Empty(t, f.todo(t, store.TodoTie), "残り 1 試合の間も未決ではない")
	f.finish(t, "D3", 2, 2)
	require.Len(t, f.todo(t, store.TodoTie), 1)
	require.ErrorIs(t, store.ConfirmGroup(ctx, f.db, f.actor, poolD, false, fixtureNow()), domain.ErrConflict)
	require.ErrorIs(t, store.DecideRank(ctx, f.db, f.actor, poolD, store.RankDecisionInput{
		Teams: []int64{d[0], d[0], d[1]}, Reason: "重複"}), domain.ErrBadRequest)
	require.NoError(t, store.DecideRank(ctx, f.db, f.actor, poolD,
		store.RankDecisionInput{Teams: d, Reason: "ディスクフリップ"}))
	require.NoError(t, store.ConfirmGroup(ctx, f.db, f.actor, poolD, false, fixtureNow()))
	assert.Equal(t, f.teams[11].ID, teamOf(f.match(t, "E3").RightTeamID), "D の 1 位はシード 12")
}

// 指定した後に訂正で成績が変わると、指定は使わない(試合前や別の成績のときの指定が黙って使われない)。
func TestTieDecisionIsDroppedWhenResultsChange(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finishPools(t)
	f.confirmPools(t)
	require.Equal(t, f.teams[11].ID, teamOf(f.match(t, "E3").RightTeamID))

	// D1(シード 4 対 5)を 3-3 にすると、シード 12 は得点で 3 位、4 と 5 は全基準で並ぶ
	f.finish(t, "D1", 3, 3)
	ties := f.todo(t, store.TodoTie)
	require.Len(t, ties, 1, "確定済みのプールに新しい未決")
	assert.Nil(t, f.match(t, "E3").RightTeamID, "D の 1 位は未決なので 1st の枠は空")
	require.NoError(t, store.DecideRank(ctx, f.db, f.actor, f.groupID(t, "Qualifying pool D"),
		store.RankDecisionInput{Teams: []int64{f.teams[4].ID, f.teams[3].ID}, Reason: "ディスクフリップ"}))
	assert.Equal(t, f.teams[4].ID, teamOf(f.match(t, "E3").RightTeamID))
}

// basis の無い指定(この仕組みより前の行)は、どの成績への指定か分からないので使わない。
func TestTieDecisionWithoutBasisIsIgnored(t *testing.T) {
	f := newFormatFixture(t)
	f.finishPools(t)
	poolD := f.groupID(t, "Qualifying pool D")
	require.NoError(t, f.db.Create(&store.GroupRankDecision{GroupID: &poolD, Reason: "古い指定",
		TeamIDs: []byte(fmt.Sprintf("[%d,%d,%d]", f.teams[11].ID, f.teams[3].ID, f.teams[4].ID)),
	}).Error)
	assert.Len(t, f.todo(t, store.TodoTie), 1)
}
