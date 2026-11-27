package entry_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

func TestCheckLink(t *testing.T) {
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	link := entry.Link{Purpose: entry.LinkTeam, ExpiresAt: now.Add(time.Minute)}
	require.NoError(t, entry.CheckLink(link, now))
	var notFound *domain.NotFoundError
	require.ErrorAs(t, entry.CheckLink(link, now.Add(time.Minute)), &notFound, "expired")
	assert.Equal(t, 72*time.Hour, entry.LinkLifetime)
}

func TestCheckPurpose(t *testing.T) {
	link := entry.Link{Purpose: entry.LinkTeam}
	require.NoError(t, entry.CheckPurpose(link, entry.LinkTeam))
	var notFound *domain.NotFoundError
	require.ErrorAs(t, entry.CheckPurpose(link, entry.LinkMember), &notFound)
}

func TestCheckNumberFree(t *testing.T) {
	members := []entry.Member{{ID: 1, Number: "7"}, {ID: 2, Number: ""}}
	require.NoError(t, entry.CheckNumberFree(members, 1, "7"), "your own number")
	require.NoError(t, entry.CheckNumberFree(members, 0, ""), "no number")
	require.NoError(t, entry.CheckNumberFree(members, 0, "8"))
	require.ErrorIs(t, entry.CheckNumberFree(members, 2, "7"), domain.ErrBadRequest)
}

func TestMemberOf(t *testing.T) {
	members := []entry.Member{{ID: 1}, {ID: 2, LineUserID: ptr(int64(50))}}
	assert.Equal(t, int64(2), entry.MemberOf(members, 50).ID)
	assert.Nil(t, entry.MemberOf(members, 51))
}

func TestCanJoin(t *testing.T) {
	require.NoError(t, entry.CanJoin(entry.StatusApproved))
	require.NoError(t, entry.CanJoin(entry.StatusReturned))
	require.ErrorIs(t, entry.CanJoin(entry.StatusCancelled), domain.ErrConflict)
}

func ptr[T any](value T) *T { return &value }

func TestCheckJoin(t *testing.T) {
	members := []entry.Member{
		{ID: 1, Name: "山田 太郎", Number: "7"},
		{ID: 2, Name: "佐藤 花子", Number: "12", Registered: true, LineUserID: ptr(int64(50))},
		{ID: 3, Name: "鈴木 一郎", Number: ""},
	}
	memberQuestions := []entry.Question{{ID: 9, Scope: entry.ScopeMember, Label: "学年", Kind: entry.KindShortText,
		Required: true}}
	answers := map[int64]string{9: "3年"}

	got, checked, err := entry.CheckJoin(entry.JoinDraft{MemberID: ptr(int64(1)), Name: " 山田 太郎 ", Number: "7",
		Answers: answers}, members, nil, nil, memberQuestions)
	require.NoError(t, err)
	assert.Equal(t, "山田 太郎", got.Name)
	assert.Equal(t, int64(1), *got.MemberID)
	assert.Equal(t, []entry.Answer{{QuestionID: 9, Value: "3年"}}, checked)

	// 新しく足す人は、空いた背番号なら通る
	got, _, err = entry.CheckJoin(entry.JoinDraft{Name: "田中 次郎", Number: "8", Answers: answers}, members, nil, nil,
		memberQuestions)
	require.NoError(t, err)
	assert.Nil(t, got.MemberID)

	// 名簿から選ばなくても、まだ誰も登録していない同じ名前の行があれば、その行で登録する(空白の違いは
	// 見ない。背番号を空にしたら代表が書いた背番号を残す)
	got, _, err = entry.CheckJoin(entry.JoinDraft{Name: "山田太郎", Answers: answers}, members, nil, nil,
		memberQuestions)
	require.NoError(t, err)
	assert.Equal(t, int64(1), *got.MemberID)
	assert.Equal(t, "7", got.Number)
	// 代表の行と、もう結ばれた行には自動で結ばない
	got, _, err = entry.CheckJoin(entry.JoinDraft{Name: "鈴木 一郎", Answers: answers}, members, nil,
		map[int64]bool{3: true}, memberQuestions)
	require.NoError(t, err)
	assert.Nil(t, got.MemberID)
	twins := []entry.Member{members[0], members[1], members[2], {ID: 4, Name: "山田太郎"}}
	got, _, err = entry.CheckJoin(entry.JoinDraft{Name: "山田 太郎", Answers: answers}, twins, nil, nil,
		memberQuestions)
	require.NoError(t, err)
	assert.Nil(t, got.MemberID, "two rows of the same name: which one is not known")

	// もう登録した人は、自分の行を直す(背番号は自分のものと重なってよい)
	mine := members[1]
	got, _, err = entry.CheckJoin(entry.JoinDraft{Name: "佐藤 花子", Number: "12", Answers: answers}, members,
		&mine, nil, memberQuestions)
	require.NoError(t, err)
	assert.Equal(t, int64(2), *got.MemberID)

	cases := []struct {
		name    string
		draft   entry.JoinDraft
		mine    *entry.Member
		wantErr error
	}{
		{"名前が空", entry.JoinDraft{Name: " ", Answers: answers}, nil, domain.ErrBadRequest},
		{"名前に改行", entry.JoinDraft{Name: "田中\nhttps://example.com", Answers: answers}, nil,
			domain.ErrBadRequest},
		{"ほかの人の背番号", entry.JoinDraft{Name: "田中", Number: "12", Answers: answers}, nil, domain.ErrBadRequest},
		{"よそのチームの行", entry.JoinDraft{MemberID: ptr(int64(99)), Name: "田中", Answers: answers}, nil,
			domain.ErrBadRequest},
		{"もう結ばれた行", entry.JoinDraft{MemberID: ptr(int64(2)), Name: "田中", Answers: answers}, nil,
			domain.ErrConflict},
		{"登録済みの人が別の行", entry.JoinDraft{MemberID: ptr(int64(1)), Name: "佐藤", Answers: answers}, &mine,
			domain.ErrBadRequest},
		{"必須の質問", entry.JoinDraft{Name: "田中"}, nil, domain.ErrBadRequest},
		{"代表の行", entry.JoinDraft{MemberID: ptr(int64(3)), Name: "鈴木", Answers: answers}, nil,
			domain.ErrConflict},
	}
	repRows := map[int64]bool{3: true}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := entry.CheckJoin(tc.draft, members, tc.mine, repRows, memberQuestions)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestRepRows(t *testing.T) {
	rep := int64(40)
	detail := entry.Detail{Summary: entry.Summary{RepName: "山田　太郎"}, RepLineUserID: &rep,
		MemberList: []entry.Member{
			{ID: 1, Name: "山田太郎"},
			{ID: 2, Name: "佐藤 花子"},
			{ID: 3, Name: "山田 太郎", Registered: true, LineUserID: ptr(int64(41))},
		}}
	assert.Equal(t, map[int64]bool{1: true}, entry.RepRows(detail, 50), "the rep's row, ignoring spaces")
	assert.Empty(t, entry.RepRows(detail, rep), "the rep can pick their own row")
	detail.RepLineUserID = nil
	assert.Empty(t, entry.RepRows(detail, 50), "no one knows the rep of a desk entry")
}

func TestDecisionNotice(t *testing.T) {
	assert.Equal(t, entry.NoticeApproved, entry.DecisionNotice(entry.ActionApprove))
	assert.Equal(t, entry.NoticeReturned, entry.DecisionNotice(entry.ActionReturn))
	assert.Equal(t, entry.NoticeCancelled, entry.DecisionNotice(entry.ActionCancel))
	assert.Equal(t, entry.NoticeReopened, entry.DecisionNotice(entry.ActionReopen))
}
