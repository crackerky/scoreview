package domain

import "fmt"

// 試合の結果の種類(docs/format-engine.md「試合の結果の種類」)。通常の試合以外の結果を、
// 得点の入力と同じ画面で最初から選べるようにする(不戦勝の選択肢が無い速報アプリが米国で
// 失格騒動の一因になった。docs/operations/research/S2-global-voices.md)。

// ResultType は試合の結果の種類。
type ResultType string

// ResultType の値。
const (
	// ResultNormal はスコアで勝敗が決まる通常の試合。
	ResultNormal ResultType = "normal"
	// ResultForfeit は不戦勝。大会の規定スコアで記録し、勝者を WinnerSide に持つ。
	ResultForfeit ResultType = "forfeit"
	// ResultDoubleForfeit は両チーム不戦敗。スコアを記録せず、両チームとも負けに数える。
	ResultDoubleForfeit ResultType = "double_forfeit"
	// ResultNoContest は不成立(人数不足など)。スコアを記録せず、勝敗だけを記録する(JFDA 規程)。
	ResultNoContest ResultType = "no_contest"
	// ResultAbandoned は途中打ち切り(雷や暑さで中断し再開しなかった)を、その時点のスコアで確定したもの。
	ResultAbandoned ResultType = "abandoned"
	// ResultUnplayed は未消化(時間切れなどで行わなかった、または途中打ち切りを未消化にした)。集計に入れない。
	ResultUnplayed ResultType = "unplayed"
)

// Valid は既知の結果の種類かどうか。
func (t ResultType) Valid() bool {
	switch t {
	case ResultNormal, ResultForfeit, ResultDoubleForfeit, ResultNoContest, ResultAbandoned,
		ResultUnplayed:
		return true
	}
	return false
}

// ForfeitScore は不戦勝の規定スコア(大会の設定)。勝者と敗者の得点。
type ForfeitScore struct {
	Winner int
	Loser  int
}

// Validate は規定スコアが「敗者 0 以上、勝者が敗者より多く MaxScore 以下」であることを確かめる。
func (s ForfeitScore) Validate(field string) error {
	if s.Loser < 0 || s.Winner <= s.Loser || s.Winner > MaxScore {
		return Invalid(field, fmt.Sprintf(
			"winner must be greater than loser and at most %d, and loser must be 0 or more", MaxScore))
	}
	return nil
}

// ExceptionalResult は通常以外の結果で試合を終えるときの入力。
type ExceptionalResult struct {
	Type ResultType
	// WinnerSide は不戦勝と不成立で必須。途中打ち切りでは同点のトーナメントの試合だけ必須。
	WinnerSide *Side
	// Forfeit は不戦勝のスコア。大会の設定か、未設定なら入力された値。
	Forfeit *ForfeitScore
	// Current は途中打ち切りで確定するときのその時点のスコア。
	Current Score
}

// ResultOutcome は試合を終えるときに保存するスコアと勝者。Score が nil ならスコアを記録しない。
type ResultOutcome struct {
	Score  *Score
	Winner *Side
}

// DecideResult は通常以外の結果で試合を終えるときに、保存するスコアと勝者を決める。
// 規則に合わない入力(勝者の無い不戦勝、規定スコアの無い不戦勝、勝者を付けた未消化など)は検証エラー。
func DecideResult(in ExceptionalResult, inBracket bool) (ResultOutcome, error) {
	needsWinner := func() error {
		if in.WinnerSide == nil {
			return Invalid("winnerSide", fmt.Sprintf("is required for %s", in.Type))
		}
		return nil
	}
	noWinner := func() error {
		if in.WinnerSide != nil {
			return Invalid("winnerSide", fmt.Sprintf("must be null for %s", in.Type))
		}
		return nil
	}
	switch in.Type {
	case ResultForfeit:
		if err := needsWinner(); err != nil {
			return ResultOutcome{}, err
		}
		if in.Forfeit == nil {
			return ResultOutcome{}, Invalid(
				"forfeitScore",
				"is required: the tournament has no forfeit score",
			)
		}
		if err := in.Forfeit.Validate("forfeitScore"); err != nil {
			return ResultOutcome{}, err
		}
		score := Score{Left: in.Forfeit.Winner, Right: in.Forfeit.Loser}
		if *in.WinnerSide == SideRight {
			score = Score{Left: in.Forfeit.Loser, Right: in.Forfeit.Winner}
		}
		winner := *in.WinnerSide
		return ResultOutcome{Score: &score, Winner: &winner}, nil
	case ResultNoContest:
		if err := needsWinner(); err != nil {
			return ResultOutcome{}, err
		}
		winner := *in.WinnerSide
		return ResultOutcome{Winner: &winner}, nil
	case ResultDoubleForfeit, ResultUnplayed:
		if err := noWinner(); err != nil {
			return ResultOutcome{}, err
		}
		return ResultOutcome{}, nil
	case ResultAbandoned:
		winner, err := DecideWinner(in.Current, inBracket, in.WinnerSide)
		if err != nil {
			return ResultOutcome{}, err
		}
		score := in.Current
		return ResultOutcome{Score: &score, Winner: winner}, nil
	case ResultNormal:
		return ResultOutcome{}, Invalid("resultType", "use events or status to finish a normal match")
	}
	return ResultOutcome{}, Invalid("resultType", fmt.Sprintf("unknown result type %q", in.Type))
}
