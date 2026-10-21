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
	assert.Equal(t, "各プールの 1 位(A1、B1、C1、D1)は 1〜4 位を決めるトーナメント(1st placement tournament)"+
		"に進みます。1 回戦は A1 対 B1、C1 対 D1。", lines[0])
	assert.Contains(t, lines[2], "9〜12 位")
}
