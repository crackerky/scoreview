package domain

import (
	"fmt"
	"time"
)

// 試合の状態とスコアを変える操作(管理画面のスコア入力)の規則。
// 保存は store の試合更新関数(store/matchupdate.go)だけが行い、ここは判定だけを持つ。

// TransitionError は許されない状態遷移を表す。errors.Is(err, ErrConflict) が真になる。
type TransitionError struct {
	From MatchStatus
	To   MatchStatus
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("cannot change match status from %s to %s", e.From, e.To)
}

func (e *TransitionError) Unwrap() error { return ErrConflict }

// statusRank は状態の進む向き。scheduled → live → finished の順に進む。
func statusRank(s MatchStatus) int {
	switch s {
	case MatchScheduled:
		return 0
	case MatchLive:
		return 1
	case MatchFinished:
		return 2
	}
	return -1
}

// ValidTransition は from から to へ変えてよいかを判定する。
//
// 前へ進む遷移(scheduled → live → finished。scheduled → finished の飛び越しも含む。結果だけを
// 登録する詳細なしの試合のため)は誰でもよい。finished → live の戻し(入力の誤りの訂正)は admin だけで、
// staff なら ErrForbidden。それ以外の戻し(→ scheduled)と同じ状態への変更は TransitionError。
func ValidTransition(from, to MatchStatus, role Role) error {
	fromRank, toRank := statusRank(from), statusRank(to)
	if fromRank < 0 || toRank < 0 {
		return Invalid("status", fmt.Sprintf("unknown status %q", to))
	}
	switch {
	case toRank > fromRank:
		return nil
	case from == MatchFinished && to == MatchLive:
		if !role.RunsDayOf() {
			return fmt.Errorf("%w: only admin or hq can reopen a finished match", ErrForbidden)
		}
		return nil
	default:
		return &TransitionError{From: from, To: to}
	}
}

// DecideWinner は試合を終了にするとき matches.winner_side に保存する値を決める。
//
// winner_side は「同点でも勝者が決まる試合」のための明示的な勝者で(00001_init.sql)、
// スコアに差があれば保存しない(nil を返す。勝者はスコアから決まる)。requested がスコアと
// 矛盾すれば検証エラー。同点で requested があればそれを返し、無ければ、トーナメント
// (bracket グループ)の試合では勝者を決めないと後続試合が埋まらないので検証エラー、
// それ以外は引き分け(nil)。
func DecideWinner(score Score, inBracket bool, requested *Side) (*Side, error) {
	switch {
	case score.Left != score.Right:
		leader := SideLeft
		if score.Right > score.Left {
			leader = SideRight
		}
		if requested != nil && *requested != leader {
			return nil, Invalid("winnerSide", "contradicts the score")
		}
		return nil, nil
	case requested != nil:
		side := *requested
		return &side, nil
	case inBracket:
		return nil, Invalid("winnerSide", "required for a tied match in a bracket group")
	default:
		return nil, nil
	}
}

// Winner は終了してスコアのある試合の勝った側を返す。引き分けか、数えない試合(終了していない、
// スコアが無い)なら nil。WinnerSide があればスコアより優先する(成績表の集計と同じ規則)。
func (r Result) Winner() *Side {
	if !r.counted() {
		return nil
	}
	return r.winner()
}

// FinalScore は得点経過の最後の累計スコア。イベントが無ければ 0-0。
func FinalScore(events []Event) Score {
	scores := RunningScores(events)
	if len(scores) == 0 {
		return Score{}
	}
	return scores[len(scores)-1]
}

// NewEvent は管理画面から追加する得点経過の 1 行。選手が side のチームに属するかは store が確かめる。
type NewEvent struct {
	Kind     EventKind
	Side     *Side
	ScorerID *int64
	AssistID *int64
	Callahan bool
}

// Validate はイベントの種類ごとの項目の組み合わせを検査する(match_events の CHECK と同じ規則を、
// DB に送る前に利用者向けの文面で返すため)。
func (e NewEvent) Validate() error {
	switch e.Kind {
	case EventGoal, EventTimeout, EventPull:
		if e.Side == nil || (*e.Side != SideLeft && *e.Side != SideRight) {
			return Invalid("side", "must be left or right")
		}
	case EventEnd:
		if e.Side != nil {
			return Invalid("side", "must be null for end")
		}
	default:
		return Invalid("kind", "must be goal, timeout, pull or end")
	}
	if e.Kind != EventGoal {
		if e.ScorerID != nil || e.AssistID != nil || e.Callahan {
			return Invalid("kind", "only goal can have scorer, assist or callahan")
		}
		return nil
	}
	if e.ScorerID != nil && e.AssistID != nil && *e.ScorerID == *e.AssistID {
		return Invalid("assistId", "must differ from scorerId")
	}
	// キャラハンゴールはディフェンスがエンドゾーン内でパスを取った得点で、アシストが無い
	if e.Callahan && e.AssistID != nil {
		return Invalid("assistId", "must be null for a callahan goal")
	}
	return nil
}

// StatusAfterEvent はイベントを追加した後の状態。未試合の試合は最初のイベントで試合中になり、
// end で終了になる。終了した試合にはイベントを足せない(先に admin が試合中へ戻す)。
func StatusAfterEvent(current MatchStatus, kind EventKind) (MatchStatus, error) {
	if current == MatchFinished {
		return "", fmt.Errorf("%w: match is finished; reopen it before adding events", ErrConflict)
	}
	if kind == EventEnd {
		return MatchFinished, nil
	}
	return MatchLive, nil
}

// StatusAfterEventRemoval はイベントを取り消した後の状態。終了した試合で取り消せるのは end だけで、
// 他に end が残らなければ試合中へ戻す(finished → live なので ValidTransition で admin に限られる)。
func StatusAfterEventRemoval(
	current MatchStatus,
	removed EventKind,
	endRemains bool,
) (MatchStatus, error) {
	if current != MatchFinished {
		return current, nil
	}
	if removed != EventEnd {
		return "", fmt.Errorf(
			"%w: match is finished; reopen it before removing events",
			ErrConflict,
		)
	}
	if endRemains {
		return current, nil
	}
	return MatchLive, nil
}

// OutcomeTeam は終了した試合の勝者(win)または敗者(lose)のチーム ID を返す。試合が終わっていない、
// 引き分けで勝者が無い、その側のチームが未確定なら nil(後続試合の未確定チームを埋められない)。
func OutcomeTeam(r Result, outcome Outcome) *int64 {
	winner := r.Winner()
	if winner == nil {
		return nil
	}
	side := *winner
	if outcome == OutcomeLose {
		side = side.Opposite()
	}
	id := r.LeftTeamID
	if side == SideRight {
		id = r.RightTeamID
	}
	if id == 0 {
		return nil
	}
	return &id
}

// ElapsedSince は start からの経過秒。時計のずれで負になったら 0 にする。
func ElapsedSince(start, now time.Time) int {
	if !now.After(start) {
		return 0
	}
	return int(now.Sub(start) / time.Second)
}
