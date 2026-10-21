package format_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/format"
)

var dayStart = time.Date(2026, 6, 14, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60))

func at(minutes int) time.Time { return dayStart.Add(time.Duration(minutes) * time.Minute) }

func byCode(as []format.Assignment) map[string]format.Assignment {
	out := map[string]format.Assignment{}
	for _, a := range as {
		out[a.Code] = a
	}
	return out
}

func kinds(vs []format.Violation) []format.ViolationKind {
	var out []format.ViolationKind
	for _, v := range vs {
		if !slices.Contains(out, v.Kind) {
			out = append(out, v.Kind)
		}
	}
	return out
}

// TestScheduleTwoPoolsFourCourts は 4 チームのプール 2 つを 4 コートで、連戦なし(休み 1 スロット)。
// ラウンドごとに 4 試合を同時に行い、各チームは 1 スロット休む。
func TestScheduleTwoPoolsFourCourts(t *testing.T) {
	plan := generate(t, format.TemplatePoolBands, format.PoolBandsParams{Teams: 8, Pools: 2})
	cfg := format.ScheduleConfig{Start: dayStart, SlotMinutes: 60, Courts: []string{"1", "2", "3", "4"},
		TeamRestSlots: 1}
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	assert.Empty(t, sched.Violations)
	got := byCode(sched.Assignments)
	// ラウンド 1(1-4、2-3)は 9:00、ラウンド 2 は 11:00、ラウンド 3 は 13:00
	assert.Equal(t, format.Assignment{Code: "A3", Start: at(0), Court: "1"}, got["A3"])
	assert.Equal(t, format.Assignment{Code: "B4", Start: at(0), Court: "4"}, got["B4"])
	assert.Equal(t, at(120), got["A2"].Start)
	assert.Equal(t, at(240), got["B1"].Start)
	assert.Equal(t, "A3", sched.Assignments[0].Code, "時刻とコートの順")
}

// TestSchedule4on4BackToBack は 4on4(休み 0 で連戦を許す)の 916 型を 4 コートで組む。
func TestSchedule4on4BackToBack(t *testing.T) {
	plan := plan916(t)
	cfg := format.ScheduleConfig{Start: dayStart, SlotMinutes: 20, Courts: []string{"1", "2", "3", "4"}}
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	assert.Empty(t, sched.Violations)
	got := byCode(sched.Assignments)
	// プール A: (1-3) 9:00 → (1-2) 9:20 で 1 シードは連戦
	assert.Equal(t, at(0), got["A2"].Start)
	assert.Equal(t, at(20), got["A1"].Start)
	assert.Equal(t, at(40), got["A3"].Start)
	// A+B と C+D を同じ時間帯に並べ、プールは 3 スロットで終わる。順位決定は次のスロットから
	assert.Equal(t, at(0), got["C2"].Start)
	assert.Equal(t, at(60), got["E2"].Start)
	assert.Equal(t, at(80), got["E1"].Start)
	assert.Equal(t, at(100), got["G2"].Start)
	assert.Equal(t, at(120), got["G1"].Start)

	// 同じ形を休み 1 で組むと連戦はなくなり、時間が延びる
	cfg.TeamRestSlots = 1
	rested, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	assert.Empty(t, rested.Violations)
	assert.Equal(t, at(40), byCode(rested.Assignments)["A1"].Start)
}

func TestScheduleReportsViolations(t *testing.T) {
	plan := generate(t, format.TemplatePoolBands, format.PoolBandsParams{Teams: 8, Pools: 2})
	cfg := format.ScheduleConfig{Start: dayStart, SlotMinutes: 60, Courts: []string{"1", "2"},
		TeamRestSlots: 1, End: at(4 * 60)}
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	// 12 試合を 2 コートで 4 スロットには入らない。生成は止めずに違反を出す
	assert.Len(t, sched.Assignments, 12)
	assert.Equal(t, []format.ViolationKind{format.ViolationAfterEnd}, kinds(sched.Violations))

	// 4 チーム 1 プールを 2 スロットに: ラウンド 2 は休みを緩めて詰め、ラウンド 3 は終了時刻の後
	single := generate(t, format.TemplateRoundRobin, format.RoundRobinParams{Teams: 4, Pools: 1})
	cfg.End = at(2 * 60)
	sched, err = format.BuildSchedule(single, cfg)
	require.NoError(t, err)
	got := byCode(sched.Assignments)
	assert.Equal(t, []time.Time{at(0), at(60), at(180)}, []time.Time{got["A3"].Start, got["A2"].Start, got["A1"].Start})
	assert.ElementsMatch(t, []format.ViolationKind{format.ViolationTeamRest, format.ViolationAfterEnd},
		kinds(sched.Violations))
}

func TestCheckScheduleAfterManualMove(t *testing.T) {
	plan := plan916(t)
	cfg := format.ScheduleConfig{Start: dayStart, SlotMinutes: 20, Courts: []string{"1", "2", "3", "4"},
		TeamRestSlots: 1}
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	require.Empty(t, sched.Violations)

	moved := slices.Clone(sched.Assignments)
	got := byCode(moved)
	for i := range moved {
		switch moved[i].Code {
		case "A1": // A2 と同じ時刻・同じコートへ
			moved[i].Start, moved[i].Court = got["A2"].Start, got["A2"].Court
		case "E1": // 準決勝より前へ
			moved[i].Start = got["E2"].Start.Add(-20 * time.Minute)
		}
	}
	violations := format.CheckSchedule(plan, cfg, moved)
	assert.Equal(t, []format.ViolationKind{format.ViolationCourtConflict, format.ViolationTeamRest,
		format.ViolationDependencyOrder}, kinds(violations))
	for _, v := range violations {
		if v.Kind == format.ViolationCourtConflict {
			assert.Equal(t, 20, v.ShortMinutes)
			assert.ElementsMatch(t, []string{"A1", "A2"}, v.Codes)
		}
	}
	assert.Equal(t, []format.ViolationKind{format.ViolationUnscheduled},
		kinds(format.CheckSchedule(plan, cfg, sched.Assignments[1:])))
}

func TestShift(t *testing.T) {
	as := []format.Assignment{
		{Code: "A1", Start: at(0), Court: "1"},
		{Code: "A2", Start: at(60), Court: "1"},
		{Code: "B1", Start: at(60), Court: "2"},
	}
	court := format.ShiftCourt(as, "1", at(30), 15)
	assert.Equal(t, []time.Time{at(0), at(75), at(60)}, []time.Time{court[0].Start, court[1].Start, court[2].Start})
	all := format.ShiftAll(as, at(60), 30)
	assert.Equal(t, []time.Time{at(0), at(90), at(90)}, []time.Time{all[0].Start, all[1].Start, all[2].Start})
	assert.Equal(t, at(60), as[1].Start, "元の時間割は変えない")
}

func TestScheduleConfigErrors(t *testing.T) {
	plan := plan916(t)
	_, err := format.BuildSchedule(plan, format.ScheduleConfig{Start: dayStart, SlotMinutes: 20})
	assert.Error(t, err)
}
