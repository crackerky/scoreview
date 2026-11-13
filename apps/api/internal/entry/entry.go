// Package entry は大会への申込(代表エントリーとメンバーエントリー)の使い方を持つ。
//
// 依存の向きは httpapi → entry ← store。httpapi は Desk だけを知り、記録の読み書き(Store)は store が
// GORM で実装する。このパッケージは、申込の中身の検査、大会ごとの質問への答えの検査、申込の状態の
// 移り方、承認で作るチームと選手の形といった決まりだけを持ち、HTTP も DB も知らない。申込の入口は
// 管理画面の代理入力と LINE(LIFF)の 2 つになるが、どちらも Desk.Submit を通るので決まりは 1 か所に
// 保たれる(docs/line-integration.md)。
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

// Tournament は申込を受ける大会。
type Tournament struct {
	ID        int64
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
	InviteCode string
	Answers    []Answer
	At         time.Time
}

// Member はメンバーエントリー。Registered はメンバー本人が LINE で登録したか。
type Member struct {
	ID         int64
	Name       string
	Number     string
	Registered bool
	PlayerID   *int64
}

// Summary は申込一覧の 1 件。
type Summary struct {
	ID          int64
	Division    Division
	FullName    string
	ShortName   string
	RepName     string
	Channel     Channel
	Status      Status
	Members     int
	Registered  int
	SubmittedAt time.Time
	TeamID      *int64
}

// Detail は申込 1 件の全部。
type Detail struct {
	Summary
	TournamentID int64
	RepPhone     string
	RepEmail     string
	ReturnNote   string
	InviteCode   string
	DecidedAt    *time.Time
	MemberList   []Member
	Answers      []Answer
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
	// Decide は Decision を当てる(Decision の説明を参照)
	Decide(ctx context.Context, id int64, decision Decision) (Detail, error)
}
