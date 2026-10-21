package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
)

func entries(ids ...int64) []domain.PoolEntry {
	out := make([]domain.PoolEntry, len(ids))
	for i, id := range ids {
		out[i] = domain.PoolEntry{TeamID: id}
	}
	return out
}

// ranks は entries と同じ順の順位。nil は 0 で表す。
func ranks(standings []domain.PoolStanding) []int {
	out := make([]int, len(standings))
	for i, s := range standings {
		if s.Rank != nil {
			out[i] = *s.Rank
		}
	}
	return out
}

func TestRankPool(t *testing.T) {
	cases := []struct {
		name    string
		entries []domain.PoolEntry
		results []domain.Result
		want    []int // entries と同じ順。0 は空欄
	}{
		{
			// clone-spec C 2.2 group 6511: 得失差は 2 が上(+9 対 +6)だが、直接対決で勝った 1 が 1 位。
			// 3 と 4 も同じ勝ち点 2 チームで、得失差は 3 が上だが直接対決の勝者 4 が上。
			name:    "6511 head-to-head beats goal difference",
			entries: entries(1, 2, 3, 4),
			results: []domain.Result{
				finished(1, 2, 5, 4), finished(1, 3, 3, 6), finished(1, 4, 9, 1),
				finished(2, 3, 8, 2), finished(2, 4, 7, 3), finished(3, 4, 2, 5),
			},
			want: []int{1, 2, 4, 3},
		},
		{
			// clone-spec C 2.2 group 6040: 1 と 2 は勝ち点も得失差(+2)も同じで、直接対決の勝者 2 が上。
			// 得点では 1 が上(17 対 14)なので、直接対決を見ていなければ逆になる。
			name:    "6040 head-to-head when points and goal difference tie",
			entries: entries(1, 2, 3, 4),
			results: []domain.Result{
				finished(1, 2, 4, 5), finished(1, 3, 7, 5), finished(1, 4, 6, 5),
				finished(2, 3, 6, 3), finished(2, 4, 3, 5), finished(3, 4, 6, 4),
			},
			want: []int{2, 1, 3, 4},
		},
		{
			// clone-spec C 2.2 group 6037: 1(+36)と 2(+42)は 2 勝 1 分で直接対決は 7-7。得失差で 2 が 1 位。
			name:    "6037 drawn head-to-head falls back to goal difference",
			entries: entries(1, 2, 3, 4),
			results: []domain.Result{
				finished(1, 2, 7, 7), finished(1, 3, 15, 0), finished(1, 4, 21, 0),
				finished(2, 3, 20, 0), finished(2, 4, 22, 0), finished(3, 4, 5, 3),
			},
			want: []int{2, 1, 3, 4},
		},
		{
			// 3 チーム同点は直接対決を飛ばす(推測)。得失差 B +2, C 0, A -2。
			name:    "three-way tie skips head-to-head",
			entries: entries(1, 2, 3),
			results: []domain.Result{finished(1, 2, 5, 4), finished(2, 3, 6, 3), finished(3, 1, 7, 4)},
			want:    []int{3, 1, 2},
		},
		{
			name:    "goals for breaks a tie after goal difference",
			entries: entries(1, 2, 3, 4),
			results: []domain.Result{
				finished(1, 3, 10, 5), finished(2, 4, 6, 1),
				finished(1, 4, 0, 1), finished(2, 3, 0, 1),
			},
			// 4 チームとも勝ち点 3。1 と 2 は得失差 +4 で並び得点 10 対 6、3 と 4 は -4 で並び 6 対 2
			want: []int{1, 2, 3, 4},
		},
		{
			name:    "complete tie shares the rank",
			entries: entries(1, 2),
			results: []domain.Result{finished(1, 2, 3, 3)},
			want:    []int{1, 1},
		},
		{
			name:    "no finished match leaves ranks blank",
			entries: entries(1, 2),
			results: []domain.Result{{LeftTeamID: 1, RightTeamID: 2, Status: domain.MatchScheduled}},
			want:    []int{0, 0},
		},
		{
			name: "rank_override wins and 0 means blank",
			entries: []domain.PoolEntry{
				{TeamID: 1, RankOverride: score(0)},
				{TeamID: 2, RankOverride: score(5)},
				{TeamID: 3},
			},
			results: []domain.Result{finished(1, 2, 9, 0), finished(1, 3, 9, 0), finished(2, 3, 9, 0)},
			want:    []int{0, 5, 3},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.RankPool(tc.entries, tc.results)
			assert.Equal(t, tc.want, ranks(got))
			for i, s := range got {
				assert.Equal(t, tc.entries[i].TeamID, s.TeamID, "rows keep entry order")
				assert.Equal(t, s.Tally.Points(), s.Points)
			}
		})
	}
}
