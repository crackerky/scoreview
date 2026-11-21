package entry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

func TestCanResubmit(t *testing.T) {
	require.NoError(t, entry.CanResubmit(entry.StatusReturned, domain.TournamentEntryOpen))
	require.NoError(t, entry.CanResubmit(entry.StatusReturned, domain.TournamentEntryClosed),
		"the staff asked for a fix, so closing entries does not block it")
	require.ErrorIs(t, entry.CanResubmit(entry.StatusReturned, domain.TournamentFinished), domain.ErrConflict)
	for _, status := range []entry.Status{entry.StatusPending, entry.StatusApproved, entry.StatusCancelled} {
		require.ErrorIs(t, entry.CanResubmit(status, domain.TournamentEntryOpen), domain.ErrConflict, status)
	}
}

func TestPlanMembers(t *testing.T) {
	current := []entry.Member{
		{ID: 1, Name: "山田 太郎", Number: "7"},
		{ID: 2, Name: "佐藤 花子", Number: "12", Registered: true, LineUserID: ptr(int64(50))},
		{ID: 3, Name: "鈴木 一郎", Number: "3"},
		{ID: 4, Name: "高橋 健", Number: ""},
	}
	drafts := []entry.MemberDraft{
		{Name: "田中 次郎", Number: "8"},
		{Name: "佐藤 花子", Number: "99"},
		{Name: "山田 太郎", Number: "70"},
	}
	plan, err := entry.PlanMembers(current, drafts)
	require.NoError(t, err)
	assert.Equal(t, []entry.PlannedMember{
		// 本人が登録した人は先頭に、本人の名前と背番号のまま(代表が書いた 99 は使わない)
		{ID: 2, Name: "佐藤 花子", Number: "12", SortOrder: 1},
		{ID: 0, Name: "田中 次郎", Number: "8", SortOrder: 2},
		// 同じ名前の行は使い回し、背番号を代表の直しに合わせる
		{ID: 1, Name: "山田 太郎", Number: "70", SortOrder: 3},
	}, plan.Rows)
	assert.Equal(t, []int64{3, 4}, plan.Remove, "rows the representative dropped, unless registered")

	_, err = entry.PlanMembers(current, []entry.MemberDraft{{Name: "田中 次郎", Number: "12"}})
	require.ErrorIs(t, err, domain.ErrBadRequest, "the number of a registered member")

	many := make([]entry.MemberDraft, entry.MaxMembers)
	for i := range many {
		many[i] = entry.MemberDraft{Name: string(rune('あ' + i))}
	}
	_, err = entry.PlanMembers(current, many)
	require.ErrorIs(t, err, domain.ErrBadRequest, "the registered member and the drafts exceed the limit")
}

func TestSameRegistered(t *testing.T) {
	members := []entry.Member{{ID: 1}, {ID: 2, Registered: true}, {ID: 3, Registered: true}}
	assert.True(t, entry.SameRegistered(members, []int64{3, 2}))
	assert.False(t, entry.SameRegistered(members, []int64{2}), "3 registered after the form opened")
	assert.False(t, entry.SameRegistered(members, []int64{2, 3, 1}), "1 is not registered")
	assert.True(t, entry.SameRegistered([]entry.Member{{ID: 1}}, nil))
}
