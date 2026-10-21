package format

import (
	"fmt"
	"time"
)

// 複数日の時間割: 会場を使う日ごとの時間、昼休み、段ごとの日分け。

// DayWindow は会場を使う 1 日の時間 [Start, End)。End がゼロなら終わりなし(最後の日だけ)。
type DayWindow struct {
	Start time.Time
	End   time.Time
}

// Break は試合を置かない時間 [Start, End)(昼休み)。
type Break struct {
	Start time.Time
	End   time.Time
}

// days は会場を使う日。Days が空なら Start から End までの 1 日(1 日の大会の設定)。
func (cfg ScheduleConfig) days() []DayWindow {
	if len(cfg.Days) > 0 {
		return cfg.Days
	}
	return []DayWindow{{Start: cfg.Start, End: cfg.End}}
}

// WithoutEnd は各日の終了時刻を外した設定を返す(当日の再検査用。遅れで終了時刻を超えるのは普通)。
func (cfg ScheduleConfig) WithoutEnd() ScheduleConfig {
	out := cfg
	out.End = time.Time{}
	if len(cfg.Days) > 0 {
		out.Days = make([]DayWindow, len(cfg.Days))
		for i, day := range cfg.Days {
			out.Days[i] = DayWindow{Start: day.Start}
		}
	}
	return out
}

// timeline は時間割のスロットの開始時刻。各日の開始から終了までを枠の長さで刻み、昼休みに重なる
// 枠は昼休みの終わりから刻み直す。会場の時間内の枠を使い切った後は、最後の枠に続けて時間外の枠を
// 作る(収まらない試合を置く場所)。
type timeline struct {
	days   []DayWindow
	breaks []Break
	pitch  time.Duration
	slots  []time.Time
	dayOf  []int
	// inHours は会場の時間内の枠の数。-1 は制限なし(最後の日に終わりがない)
	inHours int
	day     int
	next    time.Time
}

func newTimeline(days []DayWindow, breaks []Break, slotMinutes int) *timeline {
	tl := &timeline{days: days, breaks: breaks, pitch: time.Duration(slotMinutes) * time.Minute,
		inHours: -1, next: days[0].Start}
	if !days[len(days)-1].End.IsZero() {
		for grown := true; grown; {
			grown = tl.growInHours()
		}
		tl.inHours = len(tl.slots)
	}
	return tl
}

// growInHours は会場の時間内の次の枠を 1 つ足す。時間内の枠が尽きたら false。
func (tl *timeline) growInHours() bool {
	for tl.day < len(tl.days) {
		day := tl.days[tl.day]
		if tl.next.Before(day.Start) {
			tl.next = day.Start
		}
		end := tl.next.Add(tl.pitch)
		if !day.End.IsZero() && end.After(day.End) {
			tl.day++
			continue
		}
		if b, ok := tl.breakOver(tl.next, end); ok {
			tl.next = b.End
			continue
		}
		tl.slots = append(tl.slots, tl.next)
		tl.dayOf = append(tl.dayOf, tl.day)
		tl.next = end
		return true
	}
	return false
}

func (tl *timeline) breakOver(start, end time.Time) (Break, bool) {
	for _, b := range tl.breaks {
		if overlapMinutes(start, end, b.Start, b.End) > 0 {
			return b, true
		}
	}
	return Break{}, false
}

// at はスロット i の開始時刻。
func (tl *timeline) at(i int) time.Time {
	for len(tl.slots) <= i {
		if tl.growInHours() {
			continue
		}
		// 時間外: 最後の枠に続けて置く(終了の後の試合は after_end として出る)
		next := tl.days[len(tl.days)-1].Start
		if n := len(tl.slots); n > 0 {
			next = tl.slots[n-1].Add(tl.pitch)
		}
		tl.slots = append(tl.slots, next)
		tl.dayOf = append(tl.dayOf, len(tl.days)-1)
	}
	return tl.slots[i]
}

// lastInHours は会場の時間内の最後のスロット。-1 は制限なし。
func (tl *timeline) lastInHours() int {
	if tl.inHours < 0 {
		return -1
	}
	return tl.inHours - 1
}

// firstOfDay は day 日目(0 から)以降の最初のスロット。時間内に無ければ時間外の最初のスロット。
func (tl *timeline) firstOfDay(day int) int {
	for i := 0; ; i++ {
		if tl.inHours >= 0 && i >= tl.inHours {
			return tl.inHours
		}
		tl.at(i)
		if tl.dayOf[i] >= day {
			return i
		}
	}
}

// lastOfDay は day 日目の最後のスロット。-1 は制限なし(終わりのない最後の日)。
func (tl *timeline) lastOfDay(day int) int {
	if day+1 < len(tl.days) {
		return tl.firstOfDay(day+1) - 1
	}
	return tl.lastInHours()
}

// dayIndex は t の日付が何日目か(0 から)。会場を使う日でなければ -1。
func dayIndex(days []DayWindow, t time.Time) int {
	for i, day := range days {
		y1, m1, d1 := day.Start.Date()
		y2, m2, d2 := t.Date()
		if y1 == y2 && m1 == m2 && d1 == d2 {
			return i
		}
	}
	return -1
}

// dayLabel は複数日のときだけ「2 日目の」を返す。
func dayLabel(days []DayWindow, index int) string {
	if len(days) < 2 {
		return ""
	}
	return fmt.Sprintf("%d 日目の ", index+1)
}

// hoursViolations は会場の時間(日、開始、終了)、昼休み、段ごとの日分けの違反。
func hoursViolations(plan *Plan, cfg ScheduleConfig, sorted []Assignment) []Violation {
	days := cfg.days()
	length := time.Duration(cfg.SlotMinutes) * time.Minute
	stageOf := map[string]int{}
	for _, m := range plan.Matches {
		stageOf[m.Code] = plan.Groups[m.Group].Stage
	}
	var out []Violation
	for _, a := range sorted {
		finish := a.Start.Add(length)
		index := dayIndex(days, a.Start)
		if last := days[len(days)-1]; index < 0 && last.End.IsZero() && a.Start.After(last.Start) {
			index = len(days) - 1 // 終わりのない最後の日は日付が変わっても続く(終了時刻の無い 1 日の大会)
		}
		if index < 0 {
			out = append(out, Violation{Kind: ViolationOutsideHours, Codes: []string{a.Code},
				Detail: fmt.Sprintf("%s の日付(%s)は会場を使う日ではありません", a.Code,
					a.Start.Format("1/2"))})
			continue
		}
		day := days[index]
		if early := int(day.Start.Sub(a.Start) / time.Minute); early > 0 {
			out = append(out, Violation{Kind: ViolationOutsideHours, Codes: []string{a.Code},
				ShortMinutes: early, Detail: fmt.Sprintf("%s が%s開始時刻 %s より前に始まります", a.Code,
					dayLabel(days, index), day.Start.Format("15:04"))})
		}
		if !day.End.IsZero() {
			if over := int(finish.Sub(day.End) / time.Minute); over > 0 {
				out = append(out, Violation{Kind: ViolationAfterEnd, Codes: []string{a.Code}, ShortMinutes: over,
					Detail: fmt.Sprintf("%s が%s終了時刻 %s を %d 分過ぎます", a.Code, dayLabel(days, index),
						day.End.Format("15:04"), over)})
			}
		}
		for _, b := range cfg.Breaks {
			if overlap := overlapMinutes(a.Start, finish, b.Start, b.End); overlap > 0 {
				out = append(out, Violation{Kind: ViolationBreak, Codes: []string{a.Code}, ShortMinutes: overlap,
					Detail: fmt.Sprintf("%s が昼休み(%s〜%s)に %d 分重なります", a.Code,
						b.Start.Format("15:04"), b.End.Format("15:04"), overlap)})
			}
		}
		if cfg.StageDays && len(days) > 1 {
			stage, ok := stageOf[a.Code]
			if !ok {
				continue
			}
			first := min(stage, len(days)-1)
			lastDay := first
			if stage == len(plan.Stages)-1 {
				lastDay = len(days) - 1
			}
			if index < first || index > lastDay {
				out = append(out, Violation{Kind: ViolationStageDay, Codes: []string{a.Code},
					Detail: fmt.Sprintf("%s(%s)が %d 日目に入っていません(%d 日目に置かれています)", a.Code,
						plan.Stages[stage].Name, first+1, index+1)})
			}
		}
	}
	return out
}

// OverflowNote は会場の時間に収まらない試合の説明(「17:00 までにあと 6 試合が収まりません」)。
// 収まっていれば空文字。
func OverflowNote(cfg ScheduleConfig, s *Schedule) string {
	days := cfg.days()
	last := days[len(days)-1]
	if s.Overflow == 0 || last.End.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s%s までにあと %d 試合が収まりません", dayLabel(days, len(days)-1),
		last.End.Format("15:04"), s.Overflow)
}

// EndOf は時間割の最後の試合の終了時刻と、それが何日目か(0 から)。
func EndOf(cfg ScheduleConfig, s *Schedule) (time.Time, int) {
	var end time.Time
	for _, a := range s.Assignments {
		if finish := a.Start.Add(time.Duration(cfg.SlotMinutes) * time.Minute); finish.After(end) {
			end = finish
		}
	}
	days := cfg.days()
	index := dayIndex(days, end.Add(-time.Minute))
	if index < 0 {
		index = len(days) - 1
	}
	return end, index
}
