package format

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"scoreview/api/internal/domain"
)

// DefaultGroupsInParallel はいくつのグループを組にして全コートへ並べるかの既定。参照サイトの運用
// (482 は A+B → C+D、R3 5-4)に合わせて 2。
const DefaultGroupsInParallel = 2

// ScheduleConfig は時間割の生成の入力。
type ScheduleConfig struct {
	Start time.Time
	// SlotMinutes は 1 試合の枠の長さ(分)。参照サイトでは大会ごとに 15〜90 分(R3 5-2)
	SlotMinutes int
	Courts      []string
	// TeamRestSlots は同じチームの試合の間に空けるスロット数。7 人制は 1 以上、4on4 は 0 で連戦を許す
	// (R3 5-3)
	TeamRestSlots int
	// StageGapSlots は段の間に空けるスロット数
	StageGapSlots int
	// GroupsInParallel は何グループずつ組にして回すか。0 なら DefaultGroupsInParallel
	GroupsInParallel int
	// End はすべての試合が終わっていなければならない時刻。ゼロ値なら制限なし
	End time.Time
}

// Assignment は試合の時刻とコート。
type Assignment struct {
	Code  string
	Start time.Time
	Court string
}

// ViolationKind は時間割の制約違反の種類。
type ViolationKind string

// ViolationKind の値。
const (
	ViolationUnscheduled     ViolationKind = "unscheduled"      // 時刻が入っていない試合
	ViolationCourtConflict   ViolationKind = "court_conflict"   // 同じコートで時間が重なる
	ViolationTeamRest        ViolationKind = "team_rest"        // 同じチームの試合の間が足りない
	ViolationDependencyOrder ViolationKind = "dependency_order" // 前の試合(勝者・順位を待つ試合)より前に始まる
	ViolationStageGap        ViolationKind = "stage_gap"        // 段の間が足りない
	ViolationAfterEnd        ViolationKind = "after_end"        // 終了時刻を過ぎる
)

// Violation は制約違反の 1 件。ShortMinutes は足りない分数(重なり、間隔の不足、超過)。
type Violation struct {
	Kind         ViolationKind
	Codes        []string
	ShortMinutes int
	Detail       string
}

// Schedule は生成した時間割。Assignments は時刻、コートの順。
type Schedule struct {
	Assignments []Assignment
	Violations  []Violation
}

// constraintSet は試合ごとの「同じチーム」の識別子と、先に終わっていなければならない試合。
// チームは生成時点では分からないので、枠(プールの位置)を同じチームとみなし、勝者・順位を
// 待つ試合はその出どころの試合すべてを前提にする。
type constraintSet struct {
	participants map[string][]string
	prereqs      map[string][]string
}

func scheduleConstraints(plan *Plan) constraintSet {
	cs := constraintSet{participants: map[string][]string{}, prereqs: map[string][]string{}}
	groupCodes := func(g int) []string {
		var codes []string
		for _, m := range plan.GroupMatches(g) {
			codes = append(codes, m.Code)
		}
		return codes
	}
	for _, m := range plan.Matches {
		var deps []string
		for _, side := range []SideRef{m.Left, m.Right} {
			if side.From != nil {
				deps = append(deps, side.From.Code)
				continue
			}
			key := fmt.Sprintf("%d:%d", m.Group, side.Slot)
			cs.participants[m.Code] = append(cs.participants[m.Code], key)
			src := plan.SlotSource(SlotKey{Group: m.Group, Position: side.Slot})
			switch src.Kind {
			case SourceGroupRank:
				deps = append(deps, groupCodes(src.Group)...)
			case SourceMatch:
				deps = append(deps, src.Match.Code)
			case SourceBandRank:
				for _, g := range plan.StageGroups(src.Stage) {
					deps = append(deps, groupCodes(g)...)
				}
			}
		}
		slices.Sort(deps)
		cs.prereqs[m.Code] = slices.Compact(deps)
	}
	return cs
}

// BuildSchedule は段ごとに、グループを GroupsInParallel 個ずつ組にして、組の試合を
// (ラウンド、グループ、番号)の順に、制約を満たす最も早いスロットの空いているコートへ置く。
// 終了時刻までに置けない試合は、休憩の制約を緩めて置き、それでも無理なら終了時刻の後に置く。
// 違反は止めずに Violations に出す。
func BuildSchedule(plan *Plan, cfg ScheduleConfig) (*Schedule, error) {
	if cfg.SlotMinutes <= 0 || len(cfg.Courts) == 0 {
		return nil, domain.Invalid("schedule", "slot minutes and courts are required")
	}
	if cfg.TeamRestSlots < 0 || cfg.StageGapSlots < 0 || cfg.GroupsInParallel < 0 {
		return nil, domain.Invalid("schedule", "slot counts must not be negative")
	}
	parallel := cfg.GroupsInParallel
	if parallel == 0 {
		parallel = DefaultGroupsInParallel
	}
	maxSlot := -1 // 制限なし
	if !cfg.End.IsZero() {
		maxSlot = int(cfg.End.Sub(cfg.Start).Minutes())/cfg.SlotMinutes - 1
	}
	cs := scheduleConstraints(plan)
	slotOf := map[string]int{}
	courtOf := map[string]int{}
	busy := map[int][]bool{}
	last := map[string]int{}
	gap := 1 + cfg.TeamRestSlots

	earliest := func(code string, from, step int) int {
		t := from
		for _, dep := range cs.prereqs[code] {
			if s, ok := slotOf[dep]; ok {
				t = max(t, s+step)
			}
		}
		for _, p := range cs.participants[code] {
			if s, ok := last[p]; ok {
				t = max(t, s+step)
			}
		}
		return t
	}
	freeAt := func(from, limit int) (int, int, bool) {
		for t := from; limit < 0 || t <= limit; t++ {
			if busy[t] == nil {
				busy[t] = make([]bool, len(cfg.Courts))
			}
			if c := slices.Index(busy[t], false); c >= 0 {
				return t, c, true
			}
		}
		return 0, 0, false
	}

	stageStart := 0
	for stage := range plan.Stages {
		groups := plan.StageGroups(stage)
		stageLast := stageStart - 1
		for chunk := range slices.Chunk(groups, parallel) {
			var ms []Match
			for _, g := range chunk {
				ms = append(ms, plan.GroupMatches(g)...)
			}
			slices.SortStableFunc(ms, func(a, b Match) int {
				return cmp.Or(cmp.Compare(a.Round, b.Round), cmp.Compare(a.Group, b.Group),
					cmp.Compare(a.Number, b.Number))
			})
			for _, m := range ms {
				t, c, ok := freeAt(earliest(m.Code, stageStart, gap), maxSlot)
				if !ok {
					t, c, ok = freeAt(earliest(m.Code, stageStart, 1), maxSlot)
				}
				if !ok {
					t, c, _ = freeAt(earliest(m.Code, stageStart, gap), -1)
				}
				busy[t][c] = true
				slotOf[m.Code], courtOf[m.Code] = t, c
				for _, p := range cs.participants[m.Code] {
					last[p] = max(last[p], t)
				}
				stageLast = max(stageLast, t)
			}
		}
		stageStart = stageLast + 1 + cfg.StageGapSlots
	}

	out := &Schedule{}
	for _, m := range plan.Matches {
		out.Assignments = append(out.Assignments, Assignment{
			Code:  m.Code,
			Start: cfg.Start.Add(time.Duration(slotOf[m.Code]*cfg.SlotMinutes) * time.Minute),
			Court: cfg.Courts[courtOf[m.Code]],
		})
	}
	sortAssignments(out.Assignments, cfg.Courts)
	out.Violations = CheckSchedule(plan, cfg, out.Assignments)
	return out, nil
}

func sortAssignments(as []Assignment, courts []string) {
	slices.SortStableFunc(as, func(a, b Assignment) int {
		return cmp.Or(a.Start.Compare(b.Start),
			cmp.Compare(slices.Index(courts, a.Court), slices.Index(courts, b.Court)),
			cmp.Compare(a.Court, b.Court))
	})
}

// CheckSchedule は時間割の制約を検査する(運営者が手で動かした後の再検査に使う)。
// 時刻は分単位で比べ、同じチームと前提の試合は開始時刻の差が (1+TeamRestSlots)×SlotMinutes 以上、
// 段の間は (1+StageGapSlots)×SlotMinutes 以上を求める。
func CheckSchedule(plan *Plan, cfg ScheduleConfig, assignments []Assignment) []Violation {
	byCode := map[string]Assignment{}
	for _, a := range assignments {
		byCode[a.Code] = a
	}
	var out []Violation
	minutes := func(d time.Duration) int { return int(d / time.Minute) }
	for _, m := range plan.Matches {
		if _, ok := byCode[m.Code]; !ok {
			out = append(out, Violation{Kind: ViolationUnscheduled, Codes: []string{m.Code},
				Detail: m.Code + " に時刻がない"})
		}
	}

	// 同じコートの重なり
	sorted := slices.Clone(assignments)
	sortAssignments(sorted, cfg.Courts)
	for i, a := range sorted {
		for _, b := range sorted[i+1:] {
			if a.Court != b.Court {
				continue
			}
			end := a.Start.Add(time.Duration(cfg.SlotMinutes) * time.Minute)
			if overlap := minutes(end.Sub(b.Start)); overlap > 0 {
				out = append(out, Violation{
					Kind: ViolationCourtConflict, Codes: []string{a.Code, b.Code}, ShortMinutes: overlap,
					Detail: fmt.Sprintf("コート %s で %s と %s が重なる", a.Court, a.Code, b.Code),
				})
			}
		}
	}

	need := (1 + cfg.TeamRestSlots) * cfg.SlotMinutes
	cs := scheduleConstraints(plan)
	// 同じ枠(チーム)の連続する試合
	byParticipant := map[string][]Assignment{}
	var keys []string
	for _, m := range plan.Matches {
		a, ok := byCode[m.Code]
		if !ok {
			continue
		}
		for _, p := range cs.participants[m.Code] {
			if _, seen := byParticipant[p]; !seen {
				keys = append(keys, p)
			}
			byParticipant[p] = append(byParticipant[p], a)
		}
	}
	for _, p := range keys {
		list := byParticipant[p]
		slices.SortStableFunc(list, func(a, b Assignment) int { return a.Start.Compare(b.Start) })
		for i := 1; i < len(list); i++ {
			if got := minutes(list[i].Start.Sub(list[i-1].Start)); got < need {
				prev, cur := list[i-1].Code, list[i].Code
				out = append(out, Violation{
					Kind: ViolationTeamRest, Codes: []string{prev, cur}, ShortMinutes: need - got,
					Detail: fmt.Sprintf("%s と %s の間が %d 分足りない", prev, cur, need-got),
				})
			}
		}
	}
	// 前提の試合
	for _, m := range plan.Matches {
		a, ok := byCode[m.Code]
		if !ok {
			continue
		}
		for _, dep := range cs.prereqs[m.Code] {
			d, ok := byCode[dep]
			if !ok {
				continue
			}
			got := minutes(a.Start.Sub(d.Start))
			switch {
			case got < cfg.SlotMinutes:
				out = append(out, Violation{Kind: ViolationDependencyOrder, Codes: []string{dep, m.Code},
					ShortMinutes: cfg.SlotMinutes - got,
					Detail:       fmt.Sprintf("%s が %s の終了前に始まる", m.Code, dep)})
			case got < need && slices.Contains(directDeps(m), dep):
				out = append(out, Violation{Kind: ViolationTeamRest, Codes: []string{dep, m.Code},
					ShortMinutes: need - got,
					Detail:       fmt.Sprintf("%s と %s の間が %d 分足りない", dep, m.Code, need-got)})
			}
		}
	}
	out = append(out, stageGapViolations(plan, cfg, byCode)...)
	if !cfg.End.IsZero() {
		for _, a := range sorted {
			if over := minutes(a.Start.Add(time.Duration(cfg.SlotMinutes) * time.Minute).Sub(cfg.End)); over > 0 {
				out = append(out, Violation{Kind: ViolationAfterEnd, Codes: []string{a.Code}, ShortMinutes: over,
					Detail: fmt.Sprintf("%s が終了時刻を %d 分過ぎる", a.Code, over)})
			}
		}
	}
	return out
}

// directDeps は試合の左右が直接待つ試合(勝者・敗者)。順位を待つ試合(group_rank など)は、
// 前の試合に出たチームが来るとは限らないので休憩の不足には数えず、順序だけを検査する。
func directDeps(m Match) []string {
	var out []string
	for _, side := range []SideRef{m.Left, m.Right} {
		if side.From != nil {
			out = append(out, side.From.Code)
		}
	}
	return out
}

func stageGapViolations(plan *Plan, cfg ScheduleConfig, byCode map[string]Assignment) []Violation {
	var out []Violation
	need := (1 + cfg.StageGapSlots) * cfg.SlotMinutes
	var prevLast *Assignment
	for stage := range plan.Stages {
		var first, last *Assignment
		for _, g := range plan.StageGroups(stage) {
			for _, m := range plan.GroupMatches(g) {
				a, ok := byCode[m.Code]
				if !ok {
					continue
				}
				if first == nil || a.Start.Before(first.Start) {
					first = &a
				}
				if last == nil || a.Start.After(last.Start) {
					last = &a
				}
			}
		}
		if prevLast != nil && first != nil {
			if got := int(first.Start.Sub(prevLast.Start) / time.Minute); got < need {
				out = append(out, Violation{
					Kind: ViolationStageGap, Codes: []string{prevLast.Code, first.Code}, ShortMinutes: need - got,
					Detail: fmt.Sprintf("段 %s の前の間が %d 分足りない", plan.Stages[stage].Name, need-got),
				})
			}
		}
		if last != nil {
			prevLast = last
		}
	}
	return out
}

// ShiftCourt はコート court で from 以降に始まる試合を minutes 分遅らせた時間割を返す
// (当日の遅延。試合の順序は変えない)。元のスライスは変えない。
func ShiftCourt(assignments []Assignment, court string, from time.Time, minutes int) []Assignment {
	target := func(a Assignment) bool { return a.Court == court && !a.Start.Before(from) }
	return shift(assignments, target, minutes)
}

// ShiftAll は from 以降に始まるすべての試合を minutes 分遅らせた時間割を返す。
func ShiftAll(assignments []Assignment, from time.Time, minutes int) []Assignment {
	return shift(assignments, func(a Assignment) bool { return !a.Start.Before(from) }, minutes)
}

func shift(assignments []Assignment, target func(Assignment) bool, minutes int) []Assignment {
	out := slices.Clone(assignments)
	for i := range out {
		if target(out[i]) {
			out[i].Start = out[i].Start.Add(time.Duration(minutes) * time.Minute)
		}
	}
	return out
}
