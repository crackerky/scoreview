package format

import "scoreview/api/internal/domain"

// ResultType は試合の結果の種類(docs/format-engine.md「試合の結果の種類」)。
type ResultType string

// ResultType の値。空は ResultNormal と同じ。
const (
	// ResultNormal はスコアで勝敗を決める。同点はプールでは引き分け、トーナメントでは Winner が要る
	ResultNormal ResultType = "normal"
	// ResultForfeit は没収(不戦勝)。Winner が要り、敗者の棄権試合数に数える。得点と失点は記録した
	// スコアがあればそれ、無ければ規定スコア(Config.Forfeit)、どちらも無ければ数えない(勝敗だけ)
	ResultForfeit ResultType = "forfeit"
	// ResultVoid は不成立(人数不足など)。スコアを記録せず、Winner の勝敗だけを数える(JFDA 規程。R2 5.3)
	ResultVoid ResultType = "void"
	// ResultDoubleForfeit は両チーム不戦敗。両チームの負け(と棄権試合)に数え、スコアは記録しない。
	// 勝者がいないので、トーナメントの勝者・敗者を出どころにする枠は埋まらない
	ResultDoubleForfeit ResultType = "double_forfeit"
	// ResultAbandoned は途中打ち切り(雷や暑さで中断し再開しなかった)をその時点のスコアで確定したもの。
	// 数え方は通常と同じで、表示のために区別する。未消化にするなら ResultUnplayed を使う
	ResultAbandoned ResultType = "abandoned"
	// ResultUnplayed は未消化(時間切れなど)。集計に数えず、勝者・敗者を出どころにする枠は埋まらない
	ResultUnplayed ResultType = "unplayed"
)

// Result は 1 試合の結果。Status が finished でない試合はまだ結果が無いものとして扱う
// (ResultUnplayed を除く)。
type Result struct {
	Status     domain.MatchStatus
	Type       ResultType
	LeftScore  *int
	RightScore *int
	// Winner はスコアより優先する勝者。同点のトーナメントの試合(サドンデス等。clone-spec C 2.3 の 4-4)、
	// 没収、不成立で使う
	Winner *domain.Side
}

// ForfeitScore は没収試合の規定スコア。
type ForfeitScore struct {
	Winner int
	Loser  int
}

// DefaultForfeitScore は没収試合の規定スコアの初期値 15-0。docs/format-engine.md の例で、
// 運営会社に未確認(大会の設定で変える)。
func DefaultForfeitScore() ForfeitScore { return ForfeitScore{Winner: 15, Loser: 0} }

// 勝ち点(clone-spec C 2.1(b) の非表示列。3×勝+1×分 で 154 行一致)。
const (
	pointsWin  = 3
	pointsDraw = 1
)

// resultState は結果から読める試合の状態。
type resultState int

const (
	resultPending    resultState = iota // 未入力、試合前、試合中
	resultIncomplete                    // 終了しているのにスコアか勝者が足りない
	resultUnplayed                      // 未消化
	resultDraw                          // 引き分け(プールのみ有効)
	resultNoWinner                      // 両チーム不戦敗
	resultDecided                       // 勝者がいる
)

// counted は成績に数える状態か。
func (s resultState) counted() bool {
	return s == resultDecided || s == resultDraw || s == resultNoWinner
}

// decide は結果の状態と勝者を返す。
func (r *Result) decide() (resultState, domain.Side) {
	if r == nil {
		return resultPending, ""
	}
	if r.Type == ResultUnplayed {
		return resultUnplayed, ""
	}
	if r.Status != domain.MatchFinished {
		return resultPending, ""
	}
	switch r.Type {
	case ResultDoubleForfeit:
		return resultNoWinner, ""
	case ResultForfeit, ResultVoid:
		if r.Winner == nil {
			return resultIncomplete, ""
		}
		return resultDecided, *r.Winner
	}
	if r.LeftScore == nil || r.RightScore == nil {
		// スコアのない終了試合を 0-0 として数えると誤った順位が正しい顔をして出るので数えない
		return resultIncomplete, ""
	}
	if r.Winner != nil {
		return resultDecided, *r.Winner
	}
	switch {
	case *r.LeftScore > *r.RightScore:
		return resultDecided, domain.SideLeft
	case *r.LeftScore < *r.RightScore:
		return resultDecided, domain.SideRight
	}
	return resultDraw, ""
}

// Game はプールの順位計算に使う 1 試合。Left と Right はチーム ID。
type Game struct {
	Code   string
	Left   int64
	Right  int64
	Result *Result
}

// Stats はチームの成績。
type Stats struct {
	Games        int
	Wins         int
	Draws        int
	Losses       int
	GoalsFor     int
	GoalsAgainst int
	// Forfeits は棄権(没収で負け)した試合数
	Forfeits int
}

// Points は勝ち点(勝ち 3、分け 1、負け 0)。
func (s Stats) Points() int { return pointsWin*s.Wins + pointsDraw*s.Draws }

// GoalDiff は得失点差。
func (s Stats) GoalDiff() int { return s.GoalsFor - s.GoalsAgainst }

// add は game を team の側から見て加える。数えない試合なら何もしない。forfeit は不戦勝の規定スコア
// (nil は規定なし)で、スコアを記録していない不戦勝だけに使う。
func (s *Stats) add(team int64, game Game, forfeit *ForfeitScore) {
	var side domain.Side
	switch team {
	case game.Left:
		side = domain.SideLeft
	case game.Right:
		side = domain.SideRight
	default:
		return
	}
	state, winner := game.Result.decide()
	if !state.counted() {
		return
	}
	s.Games++
	if state == resultNoWinner {
		s.Losses++
		s.Forfeits++
		return
	}
	switch {
	case state == resultDraw:
		s.Draws++
	case winner == side:
		s.Wins++
	default:
		s.Losses++
	}
	switch game.Result.Type {
	case ResultVoid:
		// 不成立はスコアを数えない
	case ResultForfeit:
		if winner != side {
			s.Forfeits++
		}
		// 記録したスコア(閲覧ページの成績表と同じ値)を優先し、順位と表示を食い違わせない
		if game.Result.LeftScore != nil && game.Result.RightScore != nil {
			s.addScore(side, *game.Result.LeftScore, *game.Result.RightScore)
			return
		}
		if forfeit == nil {
			return // 規定スコアが無ければ得失点に数えない(勝敗だけ)
		}
		if winner == side {
			s.GoalsFor += forfeit.Winner
			s.GoalsAgainst += forfeit.Loser
		} else {
			s.GoalsFor += forfeit.Loser
			s.GoalsAgainst += forfeit.Winner
		}
	default:
		s.addScore(side, *game.Result.LeftScore, *game.Result.RightScore)
	}
}

func (s *Stats) addScore(side domain.Side, left, right int) {
	own, opp := left, right
	if side == domain.SideRight {
		own, opp = opp, own
	}
	s.GoalsFor += own
	s.GoalsAgainst += opp
}

// statsFor は team の games 全体の成績。
func statsFor(team int64, games []Game, forfeit *ForfeitScore) Stats {
	var s Stats
	for _, g := range games {
		s.add(team, g, forfeit)
	}
	return s
}
