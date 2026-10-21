package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParsePlaceholder(t *testing.T) {
	assert.Equal(t, &Placeholder{Code: "A1", Outcome: "win"}, parsePlaceholder("[A1-勝ち]"))
	assert.Equal(t, &Placeholder{Code: "E9", Outcome: "lose"}, parsePlaceholder("[E9-負け]"))
	assert.Equal(t, &Placeholder{Code: "S11", Outcome: "win"}, parsePlaceholder("[S11-Win]"))
	assert.Equal(t, &Placeholder{Code: "R15", Outcome: "lose"}, parsePlaceholder("[R15-Lose]"))
	assert.Nil(t, parsePlaceholder("Loquitos"))
}

func TestParseScheduleDivision(t *testing.T) {
	s, err := ParseSchedule(fixture(t, "schedule_div949_jp.html"), site.LangJP)
	require.NoError(t, err)

	assert.Equal(t, "division", s.Scope)
	assert.Equal(t, int64(949), s.ScopeID)
	assert.Equal(t, []string{"2025-10-18"}, s.Dates)
	// div 版のチーム選択肢は value がチーム ID、label が正式名(clone-spec B 4)
	require.NotEmpty(t, s.TeamOptions)
	assert.Equal(t, ScheduleTeamOption{Value: "17491", Label: Text{JA: "雨ニモマケズ"}}, s.TeamOptions[0])

	require.Len(t, s.Rows, 3)
	assert.Equal(t, MatchRow{
		MatchID: ptr(int64(45280)), HasDetail: true, Status: "finished",
		DivisionKind: "mixed", DivisionLabel: &Text{JA: "ビーチミッ"},
		Code: "A4", MonthDay: "10/18", Date: ptr("2025-10-18"), Time: ptr("09:00"),
		Field: "1", FieldNo: ptr(1),
		Left:      MatchRowSide{TeamID: ptr(int64(17322)), ShortName: Text{JA: "とびたーず"}},
		Right:     MatchRowSide{TeamID: ptr(int64(17498)), ShortName: Text{JA: "バトルロイヤル"}},
		LeftScore: ptr(2), RightScore: ptr(12),
	}, s.Rows[0])
}

func TestParseScheduleMergeEN(t *testing.T) {
	s := parseBoth(t, "schedule_tn525_jp.html", "schedule_tn525_en.html", ParseSchedule, MergeScheduleEN)

	assert.Equal(t, "tournament", s.Scope)
	assert.Equal(t, int64(525), s.ScopeID)
	require.Len(t, s.Rows, 3)
	assert.Equal(t, Text{JA: "とびたーず", EN: "Tobitazu"}, s.Rows[0].Left.ShortName)
	assert.Equal(t, &Text{JA: "ビーチミッ", EN: "Beach Mixe"}, s.Rows[0].DivisionLabel)
	// tn 版の選択肢は value がチーム名なので英語名を突き合わせない
	assert.Equal(t, "", s.TeamOptions[0].Label.EN)
}

func TestMergeScheduleRejectsDifferentRows(t *testing.T) {
	jp, err := ParseSchedule(fixture(t, "schedule_tn525_jp.html"), site.LangJP)
	require.NoError(t, err)
	en, err := ParseSchedule(fixture(t, "schedule_tn525_en.html"), site.LangEN)
	require.NoError(t, err)
	en.Rows[1].Code = "Z9"
	_, err = MergeScheduleEN(jp, en)
	assert.ErrorIs(t, err, ErrMismatch)
}

// 2017 年の旧形式: 状況「終了」(試合詳細なし)と MC_MEN(clone-spec B 6)
func TestParseScheduleOldFormat(t *testing.T) {
	s, err := ParseSchedule(fixture(t, "schedule_tn151_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotEmpty(t, s.Rows)
	kinds := map[string]bool{}
	for _, r := range s.Rows {
		assert.Equal(t, "finished", r.Status)
		assert.False(t, r.HasDetail, "%s は「終了」の行", r.Code)
		assert.Nil(t, r.MatchID)
		assert.NotNil(t, r.LeftScore)
		kinds[r.DivisionKind] = true
	}
	assert.True(t, kinds["men"], "MC_MEN の行がある")
	assert.True(t, kinds["women"])
	assert.Equal(t, []string{"2017-03-10", "2017-03-11", "2017-03-12"}, s.Dates)
}

// 未確定チーム [A1-勝ち] / [A1-Win](clone-spec B 8)
func TestParseSchedulePlaceholders(t *testing.T) {
	jp, err := ParseSchedule(fixture(t, "schedule_tn440_jp.html"), site.LangJP)
	require.NoError(t, err)
	var found *MatchRow
	for i := range jp.Rows {
		if jp.Rows[i].Left.Placeholder != nil {
			found = &jp.Rows[i]
			break
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "scheduled", found.Status)
	assert.Nil(t, found.Left.TeamID)
	assert.Equal(t, Text{}, found.Left.ShortName)
	assert.Nil(t, found.LeftScore)
	assert.Nil(t, found.MatchID)
	assert.True(t, found.HasDetail, "未試合は「終了」ではない")
	assert.Equal(t, "S2", found.Code)
	assert.Equal(t, &Placeholder{Code: "S5", Outcome: "win"}, found.Left.Placeholder)

	en, err := ParseSchedule(fixture(t, "schedule_s440_en.html"), site.LangEN)
	require.NoError(t, err)
	outcomes := map[string]bool{}
	for _, r := range en.Rows {
		for _, ph := range []*Placeholder{r.Left.Placeholder, r.Right.Placeholder} {
			if ph != nil {
				outcomes[ph.Outcome] = true
			}
		}
	}
	assert.Equal(t, map[string]bool{"win": true, "lose": true}, outcomes)
}

// 172: 日付選択肢に 2017/08/10 と 2018/08/10 があり、08/10 の年を決められない
func TestParseScheduleAmbiguousYear(t *testing.T) {
	s, err := ParseSchedule(fixture(t, "schedule_tn172_jp.html"), site.LangJP)
	require.NoError(t, err)
	var wl6 *MatchRow
	for i := range s.Rows {
		if s.Rows[i].Code == "WL6" {
			wl6 = &s.Rows[i]
		}
	}
	require.NotNil(t, wl6)
	assert.Nil(t, wl6.Date)
	assert.Equal(t, []string{"2017-08-10", "2018-08-10"}, wl6.DateCandidates)
}

// 467: コートが "A" のような文字(clone-spec に未記載の変種)
func TestParseScheduleNonNumericField(t *testing.T) {
	s, err := ParseSchedule(fixture(t, "schedule_div467_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotEmpty(t, s.Rows)
	assert.Equal(t, "A", s.Rows[0].Field)
	assert.Nil(t, s.Rows[0].FieldNo)
	assert.Equal(t, "none", s.Rows[0].DivisionKind, "オープンには種別クラスがない")
}
