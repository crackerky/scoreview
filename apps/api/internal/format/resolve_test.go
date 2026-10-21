package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// seeds はシード n のチーム ID を n にした State の Seeds。
func seeds(n int) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = int64(i + 1)
	}
	return out
}

func newState(teams int) format.State {
	return format.State{
		Seeds: seeds(teams), Results: map[string]format.Result{},
		ConfirmedGroups: map[int]bool{}, ConfirmedStages: map[int]bool{},
	}
}

func resolve(t *testing.T, plan *format.Plan, st format.State) *format.Resolution {
	t.Helper()
	res, err := format.Resolve(plan, format.DefaultConfig(), st)
	require.NoError(t, err)
	return res
}

// playAll は左右が決まった試合に「左が 2-1 で勝つ」結果を入れ、プールは全試合が終わったら確定し、
// 進められなくなるまで繰り返す。プールの枠は良いシード順なので、良いシードが勝ち続ける。
func playAll(t *testing.T, plan *format.Plan, st format.State) *format.Resolution {
	t.Helper()
	for {
		res := resolve(t, plan, st)
		changed := false
		for _, m := range plan.Matches {
			mt := res.Matches[m.Code]
			if _, done := st.Results[m.Code]; done || !mt.Left.Resolved() || !mt.Right.Resolved() {
				continue
			}
			st.Results[m.Code] = score(2, 1)
			changed = true
		}
		for _, g := range res.Groups {
			if g.State == format.GroupReady {
				st.ConfirmedGroups[g.Group] = true
				changed = true
			}
		}
		for stage := range plan.Stages {
			all := true
			for _, g := range plan.StageGroups(stage) {
				if plan.Groups[g].Kind == domain.GroupPool && !st.ConfirmedGroups[g] {
					all = false
				}
			}
			if all && !st.ConfirmedStages[stage] {
				st.ConfirmedStages[stage] = true
				changed = true
			}
		}
		if !changed {
			return res
		}
	}
}

func finalRanks(res *format.Resolution) []int {
	var out []int
	for _, f := range res.Final {
		out = append(out, f.Rank)
	}
	return out
}

func TestResolve435Flow(t *testing.T) {
	plan := plan435(t)
	st := newState(16)
	// プール A〜D は良いシードが全勝(枠の順に 1〜4 位)
	for _, m := range plan.Matches[:24] {
		st.Results[m.Code] = score(3, 1)
	}
	res := resolve(t, plan, st)
	for g := range 4 {
		assert.Equal(t, format.GroupReady, res.Groups[g].State)
	}
	// 確定までは次の段の枠は埋まらない
	e1 := res.Slots[format.SlotKey{Group: 4, Position: 1}]
	assert.Equal(t, format.TeamStatus{Pending: format.PendingGroupConfirmation, Waiting: "A"}, e1)

	for g := range 4 {
		st.ConfirmedGroups[g] = true
	}
	res = resolve(t, plan, st)
	// A = シード 1,8,9,16、C = 3,6,11,14: E4 は A1(1) 対 C2(6)
	assert.Equal(t, int64(1), res.Matches["E4"].Left.TeamID)
	assert.Equal(t, int64(6), res.Matches["E4"].Right.TeamID)
	assert.Equal(t, int64(16), res.Slots[format.SlotKey{Group: 5, Position: 8}].TeamID, "A4")
	assert.Equal(t, format.PendingMatch, res.Matches["E2"].Left.Pending)

	// トーナメントの中は試合の終了で自動に進む
	st.Results["E4"] = score(1, 2)
	res = resolve(t, plan, st)
	assert.Equal(t, int64(6), res.Matches["E2"].Left.TeamID)
	assert.Equal(t, int64(1), res.Matches["E10"].Left.TeamID)
	assert.Equal(t, format.GroupPlaying, res.Groups[4].State)

	// 同点の試合は勝者の指定が要る
	st.Results["E5"] = score(4, 4)
	res = resolve(t, plan, st)
	assert.Equal(t, format.TeamStatus{Pending: format.PendingWinner, Waiting: "E5"}, res.Matches["E2"].Right)
	right := domain.SideRight
	st.Results["E5"] = format.Result{Status: domain.MatchFinished, LeftScore: intp(4), RightScore: intp(4),
		Winner: &right}
	res = resolve(t, plan, st)
	assert.Equal(t, int64(7), res.Matches["E2"].Right.TeamID, "D1=4 と B2=7 の B2 がサドンデスで勝ち")

	// 未消化は行き先を埋めない
	st.Results["E6"] = format.Result{Type: format.ResultUnplayed}
	res = resolve(t, plan, st)
	assert.Equal(t, format.PendingUnplayed, res.Matches["E3"].Left.Pending)
}

func TestResolveTieNeedsDecision(t *testing.T) {
	plan := plan916(t)
	st := newState(12)
	// プール A(シード 1, 8, 9)はじゃんけん状態で、スコアも同じ
	st.Results["A1"] = score(5, 4) // 1-8
	st.Results["A3"] = score(5, 4) // 8-9
	st.Results["A2"] = score(4, 5) // 1-9 は 9 の勝ち
	res := resolve(t, plan, st)
	assert.Equal(t, format.GroupTied, res.Groups[0].State)
	st.ConfirmedGroups[0] = true
	res = resolve(t, plan, st)
	assert.Equal(t, format.TeamStatus{Pending: format.PendingTie, Waiting: "A"},
		res.Slots[format.SlotKey{Group: 4, Position: 1}])

	basis := res.Groups[0].Ranking.UnresolvedBasis[0]
	st.RankDecisions = map[int][]format.Decision{0: {{Order: []int64{9, 1, 8}, Basis: basis}}}
	res = resolve(t, plan, st)
	assert.Equal(t, int64(9), res.Slots[format.SlotKey{Group: 4, Position: 1}].TeamID)
	assert.Equal(t, int64(1), res.Slots[format.SlotKey{Group: 5, Position: 1}].TeamID)
	assert.Equal(t, int64(8), res.Slots[format.SlotKey{Group: 6, Position: 1}].TeamID)

	// 訂正で値が変わる(並んだままでも)と、指定は使わずに未決へ戻る
	st.Results["A1"], st.Results["A3"], st.Results["A2"] = score(6, 5), score(6, 5), score(5, 6)
	res = resolve(t, plan, st)
	assert.Equal(t, format.PendingTie, res.Slots[format.SlotKey{Group: 4, Position: 1}].Pending)
}

// TestResolve525FinalRanks はブロックから順位決定まで全試合を進め、最終順位が参照サイトの 949 の
// 順位列(1,2,3,4,5,5,7,7,…,31,31。clone-spec C 1)と同じになることを確かめる。
func TestResolve525FinalRanks(t *testing.T) {
	plan := plan525(t)
	res := playAll(t, plan, newState(32))
	want := []int{1, 2, 3, 4}
	for r := 5; r <= 31; r += 2 {
		want = append(want, r, r)
	}
	assert.Equal(t, want, finalRanks(res))
	for _, g := range res.Groups {
		assert.Equal(t, format.GroupFinished, g.State, plan.Groups[g.Group].Name)
	}
	// 1st T4 は A1(1)-B1(2)、C1(3)-D1(4) で左が勝つので、1 位 1、2 位 3、3 位 2、4 位 4
	var top []int64
	for _, f := range res.Final[:4] {
		top = append(top, f.TeamID)
	}
	assert.Equal(t, []int64{1, 3, 2, 4}, top)
	// ブロック A の 2 位(シード 8)と B の 2 位(シード 7)が 5th-9th の F1 で当たる
	assert.Equal(t, []int64{8, 7}, []int64{res.Matches["F1"].Left.TeamID, res.Matches["F1"].Right.TeamID})
}

func TestResolve916FinalRanks(t *testing.T) {
	plan := plan916(t)
	res := playAll(t, plan, newState(12))
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, finalRanks(res))
	var teams []int64
	for _, f := range res.Final {
		teams = append(teams, f.TeamID)
	}
	// 1st T4: A1=1, B1=2, C1=3, D1=4 → E2 は 1-2、E3 は 3-4 で左が勝つ → 1 位 1、2 位 3、3 位 2
	assert.Equal(t, []int64{1, 3, 2, 4}, teams[:4])
}

func TestResolveBandRank(t *testing.T) {
	plan := generate(t, format.TemplatePoolBands, format.PoolBandsParams{Teams: 14, Pools: 4, Bands: []format.Band{
		{Ranks: []int{1, 2}, Shape: format.ShapeT8},
	}})
	st := newState(14)
	for _, m := range plan.Matches {
		if plan.Groups[m.Group].Kind == domain.GroupPool {
			st.Results[m.Code] = score(3, 1)
		}
	}
	for g := range 4 {
		st.ConfirmedGroups[g] = true
	}
	res := resolve(t, plan, st)
	assert.Equal(t, format.PendingStageConfirmation, res.Slots[format.SlotKey{Group: 4, Position: 1}].Pending)
	st.ConfirmedStages[0] = true
	res = resolve(t, plan, st)
	// 1 位は全勝: 3 チームのプール(A、B)は 2 試合で勝ち点 6、4 チームのプール(C、D)は 3 試合で 9。
	// 1 試合あたりは同じ 3 で、得失点差も 1 試合あたり 2 で同じなので、全体順位の 1 番目は未決
	assert.Equal(t, format.PendingTie, res.Slots[format.SlotKey{Group: 4, Position: 1}].Pending)
	var entries []format.BandEntry
	for g := range 4 {
		team, ok := res.Groups[g].Ranking.TeamAt(1)
		require.True(t, ok)
		row, _ := res.Groups[g].Ranking.RankOf(team)
		entries = append(entries, format.BandEntry{TeamID: team, Group: g, Stats: row.Stats})
	}
	band := format.RankBand(entries, format.BandRule(), nil)
	require.Len(t, band.UnresolvedBasis, 1)
	st.BandDecisions = map[format.BandKey][]format.Decision{{Stage: 0, Rank: 1}: {
		{Order: []int64{4, 3, 2, 1}, Basis: band.UnresolvedBasis[0]}}}
	res = resolve(t, plan, st)
	assert.Equal(t, int64(4), res.Slots[format.SlotKey{Group: 4, Position: 1}].TeamID)
}

func TestResolveRejectsUnknownCode(t *testing.T) {
	plan := plan916(t)
	st := newState(12)
	st.Results["Z9"] = score(1, 0)
	_, err := format.Resolve(plan, format.DefaultConfig(), st)
	assert.ErrorIs(t, err, domain.ErrBadRequest)
}

// 結果を記録したときの対戦チーム(Played)が計算上の対戦チームと食い違う試合は、勝者と敗者を下流へ
// 流さず、最終順位にも使わない。残る最終順位は、食い違いの前と同じものだけになる。
func TestResolveMismatchWithholdsDownstream(t *testing.T) {
	plan := plan435(t)
	st := newState(16)
	res := playAll(t, plan, st)
	before := map[int64]int{}
	for _, f := range res.Final {
		before[f.TeamID] = f.Rank
	}
	require.Len(t, before, 16)
	for code, result := range st.Results {
		mt := res.Matches[code]
		result.Played = &format.PlayedTeams{Left: mt.Left.TeamID, Right: mt.Right.TeamID}
		st.Results[code] = result
	}
	assert.Equal(t, res.Final, resolve(t, plan, st).Final, "記録と計算が合っていれば同じ")

	// プール A の 1 位と 2 位が入れ替わる訂正(A1 はシード 1 対 8)
	st.Results["A1"] = format.Result{Status: domain.MatchFinished, LeftScore: intp(1), RightScore: intp(3),
		Played: &format.PlayedTeams{Left: 1, Right: 8}}
	res = resolve(t, plan, st)
	e4 := res.Matches["E4"]
	assert.True(t, e4.Mismatch, "E4 は 1 対 6 で記録したが、計算上は 8 対 6")
	assert.Zero(t, e4.Winner)
	assert.Equal(t, format.PendingMismatch, res.Matches["E2"].Left.Pending)
	assert.Contains(t, res.Groups[4].Mismatches, "E4")
	assert.False(t, res.Groups[4].Done)
	assert.Less(t, len(res.Final), 16)
	for _, f := range res.Final {
		assert.Equal(t, before[f.TeamID], f.Rank, "残る最終順位は食い違いの前と同じ(チーム %d)", f.TeamID)
	}
}
