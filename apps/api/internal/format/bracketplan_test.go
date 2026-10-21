package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

func TestBracketPlanFor(t *testing.T) {
	cases := []struct {
		kind    format.BracketType
		teams   int
		matches int
	}{
		{format.BracketT2, 2, 1}, {format.BracketT4, 4, 4}, {format.BracketT7, 7, 9},
		{format.BracketT8, 8, 12}, {format.BracketT8Block, 8, 12}, {format.BracketT8R1, 8, 4},
		{format.BracketT12, 12, 6}, {format.BracketT16, 16, 20},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			plan, err := format.BracketPlanFor(tc.kind)
			require.NoError(t, err)
			require.NoError(t, plan.Layout.Validate())
			assert.Equal(t, tc.teams, plan.Teams)
			require.Len(t, plan.Matches, tc.matches)
			require.Len(t, plan.Games, tc.matches)
			seeds := map[int]int{}
			roles := map[int]map[domain.BracketSlotRole]bool{}
			for i, m := range plan.Matches {
				assert.Equal(t, i+1, m.Number, "試合の添字は番号順(決勝が 1)")
				for _, e := range []domain.PlannedEntrant{m.Left, m.Right} {
					if e.Seed != nil {
						seeds[*e.Seed]++
					} else {
						require.NotNil(t, e.From)
						assert.Less(t, e.From.Match, len(plan.Matches))
					}
				}
			}
			for seed := range tc.teams {
				assert.Equal(t, 1, seeds[seed], "seed %d", seed)
			}
			for _, s := range plan.Slots {
				if roles[s.Match] == nil {
					roles[s.Match] = map[domain.BracketSlotRole]bool{}
				}
				roles[s.Match][s.Role] = true
			}
			for i := range plan.Matches {
				assert.True(t, roles[i][domain.SlotLeft] && roles[i][domain.SlotRight], "left/right of match %d", i+1)
			}
			assert.True(t, roles[0][domain.SlotWinner], "決勝の勝者のセル")
			for _, row := range plan.Layout.Rows {
				for _, cell := range row {
					if cell.Kind == domain.BracketTeam {
						assert.Empty(t, cell.Text)
					}
				}
			}
		})
	}
	_, err := format.BracketPlanFor("T5")
	assert.ErrorIs(t, err, domain.ErrBadRequest)
}
