package format_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

func TestParamsJSONRoundTrip(t *testing.T) {
	cases := []struct {
		template string
		params   any
	}{
		{format.TemplatePoolBands, format.PoolBandsParams{Teams: 12, Pools: 4, Bands: []format.Band{
			{Ranks: []int{1}, Shape: format.ShapeT4},
			{Ranks: []int{2, 3}, Shape: format.ShapeT8, Pairing: format.PairingSeeded, Name: "Middle"},
		}}},
		{format.TemplateBlocks, format.BlocksParams{Teams: 16, Bands: []format.Band{
			{Ranks: []int{1}, Shape: format.ShapeT2},
		}}},
		{format.TemplateRoundRobin, format.RoundRobinParams{Teams: 8, Pools: 2, FinalPoolRanks: []int{1, 2}}},
	}
	for _, tc := range cases {
		data, err := format.EncodeParams(tc.params)
		require.NoError(t, err)
		got, err := format.DecodeParams(tc.template, data)
		require.NoError(t, err)
		assert.Equal(t, tc.params, got, string(data))
		assert.Positive(t, format.TeamsOf(got))
	}
}

func TestDecodeParamsRejectsUnknownKeys(t *testing.T) {
	_, err := format.DecodeParams(format.TemplatePoolBands, []byte(`{"teams":12,"pool":4}`))
	require.ErrorIs(t, err, domain.ErrBadRequest)
	_, err = format.DecodeParams(format.TemplateRoundRobin, []byte(`{"teams":8,"pools":1,"bands":[]}`))
	require.NoError(t, err, "空の bands は無いのと同じ")
	_, err = format.DecodeParams("x", []byte(`{"teams":8}`))
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

func TestPropose4on4TwelveTeams(t *testing.T) {
	start := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	got, err := format.Propose(format.ProposalInput{Teams: 12, Courts: 4, SlotMinutes: 20, Start: start,
		Sport: format.SportFour})
	require.NoError(t, err)
	require.NotEmpty(t, got)
	first := got[0]
	assert.True(t, first.Recommended)
	assert.Equal(t, "4 プール(各 3 チーム) → 4 チームのトーナメント × 3(1〜4 位、5〜8 位、9〜12 位)", first.Title)
	assert.Equal(t, 24, first.Matches)
	assert.Equal(t, 12, first.Ranked)
	assert.Equal(t, 12, first.Unique)
	assert.True(t, first.End.After(start))
	ids := map[string]bool{}
	for _, p := range got {
		assert.False(t, ids[p.ID], "候補の ID が重複: %s", p.ID)
		ids[p.ID] = true
		assert.NotEmpty(t, p.Title)
	}
	assert.True(t, ids["rr2"], "2 プール + 決勝プールも候補に出る")
}

// 95 チーム(8 の倍数でない)でも、参照サイトの GAIA と同じブロックの形を候補に出す。
func TestProposeBlocksWithCutsForAnUnevenCount(t *testing.T) {
	start := time.Date(2026, 11, 21, 8, 30, 0, 0, time.UTC)
	got, err := format.Propose(format.ProposalInput{Teams: 95, Courts: 16, SlotMinutes: 50, Start: start,
		Sport: format.SportSeven})
	require.NoError(t, err)
	var found *format.Proposal
	for i := range got {
		if got[i].ID == "blocks12-cuts" {
			found = &got[i]
		}
	}
	require.NotNil(t, found, "the block format with cuts is proposed")
	params, ok := found.Spec.Params.(format.BlocksParams)
	require.True(t, ok)
	assert.Equal(t, []int{16, 8, 8, 8, 8, 8, 8, 8, 8, 8, 7}, params.Cuts)
	assert.Equal(t, 95, found.Ranked, "every team gets a final place")

	// 大きさを JSON にして読み直しても同じ
	data, err := format.EncodeParams(params)
	require.NoError(t, err)
	decoded, err := format.DecodeParams(format.TemplateBlocks, data)
	require.NoError(t, err)
	again, ok := decoded.(format.BlocksParams)
	require.True(t, ok)
	assert.Equal(t, params.Teams, again.Teams)
	assert.Equal(t, params.Cuts, again.Cuts)

	// 残りが 1 チームなら、最後の 8 と合わせて 4 と 5 に分ける(49 は 7 チームのブロック 7 個)
	for teams, cuts := range map[int][]int{24: {16, 8}, 49: {16, 8, 8, 8, 4, 5}, 96: {16, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8}} {
		got, err := format.Propose(format.ProposalInput{Teams: teams, Courts: 8, SlotMinutes: 50, Start: start,
			Sport: format.SportSeven})
		require.NoError(t, err)
		var spec *format.BlocksParams
		for _, p := range got {
			if params, ok := p.Spec.Params.(format.BlocksParams); ok && len(params.Cuts) > 0 {
				spec = &params
			}
		}
		require.NotNil(t, spec, "teams %d", teams)
		assert.Equal(t, cuts, spec.Cuts, "teams %d", teams)
	}
	_, err = format.DecodeParams(format.TemplatePoolBands, []byte(`{"teams":12,"pools":4,"cuts":[8,4]}`))
	require.ErrorIs(t, err, domain.ErrBadRequest, "cuts are only for blocks")
}

func TestProposeRejectsBadInput(t *testing.T) {
	_, err := format.Propose(format.ProposalInput{Teams: 12, Courts: 0, SlotMinutes: 20, Sport: format.SportFour})
	require.ErrorIs(t, err, domain.ErrBadRequest)
	// 巨大な値で計算とメモリを使い切らせない(セキュリティ点検 L-3)
	for _, in := range []format.ProposalInput{
		{Teams: 12, Courts: 1_000_000_000, SlotMinutes: 20, Sport: format.SportFour},
		{Teams: 1_000_000_000, Courts: 2, SlotMinutes: 20, Sport: format.SportFour},
		{Teams: 12, Courts: 2, SlotMinutes: 1_000_000_000, Sport: format.SportFour},
	} {
		_, err = format.Propose(in)
		assert.ErrorIs(t, err, domain.ErrBadRequest, "%+v", in)
	}
	_, err = format.Propose(format.ProposalInput{Teams: 12, Courts: 2, SlotMinutes: 20, Sport: "golf"})
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

func TestProposeUnevenPoolsMergesSingleRank(t *testing.T) {
	got, err := format.Propose(format.ProposalInput{Teams: 13, Courts: 4, SlotMinutes: 20,
		Start: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC), Sport: format.SportSeven})
	require.NoError(t, err)
	var found bool
	for _, p := range got {
		if p.ID != "pb4" {
			continue
		}
		found = true
		params, ok := p.Spec.Params.(format.PoolBandsParams)
		require.True(t, ok)
		last := params.Bands[len(params.Bands)-1]
		assert.Equal(t, []int{3, 4}, last.Ranks, "4 位が 1 チームだけなので 3 位の帯にまとめる")
		assert.Equal(t, format.ShapePool, last.Shape)
	}
	assert.True(t, found)
}

func TestDescribeRuleAndAdvancement(t *testing.T) {
	rule, err := format.RuleFor(format.TiebreakCurrent)
	require.NoError(t, err)
	text := strings.Join(format.DescribeRule(rule), "")
	assert.Contains(t, text, "勝ち点(勝ち 3、引き分け 1、負け 0)")
	assert.Contains(t, text, "(2 チームが並んだときだけ)")
	jfda, err := format.RuleFor(format.TiebreakJFDA)
	require.NoError(t, err)
	assert.Contains(t, strings.Join(format.DescribeRule(jfda), ""), "最初の基準から比べ直します")
	_, err = format.RuleFor("x")
	require.ErrorIs(t, err, domain.ErrBadRequest)

	lines := format.DescribeAdvancement(plan916(t))
	require.Len(t, lines, 4)
	assert.Equal(t, "各組の 1 位(A組1位、B組1位、C組1位、D組1位)は 1〜4 位を決めるトーナメント"+
		"に進みます。1 回戦は A組1位 対 B組1位、C組1位 対 D組1位。", lines[0])
	assert.Contains(t, lines[2], "9〜12 位")
	// 案内文はそのまま参加チームに渡すので、エンジンの英語の名前を出さない
	assert.Equal(t, "組の順位は、本部が組の結果を確定してから次の組み合わせに反映します。", lines[3])
	for _, line := range lines {
		assert.NotContains(t, line, "placement")
	}
}

// 運営が決めた組分け(組ごとのシード番号)を、そのまま予選の組にする。大きさが揃わなくてもよい。
func TestPoolBandsWithExplicitPools(t *testing.T) {
	poolSeeds := [][]int{{1, 9, 5, 10}, {2, 6, 7}, {3, 4, 8}}
	plan, err := format.NewEngine().Generate(format.Spec{Template: format.TemplatePoolBands,
		Params: format.PoolBandsParams{Teams: 10, Pools: 3, PoolSeeds: poolSeeds}})
	require.NoError(t, err)
	for g, want := range poolSeeds {
		var got []int
		for _, slot := range plan.Groups[g].Slots {
			got = append(got, slot.Source.Seed)
		}
		assert.Equal(t, want, got, "pool %d", g)
	}

	for name, bad := range map[string][][]int{
		"組の数が違う":  {{1, 2, 3, 4, 5}, {6, 7, 8, 9, 10}},
		"1 チームの組": {{1}, {2, 3, 4, 5}, {6, 7, 8, 9, 10}},
		"同じシード":   {{1, 1, 3}, {4, 5, 6}, {7, 8, 9, 10}},
		"抜けたチーム":  {{1, 2}, {3, 4}, {5, 6}},
		"範囲の外":    {{1, 2, 3}, {4, 5, 6}, {7, 8, 9, 11}},
	} {
		_, err := format.NewEngine().Generate(format.Spec{Template: format.TemplatePoolBands,
			Params: format.PoolBandsParams{Teams: 10, Pools: 3, PoolSeeds: bad}})
		require.ErrorIs(t, err, domain.ErrBadRequest, name)
	}

	data, err := format.EncodeParams(format.PoolBandsParams{Teams: 10, Pools: 3, PoolSeeds: poolSeeds})
	require.NoError(t, err)
	decoded, err := format.DecodeParams(format.TemplatePoolBands, data)
	require.NoError(t, err)
	assert.Equal(t, poolSeeds, decoded.(format.PoolBandsParams).PoolSeeds) //nolint:forcetypeassert // DecodeParams は pool_bands なら PoolBandsParams
	_, err = format.DecodeParams(format.TemplateBlocks, []byte(`{"teams":16,"poolSeeds":[[1,2]]}`))
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

// 組分けが決まっていれば、そのあとに続く順位決定戦の候補だけを出す(組分けは変えない)。
func TestProposeFollowUpsForExplicitPools(t *testing.T) {
	poolSeeds := [][]int{{1, 8, 9, 16}, {2, 7, 10, 15}, {3, 6, 11, 14}, {4, 5, 12, 13}}
	got, err := format.Propose(format.ProposalInput{Teams: 16, Courts: 4, SlotMinutes: 50,
		Start: time.Date(2027, 2, 9, 8, 30, 0, 0, time.UTC), Sport: format.SportSeven, PoolSeeds: poolSeeds})
	require.NoError(t, err)
	ids := []string{}
	for _, p := range got {
		ids = append(ids, p.ID)
		params, ok := p.Spec.Params.(format.PoolBandsParams)
		require.True(t, ok, p.ID)
		assert.Equal(t, poolSeeds, params.PoolSeeds, "the pools stay as decided")
	}
	assert.ElementsMatch(t, []string{"fixed-rank", "fixed-pairs", "fixed-none"}, ids)
	byID := map[string]format.Proposal{}
	for _, p := range got {
		byID[p.ID] = p
	}
	assert.Equal(t, 24+16, byID["fixed-rank"].Matches, "4 pools of 4 and four T4 brackets of 4 matches")
	assert.Equal(t, 24+24, byID["fixed-pairs"].Matches, "two T8 brackets of 12 matches")
	assert.Equal(t, 24, byID["fixed-none"].Matches)

	_, err = format.Propose(format.ProposalInput{Teams: 16, Courts: 4, SlotMinutes: 50, Sport: format.SportSeven,
		PoolSeeds: [][]int{{1, 2}, {3, 4}}})
	require.ErrorIs(t, err, domain.ErrBadRequest, "the pools must place every team")
}
