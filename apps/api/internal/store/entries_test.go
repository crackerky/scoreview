package store_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

type entryFixture struct {
	db         *gorm.DB
	desk       *entry.Service
	tournament store.Tournament
	division   store.Division
	admin      domain.Actor
}

func newEntryFixture(t *testing.T) entryFixture {
	t.Helper()
	db := testdb.New(t)
	b := storetest.New(t, db)
	tournament := b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen })
	division := b.Division(tournament.ID)
	b.Team(division) // 既にいるチームの後ろに並ぶことを確かめる
	user := store.User{Email: "admin@example.com", PasswordHash: "x", Role: domain.RoleAdmin}
	require.NoError(t, db.Create(&user).Error)
	return entryFixture{db: db, desk: entry.NewService(store.NewEntryStore(db), entry.WithRandom(&counter{})),
		tournament: tournament, division: division, admin: domain.Actor{UserID: user.ID, Role: domain.RoleAdmin}}
}

// counter は 1 バイトずつ増える値を返す乱数の代わり。招待コードが申込ごとに違い、しかも決まった値になる。
type counter struct{ next byte }

func (c *counter) Read(buf []byte) (int, error) {
	for i := range buf {
		c.next++
		buf[i] = c.next
	}
	return len(buf), nil
}

func (f entryFixture) draft(name string, answers map[int64]string) entry.Draft {
	return entry.Draft{DivisionID: f.division.ID, FullName: name, ShortName: name, RepName: "山田 太郎",
		RepPhone: "090-0000-0000", Members: []entry.MemberDraft{{Name: "山田 太郎", Number: "7"},
			{Name: "佐藤 花子"}}, Answers: answers}
}

func TestEntryQuestionsReplaceKeepsAnswersOfKeptQuestions(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()
	saved, err := f.desk.SaveQuestions(ctx, f.tournament.ID, []entry.Question{
		{Scope: entry.ScopeTeam, Label: "宿泊", Kind: entry.KindYesNo, Required: true},
		{Scope: entry.ScopeTeam, Label: "弁当", Kind: entry.KindChoice, Options: []string{"要る", "要らない"}},
		{Scope: entry.ScopeMember, Label: "学年", Kind: entry.KindShortText},
	})
	require.NoError(t, err)
	require.Len(t, saved, 3)
	assert.Equal(t, []string{"要る", "要らない"}, saved[1].Options)
	assert.Equal(t, []string{}, saved[0].Options)

	_, err = f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk,
		f.draft("熱海", map[int64]string{saved[0].ID: entry.Yes, saved[1].ID: "要る"}))
	require.NoError(t, err)

	// 弁当を消し、宿泊を後ろへ回し、新しい質問を前に足す
	again, err := f.desk.SaveQuestions(ctx, f.tournament.ID, []entry.Question{
		{Scope: entry.ScopeTeam, Label: "監督の名前", Kind: entry.KindShortText},
		{ID: saved[2].ID, Scope: entry.ScopeMember, Label: "学年", Kind: entry.KindShortText},
		{ID: saved[0].ID, Scope: entry.ScopeTeam, Label: "宿泊の有無", Kind: entry.KindYesNo, Required: true},
	})
	require.NoError(t, err)
	require.Len(t, again, 3)
	assert.Equal(t, "監督の名前", again[0].Label)
	assert.Equal(t, saved[0].ID, again[2].ID)
	assert.Equal(t, 1, again[2].Answered, "the answer to a kept question remains")

	// 答えのある質問の答え方は変えられない
	_, err = f.desk.SaveQuestions(ctx, f.tournament.ID, []entry.Question{
		{ID: saved[0].ID, Scope: entry.ScopeTeam, Label: "宿泊", Kind: entry.KindShortText},
	})
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestEntrySubmitRejectsTheSameTeamTwice(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()
	first, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelLine, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
	assert.Equal(t, entry.StatusPending, first.Status)
	assert.Equal(t, 2, first.Members)
	assert.Len(t, first.InviteCode, 26)

	// 確認の画面から戻って送り直すと、同じ部門に同じ名前の申込が 2 つできる。前後の空白と大文字小文字は見ない
	_, err = f.desk.Submit(ctx, f.tournament.ID, entry.ChannelLine, f.draft(" 熱海ディスクス ", nil))
	require.ErrorIs(t, err, domain.ErrConflict)

	// 取り消せば同じ名前で出し直せる
	_, err = f.desk.Decide(ctx, first.ID, entry.ActionCancel, "", f.admin)
	require.NoError(t, err)
	_, err = f.desk.Submit(ctx, f.tournament.ID, entry.ChannelLine, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
}

func TestEntryLineSubmitNeedsAnOpenTournament(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()
	require.NoError(t, f.db.Model(&store.Tournament{}).Where("id = ?", f.tournament.ID).
		Update("status", domain.TournamentEntryClosed).Error)
	_, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelLine, f.draft("熱海", nil))
	require.ErrorIs(t, err, domain.ErrConflict)
	_, err = f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("熱海", nil))
	require.NoError(t, err, "the office can still enter a late application")
}

func TestEntryApproveCreatesTeamAndPlayers(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()
	submitted, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)

	approved, err := f.desk.Decide(ctx, submitted.ID, entry.ActionApprove, "", f.admin)
	require.NoError(t, err)
	assert.Equal(t, entry.StatusApproved, approved.Status)
	require.NotNil(t, approved.TeamID)
	require.NotNil(t, approved.DecidedAt)

	team, err := store.GetAdminTeam(ctx, f.db, *approved.TeamID)
	require.NoError(t, err)
	assert.Equal(t, "熱海ディスクス", team.FullName.Ja)
	assert.Equal(t, f.division.ID, team.DivisionID)
	teams, err := store.ListTeams(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	assert.Equal(t, team.ID, teams[len(teams)-1].ID, "the approved team comes after the existing ones")

	players, err := store.ListPlayers(ctx, f.db, team.ID)
	require.NoError(t, err)
	require.Len(t, players, 2)
	assert.Equal(t, "山田 太郎", players[0].Name.Ja)
	assert.Equal(t, "7", players[0].Number)
	for i, member := range approved.MemberList {
		require.NotNil(t, member.PlayerID)
		assert.Equal(t, players[i].ID, *member.PlayerID)
	}

	// 2 回目の承認(二重押し)はチームを増やさない
	_, err = f.desk.Decide(ctx, submitted.ID, entry.ActionApprove, "", f.admin)
	require.ErrorIs(t, err, domain.ErrConflict)
	again, err := store.ListTeams(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	assert.Len(t, again, len(teams))
}

func TestEntryReturnKeepsTheNote(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()
	submitted, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("熱海", nil))
	require.NoError(t, err)
	returned, err := f.desk.Decide(ctx, submitted.ID, entry.ActionReturn, "背番号を入れてください", f.admin)
	require.NoError(t, err)
	assert.Equal(t, entry.StatusReturned, returned.Status)
	assert.Equal(t, "背番号を入れてください", returned.ReturnNote)
	assert.Nil(t, returned.TeamID)
}

func TestEntryDecideNotFound(t *testing.T) {
	f := newEntryFixture(t)
	_, err := f.desk.Decide(context.Background(), 999999, entry.ActionApprove, "", f.admin)
	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestEntryListQueryCountDoesNotGrow(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()
	count := countQueries(t, f.db)
	measure := func() int64 {
		count.Store(0)
		_, err := f.desk.Entries(ctx, f.tournament.ID)
		require.NoError(t, err)
		return count.Load()
	}
	_, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("チーム0", nil))
	require.NoError(t, err)
	before := measure()
	for i := 1; i <= 5; i++ {
		_, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("チーム"+strconv.Itoa(i), nil))
		require.NoError(t, err)
	}
	list, err := f.desk.Entries(ctx, f.tournament.ID)
	require.NoError(t, err)
	require.Len(t, list, 6)
	assert.Equal(t, 2, list[5].Members)
	assert.Equal(t, 0, list[5].Registered)
	assert.Equal(t, before, measure())
}

func TestEntryExportReadsAllEntriesWithConstantQueries(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()
	questions, err := f.desk.SaveQuestions(ctx, f.tournament.ID, []entry.Question{
		{Scope: entry.ScopeTeam, Label: "宿泊", Kind: entry.KindYesNo},
	})
	require.NoError(t, err)
	count := countQueries(t, f.db)
	measure := func() (entry.Export, int64) {
		count.Store(0)
		export, err := f.desk.Export(ctx, f.tournament.ID)
		require.NoError(t, err)
		return export, count.Load()
	}
	_, err = f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk,
		f.draft("チーム0", map[int64]string{questions[0].ID: entry.Yes}))
	require.NoError(t, err)
	_, before := measure()
	for i := 1; i <= 4; i++ {
		_, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("チーム"+strconv.Itoa(i), nil))
		require.NoError(t, err)
	}
	export, after := measure()
	assert.Equal(t, before, after)
	require.Len(t, export.Questions, 1)
	require.Len(t, export.Entries, 5)
	assert.Equal(t, "チーム0", export.Entries[0].FullName)
	assert.Equal(t, []entry.Answer{{QuestionID: questions[0].ID, Value: entry.Yes}}, export.Entries[0].Answers)
	assert.Len(t, export.Entries[4].MemberList, 2)
	assert.Equal(t, []entry.Answer{}, export.Entries[4].Answers)

	_, err = f.desk.Export(ctx, 999999)
	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)
}
