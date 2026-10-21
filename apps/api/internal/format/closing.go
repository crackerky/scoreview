package format

import "slices"

// 大会の締め(時間切れで残りの試合を未消化にする)で、決まらなかった順位を同順位として提案するための
// 計算。未消化で終わったトーナメントの試合に残ったチームは、その試合から先で取りえた順位のうち
// 最も上の順位で並ぶ(準決勝が未消化なら 4 チームが「1〜4 位は 1 位タイ」)。

// ReachablePlaces は試合 code に出るチームが、その試合と、その勝者・敗者が進む先の試合で取りえる
// グループ内の順位(1 から、昇順)。順位が決まらない試合(ブロックの途中など)だけなら空。
func (p *Plan) ReachablePlaces(code string) []int {
	m, ok := p.MatchByCode(code)
	if !ok {
		return nil
	}
	feeds := map[string][]string{}
	for _, other := range p.GroupMatches(m.Group) {
		for _, side := range []SideRef{other.Left, other.Right} {
			if side.From != nil {
				feeds[side.From.Code] = append(feeds[side.From.Code], other.Code)
			}
		}
	}
	seen := map[string]bool{}
	var places []int
	var walk func(code string)
	walk = func(code string) {
		if seen[code] {
			return
		}
		seen[code] = true
		match, _ := p.MatchByCode(code)
		for _, place := range []int{match.WinnerPlace, match.LoserPlace} {
			if place > 0 && !slices.Contains(places, place) {
				places = append(places, place)
			}
		}
		for _, next := range feeds[code] {
			walk(next)
		}
	}
	walk(code)
	slices.Sort(places)
	return places
}
