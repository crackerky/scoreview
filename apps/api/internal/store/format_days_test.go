package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

func strp(v string) *string { return &v }

// twoDaySetting は 2 日間(各日 9:00〜17:00、昼休み 12:00〜13:00)、4 コート、60 分枠、休み 1 枠で、
// 段ごとに日を分ける時間割の設定(運営者レビュー C2 の 7 人制 16 チーム)。
func twoDaySetting() *store.ScheduleSetting {
	return &store.ScheduleSetting{Date: "2026-11-14", Start: "09:00", End: strp("17:00"), SlotMinutes: 60,
		Courts: []string{"1", "2", "3", "4"}, TeamRestSlots: 1, StageDays: true, Sport: strp("seven"),
		Days: []store.ScheduleDay{
			{Date: "2026-11-14", Start: "09:00", End: strp("17:00")},
			{Date: "2026-11-15", Start: "09:00", End: strp("17:00")},
		},
		Lunch: &store.LunchSetting{Start: "12:00", Minutes: 60}}
}

// TestApplyFormatTwoDays は 2 日間の時間割を作り、プレビューと作成が同じ条件で同じ時間割になること
// (運営者レビュー C3: プレビューは 3 コート、作成は 4 コートの食い違い)と、保存した条件が
// 開き直したときにそのまま読めることを確かめる。
func TestApplyFormatTwoDays(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	b := storetest.New(t, db)
	division := b.Division(b.Tournament().ID)
	seeds := make([]int64, 16)
	for i := range seeds {
		name := fmt.Sprintf("S%02d", i+1)
		seeds[i] = b.Team(division, func(row *store.Team) { row.NameFullJa, row.NameShortJa = name, name }).ID
	}
	input := store.FormatInput{Template: format.TemplatePoolBands,
		Params: format.PoolBandsParams{Teams: 16, Pools: 4, Bands: []format.Band{
			{Ranks: []int{1, 2}, Shape: format.ShapeT8}, {Ranks: []int{3, 4}, Shape: format.ShapeT8}}},
		Seeds: seeds, Tiebreak: format.TiebreakCurrent, Forfeit: format.DefaultForfeitScore(),
		Schedule: twoDaySetting()}

	preview, err := store.PreviewFormat(ctx, db, division.ID, input)
	require.NoError(t, err)
	assert.Empty(t, preview.Violations)
	assert.Nil(t, preview.FitNote)
	require.NotNil(t, preview.EndDay)
	assert.Equal(t, 2, *preview.EndDay)
	applied, err := store.ApplyFormat(ctx, db, domain.Actor{Role: domain.RoleAdmin}, division.ID, input)
	require.NoError(t, err)
	assert.Equal(t, preview.Matches, applied.Matches, "プレビューと作成の時間割が同じ")
	assert.Equal(t, preview.Courts, applied.Courts)

	byDate := map[string]int{}
	for _, m := range applied.Matches {
		require.NotNil(t, m.Date)
		byDate[*m.Date]++
		assert.NotEqual(t, "12:00", *m.Time, "%s が昼休み", m.Code)
	}
	assert.Equal(t, map[string]int{"2026-11-14": 24, "2026-11-15": 24}, byDate,
		"1 日目に予選、2 日目に順位決定戦")

	saved, ok, err := store.GetSavedFormat(ctx, db, division.ID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, twoDaySetting(), saved.Schedule, "開き直したときに保存した条件を出す")
	violations, err := store.CheckFormatSchedule(ctx, db, division.ID)
	require.NoError(t, err)
	assert.Empty(t, violations)

	// 1 日に詰めると収まらない理由が出る
	oneDay := twoDaySetting()
	oneDay.Days, oneDay.StageDays = oneDay.Days[:1], false
	input.Schedule = oneDay
	short, err := store.PreviewFormat(ctx, db, division.ID, input)
	require.NoError(t, err)
	require.NotNil(t, short.FitNote)
	assert.Equal(t, "17:00 までにあと 20 試合が収まりません", *short.FitNote)
	assert.NotEmpty(t, short.Violations)
}

func TestScheduleSettingRejectsBadDays(t *testing.T) {
	cases := map[string]func(s *store.ScheduleSetting){
		"途中の日の終了なし": func(s *store.ScheduleSetting) { s.Days[0].End = nil },
		"日付が逆":      func(s *store.ScheduleSetting) { s.Days[1].Date = "2026-11-13" },
		"昼休みが長すぎる":  func(s *store.ScheduleSetting) { s.Lunch.Minutes = 600 },
		"休みが負":      func(s *store.ScheduleSetting) { s.TeamRestSlots = -1 },
		"競技が不明":     func(s *store.ScheduleSetting) { s.Sport = strp("golf") },
	}
	for name, edit := range cases {
		setting := twoDaySetting()
		edit(setting)
		_, err := setting.Config()
		require.ErrorIs(t, err, domain.ErrBadRequest, name)
	}
	legacy := store.ScheduleSetting{Date: "2026-10-18", Start: "09:00", SlotMinutes: 20, Courts: []string{"1"}}
	cfg, err := legacy.Config()
	require.NoError(t, err, "日数の無い古い設定は 1 日として読む")
	require.Len(t, cfg.Days, 1)
	assert.True(t, cfg.Days[0].End.IsZero())
}
