package entry_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

func TestCheckQuestions(t *testing.T) {
	current := []entry.Question{
		{ID: 1, Scope: entry.ScopeTeam, Label: "宿泊", Kind: entry.KindYesNo, Answered: 3},
		{ID: 2, Scope: entry.ScopeMember, Label: "学年", Kind: entry.KindShortText},
	}
	cases := []struct {
		name      string
		questions []entry.Question
		wantErr   error
	}{
		{"選択肢を整える", []entry.Question{{Scope: entry.ScopeTeam, Label: " 弁当 ", Kind: entry.KindChoice,
			Options: []string{" 要る", "要らない "}}}, nil},
		{"答えの無い質問は答え方を変えられる", []entry.Question{{ID: 2, Scope: entry.ScopeTeam, Label: "学年",
			Kind: entry.KindNumber}}, nil},
		{"答えのある質問は答え方を変えられない", []entry.Question{{ID: 1, Scope: entry.ScopeTeam, Label: "宿泊",
			Kind: entry.KindShortText}}, domain.ErrConflict},
		{"答えのある質問の文言は変えられる", []entry.Question{{ID: 1, Scope: entry.ScopeTeam, Label: "宿泊の有無",
			Kind: entry.KindYesNo}}, nil},
		{"よその質問 ID", []entry.Question{{ID: 99, Scope: entry.ScopeTeam, Label: "x",
			Kind: entry.KindYesNo}}, domain.ErrBadRequest},
		{"同じ ID を 2 回", []entry.Question{
			{ID: 2, Scope: entry.ScopeMember, Label: "a", Kind: entry.KindShortText},
			{ID: 2, Scope: entry.ScopeMember, Label: "b", Kind: entry.KindShortText}}, domain.ErrBadRequest},
		{"ラベルが空", []entry.Question{{Scope: entry.ScopeTeam, Label: "  ", Kind: entry.KindYesNo}},
			domain.ErrBadRequest},
		{"選択肢の無い選択式", []entry.Question{{Scope: entry.ScopeTeam, Label: "x", Kind: entry.KindChoice}},
			domain.ErrBadRequest},
		{"選択肢の重なり", []entry.Question{{Scope: entry.ScopeTeam, Label: "x", Kind: entry.KindChoice,
			Options: []string{"a", " a"}}}, domain.ErrBadRequest},
		{"選択式でないのに選択肢", []entry.Question{{Scope: entry.ScopeTeam, Label: "x",
			Kind: entry.KindNumber, Options: []string{"a"}}}, domain.ErrBadRequest},
		{"知らない相手", []entry.Question{{Scope: "coach", Label: "x", Kind: entry.KindYesNo}},
			domain.ErrBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := entry.CheckQuestions(tc.questions, current)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, len(tc.questions))
			assert.Equal(t, strings.TrimSpace(tc.questions[0].Label), got[0].Label)
			assert.NotNil(t, got[0].Options)
		})
	}
	got, err := entry.CheckQuestions(cases[0].questions, current)
	require.NoError(t, err)
	assert.Equal(t, "弁当", got[0].Label)
	assert.Equal(t, []string{"要る", "要らない"}, got[0].Options)
}

func TestCheckQuestionsLimit(t *testing.T) {
	many := make([]entry.Question, entry.MaxQuestions+1)
	for i := range many {
		many[i] = entry.Question{Scope: entry.ScopeTeam, Label: "x", Kind: entry.KindYesNo}
	}
	_, err := entry.CheckQuestions(many, nil)
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

var tournament = entry.Tournament{ID: 7, Status: domain.TournamentEntryOpen,
	Divisions: []entry.Division{{ID: 70}, {ID: 71}}}

var questions = []entry.Question{
	{ID: 1, Scope: entry.ScopeTeam, Label: "宿泊", Kind: entry.KindYesNo, Required: true},
	{ID: 2, Scope: entry.ScopeTeam, Label: "弁当", Kind: entry.KindChoice, Options: []string{"要る", "要らない"}},
	{ID: 3, Scope: entry.ScopeTeam, Label: "人数", Kind: entry.KindNumber},
	{ID: 4, Scope: entry.ScopeMember, Label: "学年", Kind: entry.KindShortText, Required: true},
}

func draft() entry.Draft {
	return entry.Draft{DivisionID: 70, FullName: " 熱海ディスクス ", ShortName: "熱海", RepName: "山田 太郎",
		Members: []entry.MemberDraft{{Name: " 山田 太郎", Number: "7"}, {Name: "佐藤 花子", Number: ""},
			{Name: "鈴木 一郎", Number: "00"}},
		Answers: map[int64]string{1: "yes", 2: "要る"}}
}

func TestCheckDraftTrimsAndOrdersAnswers(t *testing.T) {
	got, answers, err := entry.CheckDraft(draft(), tournament, questions)
	require.NoError(t, err)
	assert.Equal(t, "熱海ディスクス", got.FullName)
	assert.Equal(t, "山田 太郎", got.Members[0].Name)
	// 任意で答えの無い質問(3)は返さない。メンバー向けの必須(4)は代表の申込では問わない
	assert.Equal(t, []entry.Answer{{QuestionID: 1, Value: "yes"}, {QuestionID: 2, Value: "要る"}}, answers)
}

func TestCheckDraftRejects(t *testing.T) {
	cases := []struct {
		name   string
		change func(*entry.Draft)
	}{
		{"よその部門", func(d *entry.Draft) { d.DivisionID = 99 }},
		{"チーム名が空", func(d *entry.Draft) { d.FullName = " " }},
		{"代表の名前が空", func(d *entry.Draft) { d.RepName = "" }},
		{"メンバーの名前が空", func(d *entry.Draft) { d.Members[1].Name = "" }},
		{"背番号の重なり", func(d *entry.Draft) { d.Members[2].Number = "7" }},
		{"必須の答えが無い", func(d *entry.Draft) { delete(d.Answers, 1) }},
		{"はい/いいえ以外", func(d *entry.Draft) { d.Answers[1] = "maybe" }},
		{"選択肢に無い", func(d *entry.Draft) { d.Answers[2] = "大盛り" }},
		{"数でない", func(d *entry.Draft) { d.Answers[3] = "3.5" }},
		{"数が大きすぎる", func(d *entry.Draft) { d.Answers[3] = "10000000" }},
		{"メンバー向けの質問に答えた", func(d *entry.Draft) { d.Answers[4] = "3年" }},
		{"知らない質問", func(d *entry.Draft) { d.Answers[99] = "x" }},
		{"名前が長すぎる", func(d *entry.Draft) { d.FullName = strings.Repeat("あ", entry.MaxShortText+1) }},
		{"チーム名に改行", func(d *entry.Draft) { d.FullName = "熱海\n運営からのお知らせ" }},
		{"メンバーの名前に制御文字", func(d *entry.Draft) { d.Members[0].Name = "山田\u0007" }},
		{"背番号が長すぎる", func(d *entry.Draft) {
			d.Members[0].Number = strings.Repeat("1", entry.MaxNumberText+1)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := draft()
			tc.change(&value)
			_, _, err := entry.CheckDraft(value, tournament, questions)
			require.ErrorIs(t, err, domain.ErrBadRequest)
		})
	}
}

func TestCanSubmit(t *testing.T) {
	cases := []struct {
		status  domain.TournamentStatus
		channel entry.Channel
		wantErr error
	}{
		{domain.TournamentEntryOpen, entry.ChannelLine, nil},
		{domain.TournamentEntryClosed, entry.ChannelLine, domain.ErrConflict},
		{domain.TournamentEntryClosed, entry.ChannelDesk, nil},
		{domain.TournamentFinished, entry.ChannelDesk, domain.ErrConflict},
		{domain.TournamentEntryOpen, "mail", domain.ErrBadRequest},
	}
	for _, tc := range cases {
		err := entry.CanSubmit(tc.status, tc.channel)
		if tc.wantErr == nil {
			assert.NoError(t, err, "%s %s", tc.status, tc.channel)
		} else {
			assert.ErrorIs(t, err, tc.wantErr, "%s %s", tc.status, tc.channel)
		}
	}
}

func TestNext(t *testing.T) {
	cases := []struct {
		from    entry.Status
		action  entry.Action
		want    entry.Status
		wantErr error
	}{
		{entry.StatusPending, entry.ActionApprove, entry.StatusApproved, nil},
		{entry.StatusPending, entry.ActionReturn, entry.StatusReturned, nil},
		{entry.StatusPending, entry.ActionCancel, entry.StatusCancelled, nil},
		{entry.StatusReturned, entry.ActionCancel, entry.StatusCancelled, nil},
		// 差し戻した申込は代表が直して送り直すまで承認しない
		{entry.StatusReturned, entry.ActionApprove, "", domain.ErrConflict},
		{entry.StatusApproved, entry.ActionApprove, "", domain.ErrConflict},
		{entry.StatusApproved, entry.ActionCancel, "", domain.ErrConflict},
		{entry.StatusCancelled, entry.ActionReturn, "", domain.ErrConflict},
		{entry.StatusPending, "delete", "", domain.ErrBadRequest},
	}
	for _, tc := range cases {
		got, err := entry.Next(tc.from, tc.action)
		if tc.wantErr != nil {
			assert.ErrorIs(t, err, tc.wantErr, "%s %s", tc.from, tc.action)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}

func TestCheckNote(t *testing.T) {
	note, err := entry.CheckNote(entry.ActionReturn, " 背番号を入れてください ")
	require.NoError(t, err)
	assert.Equal(t, "背番号を入れてください", note)
	_, err = entry.CheckNote(entry.ActionReturn, " ")
	require.ErrorIs(t, err, domain.ErrBadRequest)
	_, err = entry.CheckNote(entry.ActionApprove, "ok")
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

func TestApprovalOf(t *testing.T) {
	detail := entry.Detail{Summary: entry.Summary{Division: entry.Division{ID: 70}, FullName: "熱海ディスクス",
		ShortName: "熱海"}, MemberList: []entry.Member{{ID: 5, Name: "山田 太郎", Number: "7"}}}
	assert.Equal(t, entry.Approval{DivisionID: 70, FullName: "熱海ディスクス", ShortName: "熱海",
		Players: []entry.ApprovedPlayer{{MemberID: 5, Name: "山田 太郎", Number: "7"}}}, entry.ApprovalOf(detail))
}

func TestNewInviteCode(t *testing.T) {
	code, err := entry.NewInviteCode(bytes.NewReader(make([]byte, 16)))
	require.NoError(t, err)
	assert.Equal(t, strings.Repeat("a", 26), code)
	_, err = entry.NewInviteCode(bytes.NewReader(make([]byte, 3)))
	require.Error(t, err)
	assert.False(t, errors.Is(err, domain.ErrBadRequest))
}

func TestIsInviteCode(t *testing.T) {
	code, err := entry.NewInviteCode(bytes.NewReader(bytes.Repeat([]byte{0xa7}, 16)))
	require.NoError(t, err)
	assert.True(t, entry.IsInviteCode(code), code)
	for _, bad := range []string{"", strings.Repeat("a", 25), strings.Repeat("a", 27), strings.Repeat("A", 26),
		strings.Repeat("a", 25) + "1", strings.Repeat("a", 25) + "8"} {
		assert.False(t, entry.IsInviteCode(bad), bad)
	}
}

func TestCanReturn(t *testing.T) {
	rep := int64(3)
	require.NoError(t, entry.CanReturn(entry.ActionReturn, entry.Detail{RepLineUserID: &rep}))
	require.ErrorIs(t, entry.CanReturn(entry.ActionReturn, entry.Detail{}), domain.ErrConflict,
		"no one could fix a desk entry")
	require.NoError(t, entry.CanReturn(entry.ActionCancel, entry.Detail{}))
	require.NoError(t, entry.CanReturn(entry.ActionApprove, entry.Detail{}))
}
