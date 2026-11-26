package entry

import "context"

// 代表への知らせ(docs/line-integration.md「代表への知らせ」)。LINE から出した申込の代表に、受付、送り直し、
// 承認、差し戻し、取消を公式アカウントから知らせる。知らせは申込の記録とは別で、送れなくても申込の操作は
// 失敗にしない(Service が受け取った報告の関数に渡し、main がログに残す)。

// NoticeKind は知らせる出来事。
type NoticeKind string

// NoticeKind の値。
const (
	// NoticeSubmitted は LINE からの申込を受け付けた(招待リンクを付ける)
	NoticeSubmitted NoticeKind = "submitted"
	// NoticeResubmitted は差し戻した申込の直しを受け付けた
	NoticeResubmitted NoticeKind = "resubmitted"
	// NoticeApproved は運営が承認した
	NoticeApproved NoticeKind = "approved"
	// NoticeReturned は運営が差し戻した(直してほしいことを付ける)
	NoticeReturned NoticeKind = "returned"
	// NoticeCancelled は運営が取り消した
	NoticeCancelled NoticeKind = "cancelled"
	// NoticeReopened は運営が参加の決定を取り消して、確認待ちに戻した
	NoticeReopened NoticeKind = "reopened"
)

// Notice は代表に知らせる出来事。LineUserID は代表の line_users の行の ID、Entry は出来事の後の申込
// (招待リンクと差し戻しの理由を含む)。
type Notice struct {
	Kind       NoticeKind
	LineUserID int64
	Tournament TournamentRef
	Entry      Detail
}

// Notifier は申込の出来事を代表に知らせる。talk が公式アカウントから送る形で実装する。
type Notifier interface {
	// Notify は notice を代表に知らせる。代表がブロックしていれば送らずに nil を返す
	Notify(ctx context.Context, notice Notice) error
}

// DecisionNotice は運営の操作 action のあとに代表へ知らせる出来事を返す。
func DecisionNotice(action Action) NoticeKind {
	switch action {
	case ActionApprove:
		return NoticeApproved
	case ActionReturn:
		return NoticeReturned
	case ActionReopen:
		return NoticeReopened
	default:
		return NoticeCancelled
	}
}
