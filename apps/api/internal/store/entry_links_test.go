package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
)

type linkFixture struct {
	entryFixture
	now  *time.Time
	desk *entry.Service
}

func newLinkFixture(t *testing.T) linkFixture {
	t.Helper()
	base := newEntryFixture(t)
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	clock := &now
	desk := entry.NewService(store.NewEntryStore(base.db), entry.WithRandom(&counter{}),
		entry.WithClock(func() time.Time { return *clock }),
		entry.WithInviteURL(func(code string) string { return "invite:" + code }))
	return linkFixture{entryFixture: base, now: clock, desk: desk}
}

// lineUser は公式アカウントの友だちの行を作り、その ID を返す(Webhook を受けたときに talk が作る行)。
func (f linkFixture) lineUser(t *testing.T, id string) int64 {
	t.Helper()
	row := store.LineUser{LineUserID: id, Friend: true, CreatedAt: *f.now, UpdatedAt: *f.now}
	require.NoError(t, f.db.Create(&row).Error)
	return row.ID
}

// teamLink は lineUser に代表の申込のリンクを発行し、大会 tournamentID のリンクを返す。
func (f linkFixture) teamLink(t *testing.T, lineUser, tournamentID int64) entry.TournamentLink {
	t.Helper()
	links, err := f.desk.IssueTeamLinks(context.Background(), lineUser)
	require.NoError(t, err)
	for _, link := range links {
		if link.Tournament.ID == tournamentID {
			return link
		}
	}
	require.FailNow(t, "no link for the tournament", "tournament %d in %v", tournamentID, links)
	return entry.TournamentLink{}
}

func linkedTournaments(t *testing.T, f linkFixture, lineUser int64) []int64 {
	t.Helper()
	links, err := f.desk.IssueTeamLinks(context.Background(), lineUser)
	require.NoError(t, err)
	ids := make([]int64, len(links))
	for i, link := range links {
		ids[i] = link.Tournament.ID
	}
	return ids
}

func TestEntryLinkTeamFlow(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	rep := f.lineUser(t, "Urep")

	link := f.teamLink(t, rep, f.tournament.ID)
	assert.Equal(t, f.now.Add(entry.LinkLifetime), link.ExpiresAt)

	view, err := f.desk.LinkView(ctx, link.Token)
	require.NoError(t, err)
	require.NotNil(t, view.Team)
	assert.True(t, view.Team.Open)
	assert.Empty(t, view.Team.Mine)

	submitted, err := f.desk.SubmitByLink(ctx, link.Token, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
	assert.Equal(t, entry.ChannelLine, submitted.Channel)
	assert.Equal(t, "invite:"+submitted.InviteCode, submitted.InviteURL)

	view, err = f.desk.LinkView(ctx, link.Token)
	require.NoError(t, err)
	require.Len(t, view.Team.Mine, 1, "the representative sees what they submitted")
	assert.Equal(t, submitted.ID, view.Team.Mine[0].ID)

	// ほかの人のリンクからは見えない
	other := f.teamLink(t, f.lineUser(t, "Uother"), f.tournament.ID)
	view, err = f.desk.LinkView(ctx, other.Token)
	require.NoError(t, err)
	assert.Empty(t, view.Team.Mine)

	// 期限が切れたら見つからないのと同じ
	*f.now = f.now.Add(entry.LinkLifetime)
	_, err = f.desk.LinkView(ctx, link.Token)
	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)
	_, err = f.desk.SubmitByLink(ctx, link.Token, f.draft("別のチーム", nil))
	require.ErrorAs(t, err, &notFound)
	_, err = f.desk.LinkView(ctx, "unknown")
	require.ErrorAs(t, err, &notFound)
}

func TestEntryLinkTournaments(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	b := storetest.New(t, f.db)
	b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryClosed })
	b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen; row.Listed = false })
	rep, stranger := f.lineUser(t, "Urep"), f.lineUser(t, "Ustranger")
	assert.Equal(t, []int64{f.tournament.ID}, linkedTournaments(t, f, rep),
		"only open, listed tournaments for someone who has not applied")

	// 受付を締めた後も、申込を出した代表には、その大会のリンクを出す(承認や差し戻しを見るため)
	_, err := f.desk.SubmitByLink(ctx, f.teamLink(t, rep, f.tournament.ID).Token, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
	require.NoError(t, f.db.Model(&store.Tournament{}).Where("id = ?", f.tournament.ID).
		Update("status", domain.TournamentEntryClosed).Error)
	assert.Equal(t, []int64{f.tournament.ID}, linkedTournaments(t, f, rep))
	assert.Empty(t, linkedTournaments(t, f, stranger))
	view, err := f.desk.LinkView(ctx, f.teamLink(t, rep, f.tournament.ID).Token)
	require.NoError(t, err)
	assert.False(t, view.Team.Open)
	assert.Len(t, view.Team.Mine, 1)

	// 大会が終わったら出さない
	require.NoError(t, f.db.Model(&store.Tournament{}).Where("id = ?", f.tournament.ID).
		Update("status", domain.TournamentFinished).Error)
	assert.Empty(t, linkedTournaments(t, f, rep))
}

func TestEntryLinkResubmit(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	rep := f.lineUser(t, "Urep")
	link := f.teamLink(t, rep, f.tournament.ID)
	draft := f.draft("熱海ディスクス", nil)
	draft.Members = append(draft.Members, entry.MemberDraft{Name: "鈴木 一郎", Number: "3"})
	submitted, err := f.desk.SubmitByLink(ctx, link.Token, draft)
	require.NoError(t, err)
	_, err = f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("伊東", nil))
	require.NoError(t, err)

	// 佐藤 花子が本人登録してから、運営が差し戻す
	member, err := f.desk.IssueMemberLink(ctx, f.lineUser(t, "Uhanako"), submitted.InviteCode)
	require.NoError(t, err)
	sato := submitted.MemberList[1].ID
	_, err = f.desk.JoinByLink(ctx, member.Token, entry.JoinDraft{MemberID: &sato, Name: "佐藤 花子", Number: "12"})
	require.NoError(t, err)
	seen := []int64{sato} // 代表が送り直しのフォームを開いたときに、本人が登録していた人
	_, err = f.desk.ResubmitByLink(ctx, link.Token, submitted.ID, draft, seen)
	require.ErrorIs(t, err, domain.ErrConflict, "a pending entry is not resubmitted")
	_, err = f.desk.Decide(ctx, submitted.ID, entry.ActionReturn, "チーム名を正式名に", f.admin)
	require.NoError(t, err)

	// ほかの人のリンクからは、見つからないのと同じ
	var notFound *domain.NotFoundError
	_, err = f.desk.ResubmitByLink(ctx, f.teamLink(t, f.lineUser(t, "Uother"), f.tournament.ID).Token,
		submitted.ID, draft, seen)
	require.ErrorAs(t, err, &notFound)
	// 同じ部門のほかの申込と同じ名前には直せない
	renamed := draft
	renamed.FullName = "伊東"
	_, err = f.desk.ResubmitByLink(ctx, link.Token, submitted.ID, renamed, seen)
	require.ErrorIs(t, err, domain.ErrConflict)
	// フォームを開いた後に本人の登録が変わっていたら(ここでは開いたときに佐藤の登録を知らなかった)、
	// 代表が書いた名前と突き合わせられないので、開き直させる
	_, err = f.desk.ResubmitByLink(ctx, link.Token, submitted.ID, draft, nil)
	require.ErrorIs(t, err, domain.ErrConflict)

	fixed := draft
	fixed.FullName = "熱海ディスクス フライングクラブ"
	fixed.Members = []entry.MemberDraft{{Name: "田中 次郎", Number: "8"}, {Name: "山田 太郎", Number: "70"}}
	resubmitted, err := f.desk.ResubmitByLink(ctx, link.Token, submitted.ID, fixed, seen)
	require.NoError(t, err)
	assert.Equal(t, entry.StatusPending, resubmitted.Status)
	assert.Empty(t, resubmitted.ReturnNote, "a pending entry has no return note")
	assert.Nil(t, resubmitted.DecidedAt)
	assert.Equal(t, "熱海ディスクス フライングクラブ", resubmitted.FullName)
	assert.Equal(t, submitted.InviteCode, resubmitted.InviteCode, "the invite stays the same")
	got := map[string]string{}
	for _, m := range resubmitted.MemberList {
		got[m.Name] = m.Number
	}
	assert.Equal(t, map[string]string{"佐藤 花子": "12", "田中 次郎": "8", "山田 太郎": "70"}, got,
		"the registered member stays; 鈴木 is dropped")
	assert.Equal(t, sato, resubmitted.MemberList[0].ID)
	assert.True(t, resubmitted.MemberList[0].Registered)
	assert.Equal(t, submitted.MemberList[0].ID, resubmitted.MemberList[2].ID, "山田's row is reused")
}

func TestEntryStoreJoinChecksNumbersInTheLock(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	team, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
	// entry.CheckJoin を通った後に、別の人が背番号 7 で登録していた場合と同じ(山田 太郎の 7)
	err = store.NewEntryStore(f.db).Join(ctx, entry.Join{TeamEntryID: team.ID, LineUserID: f.lineUser(t, "U1"),
		Name: "田中 次郎", Number: "7", At: *f.now})
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

func TestEntryStoreApprovalNoticesRenamedMembers(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	team, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
	approval := entry.ApprovalOf(team)
	// 運営が読んだ後、承認の前に、佐藤 花子が自分の行の名前を直した
	link, err := f.desk.IssueMemberLink(ctx, f.lineUser(t, "Uhanako"), team.InviteCode)
	require.NoError(t, err)
	sato := team.MemberList[1].ID
	_, err = f.desk.JoinByLink(ctx, link.Token, entry.JoinDraft{MemberID: &sato, Name: "佐藤 はなこ"})
	require.NoError(t, err)
	_, err = store.NewEntryStore(f.db).Decide(ctx, team.ID, entry.Decision{From: entry.StatusPending,
		To: entry.StatusApproved, DecidedBy: f.admin.UserID, At: *f.now, Approval: &approval})
	require.ErrorIs(t, err, domain.ErrConflict, "the players would have the old name")
}

func TestEntryLinkMemberJoin(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	questions, err := f.desk.SaveQuestions(ctx, f.tournament.ID, []entry.Question{
		{Scope: entry.ScopeMember, Label: "学年", Kind: entry.KindShortText, Required: true}})
	require.NoError(t, err)
	grade := questions[0].ID
	team, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
	yamada := team.MemberList[0].ID

	hanako := f.lineUser(t, "Uhanako")
	link, err := f.desk.IssueMemberLink(ctx, hanako, team.InviteCode)
	require.NoError(t, err)
	assert.Equal(t, "熱海ディスクス", link.TeamName)
	view, err := f.desk.LinkView(ctx, link.Token)
	require.NoError(t, err)
	require.NotNil(t, view.Member)
	assert.Len(t, view.Member.Unclaimed, 2)
	assert.Nil(t, view.Member.Mine)

	// 代表が書いた行(佐藤 花子)を自分の行として登録する
	sato := team.MemberList[1].ID
	form, err := f.desk.JoinByLink(ctx, link.Token, entry.JoinDraft{MemberID: &sato, Name: "佐藤 花子",
		Number: "12", Answers: map[int64]string{grade: "3年"}})
	require.NoError(t, err)
	require.NotNil(t, form.Mine)
	assert.Equal(t, sato, form.Mine.ID)
	assert.Equal(t, "12", form.Mine.Number)
	assert.Equal(t, []entry.Answer{{QuestionID: grade, Value: "3年"}}, form.MineAnswers)
	assert.Len(t, form.Unclaimed, 1)

	list, err := f.desk.Entries(ctx, f.tournament.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, list[0].Registered, "the progress counts the member")

	// 同じ人がもう一度開いて直す(行は増えない)
	form, err = f.desk.JoinByLink(ctx, link.Token, entry.JoinDraft{Name: "佐藤 はなこ", Number: "13",
		Answers: map[int64]string{grade: "3年"}})
	require.NoError(t, err)
	assert.Equal(t, sato, form.Mine.ID)
	detail, err := f.desk.Entry(ctx, team.ID)
	require.NoError(t, err)
	assert.Len(t, detail.MemberList, 2)

	// ほかの人は、結ばれた行を選べない
	taro := f.lineUser(t, "Utaro")
	other, err := f.desk.IssueMemberLink(ctx, taro, team.InviteCode)
	require.NoError(t, err)
	_, err = f.desk.JoinByLink(ctx, other.Token, entry.JoinDraft{MemberID: &sato, Name: "佐藤",
		Answers: map[int64]string{grade: "1年"}})
	require.ErrorIs(t, err, domain.ErrConflict)

	// 承認した後の登録は選手にも写る(代表の行を選べば選手を直し、新しい人なら選手を作る)
	approved, err := f.desk.Decide(ctx, team.ID, entry.ActionApprove, "", f.admin)
	require.NoError(t, err)
	_, err = f.desk.JoinByLink(ctx, other.Token, entry.JoinDraft{MemberID: &yamada, Name: "山田 太郎",
		Number: "77", Answers: map[int64]string{grade: "2年"}})
	require.NoError(t, err)
	jiro := f.lineUser(t, "Ujiro")
	third, err := f.desk.IssueMemberLink(ctx, jiro, team.InviteCode)
	require.NoError(t, err)
	_, err = f.desk.JoinByLink(ctx, third.Token, entry.JoinDraft{Name: "田中 次郎", Number: "5",
		Answers: map[int64]string{grade: "1年"}})
	require.NoError(t, err)
	players, err := store.ListPlayers(ctx, f.db, *approved.TeamID)
	require.NoError(t, err)
	byName := map[string]string{}
	for _, player := range players {
		byName[player.Name.Ja] = player.Number
	}
	assert.Equal(t, map[string]string{"山田 太郎": "77", "佐藤 はなこ": "13", "田中 次郎": "5"}, byName,
		"佐藤's edit before approval is in the player; edits after approval reach the players")

	// 取り消した申込には登録させない、リンクも出さない
	cancelled, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("伊東", nil))
	require.NoError(t, err)
	_, err = f.desk.Decide(ctx, cancelled.ID, entry.ActionCancel, "", f.admin)
	require.NoError(t, err)
	_, err = f.desk.IssueMemberLink(ctx, jiro, cancelled.InviteCode)
	require.ErrorIs(t, err, domain.ErrConflict)
	var notFound *domain.NotFoundError
	_, err = f.desk.IssueMemberLink(ctx, jiro, "zzzzzzzzzzzzzzzzzzzzzzzzzz")
	require.ErrorAs(t, err, &notFound)
}

// メンバーとして登録した人が「エントリー」と送ったときに返す、登録したチームのリンク。取り消した申込と
// 終わった大会のチームは出さない。リンクを開くと、その人の登録の画面になる。
func TestEntryLinkJoinedTeams(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	atami, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
	ito, err := f.desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("伊東", nil))
	require.NoError(t, err)
	hanako := f.lineUser(t, "Uhanako")
	none, err := f.desk.IssueJoinedLinks(ctx, hanako)
	require.NoError(t, err)
	assert.Empty(t, none, "not a member of any team yet")

	for _, team := range []entry.Detail{atami, ito} {
		link, err := f.desk.IssueMemberLink(ctx, hanako, team.InviteCode)
		require.NoError(t, err)
		_, err = f.desk.JoinByLink(ctx, link.Token, entry.JoinDraft{Name: "佐藤 花子", Number: "12",
			Answers: map[int64]string{}})
		require.NoError(t, err)
	}
	links, err := f.desk.IssueJoinedLinks(ctx, hanako)
	require.NoError(t, err)
	require.Len(t, links, 2)
	assert.Equal(t, entry.JoinedTeam{TeamEntryID: atami.ID, TeamName: "熱海ディスクス",
		Tournament: entry.TournamentRef{ID: f.tournament.ID, Name: f.tournament.NameJa}}, links[0].Team)
	view, err := f.desk.LinkView(ctx, links[0].Token)
	require.NoError(t, err)
	require.NotNil(t, view.Member)
	require.NotNil(t, view.Member.Mine, "the link opens the member's own registration")
	assert.Equal(t, "佐藤 花子", view.Member.Mine.Name)

	_, err = f.desk.Decide(ctx, atami.ID, entry.ActionCancel, "", f.admin)
	require.NoError(t, err)
	links, err = f.desk.IssueJoinedLinks(ctx, hanako)
	require.NoError(t, err)
	require.Len(t, links, 1, "a cancelled entry is left out")
	assert.Equal(t, ito.ID, links[0].Team.TeamEntryID)

	require.NoError(t, f.db.Model(&store.Tournament{}).Where("id = ?", f.tournament.ID).
		Update("status", domain.TournamentFinished).Error)
	links, err = f.desk.IssueJoinedLinks(ctx, hanako)
	require.NoError(t, err)
	assert.Empty(t, links, "a finished tournament is left out")
}

func TestEntryLinkProfileAndPastTeams(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	b := storetest.New(t, f.db)
	open := func() (store.Tournament, store.Division) {
		tournament := b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen })
		return tournament, b.Division(tournament.ID)
	}
	second, secondDivision := open()
	third, thirdDivision := open()
	rep := f.lineUser(t, "Urep")
	submitTo := func(tournament int64, division int64, name, phone string) entry.Detail {
		t.Helper()
		draft := f.draft(name, nil)
		draft.DivisionID, draft.RepPhone, draft.RepEmail = division, phone, "rep@example.com"
		detail, err := f.desk.SubmitByLink(ctx, f.teamLink(t, rep, tournament).Token, draft)
		require.NoError(t, err)
		*f.now = f.now.Add(time.Hour)
		return detail
	}

	// まだ何も入れていない人には、前回の入力も前のチームも無い
	view, err := f.desk.LinkView(ctx, f.teamLink(t, rep, second.ID).Token)
	require.NoError(t, err)
	assert.Nil(t, view.Team.Profile)
	assert.Empty(t, view.Team.PastTeams)

	atami := submitTo(f.tournament.ID, f.division.ID, "熱海ディスクス", "090-1111-1111")
	submitTo(third.ID, thirdDivision.ID, "Izu Flyers", "090-2222-2222")
	// メンバーが自分の行の名前を直すと、前のチームのメンバーにもその名前が出る
	member := f.lineUser(t, "Uhanako")
	link, err := f.desk.IssueMemberLink(ctx, member, atami.InviteCode)
	require.NoError(t, err)
	sato := atami.MemberList[1].ID
	_, err = f.desk.JoinByLink(ctx, link.Token, entry.JoinDraft{MemberID: &sato, Name: "佐藤 はなこ", Number: "12"})
	require.NoError(t, err)

	view, err = f.desk.LinkView(ctx, f.teamLink(t, rep, second.ID).Token)
	require.NoError(t, err)
	require.NotNil(t, view.Team.Profile)
	assert.Equal(t, entry.Profile{Name: "山田 太郎", Phone: "090-2222-2222", Email: "rep@example.com"},
		*view.Team.Profile, "the last input")
	require.Len(t, view.Team.PastTeams, 2)
	assert.Equal(t, "Izu Flyers", view.Team.PastTeams[0].FullName, "newest first")
	assert.Equal(t, entry.PastTeam{FullName: "熱海ディスクス", ShortName: "熱海ディスクス",
		Members: []entry.MemberDraft{{Name: "山田 太郎", Number: "7"}, {Name: "佐藤 はなこ", Number: "12"}}},
		view.Team.PastTeams[1])

	// 同じチーム名は最後の申込だけ(前後の空白と大文字小文字を除いて同じ)。今の大会の申込は出さない
	submitTo(second.ID, secondDivision.ID, " izu flyers ", "090-3333-3333")
	view, err = f.desk.LinkView(ctx, f.teamLink(t, rep, f.tournament.ID).Token)
	require.NoError(t, err)
	names := []string{}
	for _, team := range view.Team.PastTeams {
		names = append(names, team.FullName)
	}
	assert.Equal(t, []string{"izu flyers"}, names, "熱海 is in this tournament; Izu Flyers twice becomes the last one")

	// メンバー登録は名前だけを上書きし、代表として入れた連絡先は消さない
	member2, err := f.desk.IssueMemberLink(ctx, rep, atami.InviteCode)
	require.NoError(t, err)
	_, err = f.desk.JoinByLink(ctx, member2.Token, entry.JoinDraft{Name: "山田 たろう", Number: "70"})
	require.NoError(t, err)
	profile, err := store.NewEntryStore(f.db).Profile(ctx, rep)
	require.NoError(t, err)
	assert.Equal(t, entry.Profile{Name: "山田 たろう", Phone: "090-3333-3333", Email: "rep@example.com"}, *profile)
	form, err := f.desk.LinkView(ctx, member2.Token)
	require.NoError(t, err)
	assert.Equal(t, "山田 たろう", form.Member.ProfileName)
}

// recordingNotifier は代表への知らせを覚える。fail が真なら送れなかったことにする。
type recordingNotifier struct {
	notices []entry.Notice
	fail    bool
}

func (n *recordingNotifier) Notify(_ context.Context, notice entry.Notice) error {
	if n.fail {
		return errors.New("line api is down")
	}
	n.notices = append(n.notices, notice)
	return nil
}

func TestEntryNoticesGoToTheLineRepresentative(t *testing.T) {
	f := newLinkFixture(t)
	ctx := context.Background()
	notifier := &recordingNotifier{}
	var reported []error
	desk := entry.NewService(store.NewEntryStore(f.db), entry.WithRandom(&counter{}),
		entry.WithClock(func() time.Time { return *f.now }),
		entry.WithInviteURL(func(code string) string { return "invite:" + code }),
		entry.WithNotifier(notifier, func(err error) { reported = append(reported, err) }))
	rep := f.lineUser(t, "Urep")
	links, err := desk.IssueTeamLinks(ctx, rep)
	require.NoError(t, err)
	token := links[0].Token

	submitted, err := desk.SubmitByLink(ctx, token, f.draft("熱海ディスクス", nil))
	require.NoError(t, err)
	_, err = desk.Submit(ctx, f.tournament.ID, entry.ChannelDesk, f.draft("伊東", nil))
	require.NoError(t, err)
	require.Len(t, notifier.notices, 1, "a desk entry has no LINE representative to tell")
	first := notifier.notices[0]
	assert.Equal(t, entry.NoticeSubmitted, first.Kind)
	assert.Equal(t, rep, first.LineUserID)
	assert.Equal(t, f.tournament.ID, first.Tournament.ID)
	assert.Equal(t, "invite:"+submitted.InviteCode, first.Entry.InviteURL)

	_, err = desk.Decide(ctx, submitted.ID, entry.ActionReturn, "短縮名を直してください", f.admin)
	require.NoError(t, err)
	_, err = desk.ResubmitByLink(ctx, token, submitted.ID, f.draft("熱海ディスクス", nil), nil)
	require.NoError(t, err)
	_, err = desk.Decide(ctx, submitted.ID, entry.ActionApprove, "", f.admin)
	require.NoError(t, err)
	kinds := []entry.NoticeKind{}
	for _, notice := range notifier.notices {
		kinds = append(kinds, notice.Kind)
	}
	assert.Equal(t, []entry.NoticeKind{entry.NoticeSubmitted, entry.NoticeReturned, entry.NoticeResubmitted,
		entry.NoticeApproved}, kinds)
	assert.Equal(t, "短縮名を直してください", notifier.notices[1].Entry.ReturnNote)

	// 送れなくても、運営の操作は成り立ち、送れなかったことだけを報告する
	notifier.fail = true
	other, err := desk.SubmitByLink(ctx, token, f.draft("伊豆", nil))
	require.NoError(t, err)
	cancelled, err := desk.Decide(ctx, other.ID, entry.ActionCancel, "", f.admin)
	require.NoError(t, err)
	assert.Equal(t, entry.StatusCancelled, cancelled.Status)
	assert.Len(t, reported, 2)
}
