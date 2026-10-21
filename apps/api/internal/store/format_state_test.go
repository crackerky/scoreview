package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 保存した枠の出どころが、作り直した形と食い違えば(エンジンの生成が変わった等)、黙って別の形で
// 進めずに 409 にする。
func TestStoredSlotSourcesMustMatchThePlan(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	_, err := store.GetFormatProgress(ctx, f.db, f.division.ID, fixtureNow())
	require.NoError(t, err, "適用した直後は一致する")

	e := f.groupID(t, "1st placement tournament")
	require.NoError(t, f.db.Model(&store.GroupSlot{}).Where("group_id = ? AND position = 1", e).
		Update("source", []byte(`{"rank": 2, "kind": "group_rank", "group": 0}`)).Error)
	_, err = store.GetFormatProgress(ctx, f.db, f.division.ID, fixtureNow())
	require.ErrorIs(t, err, domain.ErrConflict)
}

// 先に「未消化のまま確定」したプールのある段を確定すると、段にも未消化の印が付く。
func TestStageConfirmKeepsTheUnplayedMark(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finishPools(t)
	// A の最後の試合を試合中に戻し、時間切れとして A を未消化のまま確定する
	_, err := store.ChangeMatchStatus(ctx, f.db, f.actor, f.match(t, "A3").ID,
		store.StatusInput{Status: domain.MatchLive})
	require.NoError(t, err)
	poolA := f.groupID(t, "Qualifying pool A")
	require.NoError(t, store.ConfirmGroup(ctx, f.db, f.actor, poolA, true, fixtureNow()))
	f.confirmPools(t) // A は確定済みなので何もしない
	var stage store.Stage
	require.NoError(t, f.db.Where("division_id = ? AND stage_index = 0", f.division.ID).First(&stage).Error)
	require.NoError(t, store.ConfirmStage(ctx, f.db, f.actor, stage.ID, false, fixtureNow()))
	require.NoError(t, f.db.Where("id = ?", stage.ID).First(&stage).Error)
	assert.True(t, stage.ConfirmedUnplayed)
}

func TestScheduleEndMustBeAfterStart(t *testing.T) {
	f := newFormatFixture(t)
	seeds := make([]int64, 12)
	for i, team := range f.teams {
		seeds[i] = team.ID
	}
	input := pb4Input(seeds)
	end := "08:00"
	input.Schedule.End = &end
	_, err := store.PreviewFormat(context.Background(), f.db, f.division.ID, input)
	require.ErrorIs(t, err, domain.ErrBadRequest)
}
