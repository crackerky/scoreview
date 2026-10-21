package domain

// Result は成績の集計に使う試合結果。チームが未確定の側は TeamID が 0。
type Result struct {
	LeftTeamID  int64
	RightTeamID int64
	LeftScore   *int
	RightScore  *int
	Status      MatchStatus
	// 同点でも勝者が決まる試合(トーナメント)のための明示的な勝者。不戦勝と不成立では勝者
	WinnerSide *Side
	// 結果の種類。空は通常
	ResultType ResultType
}

// Tally は成績表の 1 行(試合数、勝ち、分け、負け、得点、失点)。
type Tally struct {
	Games        int
	Wins         int
	Draws        int
	Losses       int
	GoalsFor     int
	GoalsAgainst int
}

// Points は勝ち点(勝ち 3、分け 1、負け 0)。clone-spec C 2.1(b) の非表示列の値と一致する。
func (t Tally) Points() int { return 3*t.Wins + t.Draws }

// GoalDiff は得失差。
func (t Tally) GoalDiff() int { return t.GoalsFor - t.GoalsAgainst }

// counted は集計に数える試合かどうか。終了した試合のうち、通常、不戦勝、途中打ち切りは両方のスコアが
// ある試合を、両チーム不戦敗はスコアなしで、不成立は勝者がある試合を数える。未消化は数えない。
// スコアのない通常の終了試合は不整合なデータで、0-0 の引き分けとして数えると誤った成績が正しい顔をして
// 出るため数えない。
func (r Result) counted() bool {
	if r.Status != MatchFinished {
		return false
	}
	switch r.ResultType {
	case ResultUnplayed:
		return false
	case ResultDoubleForfeit:
		return true
	case ResultNoContest:
		return r.WinnerSide != nil
	case ResultForfeit:
		// 規定スコアが無い大会の不戦勝はスコアが無く、勝者だけで数える
		return r.WinnerSide != nil || (r.LeftScore != nil && r.RightScore != nil)
	}
	return r.LeftScore != nil && r.RightScore != nil
}

// sideOf は teamID がこの試合のどちら側かを返す。出ていなければ ok=false。
func (r Result) sideOf(teamID int64) (Side, bool) {
	switch {
	case teamID == 0:
		return "", false
	case r.LeftTeamID == teamID:
		return SideLeft, true
	case r.RightTeamID == teamID:
		return SideRight, true
	default:
		return "", false
	}
}

// winner は勝った側を返す。引き分けなら nil。WinnerSide があればスコアより優先する
// (clone-spec C 2.3: 4-4 でも勝者が付く試合がある)。勝者つきの同点試合を成績表で勝ちに数えるかは
// 参照サイトで未観測で、勝ちとして数えるのは推測。
func (r Result) winner() *Side {
	if r.ResultType == ResultDoubleForfeit {
		return nil
	}
	if r.WinnerSide != nil {
		return r.WinnerSide
	}
	var side Side
	switch {
	case *r.LeftScore > *r.RightScore:
		side = SideLeft
	case *r.LeftScore < *r.RightScore:
		side = SideRight
	default:
		return nil
	}
	return &side
}

// add は 1 試合ぶんを teamID の側から見て加える。数えない試合なら何もしない。
func (t *Tally) add(teamID int64, r Result) {
	if !r.counted() {
		return
	}
	side, ok := r.sideOf(teamID)
	if !ok {
		return
	}
	switch r.ResultType {
	case ResultDoubleForfeit:
		// 両チームとも負け。スコアは記録しない(docs/format-engine.md「試合の結果の種類」)
		t.Games++
		t.Losses++
		return
	case ResultNoContest, ResultForfeit:
		if r.ResultType == ResultForfeit && r.LeftScore != nil && r.RightScore != nil {
			break
		}
		// 勝敗だけを数え、得点と失点には入れない(不成立は JFDA 規程。スコアの無い不戦勝は、
		// 本部がスコアを決めるまで得失点に数えない)
		t.Games++
		if *r.WinnerSide == side {
			t.Wins++
		} else {
			t.Losses++
		}
		return
	}
	own, opp := *r.LeftScore, *r.RightScore
	if side == SideRight {
		own, opp = opp, own
	}
	t.Games++
	t.GoalsFor += own
	t.GoalsAgainst += opp
	switch winner := r.winner(); {
	case winner == nil:
		t.Draws++
	case *winner == side:
		t.Wins++
	default:
		t.Losses++
	}
}

// TallyFor は teamID が出た試合のうち、集計に数える試合(counted)を集計する。
// 引き分けは Draws に数え、勝ちには数えない(clone-spec C 1.3)。
func TallyFor(teamID int64, results []Result) Tally {
	var t Tally
	for _, r := range results {
		t.add(teamID, r)
	}
	return t
}
