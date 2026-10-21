package domain_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
)

func str(v string) *string { return &v }

func row(matchID int64, date, clock, field, left string) domain.ScheduleRow {
	r := domain.ScheduleRow{
		MatchRow: domain.MatchRow{ID: matchID, Date: str(date), Time: str(clock), Field: str(field)},
	}
	if left != "" {
		r.Left.Team = &domain.TeamRef{ID: matchID, Name: domain.I18nText{Ja: left}}
	}
	return r
}

func ids(rows []domain.ScheduleRow) []int64 {
	out := make([]int64, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}

func TestSortSchedule(t *testing.T) {
	base := func() []domain.ScheduleRow {
		return []domain.ScheduleRow{
			row(1, "2026-10-18", "10:00", "2", "beta"),
			row(2, "2026-10-17", "09:00", "1", "Alpha"),
			row(3, "2026-10-18", "09:00", "2", ""),
			row(4, "2026-10-18", "09:00", "1", "ARROWS"),
			{MatchRow: domain.MatchRow{ID: 5}}, // 日時未定
		}
	}
	cases := []struct {
		name string
		sort *domain.ScheduleSort
		want []int64
	}{
		{"default is date, time, field ascending with nulls last", nil, []int64{2, 4, 3, 1, 5}},
		{
			// 同じコートの中では既定の並び(日付、時刻)を保つ
			"field descending keeps default order inside ties",
			&domain.ScheduleSort{Key: domain.SortByField, Dir: domain.SortDesc},
			[]int64{5, 3, 1, 2, 4},
		},
		{
			// 大文字小文字を区別しない。未確定(チームなし)は先頭
			"left team ascending is case-insensitive",
			&domain.ScheduleSort{Key: domain.SortByLeftTeam, Dir: domain.SortAsc},
			[]int64{3, 5, 2, 4, 1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := base()
			domain.SortSchedule(rows, tc.sort)
			assert.Equal(t, tc.want, ids(rows))
		})
	}
}

func TestCompareFields(t *testing.T) {
	fields := []string{"B", "10", "A", "2", "01", "1"}
	slices.SortFunc(fields, domain.CompareFields)
	assert.Equal(t, []string{"01", "1", "2", "10", "A", "B"}, fields)
}

func TestSortMatchRowsUsesNaturalFieldOrder(t *testing.T) {
	rows := []domain.MatchRow{
		{ID: 1, Field: str("10")}, {ID: 2, Field: str("B")}, {ID: 3}, {ID: 4, Field: str("9")},
	}
	domain.SortMatchRows(rows)
	var got []int64
	for _, r := range rows {
		got = append(got, r.ID)
	}
	assert.Equal(t, []int64{4, 1, 2, 3}, got)
}

func TestIsScheduleSortKey(t *testing.T) {
	for _, key := range []string{"division", "date", "time", "field", "code", "leftTeam", "rightTeam"} {
		assert.True(t, domain.IsScheduleSortKey(key), key)
	}
	for _, key := range []string{"", "status", "match_date; DROP TABLE matches"} {
		assert.False(t, domain.IsScheduleSortKey(key), key)
	}
}

func TestNextMatch(t *testing.T) {
	match := func(id int64, status domain.MatchStatus, result domain.ResultType) domain.MatchRow {
		return domain.MatchRow{ID: id, Status: status, ResultType: result}
	}
	finished := match(1, domain.MatchFinished, domain.ResultNormal)
	waiting := match(2, domain.MatchScheduled, domain.ResultNormal)
	later := match(3, domain.MatchScheduled, domain.ResultNormal)
	live := match(4, domain.MatchLive, domain.ResultNormal)
	unplayed := match(5, domain.MatchScheduled, domain.ResultUnplayed)
	cases := []struct {
		name string
		rows []domain.MatchRow
		want int
	}{
		{"試合がない", nil, -1},
		{"全試合が終了", []domain.MatchRow{finished}, -1},
		{"最初の未試合", []domain.MatchRow{finished, waiting, later}, 1},
		{"試合中が未試合より優先", []domain.MatchRow{finished, waiting, live}, 2},
		{"未消化は次の試合にしない", []domain.MatchRow{unplayed, later}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, domain.NextMatch(tc.rows))
		})
	}
}
