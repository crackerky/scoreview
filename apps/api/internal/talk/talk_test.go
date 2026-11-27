package talk_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/talk"
)

type fakeStore struct {
	seen    map[string]bool
	friends map[string]bool
	users   map[string]int64
	helped  map[string]time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{seen: map[string]bool{}, friends: map[string]bool{}, users: map[string]int64{},
		helped: map[string]time.Time{}}
}

func (s *fakeStore) FirstSeen(_ context.Context, id string, _ time.Time) (bool, error) {
	if s.seen[id] {
		return false, nil
	}
	s.seen[id] = true
	return true, nil
}

func (s *fakeStore) HelpDue(_ context.Context, userID string, at time.Time, every time.Duration) (bool, error) {
	if last, ok := s.helped[userID]; ok && at.Sub(last) < every {
		return false, nil
	}
	s.helped[userID] = at
	return true, nil
}

func (s *fakeStore) Forget(_ context.Context, id string) error {
	delete(s.seen, id)
	return nil
}

func (s *fakeStore) Follow(_ context.Context, userID string, _ time.Time) error {
	s.friends[userID] = true
	return nil
}

func (s *fakeStore) Unfollow(_ context.Context, userID string, _ time.Time) error {
	s.friends[userID] = false
	return nil
}

func (s *fakeStore) Recipient(_ context.Context, id int64) (talk.Recipient, error) {
	for userID, row := range s.users {
		if row == id {
			friend, known := s.friends[userID]
			return talk.Recipient{UserID: userID, Friend: friend || !known}, nil
		}
	}
	return talk.Recipient{}, domain.NotFound(domain.ResourceLineUser, id)
}

func (s *fakeStore) LineUser(_ context.Context, userID string, _ time.Time) (int64, error) {
	if id, ok := s.users[userID]; ok {
		return id, nil
	}
	s.users[userID] = int64(len(s.users) + 1)
	return s.users[userID], nil
}

type fakeLinks struct {
	tournaments []entry.TournamentRef
	find        []entry.TournamentRef
	joined      []entry.JoinedTeam
	scorer      []entry.ScorerTeam
	issued      []string
}

func (l *fakeLinks) IssueScorerLinks(_ context.Context, lineUser int64) ([]entry.ScorerLink, error) {
	out := make([]entry.ScorerLink, len(l.scorer))
	for i, team := range l.scorer {
		token := fmt.Sprintf("scorer-%d-%d", lineUser, team.TeamEntryID)
		l.issued = append(l.issued, token)
		out[i] = entry.ScorerLink{Team: team, Token: token}
	}
	return out, nil
}

func (l *fakeLinks) IssueFindLinks(_ context.Context, lineUser int64) ([]entry.TournamentLink, error) {
	out := make([]entry.TournamentLink, len(l.find))
	for i, tournament := range l.find {
		token := fmt.Sprintf("find-%d-%d", lineUser, tournament.ID)
		l.issued = append(l.issued, token)
		out[i] = entry.TournamentLink{Tournament: tournament, Token: token}
	}
	return out, nil
}

func (l *fakeLinks) IssueJoinedLinks(_ context.Context, lineUser int64) ([]entry.JoinedLink, error) {
	out := make([]entry.JoinedLink, len(l.joined))
	for i, team := range l.joined {
		token := fmt.Sprintf("joined-%d-%d", lineUser, team.TeamEntryID)
		l.issued = append(l.issued, token)
		out[i] = entry.JoinedLink{Team: team, Token: token}
	}
	return out, nil
}

func (l *fakeLinks) IssueTeamLinks(_ context.Context, lineUser int64) ([]entry.TournamentLink, error) {
	out := make([]entry.TournamentLink, len(l.tournaments))
	for i, tournament := range l.tournaments {
		token := fmt.Sprintf("team-%d-%d", lineUser, tournament.ID)
		l.issued = append(l.issued, token)
		out[i] = entry.TournamentLink{Tournament: tournament, Token: token}
	}
	return out, nil
}

func (l *fakeLinks) IssueMemberLink(_ context.Context, lineUser int64, code string) (entry.IssuedLink, error) {
	switch code {
	case "cccccccccccccccccccccccccc":
		return entry.IssuedLink{}, fmt.Errorf("%w: cancelled", domain.ErrConflict)
	case "aaaaaaaaaaaaaaaaaaaaaaaaaa":
		token := fmt.Sprintf("member-%d", lineUser)
		l.issued = append(l.issued, token)
		return entry.IssuedLink{Token: token, TeamName: "熱海ディスクス"}, nil
	}
	return entry.IssuedLink{}, domain.NotFound(domain.ResourceEntry, 0)
}

// fakeMessenger は返信を覚える。failures が残っている間は返信に失敗する(LINE の API が落ちたとき)。
type fakeMessenger struct {
	replies  map[string][]talk.Message
	pushes   map[string][]talk.Message
	failures int
}

func (m *fakeMessenger) Push(_ context.Context, to string, messages []talk.Message) error {
	if m.pushes == nil {
		m.pushes = map[string][]talk.Message{}
	}
	m.pushes[to] = append(m.pushes[to], messages...)
	return nil
}

func (m *fakeMessenger) Reply(_ context.Context, token string, messages []talk.Message) error {
	if m.failures > 0 {
		m.failures--
		return errors.New("line api is down")
	}
	m.replies[token] = messages
	return nil
}

func setup() (*talk.Bot, *fakeStore, *fakeLinks, *fakeMessenger) {
	store, links := newFakeStore(), &fakeLinks{tournaments: []entry.TournamentRef{{ID: 7, Name: "ラバー 2027"},
		{ID: 9, Name: "ルーキーズファイナル"}}}
	messenger := &fakeMessenger{replies: map[string][]talk.Message{}}
	bot := talk.NewBot(store, links, messenger, "https://score.example/")
	return bot, store, links, messenger
}

func message(id, user, text string) talk.Event {
	return talk.Event{ID: id, Type: talk.EventMessage, UserID: user, ReplyToken: "r-" + id, Text: text}
}

// 「エントリー」には、まず代表かメンバーかを選ぶボタンを返す(リンクはまだ作らない)。
func TestBotAsksRepOrMember(t *testing.T) {
	bot, _, links, messenger := setup()
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{message("1", "U1", " エントリー ")}))
	reply := messenger.replies["r-1"]
	require.Len(t, reply, 1)
	assert.Equal(t, talk.TextEntryChoice, reply[0].Text)
	assert.Equal(t, []talk.Choice{{Label: "代表として申し込む", Data: talk.PostbackEntryRep},
		{Label: "メンバーとして登録する", Data: talk.PostbackEntryMember}}, reply[0].Choices)
	assert.Empty(t, links.issued)
}

func postback(id, user, data string) talk.Event {
	return talk.Event{ID: id, Type: talk.EventPostback, UserID: user, ReplyToken: "r-" + id, PostbackData: data}
}

// 代表を選ぶと、大会ごとの申込のリンクを返し、メンバーの名前は書かなくてもよいことを先に書く。
func TestBotRepliesEntryLinksForEachOpenTournament(t *testing.T) {
	bot, _, _, messenger := setup()
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{postback("1", "U1", talk.PostbackEntryRep)}))
	reply := messenger.replies["r-1"]
	require.Len(t, reply, 1)
	assert.Contains(t, reply[0].Text, "書かなくても申し込めます")
	assert.Contains(t, reply[0].Text, "3 日間")
	assert.NotContains(t, reply[0].Text, "メンバー登録は", "the rep sees only the rep's way in")
	// 大会はボタンで選ぶ(ボタンが本人用の申込のリンクを開く)
	assert.Equal(t, []talk.Choice{{Label: "ラバー 2027", URL: "https://score.example/entry/team-1-7"},
		{Label: "ルーキーズファイナル", URL: "https://score.example/entry/team-1-9"}}, reply[0].Choices)

	// ボタンを押さずに「代表」と打っても同じ
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{message("2", "U1", "代表")}))
	assert.Len(t, messenger.replies["r-2"][0].Choices, 2)
}

// メンバーを選ぶと、登録したチームのリンクを返す(代表の申込のリンクは返さない。代表の申込の画面を
// 開いて、同じチームの申込を作ってしまわないように)。
func TestBotRepliesJoinedTeamsToAMember(t *testing.T) {
	bot, _, links, messenger := setup()
	links.joined = []entry.JoinedTeam{{TeamEntryID: 41, TeamName: "熱海ディスクス",
		Tournament: entry.TournamentRef{ID: 7, Name: "ラバー 2027"}}}
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{postback("1", "U1", talk.PostbackEntryMember)}))
	reply := messenger.replies["r-1"][0]
	assert.True(t, strings.HasPrefix(reply.Text, "登録したチームの確かめと直しは、チームのボタンからできます。"), reply.Text)
	assert.Equal(t, []talk.Choice{{Label: "熱海ディスクス ・ ラバー 2027", URL: "https://score.example/entry/joined-1-41"}},
		reply.Choices)
	assert.Equal(t, []string{"joined-1-41"}, links.issued, "no rep links for a member")

	// 招待リンクのほかに、大会を選んでチームを探す道も出す
	links.find = []entry.TournamentRef{{ID: 9, Name: "ルーキーズファイナル"}}
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{postback("3", "U1", talk.PostbackEntryMember)}))
	assert.Equal(t, talk.Choice{Label: "ルーキーズファイナルでチームを探す", URL: "https://score.example/entry/find-1-9"},
		messenger.replies["r-3"][0].Choices[1])
	links.joined = nil
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{postback("4", "U1", talk.PostbackEntryMember)}))
	assert.Contains(t, messenger.replies["r-4"][0].Text, "招待リンクを開くか、下のボタンで大会を選んで")
	assert.Len(t, messenger.replies["r-4"][0].Choices, 1)

	// どちらの道も無ければ(登録できるチームのある大会が無い)、登録のしかたを返す
	links.find = nil
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{message("2", "U1", "メンバー登録")}))
	assert.Equal(t, talk.TextMemberHowTo, messenger.replies["r-2"][0].Text)
}

func TestBotRichMenuPostbackAlsoStartsEntry(t *testing.T) {
	bot, _, _, messenger := setup()
	event := talk.Event{ID: "1", Type: talk.EventPostback, UserID: "U1", ReplyToken: "r", PostbackData: talk.PostbackEntry}
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{event}))
	require.Len(t, messenger.replies["r"], 1)
	assert.Len(t, messenger.replies["r"][0].Choices, 2)
}

func TestBotNoOpenTournament(t *testing.T) {
	bot, _, links, messenger := setup()
	links.tournaments = nil
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{postback("1", "U1", talk.PostbackEntryRep)}))
	text := messenger.replies["r-1"][0].Text
	assert.Equal(t, talk.TextNoOpenTournament, text)
	assert.NotContains(t, text, "日間", "no links, so no lifetime note")
}

func TestBotJoin(t *testing.T) {
	bot, _, _, messenger := setup()
	ctx := context.Background()
	require.NoError(t, bot.Handle(ctx, []talk.Event{
		message("1", "U1", "参加 AAAAAAAAAAAAAAAAAAAAAAAAAA"),
		message("2", "U1", "参加　bbbbbbbbbbbbbbbbbbbbbbbbbb"),
		message("3", "U1", "参加 cccccccccccccccccccccccccc"),
	}))
	assert.Contains(t, messenger.replies["r-1"][0].Text, "「熱海ディスクス」のメンバー登録")
	assert.Equal(t, []talk.Choice{{Label: "登録の画面を開く", URL: "https://score.example/entry/member-1"}},
		messenger.replies["r-1"][0].Choices)
	assert.Equal(t, talk.TextInviteNotFound, messenger.replies["r-2"][0].Text)
	assert.Equal(t, talk.TextTeamCancelled, messenger.replies["r-3"][0].Text)
}

func TestBotDoesNotAnswer(t *testing.T) {
	bot, _, links, messenger := setup()
	standby := message("2", "U1", "エントリー")
	standby.Standby = true
	group := message("3", "", "エントリー")
	sticker := message("4", "U1", "")
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{standby, group, sticker}))
	assert.Empty(t, messenger.replies, "standby, groups and stickers are left to the staff")
	assert.Empty(t, links.issued)
}

// 知らない言葉には使い方の案内を返す。同じ人には 1 日に 1 回だけ(運営とのやりとりの途中に割り込まない)。
func TestBotHelpsOncePerDay(t *testing.T) {
	bot, _, _, messenger := setup()
	ctx := context.Background()
	at := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	say := func(id, text string, when time.Time) talk.Event {
		event := message(id, "U1", text)
		event.At = when
		return event
	}
	require.NoError(t, bot.Handle(ctx, []talk.Event{say("1", "こんにちは", at)}))
	assert.Equal(t, talk.TextHelp, messenger.replies["r-1"][0].Text)
	require.NoError(t, bot.Handle(ctx, []talk.Event{say("2", "よろしくお願いします", at.Add(time.Hour))}))
	assert.Empty(t, messenger.replies["r-2"], "not again on the same day")
	require.NoError(t, bot.Handle(ctx, []talk.Event{say("3", "参加します", at.Add(2*time.Hour))}))
	assert.Equal(t, talk.TextJoinHowTo, messenger.replies["r-3"][0].Text, "a join without a code is always answered")
	require.NoError(t, bot.Handle(ctx, []talk.Event{say("4", "こんにちは", at.Add(25*time.Hour))}))
	assert.Equal(t, talk.TextHelp, messenger.replies["r-4"][0].Text, "again the next day")
}

func TestBotHandlesTheSameEventOnce(t *testing.T) {
	bot, _, links, _ := setup()
	event := postback("1", "U1", talk.PostbackEntryRep)
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{event, event}))
	assert.Len(t, links.issued, 2, "one link per open tournament, not two rounds")
}

func TestBotRetriesAFailedEventWhenLINERedelivers(t *testing.T) {
	bot, _, _, messenger := setup()
	messenger.failures = 1
	event := message("1", "U1", "エントリー")
	require.Error(t, bot.Handle(context.Background(), []talk.Event{event}), "the reply failed")
	assert.Empty(t, messenger.replies)

	require.NoError(t, bot.Handle(context.Background(), []talk.Event{event}), "LINE sends the same event again")
	assert.Len(t, messenger.replies["r-1"], 1, "the redelivered event is handled, not skipped")
}

func TestBotRecordsFriends(t *testing.T) {
	bot, store, _, messenger := setup()
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{
		{ID: "1", Type: talk.EventFollow, UserID: "U1", ReplyToken: "r"},
		{ID: "2", Type: talk.EventUnfollow, UserID: "U2"},
	}))
	assert.Equal(t, map[string]bool{"U1": true, "U2": false}, store.friends)
	assert.Empty(t, messenger.replies, "the greeting is left to the official account settings")
}

func TestCommandOf(t *testing.T) {
	cases := []struct {
		text string
		want talk.Command
	}{
		{"エントリー", talk.Command{Kind: talk.CommandEntry}},
		{"　申込　", talk.Command{Kind: talk.CommandEntry}},
		{"参加abcdefghijklmnopqrstuvwxyz", talk.Command{Kind: talk.CommandJoin, InviteCode: "abcdefghijklmnopqrstuvwxyz"}},
		// 招待コードが読めない「参加」には、招待リンクの開き方を返す
		{"参加 abc", talk.Command{Kind: talk.CommandJoinHelp}},
		{"参加 abcdefghijklmnopqrstuvwxy1", talk.Command{Kind: talk.CommandJoinHelp}},
		{"参加します", talk.Command{Kind: talk.CommandJoinHelp}},
		// 文の中の言葉も受ける
		{"エントリーしたいです", talk.Command{Kind: talk.CommandEntry}},
		{"大会に申し込みたい", talk.Command{Kind: talk.CommandEntry}},
		{"記録の画面はどこですか", talk.Command{Kind: talk.CommandScore}},
		// 代表とメンバーの入口は、決まった言い方だけ(運営とのやりとりの「代表の山田です」には返さない)
		{"代表として申し込む", talk.Command{Kind: talk.CommandEntryRep}},
		{"メンバー登録", talk.Command{Kind: talk.CommandEntryMember}},
		{"代表の山田です。質問があります", talk.Command{Kind: talk.CommandHelp}},
		// 知らない言葉には使い方の案内
		{"こんにちは", talk.Command{Kind: talk.CommandHelp}},
		{"", talk.Command{}},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, talk.CommandOf(talk.Event{Type: talk.EventMessage, Text: tc.text}), tc.text)
	}
	assert.Equal(t, talk.Command{}, talk.CommandOf(talk.Event{Type: talk.EventPostback, PostbackData: "other"}))
	assert.Equal(t, talk.Command{Kind: talk.CommandEntryRep},
		talk.CommandOf(talk.Event{Type: talk.EventPostback, PostbackData: talk.PostbackEntryRep}))
	assert.Equal(t, talk.Command{Kind: talk.CommandEntryMember},
		talk.CommandOf(talk.Event{Type: talk.EventPostback, PostbackData: talk.PostbackEntryMember}))
	assert.Equal(t, talk.Command{Kind: talk.CommandScore},
		talk.CommandOf(talk.Event{Type: talk.EventPostback, PostbackData: talk.PostbackScore}))
}

func notice(kind entry.NoticeKind) entry.Notice {
	return entry.Notice{Kind: kind, LineUserID: 1, Tournament: entry.TournamentRef{ID: 7, Name: "ラバー 2027"},
		Entry: entry.Detail{Summary: entry.Summary{FullName: "熱海ディスクス",
			Division: entry.Division{Name: domain.I18nText{Ja: "オープン"}}},
			InviteURL: "https://line.me/R/oaMessage/%40up/?x", ReturnNote: "短縮名を直してください"}}
}

func TestNotifierSendsNoticesToTheRepresentative(t *testing.T) {
	store, messenger := newFakeStore(), &fakeMessenger{}
	_, err := store.LineUser(context.Background(), "Urep", time.Time{})
	require.NoError(t, err)
	notifier := talk.NewNotifier(store, messenger, "https://score.example/")
	ctx := context.Background()

	require.NoError(t, notifier.Notify(ctx, notice(entry.NoticeSubmitted)))
	sent := messenger.pushes["Urep"]
	require.Len(t, sent, 2, "the receipt, and the invite to forward")
	assert.Contains(t, sent[0].Text, "「熱海ディスクス」の申込を受け付けました")
	assert.Contains(t, sent[0].Text, "部門: オープン")
	assert.Equal(t, talk.InviteText("熱海ディスクス", "https://line.me/R/oaMessage/%40up/?x"), sent[1].Text)

	require.NoError(t, notifier.Notify(ctx, notice(entry.NoticeReturned)))
	assert.Contains(t, messenger.pushes["Urep"][2].Text, "\n\n短縮名を直してください\n\n")
	require.NoError(t, notifier.Notify(ctx, notice(entry.NoticeApproved)))
	assert.Contains(t, messenger.pushes["Urep"][3].Text, "「熱海ディスクス」の「ラバー 2027」への申込を受け付け、参加が決まりました")
	assert.Contains(t, messenger.pushes["Urep"][3].Text, "https://score.example/tournaments/7")

	require.NoError(t, notifier.Notify(ctx, notice(entry.NoticeReopened)))
	assert.Contains(t, messenger.pushes["Urep"][4].Text, "「熱海ディスクス」の「ラバー 2027」への申込は、運営の確認待ちに戻りました")

	// ブロックしている人には送らない
	store.friends["Urep"] = false
	require.NoError(t, notifier.Notify(ctx, notice(entry.NoticeCancelled)))
	assert.Len(t, messenger.pushes["Urep"], 5)

	var notFound *domain.NotFoundError
	require.ErrorAs(t, notifier.Notify(ctx, entry.Notice{Kind: entry.NoticeApproved, LineUserID: 99}), &notFound)
}

func TestNoticeMessages(t *testing.T) {
	without := notice(entry.NoticeSubmitted)
	without.Entry.InviteURL = ""
	assert.Len(t, talk.NoticeMessages(without, ""), 1, "no invite to forward without the basic ID")
	for _, kind := range []entry.NoticeKind{entry.NoticeResubmitted, entry.NoticeCancelled} {
		messages := talk.NoticeMessages(notice(kind), "")
		require.Len(t, messages, 1, kind)
		assert.Contains(t, messages[0].Text, "「熱海ディスクス」", kind)
	}
	assert.Empty(t, talk.NoticeMessages(entry.Notice{Kind: "unknown"}, ""))
}

// 記録係に選ばれたメンバーへの知らせ。当日に記録の画面を開く言葉を書く。
func TestScorerChosenMessage(t *testing.T) {
	messages := talk.NoticeMessages(notice(entry.NoticeScorerChosen), "https://score.example")
	require.Len(t, messages, 1)
	assert.Contains(t, messages[0].Text, "「熱海ディスクス」の「ラバー 2027」の記録係になりました")
	assert.Contains(t, messages[0].Text, "「記録」と送ると")
}

// 「記録」と送った記録係には、記録の画面に入るリンクを返す。記録係でなければその旨を返す。
func TestBotRepliesScorerLinks(t *testing.T) {
	bot, _, links, messenger := setup()
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{message("1", "U1", "記録")}))
	assert.Equal(t, talk.TextNotScorer, messenger.replies["r-1"][0].Text)
	assert.Equal(t, []talk.Choice{{Label: "スコアサイトを見る", URL: "https://score.example/"}},
		messenger.replies["r-1"][0].Choices, "someone who is not a scorer can still see the scores")

	day := time.Date(2027, 2, 9, 0, 0, 0, 0, time.UTC)
	links.scorer = []entry.ScorerTeam{{MemberName: "佐藤 花子", JoinedTeam: entry.JoinedTeam{TeamEntryID: 41,
		TeamName: "熱海ディスクス", Tournament: entry.TournamentRef{ID: 7, Name: "ラバー 2027"}},
		Matches: []entry.ScorerMatch{{ID: 501, Date: &day, Start: "10:00", Field: "3", Opponent: "神戸"},
			{ID: 502}}}}
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{message("2", "U1", "スコアを入れる")}))
	reply := messenger.replies["r-2"][0]
	assert.Contains(t, reply.Text, "「熱海ディスクス」ラバー 2027の記録係です。")
	assert.Contains(t, reply.Text, "3 日間")
	// 試合はボタンで選び、押すとその試合のスコアの画面が開く。一覧のボタンも付ける
	assert.Equal(t, []talk.Choice{
		{Label: "2/9 10:00 コート 3 対 神戸", URL: "https://score.example/entry/scorer-1-41?match=501"},
		{Label: "相手未定", URL: "https://score.example/entry/scorer-1-41?match=502"},
		{Label: "自分たちの試合の一覧", URL: "https://score.example/entry/scorer-1-41"},
	}, reply.Choices)
}
