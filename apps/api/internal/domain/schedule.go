package domain

import (
	"cmp"
	"slices"
	"strings"
)

// ScheduleSortKey はスケジュールの並び替えの列。
type ScheduleSortKey string

// ScheduleSortKey の値。参照サイトの sortkey1(clone-spec B 2)に対応する。
const (
	SortByDivision  ScheduleSortKey = "division"
	SortByDate      ScheduleSortKey = "date"
	SortByTime      ScheduleSortKey = "time"
	SortByField     ScheduleSortKey = "field"
	SortByCode      ScheduleSortKey = "code"
	SortByLeftTeam  ScheduleSortKey = "leftTeam"
	SortByRightTeam ScheduleSortKey = "rightTeam"
)

// SortDir は並び替えの向き。
type SortDir string

// SortDir の値。
const (
	SortAsc  SortDir = "asc"
	SortDesc SortDir = "desc"
)

// ScheduleDivision は日程の行が属するディビジョン。Order はディビジョンの並び順(sort_order, id の順位)。
type ScheduleDivision struct {
	ID    int64
	Name  I18nText
	Kind  DivisionKind
	Order int
}

// ScheduleRow はスケジュールの 1 行。
type ScheduleRow struct {
	MatchRow
	Division ScheduleDivision
}

// ScheduleSort は利用者が選んだ並び替え。
type ScheduleSort struct {
	Key ScheduleSortKey
	Dir SortDir
}

// scheduleComparators は並び替えの列ごとの比較関数の許可リスト。値が無いもの(日付未定など)は
// 昇順で後ろに置く(PostgreSQL の ORDER BY ... ASC と同じ NULLS LAST)。
func scheduleComparators() map[ScheduleSortKey]func(a, b ScheduleRow) int {
	return map[ScheduleSortKey]func(a, b ScheduleRow) int{
		SortByDivision: func(a, b ScheduleRow) int { return cmp.Compare(a.Division.Order, b.Division.Order) },
		SortByDate:     func(a, b ScheduleRow) int { return compareNullable(a.Date, b.Date) },
		SortByTime:     func(a, b ScheduleRow) int { return compareNullable(a.Time, b.Time) },
		SortByField:    func(a, b ScheduleRow) int { return compareNullable(a.Field, b.Field) },
		SortByCode:     func(a, b ScheduleRow) int { return cmp.Compare(a.Code, b.Code) },
		SortByLeftTeam: func(a, b ScheduleRow) int {
			return cmp.Compare(teamSortName(a.Left), teamSortName(b.Left))
		},
		SortByRightTeam: func(a, b ScheduleRow) int {
			return cmp.Compare(teamSortName(a.Right), teamSortName(b.Right))
		},
	}
}

// IsScheduleSortKey は key が並び替えに使える列かどうかを返す。
func IsScheduleSortKey(key string) bool {
	_, ok := scheduleComparators()[ScheduleSortKey(key)]
	return ok
}

// teamSortName はチーム名の並び替えに使う値。短縮名(日本語)を大文字小文字を区別せずに比べる
// (clone-spec B 2)。未確定の側は空文字にして先頭に寄せる(参照サイトで位置は未確認で推測)。
func teamSortName(s MatchSide) string {
	if s.Team == nil {
		return ""
	}
	return strings.ToLower(s.Team.Name.Ja)
}

func compareNullable[T cmp.Ordered](a, b *T) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	return cmp.Compare(*a, *b)
}

// compareScheduleDefault は既定の並び(日付、時刻、コート番号の昇順。clone-spec B 3)。
// 最後に試合 ID で比べて順序を決定的にする。
func compareScheduleDefault(a, b ScheduleRow) int {
	if c := compareNullable(a.Date, b.Date); c != 0 {
		return c
	}
	if c := compareNullable(a.Time, b.Time); c != 0 {
		return c
	}
	if c := compareNullable(a.Field, b.Field); c != 0 {
		return c
	}
	return cmp.Compare(a.ID, b.ID)
}

// SortSchedule は rows を既定の並びにし、sort があればその列で安定ソートする
// (同じ値の中では既定の並びを保つ。clone-spec B 2)。sort.Key は IsScheduleSortKey で検査済みであること。
// 未知の列が来たらプログラムの誤りなので既定の並びのままにする。
func SortSchedule(rows []ScheduleRow, sort *ScheduleSort) {
	slices.SortFunc(rows, compareScheduleDefault)
	if sort == nil {
		return
	}
	compare, ok := scheduleComparators()[sort.Key]
	if !ok {
		return
	}
	if sort.Dir == SortDesc {
		slices.SortStableFunc(rows, func(a, b ScheduleRow) int { return compare(b, a) })
		return
	}
	slices.SortStableFunc(rows, compare)
}
