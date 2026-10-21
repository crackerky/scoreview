package format_test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

func rankMap(r format.Ranking) map[int64]int {
	out := map[int64]int{}
	for _, row := range r.Rows {
		out[row.TeamID] = row.Rank
	}
	return out
}

func order(r format.Ranking) []int64 {
	var out []int64
	for _, row := range r.Rows {
		out = append(out, row.TeamID)
	}
	return out
}

func reasonText(t *testing.T, r format.Ranking, team int64) string {
	t.Helper()
	row, ok := r.RankOf(team)
	require.True(t, ok)
	return row.Reason.Text()
}

// TestCurrentRuleGoalsFor6537 は R3 3 章の 6537: 勝ち点が同じ、直接対決は引き分け、得失点差も同じで、
// 得点 8 対 6 で決まった組。
func TestCurrentRuleGoalsFor6537(t *testing.T) {
	games := []format.Game{
		game("A1", 1, 2, 3, 3), game("A2", 1, 3, 4, 2), game("A3", 1, 4, 1, 2),
		game("A4", 2, 3, 2, 0), game("A5", 2, 4, 1, 2), game("A6", 3, 4, 5, 0),
	}
	in := format.RankInput{Teams: []int64{1, 2, 3, 4}, Games: games, Rule: format.CurrentRule(true),
		Forfeit: format.DefaultForfeitScore()}
	got := format.RankPool(in)
	assert.Equal(t, []int64{4, 1, 2, 3}, order(got))
	assert.Equal(t, map[int64]int{4: 1, 1: 2, 2: 3, 3: 4}, rankMap(got))
	assert.Empty(t, got.Unresolved)
	assert.Equal(t, "勝ち点同数、得点で上位", reasonText(t, got, 1))
	assert.Equal(t, "勝ち点同数、得点で下位", reasonText(t, got, 2))
	assert.Equal(t, "", reasonText(t, got, 4))
	row, _ := got.RankOf(1)
	assert.Equal(t, format.Stats{Games: 3, Wins: 1, Draws: 1, Losses: 1, GoalsFor: 8, GoalsAgainst: 7}, row.Stats)

	// 直接対決を外しても同じ(直接対決は引き分け)
	in.Rule = format.CurrentRule(false)
	assert.Equal(t, []int64{4, 1, 2, 3}, order(format.RankPool(in)))
}

func TestCurrentRuleHeadToHead(t *testing.T) {
	// clone-spec C 2.2 の 6511: 得失差は 2 が上だが、直接対決で 1 が上
	games := []format.Game{
		game("A1", 1, 2, 5, 4), game("A2", 1, 3, 3, 6), game("A3", 1, 4, 9, 1),
		game("A4", 2, 3, 8, 2), game("A5", 2, 4, 7, 3), game("A6", 3, 4, 2, 5),
	}
	in := format.RankInput{Teams: []int64{1, 2, 3, 4}, Games: games, Rule: format.CurrentRule(true)}
	got := format.RankPool(in)
	assert.Equal(t, map[int64]int{1: 1, 2: 2, 3: 4, 4: 3}, rankMap(got))
	assert.Equal(t, "勝ち点同数、直接対決で上位", reasonText(t, got, 1))
	// 直接対決を外すと得失差で 2 が上
	in.Rule = format.CurrentRule(false)
	assert.Equal(t, map[int64]int{1: 2, 2: 1, 3: 3, 4: 4}, rankMap(format.RankPool(in)))
}

// jfdaGames は 4 チームとも勝利数 1。D は C に没収で負けている。A-B は A、B-C は B が勝ち、A-C は引き分け。
func jfdaGames() []format.Game {
	forfeit := format.Result{Status: domain.MatchFinished, Type: format.ResultForfeit,
		Winner: new(domain.SideLeft)}
	return []format.Game{
		game("A1", 1, 2, 1, 0), game("A2", 2, 3, 10, 0), game("A3", 1, 3, 0, 0),
		{Code: "A4", Left: 3, Right: 4, Result: &forfeit},
		game("A5", 4, 1, 2, 1), game("A6", 2, 4, 3, 3),
	}
}

// TestJFDARestart は一部だけ分かれたら残りに最初から再適用することを確かめる。D は棄権で最下位、
// 残る 3 チームは該当チーム間の勝利数で C が分かれ、残る A と B を最初からやり直して A-B の勝者 A が上。
// 再適用せず次の基準(該当チーム間の得失点差)へ進むと B(+9)が A(+1)の上になる。
func TestJFDARestart(t *testing.T) {
	in := format.RankInput{Teams: []int64{1, 2, 3, 4}, Games: jfdaGames(), Rule: format.JFDARule(),
		Forfeit: format.DefaultForfeitScore()}
	got := format.RankPool(in)
	assert.Equal(t, []int64{1, 2, 3, 4}, order(got))
	assert.Equal(t, "勝利数同数、該当チーム間の勝利数で決定", reasonText(t, got, 1))
	assert.Equal(t, "勝利数同数、棄権試合数で決定", reasonText(t, got, 4))
	row, _ := got.RankOf(4)
	assert.Equal(t, 1, row.Stats.Forfeits)
	assert.Equal(t, 15+1+3, row.Stats.GoalsAgainst) // 没収の規定スコア 15-0、A5 の 1、A6 の 3

	sequential := format.JFDARule()
	sequential.Mode = format.TieSequential
	in.Rule = sequential
	assert.Equal(t, []int64{2, 1, 3, 4}, order(format.RankPool(in)))
}

// TestCycleUnresolvedAndDecision は 3 チームのじゃんけん状態。スコアも同じなら全基準で分かれず未決、
// 運営者の指定で決まる。
func TestCycleUnresolvedAndDecision(t *testing.T) {
	games := []format.Game{game("A1", 1, 2, 5, 4), game("A2", 2, 3, 5, 4), game("A3", 3, 1, 5, 4)}
	for _, rule := range []format.Rule{format.CurrentRule(true), format.JFDARule()} {
		t.Run(rule.Name, func(t *testing.T) {
			in := format.RankInput{Teams: []int64{1, 2, 3}, Games: games, Rule: rule}
			got := format.RankPool(in)
			assert.Equal(t, [][]int64{{1, 2, 3}}, got.Unresolved)
			assert.Equal(t, map[int64]int{1: 1, 2: 1, 3: 1}, rankMap(got))
			_, ok := got.TeamAt(1)
			assert.False(t, ok)
			assert.Contains(t, reasonText(t, got, 1), "全基準で並び未決")

			in.Decision = []int64{2, 3, 1}
			decided := format.RankPool(in)
			assert.Empty(t, decided.Unresolved)
			assert.Equal(t, []int64{2, 3, 1}, order(decided))
			assert.Equal(t, map[int64]int{2: 1, 3: 2, 1: 3}, rankMap(decided))
			assert.Contains(t, reasonText(t, decided, 2), "運営者の指定で")

			in.Decision = []int64{2, 3} // 一部だけの指定では決めない
			assert.Len(t, format.RankPool(in).Unresolved, 1)
		})
	}
}

func TestCycleResolvedByGoalDiff(t *testing.T) {
	games := []format.Game{game("A1", 1, 2, 5, 4), game("A2", 2, 3, 6, 3), game("A3", 3, 1, 7, 4)}
	got := format.RankPool(format.RankInput{Teams: []int64{1, 2, 3}, Games: games, Rule: format.CurrentRule(true)})
	assert.Equal(t, []int64{2, 3, 1}, order(got))
	assert.Equal(t, "勝ち点同数、得失点差で決定", reasonText(t, got, 3))
}

func TestResultTypes(t *testing.T) {
	left := domain.SideLeft
	games := []format.Game{
		{Code: "A1", Left: 1, Right: 2, Result: &format.Result{Status: domain.MatchFinished,
			Type: format.ResultForfeit, Winner: &left}},
		{Code: "A2", Left: 1, Right: 3, Result: &format.Result{Status: domain.MatchFinished,
			Type: format.ResultVoid, Winner: &left}},
		{Code: "A3", Left: 2, Right: 3, Result: &format.Result{Status: domain.MatchFinished,
			Type: format.ResultDoubleForfeit}},
		{Code: "A4", Left: 1, Right: 4, Result: &format.Result{Type: format.ResultUnplayed}},
		{Code: "A5", Left: 2, Right: 4, Result: &format.Result{Status: domain.MatchFinished,
			Type: format.ResultAbandoned, LeftScore: intp(4), RightScore: intp(6)}},
		// スコアのない終了試合は数えない
		{Code: "A6", Left: 3, Right: 4, Result: &format.Result{Status: domain.MatchFinished}},
	}
	got := format.RankPool(format.RankInput{Teams: []int64{1, 2, 3, 4}, Games: games,
		Rule: format.CurrentRule(true), Forfeit: format.ForfeitScore{Winner: 13, Loser: 0}})
	stats := map[int64]format.Stats{}
	for _, row := range got.Rows {
		stats[row.TeamID] = row.Stats
	}
	assert.Equal(t, format.Stats{Games: 2, Wins: 2, GoalsFor: 13}, stats[1])
	assert.Equal(t, format.Stats{Games: 3, Losses: 3, GoalsFor: 4, GoalsAgainst: 19, Forfeits: 2}, stats[2])
	assert.Equal(t, format.Stats{Games: 2, Losses: 2, Forfeits: 1}, stats[3])
	assert.Equal(t, format.Stats{Games: 1, Wins: 1, GoalsFor: 6, GoalsAgainst: 4}, stats[4])
}

// TestCurrentRuleMatchesDomain は現行方式が既存の domain.RankPool と同じ順位を出すことを、
// シード固定の乱数で作った 3〜6 チームのプール 2000 個で確かめる。
func TestCurrentRuleMatchesDomain(t *testing.T) {
	rng := rand.New(rand.NewPCG(20261021, 1))
	for trial := range 2000 {
		n := 3 + rng.IntN(4)
		var teams []int64
		var entries []domain.PoolEntry
		for i := range n {
			teams = append(teams, int64(i+1))
			entries = append(entries, domain.PoolEntry{TeamID: int64(i + 1)})
		}
		var games []format.Game
		var results []domain.Result
		for a := 1; a <= n; a++ {
			for b := a + 1; b <= n; b++ {
				if rng.IntN(5) == 0 {
					continue // 未消化の試合も混ぜる
				}
				ls, rs := rng.IntN(4), rng.IntN(4) // 小さいスコアで同点を起こしやすくする
				games = append(games, game("X", int64(a), int64(b), ls, rs))
				results = append(results, domain.Result{LeftTeamID: int64(a), RightTeamID: int64(b),
					LeftScore: intp(ls), RightScore: intp(rs), Status: domain.MatchFinished})
			}
		}
		if len(results) == 0 {
			continue
		}
		want := map[int64]int{}
		for _, s := range domain.RankPool(entries, results) {
			want[s.TeamID] = *s.Rank
		}
		got := format.RankPool(format.RankInput{Teams: teams, Games: games, Rule: format.CurrentRule(true)})
		require.Equal(t, want, rankMap(got), "trial %d", trial)
	}
}

func TestRankBand(t *testing.T) {
	entries := []format.BandEntry{
		{TeamID: 1, Group: 0, Stats: format.Stats{Games: 3, Wins: 2, GoalsFor: 20, GoalsAgainst: 10}},
		{TeamID: 2, Group: 1, Stats: format.Stats{Games: 2, Wins: 1, Draws: 1, GoalsFor: 10, GoalsAgainst: 5}},
		{TeamID: 3, Group: 2, Stats: format.Stats{Games: 2, Wins: 1, Draws: 1, GoalsFor: 12, GoalsAgainst: 7}},
		{TeamID: 4, Group: 3, Stats: format.Stats{Games: 2, Wins: 1, Draws: 1, GoalsFor: 12, GoalsAgainst: 7}},
	}
	got := format.RankBand(entries, format.BandRule(), nil)
	// 1 試合あたりの勝ち点: 1 は 6/3=2、他は 4/2=2。得失点差 10/3 < 5/2? 1 は 3.33、2〜4 は 2.5
	assert.Equal(t, []int64{1, 3, 4, 2}, order(got))
	assert.Equal(t, [][]int64{{3, 4}}, got.Unresolved)
	decided := format.RankBand(entries, format.BandRule(), []int64{4, 3})
	assert.Equal(t, []int64{1, 4, 3, 2}, order(decided))
}
