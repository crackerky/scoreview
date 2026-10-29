package format

import (
	"fmt"

	"scoreview/api/internal/domain"
)

// MatchStageName は試合を、観戦者と運営者の言葉で何を決める試合かが分かる名前にする(観戦アプリの
// カードと引き出し)。プールの試合はグループの名前(「予選 A 組」)。トーナメントの試合は、順位が決まる
// 試合なら「3位決定戦」、まだ次の試合へ進む試合なら、その先で決まりうる順位の範囲と回戦
// (「5〜8位 準決勝」)。最終順位にならないトーナメント(ブロック)は、グループの名前と回戦。
func MatchStageName(plan *Plan, m Match) string {
	g := plan.Groups[m.Group]
	if g.Kind != domain.GroupBracket {
		return OperatorName(g)
	}
	byCode := map[string]Match{}
	for _, other := range plan.GroupMatches(m.Group) {
		byCode[other.Code] = other
	}
	round := roundName(depthToFinal(m, byCode), m.Round)
	if !g.Final {
		return OperatorName(g) + " " + round
	}
	if m.WinnerPlace > 0 {
		winner := g.RankOffset + m.WinnerPlace
		if m.LoserPlace == m.WinnerPlace {
			return fmt.Sprintf("%d位(同順位)", winner)
		}
		return fmt.Sprintf("%d位決定戦", winner)
	}
	low, high := reachablePlaces(m, byCode)
	if low == 0 {
		return OperatorName(g) + " " + round
	}
	return fmt.Sprintf("%d〜%d位 %s", g.RankOffset+low, g.RankOffset+high, round)
}

// next は試合の勝者(win)か敗者(lose)が進む、同じグループの試合。無ければ偽。
func next(m Match, outcome domain.Outcome, byCode map[string]Match) (Match, bool) {
	for _, other := range byCode {
		for _, side := range []SideRef{other.Left, other.Right} {
			if side.From != nil && side.From.Code == m.Code && side.From.Outcome == outcome {
				return other, true
			}
		}
	}
	return Match{}, false
}

// reachablePlaces は試合の勝者と敗者がこの先で取りうるグループ内の順位の範囲。決まらなければ 0, 0。
func reachablePlaces(m Match, byCode map[string]Match) (int, int) {
	low, high := 0, 0
	add := func(place int) {
		if place == 0 {
			return
		}
		if low == 0 || place < low {
			low = place
		}
		high = max(high, place)
	}
	add(m.WinnerPlace)
	add(m.LoserPlace)
	for _, outcome := range []domain.Outcome{domain.OutcomeWin, domain.OutcomeLose} {
		if following, ok := next(m, outcome, byCode); ok {
			l, h := reachablePlaces(following, byCode)
			add(l)
			add(h)
		}
	}
	return low, high
}

// depthToFinal は勝ち進んだときに残る試合の数(この試合を含む)。決勝が 1、準決勝が 2。
func depthToFinal(m Match, byCode map[string]Match) int {
	following, ok := next(m, domain.OutcomeWin, byCode)
	if !ok {
		return 1
	}
	return 1 + depthToFinal(following, byCode)
}

func roundName(depth, round int) string {
	switch depth {
	case 1:
		return "決勝"
	case 2:
		return "準決勝"
	case 3:
		return "準々決勝"
	default:
		return fmt.Sprintf("%d回戦", round)
	}
}
