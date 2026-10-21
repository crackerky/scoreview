package format

// BandEntry は段の全体順位(band_rank)に並べる 1 チーム。段のあるプールで同じ順位だったチーム。
type BandEntry struct {
	TeamID int64
	Group  int
	Stats  Stats
}

// RankBand は段の全プールの同順位を並べる(プールの大きさが不揃いなときの全体順位)。
// rule は通常 BandRule()。該当チーム間・共通の相手の基準は値がつかず分けない。
// entries の順(プール順)を同順位の組の中の並びに使う。
func RankBand(entries []BandEntry, rule Rule, decisions []Decision) Ranking {
	teams := make([]int64, len(entries))
	stats := map[int64]Stats{}
	for i, e := range entries {
		teams[i] = e.TeamID
		stats[e.TeamID] = e.Stats
	}
	value := func(kind CriterionKind, set []int64) map[int64]ratio {
		out := map[int64]ratio{}
		for _, team := range set {
			if kind.headToHead() {
				out[team] = whole(0)
				continue
			}
			out[team] = statRatio(kind, stats[team])
		}
		return out
	}
	return rankWith(teams, stats, rule, value, decisions)
}

func (k CriterionKind) headToHead() bool {
	switch k {
	case CritH2HPoints, CritH2HWins, CritH2HGoalDiff, CritH2HGoalsFor,
		CritCommonGoalDiff, CritCommonGoalsFor:
		return true
	}
	return false
}
