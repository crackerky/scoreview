package notify

import (
	"fmt"
	"strings"

	"scoreview/api/internal/domain"
)

// 運営が文に書く差し込み。送る人ごとに、その人のチームの値に置き換えて送る。全角の括弧(｛チーム名｝)で
// 書いても同じに扱う(日本語の入力のまま打つと全角になるため)。
const (
	// VarTeam はチーム名(チームが 2 つ以上なら「、」で並べる)
	VarTeam = "チーム名"
	// VarInvite はメンバーの招待リンク(この大会に申し込んだ代表だけ)
	VarInvite = "招待リンク"
	// VarUnregistered は申込に名前があって、まだ公式 LINE で登録していないメンバー(この大会に申し込んだ
	// 代表だけ)
	VarUnregistered = "まだ登録していない人"
)

// noOne は、まだ登録していない人がいないときの差し込み。
const noOne = "いません"

// Variable は送る画面に出す差し込みと、見本の人の値。
type Variable struct {
	Name    string
	Example string
}

// RecipientEntry は宛先の人が代表の申込 1 件(この大会のもの)。差し込みの値の元。InviteCode と
// Unregistered は、代表の申込にだけ入る(メンバーと前の大会の代表では空)。
type RecipientEntry struct {
	Team         string
	InviteCode   string
	Unregistered []string
}

// VariablesFor は宛先の条件で使える差し込み。招待リンクとまだ登録していない人は、この大会に申し込んだ
// 代表に送るときだけ(メンバーと前の大会の代表には、この大会の招待リンクが無い)。
func VariablesFor(kind AudienceKind) []string {
	switch kind {
	case AudienceReps, AudienceIncomplete, AudienceNoScorer:
		return []string{VarTeam, VarInvite, VarUnregistered}
	default:
		return []string{VarTeam}
	}
}

// normalizeVariables は全角の括弧で書いた差し込みを半角の括弧にそろえる。
func normalizeVariables(body string) string {
	for _, name := range []string{VarTeam, VarInvite, VarUnregistered} {
		body = strings.ReplaceAll(body, "｛"+name+"｝", "{"+name+"}")
	}
	return body
}

// usedVariables は文に書いた差し込み(半角の括弧にそろえた後の文で数える)。
func usedVariables(body string) []string {
	var out []string
	for _, name := range []string{VarTeam, VarInvite, VarUnregistered} {
		if strings.Contains(body, "{"+name+"}") {
			out = append(out, name)
		}
	}
	return out
}

// CheckVariables は文の差し込みを宛先の条件で使えるか確かめ、括弧をそろえた文を返す。
func CheckVariables(body string, kind AudienceKind) (string, error) {
	normalized := normalizeVariables(body)
	allowed := VariablesFor(kind)
	for _, name := range usedVariables(normalized) {
		found := false
		for _, ok := range allowed {
			found = found || ok == name
		}
		if !found {
			return "", domain.Invalid("body",
				fmt.Sprintf("{%s} can be used only for the reps of this tournament", name))
		}
	}
	return normalized, nil
}

// fillValues は宛先の人の差し込みの値。invite は招待コードからリンクを作る。
func fillValues(recipient Recipient, invite func(code string) string) map[string]string {
	entries := recipient.Entries
	if len(entries) == 0 {
		for _, team := range recipient.Teams {
			entries = append(entries, RecipientEntry{Team: team})
		}
	}
	teams := make([]string, len(entries))
	invites := make([]string, len(entries))
	pending := make([]string, len(entries))
	for i, item := range entries {
		teams[i] = item.Team
		invites[i] = invite(item.InviteCode)
		names := noOne
		if len(item.Unregistered) > 0 {
			names = strings.Join(item.Unregistered, "、")
		}
		pending[i] = names
		// チームが 2 つ以上なら、どのチームの値か分かるようにチーム名を前に付けて 1 行ずつ並べる
		if len(entries) > 1 {
			invites[i] = item.Team + " " + invites[i]
			pending[i] = item.Team + ": " + names
		}
	}
	return map[string]string{
		VarTeam:         strings.Join(teams, "、"),
		VarInvite:       strings.Join(invites, "\n"),
		VarUnregistered: strings.Join(pending, "\n"),
	}
}

// Fill は文の差し込みを宛先の人の値に置き換える(差し込みが無ければ文のまま)。
func Fill(text string, recipient Recipient, invite func(code string) string) string {
	used := usedVariables(text)
	if len(used) == 0 {
		return text
	}
	values := fillValues(recipient, invite)
	for _, name := range used {
		text = strings.ReplaceAll(text, "{"+name+"}", values[name])
	}
	return text
}

// Examples は宛先の条件で使える差し込みと、送れる最初の人の値(送る画面の見本)。送れる人がいなければ
// 値は空。
func Examples(kind AudienceKind, recipients []Recipient, invite func(code string) string) (string, []Variable) {
	names := VariablesFor(kind)
	out := make([]Variable, len(names))
	for i, name := range names {
		out[i] = Variable{Name: name}
	}
	for _, recipient := range recipients {
		if !recipient.Friend {
			continue
		}
		values := fillValues(recipient, invite)
		for i := range out {
			out[i].Example = values[out[i].Name]
		}
		return recipient.Name, out
	}
	return "", out
}
