package entry

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"scoreview/api/internal/domain"
)

// 前の大会の申込(Lイベントなど、ほかの受付で受けた申込を移したもの。docs/line-integration.md
// 「Lイベントからの移行」)。終わった、一覧に載せない大会として、承認済みの申込と、参加者の台帳
// (line_users)に入れる。参加者の台帳に入れておくと、公式アカウントの Webhook をこちらに切り替えた
// 後、前に申し込んだ人の名前と連絡先と前のチームが、申込の画面に最初から入る。

// PastTournament は移す大会。Divisions は部門(Entries の Division はこの添字)。
type PastTournament struct {
	Name      string
	StartDate *time.Time
	Divisions []PastDivision
	Entries   []PastEntry
}

// PastDivision は移す部門。
type PastDivision struct {
	Name string
	Kind domain.DivisionKind
}

// PastEntry は移す申込 1 件。RepLineID は代表の、公式アカウントから見たユーザー ID。InviteCode は
// NewInviteCode で作ったもの(移した申込にもメンバーの招待リンクがあるようにする)。
type PastEntry struct {
	Division    int
	FullName    string
	RepName     string
	RepEmail    string
	RepLineID   string
	SubmittedAt time.Time
	InviteCode  string
	Members     []PastMember
}

// PastMember は移すメンバー 1 人。LineID は本人が登録した LINE のユーザー ID(代表が書いただけの
// 行なら空)。SubmittedAt は本人が登録した時刻(参加者の台帳の名前の新しさを比べる)。
type PastMember struct {
	Name        string
	Number      string
	LineID      string
	SubmittedAt time.Time
}

// IsLineUserID は id が公式アカウントから見たユーザー ID の形(U と 16 進の英小文字 32 桁)か。
func IsLineUserID(id string) bool {
	const hexDigits = 32
	if len(id) != hexDigits+1 || id[0] != 'U' {
		return false
	}
	return !strings.ContainsFunc(id[1:], notLowerHex)
}

func notLowerHex(r rune) bool {
	return (r < '0' || r > '9') && (r < 'a' || r > 'f')
}

// CheckPast は移す大会を、申込の画面と同じ上限で確かめる。誤りの場所は大会の中の位置で返す。
func CheckPast(past PastTournament) error {
	if err := checkLine("name", past.Name, MaxShortText*2, true); err != nil {
		return err
	}
	if len(past.Divisions) == 0 {
		return domain.Invalid("divisions", "needs at least one division")
	}
	for i, division := range past.Divisions {
		if err := checkLine(fmt.Sprintf("divisions[%d].name", i), division.Name, MaxShortText, true); err != nil {
			return err
		}
	}
	for i, entry := range past.Entries {
		field := fmt.Sprintf("entries[%d]", i)
		if entry.Division < 0 || entry.Division >= len(past.Divisions) {
			return domain.Invalid(field+".division", "is not one of the divisions")
		}
		if err := checkPastEntry(field, entry); err != nil {
			return err
		}
	}
	return nil
}

func checkPastEntry(field string, entry PastEntry) error {
	texts := []struct {
		name, value string
		required    bool
	}{
		{"fullName", entry.FullName, true},
		{"repName", entry.RepName, true},
		{"repEmail", entry.RepEmail, false},
	}
	for _, text := range texts {
		if err := checkLine(field+"."+text.name, text.value, MaxShortText, text.required); err != nil {
			return err
		}
	}
	if !IsLineUserID(entry.RepLineID) {
		return domain.Invalid(field+".repLineId", "must be a LINE user ID")
	}
	if !IsInviteCode(entry.InviteCode) {
		return domain.Invalid(field+".inviteCode", "must be made by NewInviteCode")
	}
	if len(entry.Members) > MaxMembers {
		return domain.Invalid(field+".members", fmt.Sprintf("must be at most %d", MaxMembers))
	}
	seen := map[string]bool{}
	for i, member := range entry.Members {
		at := fmt.Sprintf("%s.members[%d]", field, i)
		if err := checkLine(at+".name", member.Name, MaxShortText, true); err != nil {
			return err
		}
		if utf8.RuneCountInString(member.Number) > MaxNumberText {
			return domain.Invalid(at+".number", fmt.Sprintf("must be at most %d characters", MaxNumberText))
		}
		if member.LineID == "" {
			continue
		}
		if !IsLineUserID(member.LineID) {
			return domain.Invalid(at+".lineId", "must be a LINE user ID")
		}
		// 同じ人が同じチームに 2 回登録しない(member_entries の一意の索引と同じ)
		if seen[member.LineID] {
			return domain.Invalid(at+".lineId", "is registered twice in the team")
		}
		seen[member.LineID] = true
	}
	return nil
}
