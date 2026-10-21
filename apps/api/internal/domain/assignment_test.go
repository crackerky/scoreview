package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
)

func text(value string) *string { return &value }

func TestAssignmentCovers(t *testing.T) {
	match := domain.MatchSlot{ID: 7, Field: text("3"), Date: text("2026-10-18"), Time: text("10:30")}
	noTime := domain.MatchSlot{ID: 8, Field: text("3"), Date: text("2026-10-18")}
	matchID := int64(7)
	cases := []struct {
		name       string
		assignment domain.Assignment
		slot       domain.MatchSlot
		want       bool
	}{
		{"the whole tournament", domain.Assignment{}, match, true},
		{"the match itself", domain.Assignment{MatchID: &matchID}, match, true},
		{"another match", domain.Assignment{MatchID: &matchID}, noTime, false},
		{"same court", domain.Assignment{Field: text("3")}, match, true},
		{"other court", domain.Assignment{Field: text("4")}, match, false},
		{"other day", domain.Assignment{Date: text("2026-10-19")}, match, false},
		{
			"from is inclusive",
			domain.Assignment{Field: text("3"), From: text("10:30"), To: text("12:00")},
			match,
			true,
		},
		{"to is exclusive", domain.Assignment{From: text("09:00"), To: text("10:30")}, match, false},
		{"a time range does not cover a match without a time", domain.Assignment{
			From: text("09:00"),
		}, noTime, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.assignment.Covers(tc.slot))
		})
	}
	assert.True(t, domain.AnyCovers([]domain.Assignment{
		{Field: text("1")}, {Field: text("3")},
	}, match))
	assert.False(t, domain.AnyCovers(nil, match), "割り当てが無ければ担当なし")
}

func TestAssignmentValidate(t *testing.T) {
	matchID := int64(1)
	require.NoError(t, domain.Assignment{Field: text("A"), From: text("09:00")}.Validate())
	invalid := []domain.Assignment{
		{MatchID: &matchID, Field: text("1")},
		{Field: text(" ")},
		{Date: text("2026/10/18")},
		{From: text("9:00")},
		{From: text("12:00"), To: text("10:00")},
	}
	for _, assignment := range invalid {
		require.ErrorIs(t, assignment.Validate(), domain.ErrBadRequest, "%+v", assignment)
	}
}
