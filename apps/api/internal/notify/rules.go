package notify

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

// MaxBody は運営が書く文の文字数。LINE のテキストメッセージは 5000 字までなので、大会名の行を足しても
// 収まる長さにする(差し戻しの理由と同じ長さ)。
const MaxBody = entry.MaxLongText

// maxRequestKey は送る画面が作る乱数(UUID は 36 字)の長さの上限。
const maxRequestKey = 64

// CheckAudience は条件と絞り込みの組み合わせと、部門が大会のものかを確かめる。
func CheckAudience(audience Audience, tournament entry.Tournament) error {
	switch audience.Kind {
	case AudienceReps:
		switch audience.Status {
		case "", entry.StatusPending, entry.StatusReturned, entry.StatusApproved:
		default:
			return domain.Invalid("status", "must be pending, returned or approved")
		}
	case AudienceIncomplete, AudienceParticipants, AudienceNoScorer:
		if audience.Status != "" {
			return domain.Invalid("status", "is only for reps")
		}
	case AudiencePastReps:
		// まだ申し込んでいない人に送るので、この大会の部門や申込の状態では絞れない
		if audience.Status != "" || audience.DivisionID != 0 {
			return domain.Invalid("audience", "past_reps takes no status or division")
		}
	default:
		return domain.Invalid("kind", "must be reps, incomplete, participants, past_reps or no_scorer")
	}
	if audience.DivisionID == 0 {
		return nil
	}
	known := slices.ContainsFunc(tournament.Divisions, func(division entry.Division) bool {
		return division.ID == audience.DivisionID
	})
	if !known {
		return domain.Invalid("divisionId", "is not a division of the tournament")
	}
	return nil
}

// CheckBody は運営が書いた文を確かめ、届ける形(改行は LF、前後の空白なし)にして返す。改行とタブの
// ほかの制御文字は拒む(LINE の表示が崩れ、見本と違う文が届くため)。
func CheckBody(body string) (string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n"))
	if normalized == "" {
		return "", domain.Invalid("body", "is required")
	}
	if utf8.RuneCountInString(normalized) > MaxBody {
		return "", domain.Invalid("body", fmt.Sprintf("must be at most %d characters", MaxBody))
	}
	control := func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' }
	if strings.ContainsFunc(normalized, control) {
		return "", domain.Invalid("body", "must not contain control characters")
	}
	return normalized, nil
}

// CheckRequestKey は送る画面が作った乱数を確かめる。
func CheckRequestKey(key string) error {
	if key == "" || len(key) > maxRequestKey {
		return domain.Invalid("requestKey", fmt.Sprintf("must be 1 to %d characters", maxRequestKey))
	}
	if strings.ContainsFunc(key, func(r rune) bool { return !keyRune(r) }) {
		return domain.Invalid("requestKey", "must be letters, digits and hyphens")
	}
	return nil
}

func keyRune(r rune) bool {
	return r == '-' || ('0' <= r && r <= '9') || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

// Text は届ける文。1 行目に大会名を置く(大会をまたいで申し込んでいる人が、どの大会の話かを
// 取り違えないように)。
func Text(heading, body string) string {
	return heading + "\n\n" + body
}

// MaxScheduleAhead は予約できる先の長さ(大会の案内に要る長さより十分長く、忘れられた予約が残り
// 続けない長さ)。
const MaxScheduleAhead = 120 * 24 * time.Hour

// CheckSendAt は予約の送る時刻を確かめる。今より後で、MaxScheduleAhead より先でないこと。
func CheckSendAt(sendAt, now time.Time) error {
	if !sendAt.After(now) {
		return domain.Invalid("sendAt", "must be in the future")
	}
	if sendAt.After(now.Add(MaxScheduleAhead)) {
		return domain.Invalid("sendAt", "must be within 120 days")
	}
	return nil
}

// CountReachable は送れる人(友だち)と送れない人の数を返す。
func CountReachable(recipients []Recipient) (reachable, unreachable int) {
	for _, recipient := range recipients {
		if recipient.Friend {
			reachable++
		} else {
			unreachable++
		}
	}
	return reachable, unreachable
}

// CheckQuota は今月の残りの配信数で needed 通を送れるかを確かめる。足りなければ domain.ErrConflict。
func CheckQuota(quota Quota, needed int) error {
	if quota.Limit == nil {
		return nil
	}
	left := max(*quota.Limit-quota.Used, 0)
	if int64(needed) > left {
		return fmt.Errorf("%w: this needs %d messages but only %d are left this month", domain.ErrConflict,
			needed, left)
	}
	return nil
}

// MaxButtons は届く文の下に並べるボタンの数の上限(LINE の画面で押し分けやすい数)。
const MaxButtons = 3

// maxButtonLabel はボタンの文字数(LINE のボタンの表示に収まる長さ)。
const maxButtonLabel = 20

// maxButtonURL はボタンの URL の長さ(LINE の URI アクションの上限 1000 字)。
const maxButtonURL = 1000

// defaultButtonLabel はボタンの文字を書かなかったときの文字(運営が書く url のボタンには無い)。
func defaultButtonLabel(kind ButtonKind) string {
	if kind == ButtonEntry {
		return "申込を開く"
	}
	return "スコアサイトを見る"
}

// CheckButtons はボタンの数、種類、文字、URL を確かめ、文字の前後の空白を除いたものを返す。申込と
// スコアサイトのボタンは 1 つずつまで、URL は https だけ。site の行き先は Service が決めるので、ここでは
// 空にする。
func CheckButtons(buttons []Button) ([]Button, error) {
	if len(buttons) > MaxButtons {
		return nil, domain.Invalid("buttons", fmt.Sprintf("must be at most %d", MaxButtons))
	}
	out := make([]Button, 0, len(buttons))
	seen := map[ButtonKind]bool{}
	for i, button := range buttons {
		field := fmt.Sprintf("buttons[%d]", i)
		label := strings.TrimSpace(button.Label)
		switch button.Kind {
		case ButtonEntry, ButtonSite:
			if seen[button.Kind] {
				return nil, domain.Invalid(field, "the entry and site buttons can be used once each")
			}
			seen[button.Kind] = true
			if label == "" {
				label = defaultButtonLabel(button.Kind)
			}
			button.URL = ""
		case ButtonURL:
			if err := checkButtonURL(field, button.URL); err != nil {
				return nil, err
			}
			button.URL = strings.TrimSpace(button.URL)
			if label == "" {
				return nil, domain.Invalid(field+".label", "is required")
			}
		default:
			return nil, domain.Invalid(field+".kind", "must be entry, site or url")
		}
		if utf8.RuneCountInString(label) > maxButtonLabel || strings.ContainsFunc(label, unicode.IsControl) {
			return nil, domain.Invalid(field+".label", fmt.Sprintf("must be one line of at most %d characters", maxButtonLabel))
		}
		button.Label = label
		out = append(out, button)
	}
	return out, nil
}

func checkButtonURL(field, raw string) error {
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || len(trimmed) > maxButtonURL {
		return domain.Invalid(field+".url", fmt.Sprintf("must be an https URL of at most %d characters", maxButtonURL))
	}
	return nil
}
