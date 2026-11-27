package entry

import (
	"fmt"
	"strings"
	"time"

	"scoreview/api/internal/domain"
)

// 本人用のリンク(docs/line-integration.md「Webhook で本人を確かめる」)。公式アカウントに届いた
// メッセージの返信で、推測できない合言葉を入れた URL を返し、その URL を開いた人を、メッセージを
// 送った LINE の人として扱う。

// LinkPurpose は本人用のリンクで何をするか。
type LinkPurpose string

// LinkPurpose の値。
const (
	// LinkTeam は大会への代表の申込
	LinkTeam LinkPurpose = "team"
	// LinkMember は代表エントリーへのメンバー登録
	LinkMember LinkPurpose = "member"
	// LinkScorer はチームの記録係が記録の画面に入る(TeamEntryID は記録係のチームの申込)
	LinkScorer LinkPurpose = "scorer"
)

// LinkLifetimeDays は本人用のリンクを使える日数。リンクを転送されたときに、転送された人が使える期間を
// 限る。代表が週末にまとめて書くことを考えて 3 日にする。切れたら公式アカウントに送り直せば新しい
// リンクが返る。公式アカウントの返信の文もこの日数で書く。
const LinkLifetimeDays = 3

// LinkLifetime は本人用のリンクを使える長さ(LinkLifetimeDays)。
const LinkLifetime = LinkLifetimeDays * 24 * time.Hour

// Link は本人用のリンク。LineUserID は line_users の行の ID(公式アカウントから見たユーザー ID ではない)。
// Purpose が team なら TournamentID、member なら TeamEntryID が入る。
type Link struct {
	LineUserID   int64
	Purpose      LinkPurpose
	TournamentID int64
	TeamEntryID  int64
	ExpiresAt    time.Time
}

// IssuedLink は発行した本人用のリンク。TeamName はメンバー登録のリンクのときのチーム名。
type IssuedLink struct {
	Token     string
	ExpiresAt time.Time
	TeamName  string
}

// TournamentRef は大会の ID と日本語の名前。
type TournamentRef struct {
	ID   int64
	Name string
}

// TournamentLink は大会への代表の申込のリンク。
type TournamentLink struct {
	Tournament TournamentRef
	Token      string
	ExpiresAt  time.Time
}

// JoinedTeam はこの LINE の人がメンバーとして登録したチーム(取り消していない申込の、終わっていない
// 大会)。TeamEntryID は代表エントリーの ID。
type JoinedTeam struct {
	TeamEntryID int64
	TeamName    string
	Tournament  TournamentRef
}

// JoinedLink はメンバーとして登録したチームの、登録を確かめて直すリンク。
type JoinedLink struct {
	Team      JoinedTeam
	Token     string
	ExpiresAt time.Time
}

// TeamForm は代表の申込の画面に要るもの。Open は大会が申込を受け付けているか(締めた後も、出した申込は
// 見られる)。Mine はこの人がこの大会に出した申込(受けた順)。Profile はこの人の前回の入力(まだ何も
// 入れていなければ nil)、PastTeams は前にほかの大会に申し込んだチーム(新しい順、MaxPastTeams まで)。
type TeamForm struct {
	Tournament TournamentRef
	Open       bool
	Divisions  []Division
	Questions  []Question
	Mine       []Detail
	Profile    *Profile
	PastTeams  []PastTeam
	ExpiresAt  time.Time
}

// MemberForm はメンバー登録の画面に要るもの。Unclaimed は代表が書いた、まだ誰も自分の LINE を
// 結んでいない行。Mine はこの人がもう登録した行(無ければ nil)で、MineAnswers はその答え。ProfileName は
// この人が前に入れた名前(無ければ空。まだ登録していない人の名前の欄に最初から入れる)。
type MemberForm struct {
	TournamentName string
	Team           Summary
	Questions      []Question
	Unclaimed      []Member
	Mine           *Member
	MineAnswers    []Answer
	ProfileName    string
	ExpiresAt      time.Time
}

// LinkView は本人用のリンクで開く画面。Purpose に合わせて Team か Member のどちらかが入る。
type LinkView struct {
	Purpose LinkPurpose
	Team    *TeamForm
	Member  *MemberForm
	Scorer  *ScorerView
}

// ScorerTeam はこの LINE の人が記録係に選ばれたチーム(参加が決まった申込の、終わっていない大会)。
// MemberName は記録係のメンバーの名前。
type ScorerTeam struct {
	JoinedTeam
	MemberName string
}

// ScorerLink は記録係が記録の画面に入る本人用のリンク。
type ScorerLink struct {
	Team      ScorerTeam
	Token     string
	ExpiresAt time.Time
}

// ScorerView は記録係のリンクで開く画面に要るもの(記録の画面に入るボタンの前に、どのチームの記録係かを
// 見せる)。
type ScorerView struct {
	Team      ScorerTeam
	ExpiresAt time.Time
}

// JoinDraft はメンバー登録の中身。MemberID は代表が書いた行のうち自分の行(新しく足すなら nil)。
type JoinDraft struct {
	MemberID *int64
	Name     string
	Number   string
	Answers  map[int64]string
}

// Join は Store に保存させるメンバー登録。中身は検査と整形を済ませてある。Profile は同じトランザクションで
// 上書きする、この人の前回の入力。
type Join struct {
	TeamEntryID int64
	LineUserID  int64
	MemberID    *int64
	Name        string
	Number      string
	Answers     []Answer
	Profile     ProfileUpdate
	At          time.Time
}

// CheckLink はリンクが今使えるかを確かめる。期限切れは、見つからないリンクと同じエラーにする(どちらも
// 「公式アカウントに送り直す」で直るので、分けて伝える意味が無い)。
func CheckLink(link Link, now time.Time) error {
	if !now.Before(link.ExpiresAt) {
		return domain.NotFound(domain.ResourceEntryLink, 0)
	}
	return nil
}

// CheckPurpose はリンクが purpose のためのものかを確かめる。違う目的のリンクも、見つからないリンクと
// 同じエラーにする(メンバー登録のリンクで代表の申込をさせない)。
func CheckPurpose(link Link, purpose LinkPurpose) error {
	if link.Purpose != purpose {
		return domain.NotFound(domain.ResourceEntryLink, 0)
	}
	return nil
}

// CanJoin はチームにメンバーを登録してよいかを確かめる。取り消した申込には登録させない。
func CanJoin(status Status) error {
	if status == StatusCancelled {
		return fmt.Errorf("%w: the entry of this team has been cancelled", domain.ErrConflict)
	}
	return nil
}

// RepRows は、line_users の行 lineUserID の人がメンバー登録で選べない名簿の行(代表の行)の ID を返す。
// LINE で申し込んだ代表の申込で、その人が代表本人でないとき、本人がまだ登録していない、名前が代表と
// 同じ行(空白の違いは見ない)を返す。招待のリンクはチームの中で回すので、ほかの人が代表の行を選ぶと、
// 代表が自分の行で登録できなくなる。運営が代わりに入れた申込は、代表の LINE が分からないので返さない。
func RepRows(detail Detail, lineUserID int64) map[int64]bool {
	rows := map[int64]bool{}
	if detail.RepLineUserID == nil || *detail.RepLineUserID == lineUserID {
		return rows
	}
	rep := withoutSpaces(detail.RepName)
	for _, member := range detail.MemberList {
		if !member.Registered && withoutSpaces(member.Name) == rep {
			rows[member.ID] = true
		}
	}
	return rows
}

// withoutSpaces は名前の空白(全角も)を除く(「山田 太郎」と「山田太郎」を同じ人として比べる)。
func withoutSpaces(name string) string {
	return strings.Join(strings.Fields(name), "")
}

// CheckJoin はメンバー登録の中身を確かめ、前後の空白を除いた中身と、整えた答えを返す。members は
// チームの今のメンバーで、選んだ行がこのチームの、まだ誰も結んでいない行(または自分の行)であること、
// 代表の行(repRows。RepRows の結果)でないこと、背番号がほかの人と重ならないことを確かめる。mine は
// この人がもう登録した行(無ければ nil)。
func CheckJoin(
	draft JoinDraft,
	members []Member,
	mine *Member,
	repRows map[int64]bool,
	questions []Question,
) (JoinDraft, []Answer, error) {
	name, number := strings.TrimSpace(draft.Name), strings.TrimSpace(draft.Number)
	if err := checkLine("name", name, MaxShortText, true); err != nil {
		return JoinDraft{}, nil, err
	}
	if err := checkLine("number", number, MaxNumberText, false); err != nil {
		return JoinDraft{}, nil, err
	}
	// 自分の行は、もう登録していればその行、代表の行を選んだならその行、どちらでもなければ新しい行
	self := int64(0)
	if mine != nil {
		self = mine.ID
	}
	if draft.MemberID != nil {
		picked := findMember(members, *draft.MemberID)
		if picked == nil {
			return JoinDraft{}, nil, domain.Invalid("memberId", "is not a member of this team")
		}
		if repRows[picked.ID] {
			return JoinDraft{}, nil, fmt.Errorf("%w: only the representative can register as the representative's row",
				domain.ErrConflict)
		}
		if picked.Registered && picked.ID != self {
			return JoinDraft{}, nil, fmt.Errorf("%w: someone else has already registered as this member",
				domain.ErrConflict)
		}
		if mine != nil && picked.ID != mine.ID {
			return JoinDraft{}, nil, domain.Invalid("memberId", "you have already registered as another member")
		}
		self = picked.ID
	}
	if err := CheckNumberFree(members, self, number); err != nil {
		return JoinDraft{}, nil, err
	}
	answers, err := CheckAnswers(questions, ScopeMember, draft.Answers)
	if err != nil {
		return JoinDraft{}, nil, err
	}
	out := JoinDraft{Name: name, Number: number, Answers: draft.Answers}
	if self != 0 {
		out.MemberID = &self
	}
	return out, answers, nil
}

// CheckNumberFree は背番号 number が、チームの self 以外のメンバーと重ならないかを確かめる(空の背番号は
// 重ならない)。self は登録する人の行(新しい行なら 0)。
func CheckNumberFree(members []Member, self int64, number string) error {
	if number == "" {
		return nil
	}
	for _, member := range members {
		if member.ID != self && member.Number == number {
			return domain.Invalid("number", "is used by another member")
		}
	}
	return nil
}

// MemberOf は members のうち、line_users の行 lineUserID の人が登録した行を返す(無ければ nil)。
func MemberOf(members []Member, lineUserID int64) *Member {
	for i := range members {
		if members[i].LineUserID != nil && *members[i].LineUserID == lineUserID {
			return &members[i]
		}
	}
	return nil
}

func findMember(members []Member, id int64) *Member {
	for i := range members {
		if members[i].ID == id {
			return &members[i]
		}
	}
	return nil
}
