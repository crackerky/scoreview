package talk_test

import (
	"context"
	"errors"
	"fmt"
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
}

func newFakeStore() *fakeStore {
	return &fakeStore{seen: map[string]bool{}, friends: map[string]bool{}, users: map[string]int64{}}
}

func (s *fakeStore) FirstSeen(_ context.Context, id string, _ time.Time) (bool, error) {
	if s.seen[id] {
		return false, nil
	}
	s.seen[id] = true
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
	issued      []string
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

func TestBotRepliesEntryLinksForEachOpenTournament(t *testing.T) {
	bot, _, _, messenger := setup()
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{message("1", "U1", " エントリー ")}))
	reply := messenger.replies["r-1"]
	require.Len(t, reply, 1)
	assert.Contains(t, reply[0].Text, "ラバー 2027\nhttps://score.example/entry/team-1-7")
	assert.Contains(t, reply[0].Text, "ルーキーズファイナル\nhttps://score.example/entry/team-1-9")
	assert.Contains(t, reply[0].Text, "3 日間")
}

func TestBotRichMenuPostbackAlsoStartsEntry(t *testing.T) {
	bot, _, _, messenger := setup()
	event := talk.Event{ID: "1", Type: talk.EventPostback, UserID: "U1", ReplyToken: "r", PostbackData: talk.PostbackEntry}
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{event}))
	assert.Len(t, messenger.replies["r"], 1)
}

func TestBotNoOpenTournament(t *testing.T) {
	bot, _, links, messenger := setup()
	links.tournaments = nil
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{message("1", "U1", "申し込み")}))
	assert.Equal(t, talk.TextNoOpenTournament, messenger.replies["r-1"][0].Text)
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
	assert.Contains(t, messenger.replies["r-1"][0].Text, "https://score.example/entry/member-1")
	assert.Equal(t, talk.TextInviteNotFound, messenger.replies["r-2"][0].Text)
	assert.Equal(t, talk.TextTeamCancelled, messenger.replies["r-3"][0].Text)
}

func TestBotDoesNotAnswer(t *testing.T) {
	bot, _, links, messenger := setup()
	standby := message("2", "U1", "エントリー")
	standby.Standby = true
	group := message("3", "", "エントリー")
	require.NoError(t, bot.Handle(context.Background(), []talk.Event{
		message("1", "U1", "こんにちは"), standby, group,
	}))
	assert.Empty(t, messenger.replies, "unknown words, standby and groups are left to the staff")
	assert.Empty(t, links.issued)
}

func TestBotHandlesTheSameEventOnce(t *testing.T) {
	bot, _, links, _ := setup()
	event := message("1", "U1", "エントリー")
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
		{"参加 abc", talk.Command{}},
		{"参加 abcdefghijklmnopqrstuvwxy1", talk.Command{}},
		{"エントリーしたい", talk.Command{}},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, talk.CommandOf(talk.Event{Type: talk.EventMessage, Text: tc.text}), tc.text)
	}
	assert.Equal(t, talk.Command{}, talk.CommandOf(talk.Event{Type: talk.EventPostback, PostbackData: "other"}))
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

	// ブロックしている人には送らない
	store.friends["Urep"] = false
	require.NoError(t, notifier.Notify(ctx, notice(entry.NoticeCancelled)))
	assert.Len(t, messenger.pushes["Urep"], 4)

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
