package importer

import (
	"slices"
	"time"
)

const dateLayout = "2006-01-02"

// dateSlot は試合一覧の 1 行の日付。Date が決まっていれば Date、年を決められない行は Candidates
// (同じ月日の "YYYY-MM-DD" が複数)を持つ。
type dateSlot struct {
	Date       *string
	Candidates []string
}

// resolveDateSlots は年を決められない行の日付を、一覧の前後の行の日付から決める。一覧は日付順に
// 並んでいる前提で、前後で最も近い「日付が決まっている行」と連続する候補を選ぶ(pickDate)。
// 戻り値は slots と同じ長さで、決められなかった行(候補の先頭を使った行)は ambiguous に添字が入る。
// Date も Candidates もない行は nil のまま。
func resolveDateSlots(slots []dateSlot) (dates []*string, ambiguous []int) {
	dates = make([]*string, len(slots))
	for i, slot := range slots {
		switch {
		case slot.Date != nil:
			dates[i] = slot.Date
		case len(slot.Candidates) == 1:
			dates[i] = &slot.Candidates[0]
		}
	}
	for i, slot := range slots {
		if dates[i] != nil || len(slot.Candidates) == 0 {
			continue
		}
		var prev, next *string
		for k := i - 1; k >= 0 && prev == nil; k-- {
			prev = dates[k]
		}
		for k := i + 1; k < len(slots) && next == nil; k++ {
			next = dates[k]
		}
		date, ok := pickDate(slot.Candidates, prev, next)
		dates[i] = &date
		if !ok {
			ambiguous = append(ambiguous, i)
		}
	}
	return dates, ambiguous
}

// adjacentDays は「連続する」とみなす日数の差。大会は連日開催なので、前後の行と同じ日か 1 日違いなら
// 同じ大会期間の日付とみなす。
const adjacentDays = 1

// pickDate は候補から、前後の行の日付と連続する 1 つを選ぶ。同じ日を最優先し、次に adjacentDays 以内の
// 日を選ぶ。決められなければ候補の先頭を返して ok=false。
func pickDate(candidates []string, prev, next *string) (date string, ok bool) {
	neighbors := []*string{prev, next}
	for _, n := range neighbors {
		if n != nil && slices.Contains(candidates, *n) {
			return *n, true
		}
	}
	for _, n := range neighbors {
		if n == nil {
			continue
		}
		for _, c := range candidates {
			if daysBetween(c, *n) <= adjacentDays {
				return c, true
			}
		}
	}
	return candidates[0], false
}

// daysBetween は 2 つの "YYYY-MM-DD" の日数の差の絶対値。読めない値は十分に大きい差とする。
func daysBetween(a, b string) int {
	ta, errA := time.Parse(dateLayout, a)
	tb, errB := time.Parse(dateLayout, b)
	if errA != nil || errB != nil {
		return 1 << 30
	}
	d := ta.Sub(tb).Hours() / 24
	if d < 0 {
		d = -d
	}
	return int(d)
}
