package format_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

type groupShape struct {
	letter  string
	name    string
	kind    domain.GroupKind
	pattern string
	size    int
	matches int
	offset  int
	final   bool
}

func generate(t *testing.T, template string, params any) *format.Plan {
	t.Helper()
	plan, err := format.NewEngine().Generate(format.Spec{Template: template, Params: params})
	require.NoError(t, err)
	return plan
}

// plan916 は 4on4 12 チーム、4 プール(3 チーム)→ 1st / 5th / 9th の T4(R3 1-2、916 / 496)。
func plan916(t *testing.T) *format.Plan {
	return generate(t, format.TemplatePoolBands, format.PoolBandsParams{Teams: 12, Pools: 4, Bands: []format.Band{
		{Ranks: []int{1}, Shape: format.ShapeT4},
		{Ranks: []int{2}, Shape: format.ShapeT4},
		{Ranks: []int{3}, Shape: format.ShapeT4},
	}})
}

// plan435 は 16 チーム、4 プール(4 チーム)→ 1st T8(1・2 位)、9th T8(3・4 位)(R3 2-1、435 / 815)。
func plan435(t *testing.T) *format.Plan {
	return generate(t, format.TemplatePoolBands, format.PoolBandsParams{Teams: 16, Pools: 4, Bands: []format.Band{
		{Ranks: []int{1, 2}, Shape: format.ShapeT8},
		{Ranks: []int{3, 4}, Shape: format.ShapeT8},
	}})
}

// plan525 はビーチ 32 チーム、T8 ブロック × 4 → 1st T4、5th–9th T8 1 回戦のみ、13th–17th 同、
// 21st–29th T12(R3 2-1、525 / 949)。
func plan525(t *testing.T) *format.Plan {
	return generate(t, format.TemplateBlocks, format.BlocksParams{Teams: 32, Bands: []format.Band{
		{Ranks: []int{1}, Shape: format.ShapeT4},
		{Ranks: []int{2, 3}, Shape: format.ShapeT8R1},
		{Ranks: []int{4, 5}, Shape: format.ShapeT8R1},
		{Ranks: []int{6, 7, 8}, Shape: format.ShapeT12},
	}})
}

// codesOf は試合のスライスから CD を取り出す。
func codesOf(ms []format.Match) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Code
	}
	return out
}

// labels は枠の出どころの表記を並べる。
func labels(plan *format.Plan, group int) []string {
	var out []string
	for _, s := range plan.Groups[group].Slots {
		out = append(out, s.Source.Label(plan))
	}
	return out
}

func shapes(plan *format.Plan) []groupShape {
	var out []groupShape
	for _, g := range plan.Groups {
		out = append(out, groupShape{g.Letter, g.Name, g.Kind, g.Pattern, g.Size(),
			len(plan.GroupMatches(g.Index)), g.RankOffset, g.Final})
	}
	return out
}

// sideText は試合の片側を "slot 1" か "[E2-勝ち]" にする。
func sideText(plan *format.Plan, m format.Match, side format.SideRef) string {
	if side.From != nil {
		return side.From.Label()
	}
	return plan.Groups[m.Group].Slots[side.Slot-1].Source.Label(plan)
}

func matchTexts(plan *format.Plan, group int) []string {
	var out []string
	for _, m := range plan.GroupMatches(group) {
		out = append(out, fmt.Sprintf("%s r%d %s-%s w%d l%d", m.Code, m.Round,
			sideText(plan, m, m.Left), sideText(plan, m, m.Right), m.WinnerPlace, m.LoserPlace))
	}
	return out
}

// TestReference916 は 4on4 12 チームの最頻形(R3 1-2: プール戦 12 試合 + T4×3 で 12 試合)。
func TestReference916(t *testing.T) {
	plan := plan916(t)
	assert.Equal(t, []groupShape{
		{"A", "Qualifying pool A", domain.GroupPool, "", 3, 3, 0, false},
		{"B", "Qualifying pool B", domain.GroupPool, "", 3, 3, 0, false},
		{"C", "Qualifying pool C", domain.GroupPool, "", 3, 3, 0, false},
		{"D", "Qualifying pool D", domain.GroupPool, "", 3, 3, 0, false},
		{"E", "1st placement tournament", domain.GroupBracket, "T4E", 4, 4, 0, true},
		{"F", "5th placement tournament", domain.GroupBracket, "T4F", 4, 4, 4, true},
		{"G", "9th placement tournament", domain.GroupBracket, "T4G", 4, 4, 8, true},
	}, shapes(plan))
	assert.Len(t, plan.Matches, 24)
	// 蛇行のシード
	assert.Equal(t, []string{"シード1", "シード8", "シード9"}, labels(plan, 0))
	assert.Equal(t, []string{"シード4", "シード5", "シード12"}, labels(plan, 3))
	// 3 チームのプール: 連番は組み合わせ順、試合順は (1-3) → (1-2) → (2-3)
	assert.Equal(t, []string{
		"A1 r2 シード1-シード8 w0 l0", "A2 r1 シード1-シード9 w0 l0", "A3 r3 シード8-シード9 w0 l0",
	}, matchTexts(plan, 0))
	// 各帯の T4 は各プールの同じ順位(帯内の組み合わせはプール順の隣どうし。推測)
	assert.Equal(t, []string{"A1", "B1", "C1", "D1"}, labels(plan, 4))
	assert.Equal(t, []string{"A2", "B2", "C2", "D2"}, labels(plan, 5))
	assert.Equal(t, []string{"A3", "B3", "C3", "D3"}, labels(plan, 6))
	// T4: 1=決勝、2・3=準決勝、4=3 位決定
	assert.Equal(t, []string{
		"E1 r2 [E2-勝ち]-[E3-勝ち] w1 l2", "E2 r1 A1-B1 w0 l0", "E3 r1 C1-D1 w0 l0",
		"E4 r2 [E2-負け]-[E3-負け] w3 l4",
	}, matchTexts(plan, 4))
	assert.Equal(t, []string{"F1", "F2", "F3", "F4"}, codesOf(plan.GroupMatches(5)))
}

// TestReference435 は 16 チーム 4 プール → T8 ×2 のたすき掛け(R3 2-1 で確認した対応)。
func TestReference435(t *testing.T) {
	plan := plan435(t)
	assert.Equal(t, []groupShape{
		{"A", "Qualifying pool A", domain.GroupPool, "", 4, 6, 0, false},
		{"B", "Qualifying pool B", domain.GroupPool, "", 4, 6, 0, false},
		{"C", "Qualifying pool C", domain.GroupPool, "", 4, 6, 0, false},
		{"D", "Qualifying pool D", domain.GroupPool, "", 4, 6, 0, false},
		{"E", "1st placement tournament", domain.GroupBracket, "T8E", 8, 12, 0, true},
		{"F", "9th placement tournament", domain.GroupBracket, "T8F", 8, 12, 8, true},
	}, shapes(plan))
	assert.Len(t, plan.Matches, 48)
	// 4 チームのプール: (1-4, 2-3) → (1-3, 2-4) → (1-2, 3-4)、連番は組み合わせ順
	assert.Equal(t, []string{
		"A1 r3 シード1-シード8 w0 l0", "A2 r2 シード1-シード9 w0 l0", "A3 r1 シード1-シード16 w0 l0",
		"A4 r1 シード8-シード9 w0 l0", "A5 r2 シード8-シード16 w0 l0", "A6 r3 シード9-シード16 w0 l0",
	}, matchTexts(plan, 0))
	// 1 回戦 A1-C2, D1-B2, B1-D2, C1-A2。3・4 位も同じ形
	assert.Equal(t, []string{"A1", "C2", "D1", "B2", "B1", "D2", "C1", "A2"}, labels(plan, 4))
	assert.Equal(t, []string{"A3", "C4", "D3", "B4", "B3", "D4", "C3", "A4"}, labels(plan, 5))
	// T8 全順位型の番号と行き先(R3 4 章、6 章の [2勝]vs[3勝] など)
	assert.Equal(t, []string{
		"E1 r3 [E2-勝ち]-[E3-勝ち] w1 l2",
		"E2 r2 [E4-勝ち]-[E5-勝ち] w0 l0",
		"E3 r2 [E6-勝ち]-[E7-勝ち] w0 l0",
		"E4 r1 A1-C2 w0 l0",
		"E5 r1 D1-B2 w0 l0",
		"E6 r1 B1-D2 w0 l0",
		"E7 r1 C1-A2 w0 l0",
		"E8 r3 [E2-負け]-[E3-負け] w3 l4",
		"E9 r3 [E10-勝ち]-[E11-勝ち] w5 l6",
		"E10 r2 [E4-負け]-[E5-負け] w0 l0",
		"E11 r2 [E6-負け]-[E7-負け] w0 l0",
		"E12 r3 [E10-負け]-[E11-負け] w7 l8",
	}, matchTexts(plan, 4))
}

// TestReference525 はビーチ 32 チームのブロック型(R3 2-1、clone-spec C 2.3 の 949 の 8 グループ)。
func TestReference525(t *testing.T) {
	plan := plan525(t)
	assert.Equal(t, []groupShape{
		{"A", "Tournament A", domain.GroupBracket, "T8A", 8, 12, 0, false},
		{"B", "Tournament B", domain.GroupBracket, "T8B", 8, 12, 0, false},
		{"C", "Tournament C", domain.GroupBracket, "T8C", 8, 12, 0, false},
		{"D", "Tournament D", domain.GroupBracket, "T8D", 8, 12, 0, false},
		{"E", "1st placement tournament", domain.GroupBracket, "T4E", 4, 4, 0, true},
		{"F", "5th-9th placement tournament", domain.GroupBracket, "T8F", 8, 4, 4, true},
		{"G", "13th-17th placement tournament", domain.GroupBracket, "T8G", 8, 4, 12, true},
		{"H", "21st-29th placement tournament", domain.GroupBracket, "T12H", 12, 6, 20, true},
	}, shapes(plan))
	// ビーチの 1 日 66 試合(R3 5-1)
	assert.Len(t, plan.Matches, 66)
	assert.Equal(t, []string{"シード1", "シード32", "シード16", "シード17", "シード8", "シード25", "シード9", "シード24"},
		labels(plan, 0))
	assert.Equal(t, []string{"A1", "B1", "C1", "D1"}, labels(plan, 4))
	assert.Equal(t, []string{"A2", "B2", "C2", "D2", "A3", "B3", "C3", "D3"}, labels(plan, 5))
	assert.Equal(t, []string{"A4", "B4", "C4", "D4", "A5", "B5", "C5", "D5"}, labels(plan, 6))
	assert.Equal(t, []string{"A6", "B6", "C6", "D6", "A7", "B7", "C7", "D7", "A8", "B8", "C8", "D8"},
		labels(plan, 7))
	// 1 回戦のみの型: 2 試合ずつが順位の組(5th - 7th、9th - 11th)
	assert.Equal(t, []string{
		"F1 r1 A2-B2 w1 l3", "F2 r1 C2-D2 w1 l3", "F3 r1 A3-B3 w5 l7", "F4 r1 C3-D3 w5 l7",
	}, matchTexts(plan, 5))
	assert.Equal(t, []string{
		"H1 r1 A6-B6 w1 l3", "H2 r1 C6-D6 w1 l3", "H3 r1 A7-B7 w5 l7",
		"H4 r1 C7-D7 w5 l7", "H5 r1 A8-B8 w9 l11", "H6 r1 C8-D8 w9 l11",
	}, matchTexts(plan, 7))
	// ブロックの格子の Placement 列は "A rank n" と行き先(clone-spec C 2.3)
	texts := gridTexts(plan.Groups[0].Layout)
	for _, want := range []string{"A rank 1", "1st placement T →", "A rank 3", "9th placement T →",
		"A rank 6", "21st placement T →", "A rank 8", "29th placement T →"} {
		assert.Contains(t, texts, want)
	}
	assert.Contains(t, gridTexts(plan.Groups[5].Layout), "5th(tied)")
	assert.Contains(t, gridTexts(plan.Groups[5].Layout), "9th - 11th")
	assert.Contains(t, gridTexts(plan.Groups[7].Layout), "31st(tied)")
}

func gridTexts(grid *format.Grid) []string {
	var out []string
	for _, row := range grid.Layout.Rows {
		for _, cell := range row {
			if cell.Text != "" {
				out = append(out, cell.Text)
			}
		}
	}
	return out
}

func TestPoolRounds(t *testing.T) {
	cases := []struct {
		teams  int
		rounds int
		// rest は各チームの休みの回数(奇数のときだけ 1)
		rest int
	}{
		{5, 5, 1}, {6, 5, 0}, {7, 7, 1}, {8, 7, 0},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.teams), func(t *testing.T) {
			plan := generate(t, format.TemplateRoundRobin, format.RoundRobinParams{Teams: tc.teams, Pools: 1})
			ms := plan.GroupMatches(0)
			require.Len(t, ms, tc.teams*(tc.teams-1)/2)
			played := map[int]map[int]bool{}
			pairs := map[[2]int]bool{}
			lastRound := 0
			for i, m := range ms {
				assert.Equal(t, i+1, m.Number, "numbers follow the combination order")
				pairs[[2]int{m.Left.Slot, m.Right.Slot}] = true
				lastRound = max(lastRound, m.Round)
				for _, slot := range []int{m.Left.Slot, m.Right.Slot} {
					if played[slot] == nil {
						played[slot] = map[int]bool{}
					}
					assert.False(t, played[slot][m.Round], "slot %d plays twice in round %d", slot, m.Round)
					played[slot][m.Round] = true
				}
			}
			assert.Len(t, pairs, len(ms))
			assert.Equal(t, tc.rounds, lastRound)
			for slot := 1; slot <= tc.teams; slot++ {
				assert.Equal(t, tc.rest, tc.rounds-len(played[slot]), "rests of slot %d", slot)
			}
			// 上位シードどうしは最後のラウンド
			assert.Equal(t, tc.rounds, ms[0].Round)
		})
	}
}

func TestStartLetterAcrossDivisions(t *testing.T) {
	plan, err := format.NewEngine().Generate(format.Spec{Template: format.TemplateRoundRobin, StartLetter: "Y",
		Params: &format.RoundRobinParams{Teams: 8, Pools: 2}})
	require.NoError(t, err)
	var letters []string
	for _, g := range plan.Groups {
		letters = append(letters, g.Letter)
	}
	assert.Equal(t, []string{"Y", "Z", "AA"}, letters)
	assert.Equal(t, "Championship pool", plan.Groups[2].Name)
	assert.Equal(t, []string{"Y1", "Z1"}, labels(plan, 2)[:2])
	assert.Equal(t, "AA1", plan.GroupMatches(2)[0].Code)
}

// TestUnevenPoolsUseBandRank は大きさが不揃いなプールの帯が段の全体順位を出どころにすること。
func TestUnevenPoolsUseBandRank(t *testing.T) {
	plan := generate(t, format.TemplatePoolBands, format.PoolBandsParams{Teams: 14, Pools: 4, Bands: []format.Band{
		{Ranks: []int{1, 2}, Shape: format.ShapeT8},
		{Ranks: []int{3, 4}, Shape: format.ShapePool},
	}})
	var sizes []int
	for _, g := range plan.Groups[:4] {
		sizes = append(sizes, g.Size())
	}
	assert.Equal(t, []int{3, 3, 4, 4}, sizes) // 蛇行で最後の 2 チームは D、C へ
	src := plan.Groups[4].Slots[0].Source
	assert.Equal(t, format.SourceBandRank, src.Kind)
	// シード順の標準位置: 1 位の 1 番目と 2 位の 4 番目
	assert.Equal(t, format.BandRankSource(0, 1, 1), plan.Groups[4].Slots[0].Source)
	assert.Equal(t, format.BandRankSource(0, 2, 4), plan.Groups[4].Slots[1].Source)
	assert.Equal(t, 6, plan.Groups[5].Size())
	assert.Equal(t, "9th placement pool", plan.Groups[5].Name)
}

func TestT7HasBye(t *testing.T) {
	plan := generate(t, format.TemplatePoolBands, format.PoolBandsParams{Teams: 21, Pools: 7, Bands: []format.Band{
		{Ranks: []int{1}, Shape: format.ShapeT7},
	}})
	g := plan.Groups[7]
	assert.Equal(t, "T7H", g.Pattern)
	assert.Equal(t, []string{"A1", "D1", "E1", "B1", "G1", "C1", "F1"}, labels(plan, 7))
	assert.Equal(t, []string{
		"H1 r3 [H2-勝ち]-[H3-勝ち] w1 l2", "H2 r2 A1-[H4-勝ち] w0 l0", "H3 r2 [H5-勝ち]-[H6-勝ち] w0 l0",
		"H4 r1 D1-E1 w0 l0", "H5 r1 B1-G1 w0 l0", "H6 r1 C1-F1 w0 l0",
		"H7 r3 [H2-負け]-[H3-負け] w3 l4", "H8 r3 [H4-負け]-[H9-勝ち] w5 l6", "H9 r2 [H5-負け]-[H6-負け] w0 l7",
	}, matchTexts(plan, 7))
}

func TestGenerateIsDeterministic(t *testing.T) {
	assert.Equal(t, plan525(t), plan525(t))
	assert.Equal(t, plan435(t), plan435(t))
}

func TestGridsAreValidAndComplete(t *testing.T) {
	for _, plan := range []*format.Plan{plan916(t), plan435(t), plan525(t)} {
		for _, g := range plan.Groups {
			if g.Kind != domain.GroupBracket {
				assert.Nil(t, g.Layout)
				continue
			}
			require.NotNil(t, g.Layout)
			require.NoError(t, g.Layout.Layout.Validate())
			games := map[string]int{}
			for _, ref := range g.Layout.Refs {
				if ref.Game {
					games[ref.Match]++
					assert.Equal(t, domain.BracketGame, g.Layout.Layout.Rows[ref.Row][ref.Col].Kind)
				}
			}
			for _, m := range plan.GroupMatches(g.Index) {
				assert.Equal(t, 1, games[m.Code], "game cell of %s", m.Code)
			}
		}
	}
}

func TestGenerateErrors(t *testing.T) {
	cases := []struct {
		name string
		spec format.Spec
	}{
		{"unknown template", format.Spec{Template: "swiss"}},
		{"wrong params type", format.Spec{Template: format.TemplatePoolBands, Params: format.BlocksParams{}}},
		{"too few teams", format.Spec{Template: format.TemplatePoolBands,
			Params: format.PoolBandsParams{Teams: 5, Pools: 3}}},
		{"band size mismatch", format.Spec{Template: format.TemplatePoolBands, Params: format.PoolBandsParams{
			Teams: 12, Pools: 4, Bands: []format.Band{{Ranks: []int{1, 2}, Shape: format.ShapeT4}}}}},
		{"cross needs two ranks", format.Spec{Template: format.TemplatePoolBands, Params: format.PoolBandsParams{
			Teams: 12, Pools: 4, Bands: []format.Band{
				{Ranks: []int{1}, Shape: format.ShapeT4, Pairing: format.PairingCross}}}}},
		{"blocks of eight", format.Spec{Template: format.TemplateBlocks, Params: format.BlocksParams{Teams: 20}}},
		{"bad start letter", format.Spec{Template: format.TemplateRoundRobin, StartLetter: "a",
			Params: format.RoundRobinParams{Teams: 4, Pools: 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := format.NewEngine().Generate(tc.spec)
			require.Error(t, err)
			assert.True(t, errors.Is(err, domain.ErrBadRequest), "%v", err)
		})
	}
}

// customTemplate は Functional Options で足すテンプレートの例(4 チームの T4 だけ)。
type customTemplate struct{}

func (customTemplate) Name() string { return "t4_only" }

func (customTemplate) Build(b *format.Builder, _ any) error {
	b.SetTeams(4)
	stage := b.AddStage("決勝")
	_, err := b.AddGroup(format.GroupSpec{Stage: stage, Name: "Final {L}", Bracket: format.BracketT4, Final: true,
		Sources: []format.Source{format.SeedSource(1), format.SeedSource(4), format.SeedSource(2), format.SeedSource(3)}})
	return err
}

func TestWithTemplate(t *testing.T) {
	engine := format.NewEngine(format.WithTemplate(customTemplate{}))
	assert.Equal(t, []string{"blocks", "pool_bands", "round_robin", "t4_only"}, engine.Templates())
	plan, err := engine.Generate(format.Spec{Template: "t4_only", StartLetter: "M"})
	require.NoError(t, err)
	assert.Equal(t, "Final M", plan.Groups[0].Name)
	assert.Equal(t, "T4M", plan.Groups[0].Pattern)
}
