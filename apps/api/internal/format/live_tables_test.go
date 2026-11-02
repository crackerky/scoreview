package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/format"
)

func seedNames(plan *format.Plan, group int) []string {
	out := []string{}
	for _, slot := range plan.Groups[group].Slots {
		out = append(out, format.SeedName(plan, slot.Source))
	}
	return out
}

// 4 プール → T4 × 3(916): 各組の 1 位だけが 1〜4 位のトーナメントへ進む
func TestLiveTablesT4Bands(t *testing.T) {
	plan := plan916(t)
	assert.Equal(t, 1, format.TopAdvance(plan, 0))
	assert.Equal(t, 1, format.TopAdvance(plan, 3))
	assert.Equal(t, 0, format.TopAdvance(plan, 4), "トーナメントは 0")
	assert.ElementsMatch(t, []string{"予選 A 組 1位", "予選 B 組 1位", "予選 C 組 1位", "予選 D 組 1位"},
		seedNames(plan, 4))
}

// 16 チーム → T8 × 2(435): 各組の上位 2 チームが 1〜8 位のトーナメントへ進む
func TestLiveTablesT8Bands(t *testing.T) {
	plan := plan435(t)
	assert.Equal(t, 2, format.TopAdvance(plan, 0))
	assert.Contains(t, seedNames(plan, 4), "予選 A 組 2位")
}
