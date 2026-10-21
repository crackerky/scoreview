package format

import (
	"fmt"
	"slices"
)

// CriterionKind は順位の基準。
type CriterionKind string

// CriterionKind の値。H2H は「該当チーム間」(並んだチームどうしの試合だけ)、Common は
// 「共通の相手」(並んだチームの全員が対戦した、並んだチーム以外の相手との試合だけ)を数える。
const (
	CritPoints         CriterionKind = "points"
	CritWins           CriterionKind = "wins"
	CritH2HPoints      CriterionKind = "h2h_points"
	CritH2HWins        CriterionKind = "h2h_wins"
	CritFewerForfeits  CriterionKind = "fewer_forfeits"
	CritGoalDiff       CriterionKind = "goal_diff"
	CritGoalsFor       CriterionKind = "goals_for"
	CritH2HGoalDiff    CriterionKind = "h2h_goal_diff"
	CritH2HGoalsFor    CriterionKind = "h2h_goals_for"
	CritCommonGoalDiff CriterionKind = "common_goal_diff"
	CritCommonGoalsFor CriterionKind = "common_goals_for"
	// 段の全体順位(band_rank)の基準。試合数の違いを 1 試合あたりで吸収する
	CritPointsPerGame   CriterionKind = "points_per_game"
	CritGoalDiffPerGame CriterionKind = "goal_diff_per_game"
	CritGoalsForPerGame CriterionKind = "goals_for_per_game"
	// CritManual は運営者の指定(ブリックマーク投げ、ディスクフリップの結果)で決めたことを表す。
	// 基準リストには入れない
	CritManual CriterionKind = "manual"
)

// Label は画面に出す基準の名前。
func (k CriterionKind) Label() string {
	switch k {
	case CritPoints:
		return "勝ち点"
	case CritWins:
		return "勝利数"
	case CritH2HPoints:
		return "直接対決"
	case CritH2HWins:
		return "該当チーム間の勝利数"
	case CritFewerForfeits:
		return "棄権試合数"
	case CritGoalDiff:
		return "得失点差"
	case CritGoalsFor:
		return "得点"
	case CritH2HGoalDiff:
		return "該当チーム間の得失点差"
	case CritH2HGoalsFor:
		return "該当チーム間の得点"
	case CritCommonGoalDiff:
		return "共通の相手との得失点差"
	case CritCommonGoalsFor:
		return "共通の相手との得点"
	case CritPointsPerGame:
		return "1 試合あたりの勝ち点"
	case CritGoalDiffPerGame:
		return "1 試合あたりの得失点差"
	case CritGoalsForPerGame:
		return "1 試合あたりの得点"
	case CritManual:
		return "運営者の指定"
	}
	return string(k)
}

// scoreDependent は基準がスコア(得点・失点)を見るかどうか。勝ち上がり条件の列挙は
// スコアを見ない基準だけで行う。
func (k CriterionKind) scoreDependent() bool {
	switch k {
	case CritPoints, CritWins, CritH2HPoints, CritH2HWins, CritFewerForfeits:
		return false
	}
	return true
}

// Criterion は基準リストの 1 項目。MaxTied が 0 でなければ、並んだチームが MaxTied 以下の
// ときだけ使う(現行方式の直接対決は 2 チームのときだけ。clone-spec C 2.2)。
type Criterion struct {
	Kind    CriterionKind
	MaxTied int
}

// TieMode は一部のチームだけが分かれたときの進め方。
type TieMode string

// TieMode の値。
const (
	// TieRestart は残ったチームの間で最初の基準からやり直す(JFDA、UPA 2.5。R2 2.1)
	TieRestart TieMode = "restart"
	// TieSequential は残ったチームの間で次の基準へ進む(基準を辞書式に比べるのと同じ)
	TieSequential TieMode = "sequential"
)

// Rule はタイブレークの設定。すべての基準で分かれなければ未決にする(勝手に決めない)。
type Rule struct {
	Name     string
	Mode     TieMode
	Criteria []Criterion
}

// CurrentRule は「現行方式」(初期値): 勝ち点 → 直接対決(2 チームのときだけ)→ 得失点差 → 得点。
// 参照サイトの 56 プールで全一致(R3 3 章)。直接対決を見ているかはデータから判別できないので
// headToHead で外せる。基準の比べ方は辞書式(TieSequential)で、domain.RankPool と同じ順位になる。
func CurrentRule(headToHead bool) Rule {
	criteria := []Criterion{{Kind: CritPoints}}
	if headToHead {
		criteria = append(criteria, Criterion{Kind: CritH2HPoints, MaxTied: 2})
	}
	criteria = append(criteria, Criterion{Kind: CritGoalDiff}, Criterion{Kind: CritGoalsFor})
	return Rule{Name: "current", Mode: TieSequential, Criteria: criteria}
}

// JFDARule は JFDA 競技規則(2025 全日本ミックス。R2 2.1): 勝利数 → 該当チーム間の勝利数 →
// 棄権試合数の少なさ → 該当チーム間の得失点差 → 共通の相手との得失点差 → 該当チーム間の得点 →
// 共通の相手との得点。一部だけ分かれたら残りに最初から再適用する。最後のブリックマーク投げは
// 運営者の指定(Decision)で入れる。
func JFDARule() Rule {
	return Rule{Name: "jfda", Mode: TieRestart, Criteria: []Criterion{
		{Kind: CritWins}, {Kind: CritH2HWins}, {Kind: CritFewerForfeits},
		{Kind: CritH2HGoalDiff}, {Kind: CritCommonGoalDiff},
		{Kind: CritH2HGoalsFor}, {Kind: CritCommonGoalsFor},
	}}
}

// BandRule は段の全体順位(band_rank)の初期値: 1 試合あたりの勝ち点 → 1 試合あたりの得失点差 →
// 1 試合あたりの得点 → 未決。参照サイトはこの順位を画像で掲示しており基準を読めないので推測
// (docs/format-engine.md「順位帯をまたぐ全体順位」)。運営会社に確認する。
func BandRule() Rule {
	return Rule{Name: "band", Mode: TieSequential, Criteria: []Criterion{
		{Kind: CritPointsPerGame}, {Kind: CritGoalDiffPerGame}, {Kind: CritGoalsForPerGame},
	}}
}

// RankInput はプール順位の計算の入力。Teams はグループの枠の順。
type RankInput struct {
	Teams   []int64
	Games   []Game
	Rule    Rule
	Forfeit ForfeitScore
	// Decision は運営者が決めた順(group_rank_decisions)。未決の同順位のチームが全員ここに
	// あれば、この並びで順位を決める
	Decision []int64
}

// Ranking は順位の計算結果。
type Ranking struct {
	// Rows は順位順。未決で並んだチームは同じ Rank を持ち、Teams の順に並ぶ
	Rows []RankRow
	// Unresolved は未決のまま残った同順位の組
	Unresolved [][]int64
}

// RankRow は順位表の 1 行。
type RankRow struct {
	TeamID int64
	Rank   int
	Stats  Stats
	Reason Reason
}

// Reason はどの基準で順位が決まったか(画面の「勝ち点同数、得失点差で上位」)。
type Reason struct {
	// TiedOn は最初の基準。TiedWith はその基準で並んだ他のチーム(空なら並ばなかった)
	TiedOn   CriterionKind
	TiedWith []int64
	// DecidedBy は最後にこのチームを分けた基準。運営者の指定なら CritManual
	DecidedBy  CriterionKind
	Unresolved bool
	// Above は 2 チームで並んで上位になったか(説明の「上位」「下位」に使う)
	Above bool
}

// Text は順位の根拠の説明。並ばなかったチームは空文字。
func (r Reason) Text() string {
	if len(r.TiedWith) == 0 {
		return ""
	}
	head := r.TiedOn.Label() + "同数"
	if r.Unresolved {
		return head + "、全基準で並び未決"
	}
	tail := "で決定"
	if len(r.TiedWith) == 1 {
		tail = "で下位"
		if r.Above {
			tail = "で上位"
		}
	}
	return head + "、" + r.DecidedBy.Label() + tail
}

// TeamAt は rank 位のチームを返す。その順位が未決の同順位か、該当が無ければ ok=false。
func (r Ranking) TeamAt(rank int) (int64, bool) {
	var found []int64
	for _, row := range r.Rows {
		if row.Rank == rank {
			found = append(found, row.TeamID)
		}
	}
	if len(found) != 1 {
		return 0, false
	}
	return found[0], true
}

// RankOf は team の順位を返す。
func (r Ranking) RankOf(team int64) (RankRow, bool) {
	for _, row := range r.Rows {
		if row.TeamID == team {
			return row, true
		}
	}
	return RankRow{}, false
}

// RankPool はプールの順位を計算する。数えるのは終了した試合(没収、不成立を含む)だけ。
// 基準で分かれない同順位は Unresolved に入れ、同じ順位(次の順位は飛ばす)にする。
func RankPool(in RankInput) Ranking {
	stats := map[int64]Stats{}
	for _, team := range in.Teams {
		stats[team] = statsFor(team, in.Games, in.Forfeit)
	}
	ev := poolEvaluator{games: in.Games, forfeit: in.Forfeit, stats: stats}
	return rankWith(in.Teams, stats, in.Rule, ev.value, in.Decision)
}

// rankWith は value で基準の値を出して順位をつける共通部分(プールと段の全体順位)。
func rankWith(
	teams []int64,
	stats map[int64]Stats,
	rule Rule,
	value func(CriterionKind, []int64) map[int64]ratio,
	decision []int64,
) Ranking {
	s := &splitter{rule: rule, value: value, decidedBy: map[int64]CriterionKind{}}
	tiers := s.split(teams, 0)

	tiedWith := map[int64][]int64{}
	if len(rule.Criteria) > 0 {
		for _, tier := range groupByValue(teams, value(rule.Criteria[0].Kind, teams)) {
			for _, team := range tier {
				tiedWith[team] = without(tier, team)
			}
		}
	}

	var out Ranking
	position := 1
	for _, tier := range tiers {
		resolved := tier
		manual := false
		if len(tier) > 1 {
			if ordered, ok := applyDecision(tier, decision); ok {
				resolved, manual = ordered, true
			}
		}
		if len(resolved) > 1 && !manual {
			out.Unresolved = append(out.Unresolved, slices.Clone(tier))
		}
		for i, team := range resolved {
			rank := position
			if manual {
				rank = position + i
			}
			reason := Reason{TiedWith: tiedWith[team], DecidedBy: s.decidedBy[team],
				Unresolved: len(resolved) > 1 && !manual}
			if len(rule.Criteria) > 0 {
				reason.TiedOn = rule.Criteria[0].Kind
			}
			if manual {
				reason.DecidedBy = CritManual
			}
			out.Rows = append(out.Rows, RankRow{TeamID: team, Rank: rank, Stats: stats[team], Reason: reason})
		}
		position += len(tier)
	}
	// 2 チームの並びの上位・下位
	for i := range out.Rows {
		row := &out.Rows[i]
		if len(row.Reason.TiedWith) == 1 {
			if other, ok := out.RankOf(row.Reason.TiedWith[0]); ok {
				row.Reason.Above = row.Rank < other.Rank
			}
		}
	}
	return out
}

func applyDecision(tier, decision []int64) ([]int64, bool) {
	index := map[int64]int{}
	for i, team := range decision {
		index[team] = i
	}
	for _, team := range tier {
		if _, ok := index[team]; !ok {
			return nil, false
		}
	}
	ordered := slices.Clone(tier)
	slices.SortFunc(ordered, func(a, b int64) int { return index[a] - index[b] })
	return ordered, true
}

func without(set []int64, team int64) []int64 {
	var out []int64
	for _, t := range set {
		if t != team {
			out = append(out, t)
		}
	}
	return out
}

// splitter は基準リストで同順位を分ける。
type splitter struct {
	rule      Rule
	value     func(CriterionKind, []int64) map[int64]ratio
	decidedBy map[int64]CriterionKind
}

// split は set を基準 from 以降で分け、上位から並べた組の列を返す。1 チームの組は決まった順位、
// 2 チーム以上の組は未決。
func (s *splitter) split(set []int64, from int) [][]int64 {
	if len(set) <= 1 {
		return [][]int64{set}
	}
	for ci := from; ci < len(s.rule.Criteria); ci++ {
		crit := s.rule.Criteria[ci]
		if crit.MaxTied > 0 && len(set) > crit.MaxTied {
			continue
		}
		tiers := groupByValue(set, s.value(crit.Kind, set))
		if len(tiers) == 1 {
			continue
		}
		for _, team := range set {
			s.decidedBy[team] = crit.Kind
		}
		var out [][]int64
		for _, tier := range tiers {
			next := ci + 1
			if s.rule.Mode == TieRestart {
				next = 0
			}
			out = append(out, s.split(tier, next)...)
		}
		return out
	}
	return [][]int64{set}
}

// groupByValue は値の大きい順に組を作る。組の中は set の順。
func groupByValue(set []int64, values map[int64]ratio) [][]int64 {
	ordered := slices.Clone(set)
	slices.SortStableFunc(ordered, func(a, b int64) int { return values[b].cmp(values[a]) })
	var tiers [][]int64
	for i, team := range ordered {
		if i > 0 && values[ordered[i-1]].cmp(values[team]) == 0 {
			tiers[len(tiers)-1] = append(tiers[len(tiers)-1], team)
			continue
		}
		tiers = append(tiers, []int64{team})
	}
	return tiers
}

// ratio は分数の値。1 試合あたりの基準を浮動小数点の誤差なしに比べるため。den は正。
type ratio struct {
	num int
	den int
}

func whole(n int) ratio { return ratio{num: n, den: 1} }

func (r ratio) cmp(o ratio) int {
	left, right := r.num*o.den, o.num*r.den
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	}
	return 0
}

// poolEvaluator はプールの基準の値を出す。
type poolEvaluator struct {
	games   []Game
	forfeit ForfeitScore
	stats   map[int64]Stats
}

func (ev poolEvaluator) value(kind CriterionKind, set []int64) map[int64]ratio {
	out := map[int64]ratio{}
	switch kind {
	case CritH2HPoints, CritH2HWins, CritH2HGoalDiff, CritH2HGoalsFor:
		mutual := ev.gamesWithin(set)
		for _, team := range set {
			out[team] = whole(statValue(kind, statsFor(team, mutual, ev.forfeit)))
		}
	case CritCommonGoalDiff, CritCommonGoalsFor:
		common := ev.commonGames(set)
		for _, team := range set {
			out[team] = whole(statValue(kind, statsFor(team, common[team], ev.forfeit)))
		}
	default:
		for _, team := range set {
			out[team] = statRatio(kind, ev.stats[team])
		}
	}
	return out
}

func statValue(kind CriterionKind, s Stats) int {
	switch kind {
	case CritPoints, CritH2HPoints:
		return s.Points()
	case CritWins, CritH2HWins:
		return s.Wins
	case CritFewerForfeits:
		return -s.Forfeits
	case CritGoalDiff, CritH2HGoalDiff, CritCommonGoalDiff:
		return s.GoalDiff()
	case CritGoalsFor, CritH2HGoalsFor, CritCommonGoalsFor:
		return s.GoalsFor
	}
	// 基準の一覧と switch の不一致はプログラムの誤り
	panic(fmt.Sprintf("format: criterion %q has no team value", kind))
}

func statRatio(kind CriterionKind, s Stats) ratio {
	games := max(s.Games, 1) // 試合数 0 は値 0 として比べる
	switch kind {
	case CritPointsPerGame:
		return ratio{num: s.Points(), den: games}
	case CritGoalDiffPerGame:
		return ratio{num: s.GoalDiff(), den: games}
	case CritGoalsForPerGame:
		return ratio{num: s.GoalsFor, den: games}
	}
	return whole(statValue(kind, s))
}

func (ev poolEvaluator) gamesWithin(set []int64) []Game {
	var out []Game
	for _, g := range ev.games {
		if slices.Contains(set, g.Left) && slices.Contains(set, g.Right) {
			out = append(out, g)
		}
	}
	return out
}

// commonGames はチームごとの「共通の相手」との試合。共通の相手は set の全チームと数える試合を
// した set 外のチーム。同じ相手と複数回当たった場合の平均(UPA 4a)はプールでは起きないので扱わない。
func (ev poolEvaluator) commonGames(set []int64) map[int64][]Game {
	opponents := map[int64]map[int64]bool{}
	for _, team := range set {
		opponents[team] = map[int64]bool{}
	}
	for _, g := range ev.games {
		if state, _ := g.Result.decide(); !state.counted() {
			continue
		}
		if _, ok := opponents[g.Left]; ok && !slices.Contains(set, g.Right) {
			opponents[g.Left][g.Right] = true
		}
		if _, ok := opponents[g.Right]; ok && !slices.Contains(set, g.Left) {
			opponents[g.Right][g.Left] = true
		}
	}
	isCommon := func(opp int64) bool {
		for _, team := range set {
			if !opponents[team][opp] {
				return false
			}
		}
		return true
	}
	out := map[int64][]Game{}
	for _, g := range ev.games {
		for _, team := range set {
			switch {
			case g.Left == team && isCommon(g.Right), g.Right == team && isCommon(g.Left):
				out[team] = append(out[team], g)
			}
		}
	}
	return out
}
