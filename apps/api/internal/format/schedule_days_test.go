package format_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/format"
)

// clock は 2026-11-14 を 1 日目とする day 日目(1 から)の hh:mm(UTC、試合の時刻は現地時刻のまま扱う)。
func clock(day, hour, minute int) time.Time {
	return time.Date(2026, 11, 13+day, hour, minute, 0, 0, time.UTC)
}

// twoDays7s は 7 人制 16 チームを 2 日間(各日 9:00〜17:00、昼休み 12:00〜13:00)、4 コート、60 分枠で
// 組む設定(運営者レビュー C2 の大会)。
func twoDays7s(days int) format.ScheduleConfig {
	cfg := format.ScheduleConfig{SlotMinutes: 60, Courts: []string{"1", "2", "3", "4"}, TeamRestSlots: 1}
	for day := 1; day <= days; day++ {
		cfg.Days = append(cfg.Days, format.DayWindow{Start: clock(day, 9, 0), End: clock(day, 17, 0)})
		cfg.Breaks = append(cfg.Breaks, format.Break{Start: clock(day, 12, 0), End: clock(day, 13, 0)})
	}
	return cfg
}

func inLunch(a format.Assignment) bool {
	return a.Start.Hour() == 12
}

// TestScheduleTwoDaysStagePerDay は「段ごとに日を分ける」で、1 日目に予選、2 日目に順位決定戦が
// 各日の時間内に、昼休みを避けて収まることを確かめる。
func TestScheduleTwoDaysStagePerDay(t *testing.T) {
	plan := plan435(t)
	cfg := twoDays7s(2)
	cfg.StageDays = true
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	assert.Empty(t, sched.Violations)
	assert.Zero(t, sched.Overflow)
	assert.Empty(t, format.OverflowNote(cfg, sched))
	stageOf := map[string]int{}
	for _, m := range plan.Matches {
		stageOf[m.Code] = plan.Groups[m.Group].Stage
	}
	for _, a := range sched.Assignments {
		wantDay := 14 + stageOf[a.Code]
		assert.Equal(t, wantDay, a.Start.Day(), "%s は %d 日目", a.Code, stageOf[a.Code]+1)
		assert.False(t, inLunch(a), "%s が昼休みに入っている", a.Code)
		assert.False(t, a.Start.Before(clock(1, 9, 0).AddDate(0, 0, a.Start.Day()-14)), a.Code)
		assert.False(t, a.Start.Add(time.Hour).After(clock(1, 17, 0).AddDate(0, 0, a.Start.Day()-14)), a.Code)
	}
	end, day := format.EndOf(cfg, sched)
	assert.Equal(t, 1, day, "最後の試合は 2 日目")
	assert.False(t, end.After(clock(2, 17, 0)))
	first := clock(2, 17, 0)
	for _, a := range sched.Assignments {
		if stageOf[a.Code] == 1 && a.Start.Before(first) {
			first = a.Start
		}
	}
	assert.Equal(t, clock(2, 9, 0), first, "順位決定戦は 2 日目の 9:00 から")
}

// TestScheduleTwoDaysWithoutStageSplit は日分けをしなければ、予選に続けて順位決定戦を 1 日目に
// 詰め、試合は日をまたがない(1 日目の 17:00 を越える試合は 2 日目の 9:00 から)。
func TestScheduleTwoDaysWithoutStageSplit(t *testing.T) {
	plan := plan435(t)
	cfg := twoDays7s(2)
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	assert.Empty(t, sched.Violations)
	var firstDay, secondDay int
	for _, a := range sched.Assignments {
		switch a.Start.Day() {
		case 14:
			firstDay++
			assert.False(t, a.Start.Add(time.Hour).After(clock(1, 17, 0)), a.Code)
		case 15:
			secondDay++
		default:
			t.Errorf("%s が会場を使う日の外 %s", a.Code, a.Start)
		}
		assert.False(t, inLunch(a), a.Code)
	}
	assert.Equal(t, 48, firstDay+secondDay)
	assert.Positive(t, secondDay)
	assert.GreaterOrEqual(t, firstDay, 24, "予選は 1 日目に収まる")
}

// TestScheduleOverflowReason は収まらないとき、生成を止めずに時間外へ置き、理由の文を返す。
func TestScheduleOverflowReason(t *testing.T) {
	plan := plan435(t)
	cfg := twoDays7s(1)
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	// 1 日 7 枠 × 4 コート = 28 試合しか入らない
	assert.Equal(t, 48-28, sched.Overflow)
	assert.Equal(t, "17:00 までにあと 20 試合が収まりません", format.OverflowNote(cfg, sched))
	assert.Contains(t, kinds(sched.Violations), format.ViolationAfterEnd)

	cfg2 := twoDays7s(2)
	cfg2.Days[1].End = clock(2, 10, 0) // 2 日目は 9:00〜10:00 だけ
	sched2, err := format.BuildSchedule(plan, cfg2)
	require.NoError(t, err)
	assert.Equal(t, 48-28-4, sched2.Overflow)
	assert.Equal(t, "2 日目の 10:00 までにあと 16 試合が収まりません", format.OverflowNote(cfg2, sched2))
}

// TestCheckScheduleHours は会場の時間外、昼休み、段の日の違反を検査で出す(手で動かした後)。
func TestCheckScheduleHours(t *testing.T) {
	plan := plan435(t)
	cfg := twoDays7s(2)
	cfg.StageDays = true
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	require.Empty(t, sched.Violations)

	moved := slices.Clone(sched.Assignments)
	got := byCode(moved)
	for i := range moved {
		switch moved[i].Code {
		case "A1": // 昼休みへ
			moved[i].Start, moved[i].Court = clock(1, 12, 0), "1"
		case "A2": // 開始より前
			moved[i].Start = clock(1, 8, 0)
		case "F1": // 決勝を 1 日目の夕方へ(段の日と終了時刻の両方)
			moved[i].Start = clock(1, 16, 30)
		case "F2": // 会場を使わない日
			moved[i].Start = got["F2"].Start.AddDate(0, 0, 1)
		}
	}
	violations := format.CheckSchedule(plan, cfg, moved)
	byKind := map[format.ViolationKind][]string{}
	for _, v := range violations {
		byKind[v.Kind] = append(byKind[v.Kind], v.Codes...)
	}
	assert.Contains(t, byKind[format.ViolationBreak], "A1")
	assert.ElementsMatch(t, []string{"A2", "F2"}, byKind[format.ViolationOutsideHours])
	assert.Contains(t, byKind[format.ViolationAfterEnd], "F1")
	assert.Contains(t, byKind[format.ViolationStageDay], "F1")

	// 当日の再検査は終了時刻を見ない(遅れで超えるのは普通)
	for _, v := range format.CheckSchedule(plan, cfg.WithoutEnd(), moved) {
		assert.NotEqual(t, format.ViolationAfterEnd, v.Kind)
	}
}

// TestScheduleTeamRestViolation は「同じチームの試合の間に最低 N 枠空ける」を守れない試合を
// 違反として一覧に出す(運営者レビュー H8。4on4 の 09:00 と 09:20 の連戦)。
func TestScheduleTeamRestViolation(t *testing.T) {
	plan := plan916(t)
	cfg := format.ScheduleConfig{Start: dayStart, SlotMinutes: 20, Courts: []string{"1", "2", "3", "4"}}
	sched, err := format.BuildSchedule(plan, cfg)
	require.NoError(t, err)
	require.Empty(t, sched.Violations, "4on4 の既定(0 枠)は連戦を許す")

	cfg.TeamRestSlots = 1
	violations := format.CheckSchedule(plan, cfg, sched.Assignments)
	var pairs [][]string
	for _, v := range violations {
		if v.Kind == format.ViolationTeamRest {
			pairs = append(pairs, v.Codes)
			assert.Equal(t, 20, v.ShortMinutes)
		}
	}
	assert.Contains(t, pairs, []string{"A2", "A1"}, "プール A の 1 シードの 9:00 と 9:20")
	assert.Contains(t, pairs, []string{"E2", "E1"}, "準決勝の 20 分後の決勝")
}

func TestScheduleRejectsBadDays(t *testing.T) {
	plan := plan916(t)
	cfg := twoDays7s(2)
	cfg.Days[0].End = time.Time{}
	_, err := format.BuildSchedule(plan, cfg)
	require.Error(t, err, "最後の日以外は終了時刻が要る")

	cfg = twoDays7s(2)
	cfg.Days[0], cfg.Days[1] = cfg.Days[1], cfg.Days[0]
	_, err = format.BuildSchedule(plan, cfg)
	require.Error(t, err, "日は順に")

	cfg = twoDays7s(1)
	cfg.Days[0].End = clock(1, 9, 30)
	_, err = format.BuildSchedule(plan, cfg)
	require.Error(t, err, "1 試合も入らない")
}

// TestProposeConsidersDays は形の提案が日数と会場の時間を考慮し、収まらない候補に理由を付けて
// 収まる候補の後ろへ回すことを確かめる。
func TestProposeConsidersDays(t *testing.T) {
	cfg := twoDays7s(2)
	cfg.StageDays = true
	got, err := format.Propose(format.ProposalInput{Teams: 16, Sport: format.SportSeven, Schedule: &cfg})
	require.NoError(t, err)
	require.NotEmpty(t, got)
	assert.Equal(t, "pb4-t8", got[0].ID)
	assert.True(t, got[0].Recommended)
	assert.Zero(t, got[0].Overflow)
	assert.Empty(t, got[0].FitNote)
	assert.Equal(t, 1, got[0].EndDay, "順位決定戦は 2 日目")

	one := twoDays7s(1)
	got, err = format.Propose(format.ProposalInput{Teams: 16, Sport: format.SportSeven, Schedule: &one})
	require.NoError(t, err)
	seenOverflow := false
	for _, p := range got {
		if p.Overflow > 0 {
			seenOverflow = true
			assert.Contains(t, p.FitNote, "17:00 までにあと")
			assert.False(t, p.Recommended)
			continue
		}
		assert.False(t, seenOverflow, "収まる候補 %s は収まらない候補より前", p.ID)
	}
	assert.True(t, seenOverflow, "48 試合の形は 1 日に収まらない")
}
