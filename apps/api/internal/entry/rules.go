package entry

import (
	"encoding/base32"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"scoreview/api/internal/domain"
)

// 申込の上限。現実の値を十分に超える範囲で、保存の前に検証エラーにする(大きすぎる入力で DB と画面を
// 重くしないため)。web の entryLimits(apps/web/src/api/entry.types.ts)と揃える。
const (
	// MaxQuestions は大会 1 つの質問の数。Lイベントの申込フォームでも 10 問前後が普通
	MaxQuestions = 30
	// MaxOptions は選択式の質問 1 つの選択肢の数
	MaxOptions = 30
	// MaxMembers はチーム 1 つのメンバーの数。アルティメットの登録は多くて 30 人前後
	MaxMembers = 60
	// MaxShortText は名前や短い答えの文字数
	MaxShortText = 100
	// MaxNumberText は背番号の文字数(「00」や「#7」のような書き方も入る長さ)
	MaxNumberText = 10
	// MaxLongText は長い答えと差し戻しの理由の文字数
	MaxLongText = 2000
	// MaxNumber は数の答えの絶対値の上限
	MaxNumber = 1_000_000
)

// 招待コードの長さ(バイト)。16 バイト(128 ビット)の乱数は総当たりで当てられない。
const inviteCodeBytes = 16

// Yes と No は「はい/いいえ」の質問の答えの値。
const (
	Yes = "yes"
	No  = "no"
)

// CheckQuestions は保存する質問の並びを確かめ、ラベルと選択肢の前後の空白を除いた並びを返す。
// current は今の質問(答えの数つき)。current に無い ID は domain.NotFound ではなく検証エラーにする
// (別の大会の質問 ID を混ぜても、その質問を書き換えられないように)。
func CheckQuestions(questions, current []Question) ([]Question, error) {
	if len(questions) > MaxQuestions {
		return nil, domain.Invalid("questions", fmt.Sprintf("must be at most %d", MaxQuestions))
	}
	byID := make(map[int64]Question, len(current))
	for _, question := range current {
		byID[question.ID] = question
	}
	seen := make(map[int64]bool, len(questions))
	out := make([]Question, len(questions))
	for i, question := range questions {
		field := fmt.Sprintf("questions[%d]", i)
		if question.ID != 0 {
			before, ok := byID[question.ID]
			if !ok || seen[question.ID] {
				return nil, domain.Invalid(field+".id", "is not a question of this tournament")
			}
			seen[question.ID] = true
			// 答えのある質問の答え方や相手を変えると、集めた答えが別の意味になる
			if before.Answered > 0 && (before.Kind != question.Kind || before.Scope != question.Scope) {
				return nil, fmt.Errorf("%w: %s already has answers; its kind and scope cannot change",
					domain.ErrConflict, field)
			}
		}
		checked, err := checkQuestion(field, question)
		if err != nil {
			return nil, err
		}
		out[i] = checked
	}
	return out, nil
}

func checkQuestion(field string, question Question) (Question, error) {
	out := question
	out.Label = strings.TrimSpace(question.Label)
	if err := checkLine(field+".label", out.Label, MaxShortText*2, true); err != nil {
		return Question{}, err
	}
	switch question.Scope {
	case ScopeTeam, ScopeMember:
	default:
		return Question{}, domain.Invalid(field+".scope", "must be team or member")
	}
	switch question.Kind {
	case KindShortText, KindLongText, KindNumber, KindYesNo:
		if len(question.Options) > 0 {
			return Question{}, domain.Invalid(field+".options", "must be empty unless kind is choice")
		}
		out.Options = []string{}
	case KindChoice:
		options, err := checkOptions(field+".options", question.Options)
		if err != nil {
			return Question{}, err
		}
		out.Options = options
	default:
		return Question{}, domain.Invalid(field+".kind",
			"must be short_text, long_text, choice, number or yes_no")
	}
	return out, nil
}

func checkOptions(field string, options []string) ([]string, error) {
	if len(options) == 0 || len(options) > MaxOptions {
		return nil, domain.Invalid(field, fmt.Sprintf("must have 1 to %d options", MaxOptions))
	}
	out := make([]string, len(options))
	seen := make(map[string]bool, len(options))
	for i, option := range options {
		value := strings.TrimSpace(option)
		if err := checkLine(fmt.Sprintf("%s[%d]", field, i), value, MaxShortText, true); err != nil {
			return nil, err
		}
		if seen[value] {
			return nil, domain.Invalid(fmt.Sprintf("%s[%d]", field, i), "is duplicated")
		}
		seen[value] = true
		out[i] = value
	}
	return out, nil
}

// CheckDraft は代表エントリーの中身を確かめ、前後の空白を除いた中身と、整えた答えを返す。
// チーム名は前後の空白だけを除き、それ以外は加工しない(scoreview-design 8)。背番号は空でもよいが、
// 同じチームで重ねない。答えは CheckAnswers でチーム向けの質問に照らす。
func CheckDraft(draft Draft, tournament Tournament, questions []Question) (Draft, []Answer, error) {
	out := draft
	if !hasDivision(tournament, draft.DivisionID) {
		return Draft{}, nil, domain.Invalid("divisionId", "is not a division of this tournament")
	}
	texts := []struct {
		field    string
		value    *string
		required bool
	}{
		{"fullName", &out.FullName, true},
		{"shortName", &out.ShortName, true},
		{"repName", &out.RepName, true},
		{"repPhone", &out.RepPhone, false},
		{"repEmail", &out.RepEmail, false},
	}
	for _, text := range texts {
		*text.value = strings.TrimSpace(*text.value)
		if err := checkLine(text.field, *text.value, MaxShortText, text.required); err != nil {
			return Draft{}, nil, err
		}
	}
	members, err := checkMembers(draft.Members)
	if err != nil {
		return Draft{}, nil, err
	}
	out.Members = members
	answers, err := CheckAnswers(questions, ScopeTeam, draft.Answers)
	if err != nil {
		return Draft{}, nil, err
	}
	return out, answers, nil
}

func hasDivision(tournament Tournament, divisionID int64) bool {
	for _, division := range tournament.Divisions {
		if division.ID == divisionID {
			return true
		}
	}
	return false
}

func checkMembers(members []MemberDraft) ([]MemberDraft, error) {
	if len(members) > MaxMembers {
		return nil, domain.Invalid("members", fmt.Sprintf("must be at most %d", MaxMembers))
	}
	out := make([]MemberDraft, len(members))
	numbers := make(map[string]bool, len(members))
	for i, member := range members {
		field := fmt.Sprintf("members[%d]", i)
		name, number := strings.TrimSpace(member.Name), strings.TrimSpace(member.Number)
		if err := checkLine(field+".name", name, MaxShortText, true); err != nil {
			return nil, err
		}
		if err := checkLine(field+".number", number, MaxNumberText, false); err != nil {
			return nil, err
		}
		if number != "" {
			if numbers[number] {
				return nil, domain.Invalid(field+".number", "is used by another member")
			}
			numbers[number] = true
		}
		out[i] = MemberDraft{Name: name, Number: number}
	}
	return out, nil
}

// CheckAnswers は scope の質問への答えを確かめ、質問の並び順の答えを返す。必須の質問に答えが無ければ
// 検証エラー。答えの無い任意の質問は返さない。scope の違う質問や、知らない質問への答えも検証エラーに
// する(画面の作り違いを黙って捨てないため)。
func CheckAnswers(questions []Question, scope Scope, answers map[int64]string) ([]Answer, error) {
	known := make(map[int64]bool, len(questions))
	out := []Answer{}
	for _, question := range questions {
		if question.Scope != scope {
			continue
		}
		known[question.ID] = true
		field := fmt.Sprintf("answers[%d]", question.ID)
		value := strings.TrimSpace(answers[question.ID])
		if value == "" {
			if question.Required {
				return nil, domain.Invalid(field, "is required")
			}
			continue
		}
		if err := checkAnswer(field, question, value); err != nil {
			return nil, err
		}
		out = append(out, Answer{QuestionID: question.ID, Value: value})
	}
	for id := range answers {
		if !known[id] {
			return nil, domain.Invalid(fmt.Sprintf("answers[%d]", id), "is not a question for this form")
		}
	}
	return out, nil
}

func checkAnswer(field string, question Question, value string) error {
	switch question.Kind {
	case KindShortText:
		return checkLine(field, value, MaxShortText*2, true)
	case KindLongText:
		return checkText(field, value, MaxLongText, true)
	case KindNumber:
		number, err := strconv.Atoi(value)
		if err != nil {
			return domain.Invalid(field, "must be a whole number")
		}
		return domain.CheckRange(field, number, -MaxNumber, MaxNumber)
	case KindYesNo:
		if value != Yes && value != No {
			return domain.Invalid(field, "must be yes or no")
		}
		return nil
	case KindChoice:
		for _, option := range question.Options {
			if option == value {
				return nil
			}
		}
		return domain.Invalid(field, "must be one of the options")
	}
	return domain.Invalid(field, "has an unknown kind")
}

// checkText は文字数(バイトではなく文字)と必須を確かめる。
func checkText(field, value string, maxRunes int, required bool) error {
	if required && value == "" {
		return domain.Invalid(field, "is required")
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return domain.Invalid(field, fmt.Sprintf("must be at most %d characters", maxRunes))
	}
	return nil
}

// checkLine は 1 行の文(名前、チーム名、短い答え)を確かめる。checkText に加えて、改行と制御文字を
// 拒む。チーム名は公式アカウントの返信の文に入るので、改行で運営の名義の文や URL を足させないため。
func checkLine(field, value string, maxRunes int, required bool) error {
	if err := checkText(field, value, maxRunes, required); err != nil {
		return err
	}
	if strings.ContainsFunc(value, unicode.IsControl) {
		return domain.Invalid(field, "must not contain line breaks or control characters")
	}
	return nil
}

// CanSubmit は申込を受けてよいかを確かめる。LINE からの申込は大会が受付中のときだけ受ける。運営の
// 代理入力は、締め切った後の電話の申込や遅れて届いた紙を入れるため、終わった大会以外なら受ける。
func CanSubmit(status domain.TournamentStatus, channel Channel) error {
	switch {
	case status == domain.TournamentFinished:
		return fmt.Errorf("%w: the tournament has finished", domain.ErrConflict)
	case channel == ChannelLine && status != domain.TournamentEntryOpen:
		return fmt.Errorf("%w: the tournament is not accepting entries", domain.ErrConflict)
	case channel != ChannelLine && channel != ChannelDesk:
		return domain.Invalid("channel", "must be line or desk")
	}
	return nil
}

// Next は申込の状態 from に action をしたあとの状態を返す。できない操作は domain.ErrConflict。
//
// 承認と差し戻しは運営の確認を待っている申込(申込中)だけにできる。取消は承認する前ならいつでも
// できる。承認した申込を取り消さないのは、チームがもう組分けと試合に入っているかもしれないからで、
// そのときは部門の画面でチームを棄権にする(既存の棄権の扱い。docs/line-integration.md)。
func Next(from Status, action Action) (Status, error) {
	switch action {
	case ActionApprove:
		if from == StatusPending {
			return StatusApproved, nil
		}
	case ActionReturn:
		if from == StatusPending {
			return StatusReturned, nil
		}
	case ActionCancel:
		if from == StatusPending || from == StatusReturned {
			return StatusCancelled, nil
		}
		if from == StatusApproved {
			return "", fmt.Errorf("%w: an approved entry cannot be cancelled; withdraw the team instead",
				domain.ErrConflict)
		}
	default:
		return "", domain.Invalid("action", "must be approve, return or cancel")
	}
	return "", fmt.Errorf("%w: cannot %s an entry that is %s", domain.ErrConflict, action, from)
}

// CanReturn は申込を差し戻せるかを確かめる。差し戻した申込を直して送り直せるのは、LINE で申し込んだ
// 代表だけなので、代表の LINE が無い申込(運営が代わりに入れた申込や、代表の LINE を消した申込)は
// 差し戻せない(差し戻すと、誰も直せず、取り消すしか無くなる)。そのときは取り消して入れ直す。
func CanReturn(action Action, current Detail) error {
	if action == ActionReturn && current.RepLineUserID == nil {
		return fmt.Errorf("%w: only an entry from a LINE representative can be returned; cancel it and enter it again",
			domain.ErrConflict)
	}
	return nil
}

// CheckNote は差し戻しの理由を確かめる。差し戻しには代表が何を直せばよいかが要るので必須にし、
// それ以外の操作には付けさせない。
func CheckNote(action Action, note string) (string, error) {
	trimmed := strings.TrimSpace(note)
	if action != ActionReturn {
		if trimmed != "" {
			return "", domain.Invalid("note", "is only for returning an entry")
		}
		return "", nil
	}
	if err := checkText("note", trimmed, MaxLongText, true); err != nil {
		return "", err
	}
	return trimmed, nil
}

// ApprovalOf は承認で作るチームと選手を申込から決める。チームの正式名と短縮名は申込のまま、選手は
// メンバーの並び順で、名前と背番号を写す。
func ApprovalOf(detail Detail) Approval {
	players := make([]ApprovedPlayer, len(detail.MemberList))
	for i, member := range detail.MemberList {
		players[i] = ApprovedPlayer{MemberID: member.ID, Name: member.Name, Number: member.Number}
	}
	return Approval{DivisionID: detail.Division.ID, FullName: detail.FullName,
		ShortName: detail.ShortName, Players: players}
}

// NewInviteCode は random から招待コードを作る。英小文字と数字の 26 文字で、URL にそのまま入れられる。
func NewInviteCode(random io.Reader) (string, error) {
	buf := make([]byte, inviteCodeBytes)
	if _, err := io.ReadFull(random, buf); err != nil {
		return "", fmt.Errorf("read random bytes for invite code: %w", err)
	}
	return strings.ToLower(inviteEncoding().EncodeToString(buf)), nil
}

// IsInviteCode は code が NewInviteCode の作る形(base32 の英小文字と 2〜7 の数字を 26 文字)かどうか。
// 形の違うものは DB に聞かずに「見つからない」にできる。
func IsInviteCode(code string) bool {
	if len(code) != inviteEncoding().EncodedLen(inviteCodeBytes) {
		return false
	}
	for _, char := range code {
		if (char < 'a' || char > 'z') && (char < '2' || char > '7') {
			return false
		}
	}
	return true
}

func inviteEncoding() *base32.Encoding {
	return base32.StdEncoding.WithPadding(base32.NoPadding)
}
