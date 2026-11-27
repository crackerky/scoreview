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

// Message は返すメッセージ。Choices があれば、文の下に押せるボタンを並べる(押すとそのボタンの Data が
// ポストバックで届き、トークにはボタンの文字が本人の発言として出る)。
type Message struct {
	Text    string
	Choices []Choice
}

// Choice は文の下のボタン 1 つ。Label はボタンの文字(MaxChoiceLabel 字まで)。URL があれば押すとその
// ページを開き、無ければ Data(PostbackEntryRep など)がポストバックで届く。
type Choice struct {
	Label string
	Data  string
	URL   string
}

// MaxChoiceLabel はボタンの文字の上限(LINE の Flex Message のボタンは 40 字まで)。
const MaxChoiceLabel = 40

// Messenger は公式アカウントとして送る。line が実装する。
type Messenger interface {
	// Reply は届いたものに返信する。返信のトークンは 1 回だけ、届いてから 1 分以内に使える
	Reply(ctx context.Context, replyToken string, messages []Message) error
	// Push は公式アカウントから to(公式アカウントから見たユーザー ID)に送る。月の配信数に数えられる
	Push(ctx context.Context, to string, messages []Message) error
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
	// HelpDue は使い方の案内をこの人に返してよいか(まだ返していないか、前に返してから every 以上たった)を
	// 確かめ、返してよければ at を返した時刻として記録して真を返す
	HelpDue(ctx context.Context, userID string, at time.Time, every time.Duration) (bool, error)
	// Recipient は line_users の行 ID の人の、公式アカウントから見たユーザー ID と、今も友だちかを読む。
	// 行が無ければ domain.NotFound
	Recipient(ctx context.Context, id int64) (Recipient, error)
}

// Recipient は知らせる相手。UserID は公式アカウントから見たユーザー ID。Friend が偽(ブロック中)なら送らない。
type Recipient struct {
	UserID string
	Friend bool
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
			return nil
		}
		messages, err := b.answer(ctx, event, command)
		if err != nil || len(messages) == 0 {
			return err
		}
		return b.messenger.Reply(ctx, event.ReplyToken, messages)
	}
	return nil
}

func (b *Bot) answer(ctx context.Context, event Event, command Command) ([]Message, error) {
	switch command.Kind {
	case CommandEntry:
		return []Message{EntryChoiceMessage()}, nil
	case CommandJoinHelp:
		return []Message{{Text: TextJoinHowTo}}, nil
	case CommandHelp:
		// 知らない言葉への使い方の案内は、同じ人に 1 日 1 回だけ(運営とのやりとりの途中に割り込まない。
		// それ以外は返さず、運営がトークの画面から手で返す)
		due, err := b.store.HelpDue(ctx, event.UserID, event.At, HelpInterval)
		if err != nil || !due {
			return nil, err
		}
		return []Message{{Text: TextHelp}}, nil
	}
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
		return []Message{MemberLinkMessage(link.TeamName, b.linkURL(link.Token))}, nil
	}
	if command.Kind == CommandScore {
		links, err := b.links.IssueScorerLinks(ctx, lineUser)
		if err != nil {
			return nil, err
		}
		scorer := make([]ScorerTeamLink, len(links))
		for i, link := range links {
			scorer[i] = ScorerTeamLink{Team: link.Team.TeamName, Tournament: link.Team.Tournament.Name,
				URL: b.linkURL(link.Token), Matches: link.Team.Matches}
		}
		return ScorerMessages(scorer, b.siteURL), nil
	}
	if command.Kind == CommandEntryMember {
		joinedLinks, err := b.links.IssueJoinedLinks(ctx, lineUser)
		if err != nil {
			return nil, err
		}
		joined := make([]JoinedTeamLink, len(joinedLinks))
		for i, link := range joinedLinks {
			joined[i] = JoinedTeamLink{Team: link.Team.TeamName, Tournament: link.Team.Tournament.Name,
				URL: b.linkURL(link.Token)}
		}
		findLinks, err := b.links.IssueFindLinks(ctx, lineUser)
		if err != nil {
			return nil, err
		}
		find := make([]TeamLink, len(findLinks))
		for i, link := range findLinks {
			find[i] = TeamLink{Tournament: link.Tournament.Name, URL: b.linkURL(link.Token)}
		}
		return []Message{MemberReplyMessage(joined, find)}, nil
	}
	links, err := b.links.IssueTeamLinks(ctx, lineUser)
	if err != nil {
		return nil, err
	}
	issued := make([]TeamLink, len(links))
	for i, link := range links {
		issued[i] = TeamLink{Tournament: link.Tournament.Name, URL: b.linkURL(link.Token)}
	}
	return []Message{RepReplyMessage(issued)}, nil
}

func (b *Bot) linkURL(token string) string {
	return b.siteURL + "/entry/" + token
}

// CommandKind は届いた言葉が何を頼んでいるか。
type CommandKind int

// CommandKind の値。
const (
	CommandNone CommandKind = iota
	// CommandEntry は申込の入口(リッチメニューの「エントリー」か、トークで「エントリー」と送った)。
	// 代表として申し込むか、メンバーとして登録するかを選ぶボタンを返す
	CommandEntry
	// CommandEntryRep は代表の入口(「代表として申し込む」を押した)。大会ごとの申込のリンクを返す
	CommandEntryRep
	// CommandEntryMember はメンバーの入口(「メンバーとして登録する」を押した)。登録したチームのリンクか、
	// 登録のしかたを返す
	CommandEntryMember
	// CommandJoin はメンバー登録(招待リンクで開いたトークの「参加 招待コード」)
	CommandJoin
	// CommandScore はチームの記録係が記録の画面を開く(トークで「記録」と送った)
	CommandScore
	// CommandJoinHelp は「参加」と送ったが招待コードが読めない(招待リンクを開かずに打った、など)
	CommandJoinHelp
	// CommandHelp は知らない言葉(使い方の案内を返す)
	CommandHelp
)

// HelpInterval は使い方の案内を同じ人に返す間隔。
const HelpInterval = 24 * time.Hour

// Command は届いた言葉の解釈。
type Command struct {
	Kind       CommandKind
	InviteCode string
}

// ポストバックのデータ。PostbackEntry はリッチメニューの「エントリー」、PostbackEntryRep と
// PostbackEntryMember は「エントリー」への返信のボタン(リッチメニューに置いてもよい)、PostbackScore は
// リッチメニューの「スコアを入れる」。
const (
	PostbackEntry       = "action=entry"
	PostbackEntryRep    = "action=entry-rep"
	PostbackEntryMember = "action=entry-member"
	PostbackScore       = "action=score"
)

// isRepWord は代表の入口として受け取る言葉か(ボタンを押さずに手で打ったとき。文の中の「代表」は運営との
// やりとりにもよく出るので、決まった言い方だけを受ける)。
func isRepWord(text string) bool {
	switch text {
	case "代表", "代表です", "代表として申し込む", "代表の申込", "代表の申し込み", "代表エントリー":
		return true
	}
	return false
}

// isMemberWord はメンバーの入口として受け取る言葉か(isRepWord と同じく、決まった言い方だけ)。
func isMemberWord(text string) bool {
	switch text {
	case "メンバー", "メンバーです", "メンバーとして登録する", "メンバー登録", "メンバーエントリー", "選手登録":
		return true
	}
	return false
}

// isEntryWord は代表の申込として受け取る言葉か。リッチメニューの文のほか、手で打ちそうな言い方も受ける。
func isEntryWord(text string) bool {
	switch text {
	case "エントリー", "エントリ", "申込", "申込み", "申し込み", "もうしこみ":
		return true
	}
	return false
}

// isScoreWord は記録係が記録の画面を開く言葉か。
func isScoreWord(text string) bool {
	switch text {
	case "記録", "きろく", "記録係", "スコア", "スコア入力", "スコアを入れる":
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
		switch event.PostbackData {
		case PostbackEntry:
			return Command{Kind: CommandEntry}
		case PostbackEntryRep:
			return Command{Kind: CommandEntryRep}
		case PostbackEntryMember:
			return Command{Kind: CommandEntryMember}
		case PostbackScore:
			return Command{Kind: CommandScore}
		}
		return Command{}
	}
	text := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(event.Text, "　", " ")))
	if code := inviteCodeOf(text); code != "" {
		return Command{Kind: CommandJoin, InviteCode: code}
	}
	if isRepWord(text) {
		return Command{Kind: CommandEntryRep}
	}
	if isMemberWord(text) {
		return Command{Kind: CommandEntryMember}
	}
	// 決まった言葉のほか、文の中にその言葉があれば同じものとして受ける(「エントリーしたいです」など)
	if isEntryWord(text) || containsAny(text, "エントリ", "申込", "申し込", "もうしこ") {
		return Command{Kind: CommandEntry}
	}
	if strings.HasPrefix(text, "参加") {
		return Command{Kind: CommandJoinHelp}
	}
	if isScoreWord(text) || containsAny(text, "記録", "スコア") {
		return Command{Kind: CommandScore}
	}
	if text != "" {
		return Command{Kind: CommandHelp}
	}
	return Command{}
}

// containsAny は text が words のどれかを含むか。
func containsAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}
