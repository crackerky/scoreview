package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/format"
)

func confirmedPools435(t *testing.T) (*format.Plan, format.State) {
	t.Helper()
	plan := plan435(t)
	st := newState(16)
	for _, m := range plan.Matches[:24] {
		st.Results[m.Code] = score(3, 1)
	}
	for g := range 4 {
		st.ConfirmedGroups[g] = true
	}
	return plan, st
}

func TestImpactOfCorrection(t *testing.T) {
	plan, st := confirmedPools435(t)
	st.Results["E4"] = score(3, 1) // A1(1) 対 C2(6) は既に終わっている

	// A1(シード 1 対 8)の結果を訂正し、8 が勝つ: プール A の 1 位と 2 位が入れ替わる
	impact, err := format.ImpactOfResult(plan, format.DefaultConfig(), st, "A1", score(1, 3))
	require.NoError(t, err)
	assert.Equal(t, []format.RankChange{
		{Group: 0, TeamID: 8, Before: 2, After: 1},
		{Group: 0, TeamID: 1, Before: 1, After: 2},
	}, impact.RankChanges)
	assert.Equal(t, []format.SlotChange{
		{Slot: format.SlotKey{Group: 4, Position: 1}, Before: 1, After: 8},
		{Slot: format.SlotKey{Group: 4, Position: 8}, Before: 8, After: 1},
	}, impact.SlotChanges)
	affected := map[string]format.AffectedMatch{}
	for _, a := range impact.AffectedMatches {
		affected[a.Code] = a
	}
	assert.Equal(t, format.AffectedMatch{Code: "E4", BeforeLeft: 1, BeforeRight: 6, AfterLeft: 8, AfterRight: 6,
		HasResult: true}, affected["E4"])
	assert.False(t, affected["E7"].HasResult)
	assert.Equal(t, int64(1), affected["E7"].AfterRight)
	assert.Contains(t, affected, "E2", "E4 の勝者が変わるので準決勝の左も変わる")
	assert.True(t, impact.NeedsApproval)
	// 元の State は変えない
	assert.Equal(t, score(3, 1), st.Results["A1"])
}

func TestImpactBeforeConfirmation(t *testing.T) {
	plan, st := confirmedPools435(t)
	st.ConfirmedGroups[0] = false
	impact, err := format.ImpactOfResult(plan, format.DefaultConfig(), st, "A1", score(1, 3))
	require.NoError(t, err)
	assert.Len(t, impact.RankChanges, 2)
	assert.Empty(t, impact.SlotChanges)
	assert.Empty(t, impact.AffectedMatches)
	assert.False(t, impact.NeedsApproval)
}
