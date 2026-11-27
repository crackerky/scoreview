// Package notify は、管理画面から条件で選んだ相手に公式アカウントから送る(docs/line-integration.md
// 「管理画面から送る仕組み」)。
//
// 依存の向きは httpapi → notify ← store。宛先の条件はコードで名前を付けたものだけにし、管理画面から
// 自由な SQL は書かせない(宛先の選び方に権限の確認が効かなくなるため)。送る口(Sender)は LINE の
// HTTP の形を知る line が実装する。
package notify

import (
	"context"
	"errors"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

// AudienceKind は宛先の条件。
type AudienceKind string

// AudienceKind の値。どれも LINE から申し込んだ人(公式アカウントの友だちとして記録した人)だけで、
// 運営が代わりに入れた申込の代表には LINE が無いので含まない。
const (
	// AudienceReps はこの大会に申し込んだ代表(取り消した申込を除く。Status で状態を絞れる)
	AudienceReps AudienceKind = "reps"
	// AudienceIncomplete はメンバーの登録が揃っていない申込の代表(取り消した申込を除く)
	AudienceIncomplete AudienceKind = "incomplete"
	// AudienceParticipants は承認した申込の代表と、LINE で登録したメンバー
	AudienceParticipants AudienceKind = "participants"
	// AudiencePastReps はほかの大会に申し込んだことがあり、この大会にはまだ申し込んでいない代表
	// (受付を始めたことの告知)
	AudiencePastReps AudienceKind = "past_reps"
	// AudienceNoScorer は参加が決まった申込のうち、まだ記録係を選んでいないチームの代表(記録係を選んで
	// もらう頼みと催促)
	AudienceNoScorer AudienceKind = "no_scorer"
)

// ButtonKind は届く文の下のボタンの種類。
type ButtonKind string

// ButtonKind の値。
const (
	// ButtonEntry は公式アカウントに「エントリー」と送るボタン(押すと本人用の申込のリンクが返る)
	ButtonEntry ButtonKind = "entry"
	// ButtonSite はスコアサイトの大会のページを開くボタン(URL は Service が決める)
	ButtonSite ButtonKind = "site"
	// ButtonURL は運営が書いた URL を開くボタン
	ButtonURL ButtonKind = "url"
)

// Button は届く文の下のボタン。URL は site と url のときの行き先(entry では空)。
type Button struct {
	Kind  ButtonKind
	Label string
	URL   string
}

// Message は届けるもの。Buttons が空なら文だけ、あれば文の下にボタンを並べたカードで届ける。
type Message struct {
	Text    string
	Buttons []Button
}

// Audience は宛先の条件と絞り込み。
type Audience struct {
	Kind AudienceKind
	// Status は Kind が reps のときだけ使う申込の状態。空なら取り消した申込を除く全部
	Status entry.Status
	// DivisionID は部門で絞る。0 なら全部の部門(past_reps では使わない)
	DivisionID int64
}

// Recipient は宛先の 1 人。LineUserID は line_users の行の ID、UserID は公式アカウントから見たユーザー ID。
// Friend が偽(ブロック中)の人には送らない(送っても届かず、配信数だけが減る)。Name はその人が最後に
// 入れた名前(無ければ申込に書かれた名前)、Teams は条件に当てはまった申込のチーム名。
type Recipient struct {
	LineUserID int64
	UserID     string
	Friend     bool
	Name       string
	Teams      []string
}

// Quota は公式アカウントの今月の配信数。Limit が nil なら上限が無い(追加メッセージのあるプラン)。
type Quota struct {
	Limit *int64
	Used  int64
}

// Preview は送る前に確かめるもの。Heading は届く文の 1 行目に付ける大会名。Quota は LINE から読めな
// かったときと、LINE の設定が無いときは nil。CanSend は LINE の設定があって送れるか。
type Preview struct {
	Heading     string
	Recipients  []Recipient
	Reachable   int
	Unreachable int
	Quota       *Quota
	CanSend     bool
}

// JobStatus は宛先ごとの結果。
type JobStatus string

// JobStatus の値。
const (
	// JobPending は送る前か、送っている途中
	JobPending JobStatus = "pending"
	// JobSent は LINE が受け付けた(相手の端末に届いたかまでは LINE が返さない)
	JobSent JobStatus = "sent"
	// JobSkipped は友だちでないので送らなかった
	JobSkipped JobStatus = "skipped"
	// JobFailed は LINE が受け付けなかった
	JobFailed JobStatus = "failed"
)

// Job は宛先 1 人への結果。Name と Teams は送った時点のもの。
type Job struct {
	Name   string
	Teams  string
	Status JobStatus
	Error  string
	SentAt *time.Time
}

// Counts は 1 回の送信の結果ごとの人数。Resendable は今送り直せる人数(送れなかった人と、送っている
// 送信の持ち時間が切れた途中の人。Store が now で数える)。
type Counts struct {
	Pending    int
	Sent       int
	Skipped    int
	Failed     int
	Resendable int
}

// BatchState は送信の段階。
type BatchState string

// BatchState の値。
const (
	// BatchScheduled は送る時刻を待っている予約(宛先はまだ無い)
	BatchScheduled BatchState = "scheduled"
	// BatchSent は宛先を決めて送った(すぐ送った送信と、送る時刻の来た予約)
	BatchSent BatchState = "sent"
	// BatchCancelled は運営が取りやめた予約
	BatchCancelled BatchState = "cancelled"
	// BatchSkipped は送る時刻に送れなかった予約(理由は Note)
	BatchSkipped BatchState = "skipped"
)

// SkipReason は送る時刻に送れなかった理由。
type SkipReason string

// SkipReason の値。
const (
	// SkipNoRecipients は条件に当てはまる友だちがいなかった
	SkipNoRecipients SkipReason = "no_recipients"
	// SkipQuota は今月の残りの配信数が足りなかった(一部の人にだけ届くのを避けて、何も送らない)
	SkipQuota SkipReason = "quota"
	// SkipNotConfigured は LINE の設定が無かった
	SkipNotConfigured SkipReason = "line_not_configured"
)

// Batch は 1 回の送信の記録。Body は実際に届けた文(大会名の行を含む)。SentByName は送った運営の
// 表示名(運営を消していれば空)。Jobs は Store.Batch で読んだときだけ入る。CanResend は送れなかった
// 宛先を今送り直せるか(Service が決める。Store は偽のまま返す)。SendAt は予約の送る時刻(すぐ送った
// 送信は nil)、Note は送れなかった予約の理由。
type Batch struct {
	ID           int64
	TournamentID int64
	Audience     Audience
	Body         string
	Buttons      []Button
	Automatic    bool
	SentByName   string
	CreatedAt    time.Time
	SendAt       *time.Time
	State        BatchState
	Note         SkipReason
	Counts       Counts
	Jobs         []Job
	CanResend    bool
}

// History は大会の送信の記録(新しい順)と、今送れるか。
type History struct {
	Batches []Batch
	CanSend bool
}

// Request は運営の「送る」。Body は運営が書いた文(大会名の行は Service が付ける)。RequestKey は送る
// 画面を開くたびに作る乱数で、同じ画面から 2 回届いても 2 回目は送らない。SendAt は送る時刻(nil なら
// すぐ送る)。
type Request struct {
	Audience   Audience
	Body       string
	Buttons    []Button
	RequestKey string
	SendAt     *time.Time
}

// Due は送る時刻の来た予約。
type Due struct {
	ID           int64
	TournamentID int64
	Audience     Audience
	Body         string
	Buttons      []Button
}

// DispatchResult は定期の呼び出しで扱った予約の数。
type DispatchResult struct {
	Sent    int
	Skipped int
}

// NewJob は記録する宛先 1 人。RetryKey はその人が入るまとまりの送り直しのキー(送らない人は空)。
type NewJob struct {
	LineUserID int64
	Name       string
	Teams      string
	Status     JobStatus
	RetryKey   string
}

// Group は 1 回のマルチキャストで送る宛先のまとまり。送り直すときは、同じキーで同じ宛先に同じ文を
// 送る(LINE は宛先や文を変えた送り直しの結果を決めていない)。
type Group struct {
	RetryKey    string
	LineUserIDs []int64
	UserIDs     []string
}

// NewBatch は記録する送信。Body は届ける文。SentBy は送った運営(0 なら運営なし)。ClaimedUntil は、
// この送信が送る宛先(pending)を持つ期限。SendAt があれば予約で、宛先(Jobs)を持たない。
type NewBatch struct {
	TournamentID int64
	RequestKey   string
	Audience     Audience
	Body         string
	Buttons      []Button
	SentBy       int64
	At           time.Time
	SendAt       *time.Time
	ClaimedUntil time.Time
	Jobs         []NewJob
}

// JobResult は送った結果。
type JobResult struct {
	Status JobStatus
	Error  string
	At     time.Time
}

// Store は宛先を選び、送信を記録する。store が実装する。
type Store interface {
	// Tournament は大会の名前と部門を読む。無ければ domain.NotFound
	Tournament(ctx context.Context, id int64) (entry.Tournament, error)
	// Recipients は条件に当てはまる人を、同じ人を 1 人にまとめて返す(条件は CheckAudience で確かめてある)
	Recipients(ctx context.Context, tournamentID int64, audience Audience) ([]Recipient, error)
	// FindBatch は RequestKey の送信の記録の ID を返す。無ければ偽
	FindBatch(ctx context.Context, requestKey string) (id int64, found bool, err error)
	// CreateBatch は送信と宛先ごとの行を 1 つのトランザクションで記録する。同じ RequestKey の記録が
	// もうあれば作らず、その ID と偽を返す
	CreateBatch(ctx context.Context, batch NewBatch) (id int64, created bool, err error)
	// Claim は送信 batchID の、送れなかった(failed)宛先と、持ち時間が now で切れた途中(pending)の
	// 宛先を、until まで持つことにして(pending にして)、記録した順のまとまりで返す。ほかの送信が
	// 持っている宛先は返さない(同時に呼ばれても、1 つのまとまりは 1 つの呼び出しにだけ返す)
	Claim(ctx context.Context, batchID int64, now, until time.Time) ([]Group, error)
	// FinishJobs は送信 batchID の、line_users の行の ID が lineUserIDs の宛先の結果を書く
	FinishJobs(ctx context.Context, batchID int64, lineUserIDs []int64, result JobResult) error
	// Batches は大会の送信を新しい順に、結果ごとの人数(送り直せる人数は now で数える)とともに読む
	Batches(ctx context.Context, tournamentID int64, now time.Time) ([]Batch, error)
	// Batch は送信 1 回を宛先ごとの結果とともに読む。無ければ domain.NotFound
	Batch(ctx context.Context, id int64, now time.Time) (Batch, error)
	// DueBatches は送る時刻が now までの予約を、送る時刻の順に limit 件まで読む
	DueBatches(ctx context.Context, now time.Time, limit int) ([]Due, error)
	// StartScheduled は予約 id を送ったことにし、宛先を同じトランザクションで書く(宛先の持ち時間は
	// claimedUntil まで)。予約がもう予約でなければ(ほかの呼び出しが送った、運営が取りやめた)書かずに偽
	StartScheduled(ctx context.Context, id int64, jobs []NewJob, claimedUntil time.Time) (bool, error)
	// SkipScheduled は予約 id を送れなかったことにする。もう予約でなければ偽
	SkipScheduled(ctx context.Context, id int64, reason SkipReason) (bool, error)
	// CancelScheduled は予約 id を取りやめる。もう予約でなければ偽
	CancelScheduled(ctx context.Context, id int64) (bool, error)
}

// MaxMulticast は 1 回の Multicast に渡せる宛先の数(Messaging API のマルチキャストの上限)。
// https://developers.line.biz/ja/reference/messaging-api/#send-multicast-message
const MaxMulticast = 500

// ResendWindow は送れなかった宛先を送り直せる期間。LINE の送り直しのキーが有効な 24 時間。
// https://developers.line.biz/ja/docs/messaging-api/retrying-api-request/
const ResendWindow = 24 * time.Hour

// SendLease は送信がまとまりを持つ時間。api の 1 回の要求が続きうる時間(Vercel の関数の最長 800 秒)
// より長くし、持ち時間の切れたまとまりを送っている送信が残らないようにする。
const SendLease = 15 * time.Minute

// ErrTemporary は LINE が受け付けたか分からない失敗(LINE の 5xx、応答が無い)。同じ送り直しのキーで
// もう一度送ってよい。LINE が受け付けなかったと分かる失敗(4xx)は、送り直しても結果が変わらない。
var ErrTemporary = errors.New("temporary LINE failure")

// Sender は公式アカウントとして送る。line が実装する。
type Sender interface {
	// Multicast は to(公式アカウントから見たユーザー ID。MaxMulticast 人まで)に message を送る。送った
	// 人数が月の配信数に数えられる。retryKey(UUID)が同じ送り直しは、LINE が 1 回目を受け付けて
	// いれば送らずに成功を返す。受け付けたか分からない失敗は ErrTemporary を包んで返す
	Multicast(ctx context.Context, to []string, message Message, retryKey string) error
	// Quota は今月の配信数の上限と使った数を読む
	Quota(ctx context.Context) (Quota, error)
}

// Desk は管理画面の「送る」。Service が実装する。
type Desk interface {
	// Preview は条件に当てはまる人と、今回使う配信数を返す
	Preview(ctx context.Context, tournamentID int64, audience Audience) (Preview, error)
	// Send は条件に当てはまる人に送り、記録を返す。同じ RequestKey の送信がもうあれば、送らずに
	// その記録を返す
	Send(ctx context.Context, tournamentID int64, request Request, actor domain.Actor) (Batch, error)
	// Resend は送信 id の送れなかった宛先に、同じまとまり、同じキー、同じ文で送り直す。送ってから
	// ResendWindow を過ぎていれば domain.ErrConflict
	Resend(ctx context.Context, id int64) (Batch, error)
	// History は大会の送信の記録を返す
	History(ctx context.Context, tournamentID int64) (History, error)
	// Batch は送信 1 回の記録を返す
	Batch(ctx context.Context, id int64) (Batch, error)
	// Cancel は予約 id を取りやめる。もう予約でなければ domain.ErrConflict
	Cancel(ctx context.Context, id int64) (Batch, error)
	// Dispatch は送る時刻の来た予約を送る(定期の呼び出しから)
	Dispatch(ctx context.Context) (DispatchResult, error)
}
