package format

import (
	"fmt"

	"scoreview/api/internal/domain"
)

// 観戦アプリの「表で見る」(星取表とトーナメント表)に要る、大会の形から読む情報。

// SeedName は枠に入るチームの出どころを、観戦者の言葉にする(トーナメント表の 1 回戦の枠の上に出す
// 「予選 A 組 1位」)。別のグループの試合の勝者と敗者は、その試合の名前で呼ぶ(「1〜4位 準決勝の勝者」)。
func SeedName(plan *Plan, src Source) string {
	switch src.Kind {
	case SourceSeed:
		return fmt.Sprintf("シード%d", src.Seed)
	case SourceGroupRank:
		return fmt.Sprintf("%s %d位", OperatorName(plan.Groups[src.Group]), src.Rank)
	case SourceMatch:
		m, ok := plan.MatchByCode(src.Match.Code)
		if !ok {
			return src.Label(plan)
		}
		if src.Match.Outcome == domain.OutcomeWin {
			return MatchStageName(plan, m) + "の勝者"
		}
		return MatchStageName(plan, m) + "の負けチーム"
	default:
		return src.Label(plan)
	}
}

// TopAdvance はプール g の上位何チームが、優勝を争うグループ(最終順位の 1 位を決めうるグループ)へ
// 進むか(星取表の「進出圏の色」)。プールでないグループや、どの順位も優勝を争わないプールは 0。
// 段の全体順位(band_rank)から入る枠は、同じ段のどのプールの同じ順位も進むものとして数える。
func TopAdvance(plan *Plan, g int) int {
	if plan.Groups[g].Kind != domain.GroupPool {
		return 0
	}
	best := bestPlaces(plan)
	advance := 0
	for target, group := range plan.Groups {
		if best[target] != 1 {
			continue
		}
		for _, slot := range group.Slots {
			src := slot.Source
			fed := (src.Kind == SourceGroupRank && src.Group == g) ||
				(src.Kind == SourceBandRank && src.Stage == plan.Groups[g].Stage)
			if fed && src.Rank > advance {
				advance = src.Rank
			}
		}
	}
	return advance
}

// bestPlaces はグループごとの、そのグループのチームが取りうる最も良い最終順位。最終順位になる
// グループは RankOffset+1、ならないグループはチームを送る先のグループのうち最も良いもの。どこにも
// 送らないグループは 0。
func bestPlaces(plan *Plan) []int {
	best := make([]int, len(plan.Groups))
	for i, g := range plan.Groups {
		if g.Final {
			best[i] = g.RankOffset + 1
		}
	}
	// 送り先は必ず後の段なので、段の数だけ繰り返せば前の段まで伝わる
	for range plan.Stages {
		for target, group := range plan.Groups {
			if best[target] == 0 {
				continue
			}
			for _, slot := range group.Slots {
				for _, from := range feeders(plan, slot.Source) {
					if !plan.Groups[from].Final && (best[from] == 0 || best[target] < best[from]) {
						best[from] = best[target]
					}
				}
			}
		}
	}
	return best
}

// feeders は枠の出どころになるグループ(序数)。
func feeders(plan *Plan, src Source) []int {
	switch src.Kind {
	case SourceGroupRank:
		return []int{src.Group}
	case SourceBandRank:
		return plan.StageGroups(src.Stage)
	case SourceMatch:
		if m, ok := plan.MatchByCode(src.Match.Code); ok {
			return []int{m.Group}
		}
	}
	return nil
}
