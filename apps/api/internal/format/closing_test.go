package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/format"
)

func fourOnFourPlan(t *testing.T) *format.Plan {
	t.Helper()
	return generate(t, format.TemplatePoolBands, format.PoolBandsParams{Teams: 12, Pools: 4, Bands: []format.Band{
		{Ranks: []int{1}, Shape: format.ShapeT4},
		{Ranks: []int{2}, Shape: format.ShapeT4},
		{Ranks: []int{3}, Shape: format.ShapeT4},
	}})
}

// 未消化で終わった試合から先で取りえた順位(大会の締めの同順位の提案。レビュー H10)。
func TestReachablePlaces(t *testing.T) {
	plan := fourOnFourPlan(t)
	// G は 9〜12 位の T4。G2 と G3 が準決勝、G1 が決勝、G4 が 3 位決定戦
	assert.Equal(t, []int{1, 2, 3, 4}, plan.ReachablePlaces("G2"))
	assert.Equal(t, []int{1, 2}, plan.ReachablePlaces("G1"))
	assert.Equal(t, []int{3, 4}, plan.ReachablePlaces("G4"))
	assert.Empty(t, plan.ReachablePlaces("A1"), "プールの試合は順位を決めない")
	assert.Empty(t, plan.ReachablePlaces("Z9"))
}

// 管理画面のグループ名は運営の言葉にする(グループ名のデータは変えない。レビュー M1)。
func TestOperatorNames(t *testing.T) {
	plan := fourOnFourPlan(t)
	assert.Equal(t, "予選 A 組", format.OperatorName(plan.Groups[0]))
	assert.Equal(t, "1〜4 位を決めるトーナメント", format.OperatorName(plan.Groups[4]))
	assert.Equal(t, "9〜12 位を決めるトーナメント", format.OperatorName(plan.Groups[6]))
	custom := plan.Groups[0]
	custom.Name = "本部の付けた名前"
	assert.Equal(t, "本部の付けた名前", format.OperatorName(custom))
}
