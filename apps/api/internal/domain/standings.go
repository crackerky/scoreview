package domain

import "slices"

// PoolEntry はプール(総当たりグループ)の 1 チーム。RankOverride は group_teams.rank_override。
type PoolEntry struct {
	TeamID       int64
	RankOverride *int
}

// PoolStanding はプールのリーグ結果の 1 行。Rank が nil なら順位欄を空にする。
type PoolStanding struct {
	TeamID int64
	Tally  Tally
	Points int
	Rank   *int
}

// RankPool はプールの順位を計算する。戻り値は entries と同じ順(リーグ結果は登録順で並べ、
// 順位順ではない。clone-spec C 2.1(b))。results はそのグループの試合。
//
// 比べる順は clone-spec C 2.2(41 プールで不一致 0):
//  1. 勝ち点(勝ち 3、分け 1)の降順
//  2. 同じ勝ち点が 2 チームだけなら直接対決の勝ち点の降順(引き分けなら次へ)
//  3. 得失差の降順
//  4. 得点の降順(参照サイトでこれが効く例は無く推測)
//
// 同じ勝ち点が 3 チーム以上のときは直接対決を飛ばして得失差、得点で比べる。参照サイトに例が無く推測。
// すべて同じなら同順位にし、次の順位は飛ばす(1, 2, 2, 4。推測)。
// 終了した試合が 1 つもないプールは順位を出さない(未開催のディビジョン表が空欄になることからの推測)。
// RankOverride があれば計算より優先し、0 なら空欄(nil)にする。
func RankPool(entries []PoolEntry, results []Result) []PoolStanding {
	standings := make([]PoolStanding, len(entries))
	anyCounted := false
	for i, entry := range entries {
		tally := TallyFor(entry.TeamID, results)
		standings[i] = PoolStanding{TeamID: entry.TeamID, Tally: tally, Points: tally.Points()}
		if tally.Games > 0 {
			anyCounted = true
		}
	}

	if anyCounted {
		assignComputedRanks(standings, results)
	}
	for i, entry := range entries {
		if entry.RankOverride == nil {
			continue
		}
		if *entry.RankOverride == 0 {
			standings[i].Rank = nil
			continue
		}
		rank := *entry.RankOverride
		standings[i].Rank = &rank
	}
	return standings
}

func assignComputedRanks(standings []PoolStanding, results []Result) {
	samePoints := map[int]int{}
	for _, s := range standings {
		samePoints[s.Points]++
	}
	compare := func(a, b PoolStanding) int {
		if a.Points != b.Points {
			return b.Points - a.Points
		}
		if samePoints[a.Points] == 2 {
			if c := compareHeadToHead(a.TeamID, b.TeamID, results); c != 0 {
				return c
			}
		}
		if a.Tally.GoalDiff() != b.Tally.GoalDiff() {
			return b.Tally.GoalDiff() - a.Tally.GoalDiff()
		}
		return b.Tally.GoalsFor - a.Tally.GoalsFor
	}

	order := make([]int, len(standings))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(i, j int) int { return compare(standings[i], standings[j]) })

	for pos, idx := range order {
		rank := pos + 1
		if pos > 0 {
			prev := order[pos-1]
			if compare(standings[prev], standings[idx]) == 0 {
				rank = *standings[prev].Rank
			}
		}
		standings[idx].Rank = &rank
	}
}

// compareHeadToHead は a と b の直接対決の勝ち点で比べる。a が上なら負を返す。
func compareHeadToHead(a, b int64, results []Result) int {
	var mutual []Result
	for _, r := range results {
		_, aIn := r.sideOf(a)
		_, bIn := r.sideOf(b)
		if aIn && bIn {
			mutual = append(mutual, r)
		}
	}
	return TallyFor(b, mutual).Points() - TallyFor(a, mutual).Points()
}
