package format

import (
	"cmp"
	"fmt"
	"slices"

	"scoreview/api/internal/domain"
)

// Config は順位と結果の扱いの設定(division_formats のタイブレーク設定)。Forfeit は不戦勝の規定スコアで、
// スコアを記録していない不戦勝だけに使う(記録したスコアがあればそちらで数える)。nil は規定なしで、
// その不戦勝は勝敗だけを数える。
type Config struct {
	Tiebreak Rule
	Band     Rule
	Forfeit  *ForfeitScore
}

// DefaultConfig は初期値: 現行方式(直接対決あり)、全体順位の推測の基準、没収 15-0。
func DefaultConfig() Config {
	forfeit := DefaultForfeitScore()
	return Config{Tiebreak: CurrentRule(true), Band: BandRule(), Forfeit: &forfeit}
}

// BandKey は段の全体順位の組(段の r 位を並べたもの)を指す。
type BandKey struct {
	Stage int
	Rank  int
}

// State は進行の現在。Seeds[n-1] がシード n のチーム(0 は未登録)。Results は試合 CD ごとの結果。
type State struct {
	Seeds           []int64
	Results         map[string]Result
	ConfirmedGroups map[int]bool
	ConfirmedStages map[int]bool
	// Manual は出どころが manual の枠に運営者が入れたチーム
	Manual map[SlotKey]int64
	// RankDecisions はグループごとの運営者の順位指定(group_rank_decisions)。RankInput.Decision と同じ
	RankDecisions map[int][]int64
	// BandDecisions は段の全体順位の運営者の指定
	BandDecisions map[BandKey][]int64
}

// clone は map を複製した State を返す(影響の計算で元の State を変えないため)。
func (s State) clone() State {
	out := s
	out.Seeds = slices.Clone(s.Seeds)
	out.Results = cloneMap(s.Results)
	out.ConfirmedGroups = cloneMap(s.ConfirmedGroups)
	out.ConfirmedStages = cloneMap(s.ConfirmedStages)
	out.Manual = cloneMap(s.Manual)
	out.RankDecisions = cloneMap(s.RankDecisions)
	out.BandDecisions = cloneMap(s.BandDecisions)
	return out
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// PendingReason は枠や試合の片側のチームが決まらない理由。
type PendingReason string

// PendingReason の値。
const (
	PendingSeed              PendingReason = "seed_missing"
	PendingManual            PendingReason = "manual_missing"
	PendingMatch             PendingReason = "match_not_finished"
	PendingResultIncomplete  PendingReason = "result_incomplete"
	PendingWinner            PendingReason = "winner_missing"
	PendingNoWinner          PendingReason = "no_winner"
	PendingUnplayed          PendingReason = "match_unplayed"
	PendingGroupConfirmation PendingReason = "group_not_confirmed"
	PendingStageConfirmation PendingReason = "stage_not_confirmed"
	PendingTie               PendingReason = "tie_unresolved"
	PendingRankMissing       PendingReason = "rank_missing"
	PendingUpstream          PendingReason = "upstream_pending"
	// PendingMismatch は待っている試合の記録した対戦チームが、計算上の対戦チームと食い違っている
	// (確定後の訂正で順位が変わった)。運営者が食い違いを解消するまで結果を下流へ流さない
	PendingMismatch PendingReason = "teams_mismatch"
)

// TeamStatus は枠や試合の片側の解決結果。TeamID が 0 なら未確定で、Pending と Waiting
// (待っている試合 CD やグループ文字)が理由を表す。
type TeamStatus struct {
	TeamID  int64
	Pending PendingReason
	Waiting string
}

// Resolved はチームが確定したかどうか。
func (t TeamStatus) Resolved() bool { return t.TeamID != 0 }

func pending(reason PendingReason, waiting string) TeamStatus {
	return TeamStatus{Pending: reason, Waiting: waiting}
}

// MatchTeams は試合の左右のチーム(計算上の対戦チーム)と、決まっていれば勝者と敗者。勝者と敗者は
// 結果を記録したときの対戦チーム(Result.Played)で決める。Mismatch は、始まった試合の記録した
// 対戦チームが計算上の対戦チームと食い違うこと。食い違う試合の結果は数えず、勝者と敗者を
// 出どころにする枠と最終順位は埋めない。
type MatchTeams struct {
	Code     string
	Left     TeamStatus
	Right    TeamStatus
	Winner   int64
	Loser    int64
	Mismatch bool
}

// GroupState はグループの進み具合。
type GroupState string

// GroupState の値。
const (
	GroupWaiting   GroupState = "waiting"   // 枠が埋まっていない
	GroupPlaying   GroupState = "playing"   // 試合が残っている
	GroupTied      GroupState = "tied"      // 全試合が終わったが未決の同順位がある(プール)
	GroupReady     GroupState = "ready"     // 確定待ち(プール)
	GroupConfirmed GroupState = "confirmed" // 運営者が確定した(プール)
	GroupFinished  GroupState = "finished"  // 全試合が終わった(トーナメント。確定の操作は要らない)
)

// GroupStatus はグループの状態。Ranking はプールの枠がすべて埋まっていれば入る。Done は全試合が
// 終わった(結果が数えられる。食い違いの試合は終わっていないとみなす)こと、Mismatches は対戦チームが
// 食い違っている試合の CD。食い違いのあるグループの順位は、出どころの枠と最終順位に使わない。
type GroupStatus struct {
	Group      int
	State      GroupState
	Ranking    *Ranking
	Done       bool
	Mismatches []string
}

// FinalRank は最終順位の 1 行。Tied は同順位(トーナメントの同順位、プールの未決)。
type FinalRank struct {
	TeamID int64
	Rank   int
	Tied   bool
	Group  int
}

// Resolution は進行の解決結果。
type Resolution struct {
	Slots   map[SlotKey]TeamStatus
	Matches map[string]MatchTeams
	Groups  []GroupStatus
	Final   []FinalRank
}

// Resolve は現在の結果と確定状態から、全枠のチーム、各試合の左右、グループの状態と順位、
// 最終順位を求める。トーナメントの中の勝ち上がりは試合の終了で自動に決まり、プールから次の段へは
// グループの確定(band_rank は段の確定)が要る。State に plan に無い試合 CD があればエラー。
func Resolve(plan *Plan, cfg Config, st State) (*Resolution, error) {
	for code := range st.Results {
		if _, ok := plan.MatchByCode(code); !ok {
			return nil, domain.Invalid("results", "unknown match code "+code)
		}
	}
	r := &resolver{
		plan: plan, cfg: cfg, st: st,
		out:      &Resolution{Slots: map[SlotKey]TeamStatus{}, Matches: map[string]MatchTeams{}},
		matches:  map[string]Match{},
		rankings: map[int]*Ranking{},
	}
	for _, m := range plan.Matches {
		r.matches[m.Code] = m
	}
	// 出どころは前のグループだけを指す(validatePlan)ので、グループ順に解けば足りる
	for _, g := range plan.Groups {
		for _, slot := range g.Slots {
			key := SlotKey{Group: g.Index, Position: slot.Position}
			r.out.Slots[key] = r.source(key, slot.Source)
		}
		for _, m := range plan.GroupMatches(g.Index) {
			r.matchTeams(m.Code)
		}
		r.out.Groups = append(r.out.Groups, r.groupStatus(g))
	}
	r.out.Final = r.finalRanks()
	return r.out, nil
}

type resolver struct {
	plan     *Plan
	cfg      Config
	st       State
	out      *Resolution
	matches  map[string]Match
	rankings map[int]*Ranking
}

func (r *resolver) source(key SlotKey, src Source) TeamStatus {
	switch src.Kind {
	case SourceSeed:
		if src.Seed <= len(r.st.Seeds) && r.st.Seeds[src.Seed-1] != 0 {
			return TeamStatus{TeamID: r.st.Seeds[src.Seed-1]}
		}
		return pending(PendingSeed, fmt.Sprintf("seed %d", src.Seed))
	case SourceManual:
		if team := r.st.Manual[key]; team != 0 {
			return TeamStatus{TeamID: team}
		}
		return pending(PendingManual, "")
	case SourceGroupRank:
		return r.groupRank(src.Group, src.Rank, true)
	case SourceMatch:
		return r.outcomeTeam(src.Match)
	case SourceBandRank:
		return r.bandRank(src)
	}
	return pending(PendingManual, "")
}

// groupRank はグループ g の rank 位。プールは confirmed が真なら確定を求める。
func (r *resolver) groupRank(g, rank int, needConfirm bool) TeamStatus {
	group := r.plan.Groups[g]
	if group.Kind == domain.GroupBracket {
		var deciders []MatchRef
		for _, m := range r.plan.GroupMatches(g) {
			if m.WinnerPlace == rank {
				deciders = append(deciders, MatchRef{Code: m.Code, Outcome: domain.OutcomeWin})
			}
			if m.LoserPlace == rank {
				deciders = append(deciders, MatchRef{Code: m.Code, Outcome: domain.OutcomeLose})
			}
		}
		if len(deciders) != 1 {
			return pending(PendingRankMissing, group.Letter)
		}
		return r.outcomeTeam(deciders[0])
	}
	if needConfirm && !r.st.ConfirmedGroups[g] {
		return pending(PendingGroupConfirmation, group.Letter)
	}
	if len(r.out.Groups) > g && len(r.out.Groups[g].Mismatches) > 0 {
		return pending(PendingMismatch, group.Letter)
	}
	ranking := r.rankings[g]
	if ranking == nil {
		return pending(PendingUpstream, group.Letter)
	}
	if team, ok := ranking.TeamAt(rank); ok {
		return TeamStatus{TeamID: team}
	}
	for _, row := range ranking.Rows {
		if row.Rank == rank {
			return pending(PendingTie, group.Letter)
		}
	}
	return pending(PendingRankMissing, group.Letter)
}

func (r *resolver) bandRank(src Source) TeamStatus {
	stageName := r.plan.Stages[src.Stage].Name
	if !r.st.ConfirmedStages[src.Stage] {
		return pending(PendingStageConfirmation, stageName)
	}
	var entries []BandEntry
	for _, g := range r.plan.StageGroups(src.Stage) {
		group := r.plan.Groups[g]
		if group.Kind != domain.GroupPool || group.Size() < src.Rank {
			continue
		}
		status := r.groupRank(g, src.Rank, false)
		if !status.Resolved() {
			return status
		}
		row, _ := r.rankings[g].RankOf(status.TeamID)
		entries = append(entries, BandEntry{TeamID: status.TeamID, Group: g, Stats: row.Stats})
	}
	ranking := RankBand(entries, r.cfg.Band, r.st.BandDecisions[BandKey{Stage: src.Stage, Rank: src.Rank}])
	if team, ok := ranking.TeamAt(src.Order); ok {
		return TeamStatus{TeamID: team}
	}
	if src.Order <= len(entries) {
		return pending(PendingTie, stageName)
	}
	return pending(PendingRankMissing, stageName)
}

func (r *resolver) side(m Match, side SideRef) TeamStatus {
	if side.From != nil {
		return r.outcomeTeam(*side.From)
	}
	return r.out.Slots[SlotKey{Group: m.Group, Position: side.Slot}]
}

func (r *resolver) matchTeams(code string) MatchTeams {
	if mt, ok := r.out.Matches[code]; ok {
		return mt
	}
	m := r.matches[code]
	mt := MatchTeams{Code: code, Left: r.side(m, m.Left), Right: r.side(m, m.Right)}
	res := r.result(code)
	mt.Mismatch = res.started() && !agrees(mt, res)
	if mt.Left.Resolved() && mt.Right.Resolved() && !mt.Mismatch {
		if state, winner := res.decide(); state == resultDecided {
			mt.Winner, mt.Loser = mt.Left.TeamID, mt.Right.TeamID
			if res.Played != nil {
				mt.Winner, mt.Loser = res.Played.Left, res.Played.Right
			}
			if winner == domain.SideRight {
				mt.Winner, mt.Loser = mt.Loser, mt.Winner
			}
		}
	}
	r.out.Matches[code] = mt
	return mt
}

// agrees は結果を記録したときの対戦チームが、計算上の対戦チームと同じかどうか。計算上まだ決まらない
// 側は比べない(上流の結果待ちで、その試合の結果はどのみち数えない)。記録が無い(Played が nil)
// なら照合しない。
func agrees(mt MatchTeams, res *Result) bool {
	if res == nil || res.Played == nil {
		return true
	}
	same := func(side TeamStatus, played int64) bool {
		return !side.Resolved() || side.TeamID == played
	}
	return same(mt.Left, res.Played.Left) && same(mt.Right, res.Played.Right)
}

func (r *resolver) result(code string) *Result {
	if res, ok := r.st.Results[code]; ok {
		return &res
	}
	return nil
}

// outcomeTeam は試合 ref の勝者か敗者。
func (r *resolver) outcomeTeam(ref MatchRef) TeamStatus {
	mt := r.matchTeams(ref.Code)
	if mt.Mismatch {
		return pending(PendingMismatch, ref.Code)
	}
	if !mt.Left.Resolved() || !mt.Right.Resolved() {
		return pending(PendingUpstream, ref.Code)
	}
	state, _ := r.result(ref.Code).decide()
	switch state {
	case resultDecided:
		if ref.Outcome == domain.OutcomeWin {
			return TeamStatus{TeamID: mt.Winner}
		}
		return TeamStatus{TeamID: mt.Loser}
	case resultDraw:
		return pending(PendingWinner, ref.Code)
	case resultNoWinner:
		return pending(PendingNoWinner, ref.Code)
	case resultUnplayed:
		return pending(PendingUnplayed, ref.Code)
	case resultIncomplete:
		return pending(PendingResultIncomplete, ref.Code)
	}
	return pending(PendingMatch, ref.Code)
}

// poolGames はプールの試合を順位計算の入力にする。枠がすべて埋まっていなければ ok=false。
func (r *resolver) poolGames(g int) ([]int64, []Game, bool) {
	group := r.plan.Groups[g]
	teams := make([]int64, group.Size())
	for i := range group.Slots {
		status := r.out.Slots[SlotKey{Group: g, Position: i + 1}]
		if !status.Resolved() {
			return nil, nil, false
		}
		teams[i] = status.TeamID
	}
	var games []Game
	for _, m := range r.plan.GroupMatches(g) {
		mt := r.out.Matches[m.Code]
		game := Game{Code: m.Code, Round: m.Round, Left: mt.Left.TeamID, Right: mt.Right.TeamID}
		if !mt.Mismatch {
			// 食い違う試合の結果は、計算上のチームのものとして数えない
			game.Result = r.result(m.Code)
		}
		games = append(games, game)
	}
	return teams, games, true
}

func (r *resolver) groupStatus(g Group) GroupStatus {
	status := GroupStatus{Group: g.Index}
	done, playable := true, false
	for _, m := range r.plan.GroupMatches(g.Index) {
		mt := r.out.Matches[m.Code]
		if mt.Left.Resolved() && mt.Right.Resolved() {
			playable = true
		}
		if mt.Mismatch {
			status.Mismatches = append(status.Mismatches, m.Code)
			done = false
			continue
		}
		state, _ := r.result(m.Code).decide()
		draw := state == resultDraw && g.Kind == domain.GroupPool
		if state != resultDecided && state != resultUnplayed && state != resultNoWinner && !draw {
			done = false
		}
	}
	status.Done = done
	if g.Kind == domain.GroupBracket {
		switch {
		case done:
			status.State = GroupFinished
		case playable:
			status.State = GroupPlaying
		default:
			status.State = GroupWaiting
		}
		return status
	}
	teams, games, ok := r.poolGames(g.Index)
	if !ok {
		status.State = GroupWaiting
		return status
	}
	ranking := RankPool(RankInput{Teams: teams, Games: games, Rule: r.cfg.Tiebreak,
		Forfeit: r.cfg.Forfeit, Decision: r.st.RankDecisions[g.Index]})
	r.rankings[g.Index] = &ranking
	status.Ranking = &ranking
	switch {
	case r.st.ConfirmedGroups[g.Index]:
		status.State = GroupConfirmed
	case !done:
		status.State = GroupPlaying
	case len(ranking.Unresolved) > 0:
		status.State = GroupTied
	default:
		status.State = GroupReady
	}
	return status
}

// finalRanks は最終順位が決まったチームを順位順に返す。プールは確定済みのものだけ。
func (r *resolver) finalRanks() []FinalRank {
	var out []FinalRank
	for _, g := range r.plan.Groups {
		if !g.Final {
			continue
		}
		if g.Kind == domain.GroupPool {
			ranking := r.rankings[g.Index]
			if ranking == nil || !r.st.ConfirmedGroups[g.Index] || len(r.out.Groups[g.Index].Mismatches) > 0 {
				continue
			}
			for _, row := range ranking.Rows {
				out = append(out, FinalRank{TeamID: row.TeamID, Rank: g.RankOffset + row.Rank,
					Tied: row.Reason.Unresolved, Group: g.Index})
			}
			continue
		}
		def, err := bracketDefFor(g.Bracket)
		if err != nil {
			continue // Generate が作った型なので起きない
		}
		for _, m := range r.plan.GroupMatches(g.Index) {
			mt := r.out.Matches[m.Code]
			if mt.Winner == 0 {
				continue
			}
			if m.WinnerPlace > 0 {
				out = append(out, FinalRank{TeamID: mt.Winner, Rank: g.RankOffset + m.WinnerPlace,
					Tied: def.placeTied(m.WinnerPlace), Group: g.Index})
			}
			if m.LoserPlace > 0 {
				out = append(out, FinalRank{TeamID: mt.Loser, Rank: g.RankOffset + m.LoserPlace,
					Tied: def.placeTied(m.LoserPlace), Group: g.Index})
			}
		}
	}
	slices.SortStableFunc(out, func(a, b FinalRank) int {
		return cmp.Or(cmp.Compare(a.Rank, b.Rank), cmp.Compare(a.TeamID, b.TeamID))
	})
	return out
}
