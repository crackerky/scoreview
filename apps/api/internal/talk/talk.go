// Package talk は公式アカウントのトークの受け答えを持つ(docs/line-integration.md「本人の確かめ方」)。
//
// 依存の向きは httpapi → line(LINE の HTTP の形を知る層)→ talk ← store。talk は LINE の HTTP も DB も
// 知らず、届いたもの(Event)にどう返すかの決まりだけを持つ。返す本人用のリンクの発行は entry に任せる。
package talk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

// EventType は公式アカウントに届いたものの種類。
type EventType string

// EventType の値。Other は受け答えしないもの(画像、スタンプ、グループへの参加など)。
const (
	EventFollow   EventType = "follow"
	EventUnfollow EventType = "unfollow"
	EventMessage  EventType = "message"
	EventPostback EventType = "postback"
	EventOther    EventType = "other"
)

// Event は公式アカウントに届いたもの 1 つ。UserID は 1 対 1 のトークから届いたときの、公式アカウント
// から見たユーザー ID(グループやトークルームからなら空)。Standby は、別のツールがトークを受け持って
// いて、こちらは送ってはいけない状態。
type Event struct {
	ID           string
	Type         EventType
	Standby      bool
	UserID       string
	ReplyToken   string
	Text         string
	PostbackData string
	At           time.Time
}

// Message は返すメッセージ。今は文だけ。
type Message struct {
	Text string
}

// Messenger は公式アカウントとして返す。line が実装する。
type Messenger interface {
	// Reply は届いたものに返信する。返信のトークンは 1 回だけ、届いてから 1 分以内に使える
	Reply(ctx context.Context, replyToken string, messages []Message) error
}

// Store は友だちの状態と、届いたものの記録を読み書きする。store が実装する。
type Store interface {
	// FirstSeen は届いたものを記録し、初めて届いたものなら真を返す(LINE は同じものを 2 回送ることがある)
	FirstSeen(ctx context.Context, eventID string, at time.Time) (bool, error)
	// Forget は FirstSeen の記録を消す(処理に失敗したものを、LINE が送り直したときにやり直せるように)
	Forget(ctx context.Context, eventID string) error
	// Follow は友だち追加(またはブロックの解除)を記録する
	Follow(ctx context.Context, userID string, at time.Time) error
	// Unfollow はブロックを記録する
	Unfollow(ctx context.Context, userID string, at time.Time) error
	// LineUser は公式アカウントから見たユーザー ID の line_users の行の ID を返す(無ければ作る)
	LineUser(ctx context.Context, userID string, at time.Time) (int64, error)
}

// Bot は届いたものに、決まりに沿って返す。
type Bot struct {
	store     Store
	links     entry.Links
	messenger Messenger
	siteURL   string
}

// NewBot は Bot を返す。siteURL は本人用のリンクの前に付けるサイトの URL(末尾の / は無くてよい)。
func NewBot(store Store, links entry.Links, messenger Messenger, siteURL string) *Bot {
	return &Bot{store: store, links: links, messenger: messenger, siteURL: strings.TrimRight(siteURL, "/")}
}

// Handle は届いたものを順に処理する。1 つの失敗で残りを止めず、失敗はまとめて返す。失敗したものは
// 届いた記録を消すので、LINE Developers で Webhook の再送を有効にしていれば、LINE が送り直したときに
// 処理できなかったものだけをやり直せる(再送が無効なら、失敗はログに残るだけになる)。
func (b *Bot) Handle(ctx context.Context, events []Event) error {
	var failed []error
	for _, event := range events {
		if err := b.handle(ctx, event); err != nil {
			failed = append(failed, fmt.Errorf("event %s: %w", event.ID, err))
		}
	}
	return errors.Join(failed...)
}

func (b *Bot) handle(ctx context.Context, event Event) error {
	if event.UserID == "" {
		return nil
	}
	first, err := b.store.FirstSeen(ctx, event.ID, event.At)
	if err != nil || !first {
		return err
	}
	if err := b.respond(ctx, event); err != nil {
		if forgetErr := b.store.Forget(ctx, event.ID); forgetErr != nil {
			return errors.Join(err, forgetErr)
		}
		return err
	}
	return nil
}

// respond は初めて届いたものに応じる(友だちの状態を記録し、頼まれたことに返信する)。
func (b *Bot) respond(ctx context.Context, event Event) error {
	switch event.Type {
	case EventFollow:
		return b.store.Follow(ctx, event.UserID, event.At)
	case EventUnfollow:
		return b.store.Unfollow(ctx, event.UserID, event.At)
	case EventMessage, EventPostback:
		if event.Standby || event.ReplyToken == "" {
			return nil
		}
		command := CommandOf(event)
		if command.Kind == CommandNone {
			// 知らない言葉には返さない。運営がトークの画面から手で返せるように
			return nil
		}
		messages, err := b.answer(ctx, event, command)
		if err != nil {
			return err
		}
		return b.messenger.Reply(ctx, event.ReplyToken, messages)
	}
	return nil
}

func (b *Bot) answer(ctx context.Context, event Event, command Command) ([]Message, error) {
	lineUser, err := b.store.LineUser(ctx, event.UserID, event.At)
	if err != nil {
		return nil, err
	}
	if command.Kind == CommandJoin {
		link, err := b.links.IssueMemberLink(ctx, lineUser, command.InviteCode)
		var notFound *domain.NotFoundError
		switch {
		case errors.As(err, &notFound):
			return []Message{{Text: TextInviteNotFound}}, nil
		case errors.Is(err, domain.ErrConflict):
			return []Message{{Text: TextTeamCancelled}}, nil
		case err != nil:
			return nil, err
		}
		return []Message{{Text: MemberLinkText(link.TeamName, b.linkURL(link.Token))}}, nil
	}
	links, err := b.links.IssueTeamLinks(ctx, lineUser)
	if err != nil {
		return nil, err
	}
	issued := make([]TeamLink, len(links))
	for i, link := range links {
		issued[i] = TeamLink{Tournament: link.Tournament.Name, URL: b.linkURL(link.Token)}
	}
	return []Message{{Text: TeamLinksText(issued)}}, nil
}

func (b *Bot) linkURL(token string) string {
	return b.siteURL + "/entry/" + token
}

// CommandKind は届いた言葉が何を頼んでいるか。
type CommandKind int

// CommandKind の値。
const (
	CommandNone CommandKind = iota
	// CommandEntry は代表の申込(リッチメニューの「エントリー」か、トークで「エントリー」と送った)
	CommandEntry
	// CommandJoin はメンバー登録(招待リンクで開いたトークの「参加 招待コード」)
	CommandJoin
)

// Command は届いた言葉の解釈。
type Command struct {
	Kind       CommandKind
	InviteCode string
}

// PostbackEntry はリッチメニューの「エントリー」のポストバックのデータ。
const PostbackEntry = "action=entry"

// isEntryWord は代表の申込として受け取る言葉か。リッチメニューの文のほか、手で打ちそうな言い方も受ける。
func isEntryWord(text string) bool {
	switch text {
	case "エントリー", "エントリ", "申込", "申込み", "申し込み", "もうしこみ":
		return true
	}
	return false
}

// inviteCodeOf は「参加 招待コード」の招待コードを返す。招待コードの形でなければ空(entry.IsInviteCode)。
func inviteCodeOf(text string) string {
	rest, ok := strings.CutPrefix(text, "参加")
	if !ok {
		return ""
	}
	code := strings.TrimSpace(rest)
	if !entry.IsInviteCode(code) {
		return ""
	}
	return code
}

// CommandOf は届いたものを解釈する。前後と間の空白(全角を含む)を除き、英字は小文字にして比べる
// (スマートフォンの入力で招待コードの先頭が大文字になることがあるため)。
func CommandOf(event Event) Command {
	if event.Type == EventPostback {
		if event.PostbackData == PostbackEntry {
			return Command{Kind: CommandEntry}
		}
		return Command{}
	}
	text := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(event.Text, "　", " ")))
	if isEntryWord(text) {
		return Command{Kind: CommandEntry}
	}
	if code := inviteCodeOf(text); code != "" {
		return Command{Kind: CommandJoin, InviteCode: code}
	}
	return Command{}
}
