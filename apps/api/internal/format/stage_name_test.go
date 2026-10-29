package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/format"
)

func stageNames(t *testing.T, plan *format.Plan, codes ...string) []string {
	t.Helper()
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		m, ok := plan.MatchByCode(code)
		require.True(t, ok, code)
		out = append(out, format.MatchStageName(plan, m))
	}
	return out
}

// 4 プール → T4 × 3(916): 順位が決まる試合は「何位決定戦」、1 回戦は決まりうる順位の範囲と回戦
func TestMatchStageNameT4Bands(t *testing.T) {
	plan := plan916(t)
	assert.Equal(t,
		[]string{"予選 A 組", "1〜4位 準決勝", "1位決定戦", "3位決定戦", "5〜8位 準決勝", "7位決定戦", "11位決定戦"},
		stageNames(t, plan, "A1", "E2", "E1", "E4", "F2", "F4", "G4"))
}

// 16 チーム → T8 × 2(435): 負けた側の試合も、その先で決まる順位の範囲で呼ぶ
func TestMatchStageNameT8Bands(t *testing.T) {
	plan := plan435(t)
	assert.Equal(t,
		[]string{"1〜8位 準々決勝", "1〜4位 準決勝", "5〜8位 準決勝", "5位決定戦", "9〜16位 準々決勝", "15位決定戦"},
		stageNames(t, plan, "E4", "E2", "E10", "E9", "F4", "F12"))
}
