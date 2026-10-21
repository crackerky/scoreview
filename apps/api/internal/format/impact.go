package format

import (
	"cmp"
	"slices"

	"scoreview/api/internal/domain"
)

// RankChange はプール順位の変化。Before と After の 0 は順位なし(枠が埋まっていない)。
type RankChange struct {
	Group  int
	TeamID int64
	Before int
	After  int
}

// SlotChange は枠に入るチームの変化。0 は未確定。
type SlotChange struct {
	Slot   SlotKey
	Before int64
	After  int64
}

// AffectedMatch は対戦チームが変わる試合。HasResult は既に結果(試合中を含む)がある試合で、
// この訂正は運営者の明示的な承認なしに保存しない。
type AffectedMatch struct {
	Code        string
	BeforeLeft  int64
	BeforeRight int64
	AfterLeft   int64
	AfterRight  int64
	HasResult   bool
}

// Impact は状態を変えたときに変わるもの(docs/format-engine.md「訂正の影響を先に見せる」)。
type Impact struct {
	RankChanges     []RankChange
	SlotChanges     []SlotChange
	AffectedMatches []AffectedMatch
	// NeedsApproval は既に結果の出た下流の試合に影響するかどうか
	NeedsApproval bool
}

// ComputeImpact は before から after へ変えたときの差分を返す。
func ComputeImpact(plan *Plan, cfg Config, before, after State) (*Impact, error) {
	old, err := Resolve(plan, cfg, before)
	if err != nil {
		return nil, err
	}
	next, err := Resolve(plan, cfg, after)
	if err != nil {
		return nil, err
	}
	out := &Impact{}
	for i := range plan.Groups {
		changes := rankChanges(i, old.Groups[i].Ranking, next.Groups[i].Ranking)
		out.RankChanges = append(out.RankChanges, changes...)
	}
	for _, g := range plan.Groups {
		for _, slot := range g.Slots {
			key := SlotKey{Group: g.Index, Position: slot.Position}
			if a, b := old.Slots[key].TeamID, next.Slots[key].TeamID; a != b {
				out.SlotChanges = append(out.SlotChanges, SlotChange{Slot: key, Before: a, After: b})
			}
		}
	}
	for _, m := range plan.Matches {
		a, b := old.Matches[m.Code], next.Matches[m.Code]
		if a.Left.TeamID == b.Left.TeamID && a.Right.TeamID == b.Right.TeamID {
			continue
		}
		res, ok := after.Results[m.Code]
		hasResult := ok && (res.Status == domain.MatchLive || res.Status == domain.MatchFinished)
		out.AffectedMatches = append(out.AffectedMatches, AffectedMatch{
			Code: m.Code, BeforeLeft: a.Left.TeamID, BeforeRight: a.Right.TeamID,
			AfterLeft: b.Left.TeamID, AfterRight: b.Right.TeamID, HasResult: hasResult,
		})
		if hasResult {
			out.NeedsApproval = true
		}
	}
	return out, nil
}

// ImpactOfResult は試合 code の結果を result に変えたときの差分を返す。st は変えない。
func ImpactOfResult(plan *Plan, cfg Config, st State, code string, result Result) (*Impact, error) {
	after := st.clone()
	after.Results[code] = result
	return ComputeImpact(plan, cfg, st, after)
}

func rankChanges(group int, before, after *Ranking) []RankChange {
	ranks := map[int64][2]int{}
	if before != nil {
		for _, row := range before.Rows {
			ranks[row.TeamID] = [2]int{row.Rank, 0}
		}
	}
	if after != nil {
		for _, row := range after.Rows {
			pair := ranks[row.TeamID]
			pair[1] = row.Rank
			ranks[row.TeamID] = pair
		}
	}
	var out []RankChange
	for team, pair := range ranks {
		if pair[0] != pair[1] {
			out = append(out, RankChange{Group: group, TeamID: team, Before: pair[0], After: pair[1]})
		}
	}
	slices.SortFunc(out, func(a, b RankChange) int {
		return cmp.Or(cmp.Compare(a.After, b.After), cmp.Compare(a.TeamID, b.TeamID))
	})
	return out
}
