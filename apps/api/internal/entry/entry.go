// Package entry は大会への申込(代表エントリーとメンバーエントリー)の使い方を持つ。
//
// 依存の向きは httpapi → entry ← store。httpapi は Desk だけを知り、記録の読み書き(Store)は store が
// GORM で実装する。このパッケージは、申込の中身の検査、大会ごとの質問への答えの検査、申込の状態の
// 移り方、承認で作るチームと選手の形といった決まりだけを持ち、HTTP も DB も知らない。申込の入口は
// 管理画面の代理入力(Desk.Submit)と、公式 LINE が返す本人用のリンク(Desk.SubmitByLink)の 2 つで、
// どちらも同じ受付(Service.submit)を通るので決まりは 1 か所に保たれる(docs/line-integration.md)。
package entry

import (
	"context"
	"time"

	"scoreview/api/internal/domain"
)

// Scope は質問に誰が答えるか。
type Scope string

// Scope の値。
const (
	// ScopeTeam は代表がチームについて答える
	ScopeTeam Scope = "team"
	// ScopeMember はメンバーが各自答える
	ScopeMember Scope = "member"
)

// Kind は質問の答え方。
type Kind string

// Kind の値。
const (
	KindShortText Kind = "short_text"
	KindLongText  Kind = "long_text"
	KindChoice    Kind = "choice"
	KindNumber    Kind = "number"
	KindYesNo     Kind = "yes_no"
)

// Status は申込の状態。
type Status string

// Status の値(docs/line-integration.md「申込とチームを分ける」)。
const (
	// StatusPending は運営の確認を待っている
	StatusPending Status = "pending"
	// StatusReturned は運営が直してほしい点を付けて代表に返した
	StatusReturned Status = "returned"
	// StatusApproved は承認して部門のチームと選手ができた
	StatusApproved Status = "approved"
	// StatusCancelled は取り消した
	StatusCancelled Status = "cancelled"
)

// Channel は申込を受けた経路。
type Channel string

// Channel の値。
const (
	// ChannelLine は代表が本人の LINE から申し込んだ
	ChannelLine Channel = "line"
	// ChannelDesk は電話や紙で受けた申込を運営が代わりに入れた
	ChannelDesk Channel = "desk"
)

// Action は運営が申込に対してする操作。
type Action string

// Action の値。
const (
	ActionApprove Action = "approve"
	ActionReturn  Action = "return"
	ActionCancel  Action = "cancel"
	// ActionReopen は参加の決定(承認)を取り消して確認待ちに戻す。承認で作ったチームと選手は消す
	// (押し間違いを戻すため。チームが組や試合に入った後は戻せない)
	ActionReopen Action = "reopen"
)

// Question は大会ごとの質問。ID が 0 の質問は新しく作る。Answered は答えの数で、読むときだけ入る。
type Question struct {
	ID       int64
	Scope    Scope
	Label    string
	Kind     Kind
	Required bool
	// Options は choice の選択肢。choice 以外は空
	Options  []string
	Answered int
}

// Tournament は申込を受ける大会。Name は日本語の名前。
type Tournament struct {
	ID        int64
	Name      string
	Status    domain.TournamentStatus
	Divisions []Division
}

// Division は申込で選べる部門。
type Division struct {
	ID   int64
	Name domain.I18nText
}

// MemberDraft は申込に書くメンバー 1 人。
type MemberDraft struct {
	Name   string
	Number string
}

// Draft は代表エントリーの中身。Answers は質問 ID ごとの答え(チーム向けの質問だけ)。
type Draft struct {
	DivisionID int64
	FullName   string
	ShortName  string
	RepName    string
	RepPhone   string
	RepEmail   string
	Members    []MemberDraft
	Answers    map[int64]string
}

// Answer は質問への答え。
type Answer struct {
	QuestionID int64
	Value      string
}

// NewEntry は Store に保存させる新しい申込。中身は検査と整形を済ませてある。
type NewEntry struct {
	DivisionID int64
	FullName   string
	ShortName  string
	RepName    string
	RepPhone   string
	RepEmail   string
	Members    []MemberDraft
	Channel    Channel
	// RepLineUserID は LINE から申し込んだ代表の line_users の行の ID。運営の代理入力なら nil
	RepLineUserID *int64
	// Profile は同じトランザクションで上書きする、代表の前回の入力(RepLineUserID があるときだけ)
	Profile    *ProfileUpdate
	InviteCode string
	Answers    []Answer
	At         time.Time
}

// Resubmission は Store に保存させる、差し戻した申込の直し。中身は検査と整形を済ませてある(メンバーは
// 代表が書いた並びで、本人が登録した行との合わせ方は Store がロックの中で PlanMembers で決める)。
// Registered は代表がフォームを開いたときに本人が登録していた行の ID で、Store はロックの中の登録と
// 比べる(その間に登録したメンバーが名前を直していると、代表の書いた名前と突き合わせられないため)。
// RepLineUserID と Profile は、同じトランザクションで上書きする代表の前回の入力。
type Resubmission struct {
	Draft         Draft
	Answers       []Answer
	Registered    []int64
	RepLineUserID int64
	Profile       ProfileUpdate
	At            time.Time
}

// Member はメンバーエントリー。Registered はメンバー本人が LINE で登録したか。LineUserID はその人の
// line_users の行の ID(本人用のリンクで「自分の行」を見分けるため。画面には出さない)。
type Member struct {
	ID         int64
	Name       string
	Number     string
	Registered bool
	PlayerID   *int64
	LineUserID *int64
	// Answers は本人が答えたメンバー向けの質問(申込の中身として読んだときだけ入る)。運営の画面にだけ
	// 出し、本人用のリンクでは出さない(ほかのメンバーや代表に、学年などの答えを見せない)
	Answers []Answer
}

// Summary は申込一覧の 1 件。ScorerChosen はチームの記録係を選んだか。
type Summary struct {
	ScorerChosen bool
	ID           int64
	Division     Division
	FullName     string
	ShortName    string
	RepName      string
	Channel      Channel
	Status       Status
	Members      int
	Registered   int
	SubmittedAt  time.Time
	TeamID       *int64
}

// Detail は申込 1 件の全部。RepLineUserID は LINE から申し込んだ代表の line_users の行の ID(運営の代理
// 入力なら nil。代表に知らせるため。画面には出さない)。
type Detail struct {
	Summary
	TournamentID  int64
	RepLineUserID *int64
	RepPhone      string
	RepEmail      string
	ReturnNote    string
	InviteCode    string
	// InviteURL はメンバーの招待リンク(公式アカウントのトークを「参加 招待コード」の入った状態で開く)。
	// 公式アカウントのベーシック ID を設定していなければ空
	InviteURL string
	DecidedAt *time.Time
	// ScorerMemberID はチームの記録係のメンバー(MemberList の ID)。まだ選んでいなければ nil
	ScorerMemberID *int64
	MemberList     []Member
	Answers        []Answer
}

// Decision は申込の状態の変更。Store は申込が From の状態のときだけ To に変える(同時に別の運営が
// 変えていたら domain.ErrConflict)。Approval があれば同じトランザクションでチームと選手を作る。
type Decision struct {
	From      Status
	To        Status
	Note      string
	DecidedBy int64
	At        time.Time
	Approval  *Approval
	// Reopen は参加の決定を取り消して確認待ちに戻す(ActionReopen)。承認で作ったチームと選手を消す
	Reopen *Reopen
}

// Reopen は参加の決定を取り消すときの、申込の部門(組や試合を作る操作と順番に行うため、部門を
// ロックする)。
type Reopen struct {
	DivisionID int64
}

// Approval は承認で作る部門のチームと選手。
type Approval struct {
	DivisionID int64
	FullName   string
	ShortName  string
	// Players は MemberID の順で、作った選手をメンバーエントリーに結びつける
	Players []ApprovedPlayer
}

// ApprovedPlayer は承認で作る選手 1 人。
type ApprovedPlayer struct {
	MemberID int64
	Name     string
	Number   string
}

// Desk は申込の受付。httpapi が使う唯一の口で、Service が実装する。
type Desk interface {
	// Questions は大会の質問を並び順で返す。大会が無ければ domain.NotFound
	Questions(ctx context.Context, tournamentID int64) ([]Question, error)
	// SaveQuestions は大会の質問を questions の並びで置き換える。ID が 0 の質問は作り、無くなった
	// 質問は答えごと消す。答えのある質問の答え方は変えられない(domain.ErrConflict)
	SaveQuestions(ctx context.Context, tournamentID int64, questions []Question) ([]Question, error)
	// Entries は大会の申込を受けた順で返す
	Entries(ctx context.Context, tournamentID int64) ([]Summary, error)
	// Entry は申込 1 件を返す。無ければ domain.NotFound
	Entry(ctx context.Context, id int64) (Detail, error)
	// Submit は代表エントリーを受け付ける。LINE からの申込は大会が受付中のときだけ受ける
	Submit(ctx context.Context, tournamentID int64, channel Channel, draft Draft) (Detail, error)
	// Decide は運営の操作で申込の状態を変える。承認なら部門のチームと選手を作る
	Decide(ctx context.Context, id int64, action Action, note string, actor domain.Actor) (Detail, error)
	// Export は大会の質問と、申込の全部(中身つき、受けた順)を返す。書き出し(CSV)のため
	Export(ctx context.Context, tournamentID int64) (Export, error)
	// LinkView は本人用のリンクで開く画面に要るものを返す。期限切れと見つからないリンクは domain.NotFound
	LinkView(ctx context.Context, token string) (LinkView, error)
	// SubmitByLink は本人用のリンクから代表の申込を受け付ける(LINE からの申込)
	SubmitByLink(ctx context.Context, token string, draft Draft) (Detail, error)
	// ResubmitByLink は差し戻された申込を、本人用のリンクから代表が直して送り直す。申込は申込中に戻る。
	// registered は代表がフォームを開いたときに本人が登録していたメンバーの行の ID。この代表のこの大会の
	// 申込でなければ domain.NotFound、差し戻しでないか、本人の登録がその後に変わっていれば domain.ErrConflict
	ResubmitByLink(ctx context.Context, token string, entryID int64, draft Draft, registered []int64) (Detail, error)
	// JoinByLink は本人用のリンクからメンバー登録を受け付け、登録後の画面を返す
	JoinByLink(ctx context.Context, token string, draft JoinDraft) (MemberForm, error)
	// ChooseScorerByLink は代表が、参加が決まった自分の申込のチームの記録係を、LINE で登録したメンバー
	// から選ぶ(memberID が nil なら外す)。この代表のこの大会の申込でなければ domain.NotFound、参加が
	// 決まっていなければ domain.ErrConflict
	ChooseScorerByLink(ctx context.Context, token string, entryID int64, memberID *int64) (Detail, error)
	// ScorerSignIn は記録係のリンクを確かめ、記録係の利用者を用意してその ID を返す(呼ぶ側がセッション
	// を発行する)。リンクの人がもうそのチームの記録係でなければ domain.NotFound
	ScorerSignIn(ctx context.Context, token string) (int64, error)
}

// Links は公式アカウントのトークの受け答え(talk)が使う、本人用のリンクの発行。Service が実装する。
type Links interface {
	// IssueTeamLinks は代表の申込のリンクを、申込を受け付けている大会と、この人が申込を出した大会
	// (受付を締めた後も、承認や差し戻しを見て招待を送れるように。終わった大会は除く)ごとに発行する。
	// 大会は開催日の順
	IssueTeamLinks(ctx context.Context, lineUserID int64) ([]TournamentLink, error)
	// IssueMemberLink は招待コードのチームへのメンバー登録のリンクを発行する。招待コードが無ければ
	// domain.NotFound、取り消した申込なら domain.ErrConflict
	IssueMemberLink(ctx context.Context, lineUserID int64, inviteCode string) (IssuedLink, error)
	// IssueJoinedLinks はこの人がメンバーとして登録したチームごとに、登録を確かめて直すリンクを発行する
	// (「エントリー」と送ったメンバーに、チームの申込ではなく自分の登録を返すため)。大会は開催日の順
	IssueJoinedLinks(ctx context.Context, lineUserID int64) ([]JoinedLink, error)
	// IssueScorerLinks はこの人が記録係に選ばれたチームごとに、記録の画面に入るリンクを発行する
	// (公式アカウントに「記録」と送ったときの返信)。大会は開催日の順
	IssueScorerLinks(ctx context.Context, lineUserID int64) ([]ScorerLink, error)
}

// Export は書き出す大会の申込。
type Export struct {
	Questions []Question
	Entries   []Detail
}

// Store は申込の記録を読み書きする。store が実装する。
type Store interface {
	// Tournament は大会の状態と部門を読む。無ければ domain.NotFound
	Tournament(ctx context.Context, id int64) (Tournament, error)
	// Questions は大会の質問を並び順で、答えの数とともに読む
	Questions(ctx context.Context, tournamentID int64) ([]Question, error)
	// ReplaceQuestions は大会の質問を並びどおりに置き換える(検査は済ませてある)
	ReplaceQuestions(ctx context.Context, tournamentID int64, questions []Question) ([]Question, error)
	// Create は申込を保存する。同じ部門に同じチーム名の取り消していない申込があれば domain.ErrConflict
	Create(ctx context.Context, tournamentID int64, entry NewEntry) (Detail, error)
	// List は大会の申込を受けた順で読む。問い合わせの数は申込の数によらないこと
	List(ctx context.Context, tournamentID int64) ([]Summary, error)
	// Get は申込 1 件を読む。無ければ domain.NotFound
	Get(ctx context.Context, id int64) (Detail, error)
	// Details は大会の申込を中身つきで受けた順に読む。問い合わせの数は申込の数によらないこと
	Details(ctx context.Context, tournamentID int64) ([]Detail, error)
	// Decide は Decision を当てる(Decision の説明を参照)
	Decide(ctx context.Context, id int64, decision Decision) (Detail, error)
	// LinkTournaments は一覧に載せていて申込を受け付けている(状態が entry_open の)大会と、この LINE の
	// 人が代表として取り消していない申込を出した、終わっていない大会を、開催日の順に読む
	LinkTournaments(ctx context.Context, lineUserID int64) ([]TournamentRef, error)
	// JoinedTeams はこの LINE の人がメンバーとして登録したチームを、取り消した申込と終わった大会を除いて、
	// 開催日の順に読む
	JoinedTeams(ctx context.Context, lineUserID int64) ([]JoinedTeam, error)
	// ScorerTeams はこの LINE の人が記録係に選ばれたチームを、参加が決まった申込の終わっていない大会に
	// 限って、開催日の順に読む
	ScorerTeams(ctx context.Context, lineUserID int64) ([]ScorerTeam, error)
	// ScorerMatches はチーム teamIDs のそれぞれが記録する、まだ終わっていない試合を時間割の順に読む
	// (記録するのは、左のチームに記録係がいれば左、いなければ右のチーム)
	ScorerMatches(ctx context.Context, teamIDs []int64) (map[int64][]ScorerMatch, error)
	// EnsureScorer は記録係の利用者(users.team_scorer。パスワードは持たない)を、この LINE の人に
	// 結んで用意し(あれば使う)、大会 tournamentID の担当にして、利用者の ID を返す。LINE の人が
	// もう別の利用者(運営)に結ばれていれば、その利用者を担当にする
	EnsureScorer(ctx context.Context, lineUserID, tournamentID int64, name string) (int64, error)
	// SaveLink は本人用のリンクを、合言葉の SHA-256 とともに保存する
	SaveLink(ctx context.Context, tokenHash []byte, link Link) error
	// FindLink は合言葉の SHA-256 から本人用のリンクを読む。無ければ domain.NotFound
	FindLink(ctx context.Context, tokenHash []byte) (Link, error)
	// FindByInvite は招待コードの申込を読む。無ければ domain.NotFound
	FindByInvite(ctx context.Context, inviteCode string) (Summary, error)
	// RepEntries は大会にこの LINE の人が代表として出した申込を、受けた順に中身つきで読む
	RepEntries(ctx context.Context, tournamentID, lineUserID int64) ([]Detail, error)
	// MemberAnswers はメンバーエントリーの答えを読む
	MemberAnswers(ctx context.Context, memberID int64) ([]Answer, error)
	// Profile は line_users の行 lineUserID の人の前回の入力を読む。まだ何も入れていなければ nil
	Profile(ctx context.Context, lineUserID int64) (*Profile, error)
	// PastTeams はこの人が代表として申し込んだチームを、大会 exceptTournamentID の申込を除いて、同じ
	// チーム名(前後の空白と大文字小文字を除く)は最後の申込だけにして新しい順に limit まで読む
	PastTeams(ctx context.Context, lineUserID, exceptTournamentID int64, limit int) ([]PastTeam, error)
	// Join はメンバー登録を当てる。代表の行を選んだならその行に LINE の人を結び、名前と背番号を直す。
	// その行がこの間に別の人に結ばれていたら domain.ErrConflict。背番号の重なりはロックの中で確かめ直す。
	// 承認した申込なら選手も直す(無ければ作る)
	Join(ctx context.Context, join Join) error
	// SetScorer は申込のチームの記録係を memberID(nil なら外す)にする。メンバーがこの申込の行で、
	// LINE で登録しているかは呼ぶ側が確かめる(CanChooseScorer)
	SetScorer(ctx context.Context, entryID int64, memberID *int64, at time.Time) (Detail, error)
	// Resubmit は差し戻した申込を直した中身で置き換え、申込中に戻す。申込がこの間に差し戻しでなく
	// なっていたら domain.ErrConflict。メンバーの行はロックの中で PlanMembers に沿って変える
	Resubmit(ctx context.Context, id int64, resubmission Resubmission) (Detail, error)
}
