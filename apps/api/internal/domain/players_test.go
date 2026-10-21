package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
)

func TestSortPlayers(t *testing.T) {
	cases := []struct {
		name  string
		input []domain.PlayerSortKey
		want  []string // 背番号の並び
	}{
		{
			name: "total, then goals, then number",
			input: []domain.PlayerSortKey{
				{Goals: 1, Assists: 1, Number: "7"},
				{Goals: 2, Assists: 0, Number: "9"},
				{Goals: 0, Assists: 5, Number: "3"},
				{Goals: 2, Assists: 0, Number: "4"},
			},
			want: []string{"3", "4", "9", "7"},
		},
		{
			// clone-spec D 1.1: 0 点だけなら背番号の数値昇順(#0,#3,#6,#9,#51,#90)
			name: "numbers compare numerically",
			input: []domain.PlayerSortKey{
				{Number: "90"}, {Number: "9"}, {Number: "51"}, {Number: "0"}, {Number: "6"}, {Number: "3"},
			},
			want: []string{"0", "3", "6", "9", "51", "90"},
		},
		{
			name:  "non-numeric numbers go last in string order",
			input: []domain.PlayerSortKey{{Number: "B"}, {Number: "12"}, {Number: ""}, {Number: "A"}, {Number: "2"}},
			want:  []string{"2", "12", "", "A", "B"},
		},
		{
			name:  "same numeric value falls back to string order",
			input: []domain.PlayerSortKey{{Number: "00"}, {Number: "0"}},
			want:  []string{"0", "00"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := append([]domain.PlayerSortKey(nil), tc.input...)
			domain.SortPlayers(items, func(k domain.PlayerSortKey) domain.PlayerSortKey { return k })
			got := make([]string, len(items))
			for i, it := range items {
				got[i] = it.Number
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
