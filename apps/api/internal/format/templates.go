package format

import (
	"fmt"
	"slices"

	"scoreview/api/internal/domain"
)

// テンプレート名。division_formats に保存する値になる。
const (
	TemplatePoolBands  = "pool_bands"
	TemplateBlocks     = "blocks"
	TemplateRoundRobin = "round_robin"
)

// blockSize は「ノックアウトブロック」テンプレートの 1 ブロックのチーム数(R3 1-2、T8 ブロック型)。
const blockSize = 8

// Shape は順位帯の形。トーナメントの型か総当たり。
type Shape string

// Shape の値。
const (
	ShapePool Shape = "pool"
	ShapeT2   Shape = Shape(BracketT2)
	ShapeT4   Shape = Shape(BracketT4)
	ShapeT7   Shape = Shape(BracketT7)
	ShapeT8   Shape = Shape(BracketT8)
	ShapeT8R1 Shape = Shape(BracketT8R1)
	ShapeT12  Shape = Shape(BracketT12)
	ShapeT16  Shape = Shape(BracketT16)
)

// Pairing は順位帯の中の組み合わせの決め方。
type Pairing string

// Pairing の値。空は形に応じた既定(Band のコメント)。
const (
	// PairingCross はたすき掛け。2 つの順位を集めた帯で、上位の順位を標準のシード位置に置き、
	// 下位の順位を反対の山から当てる(435: A1-C2, D1-B2, B1-D2, C1-A2)。同じプールは決勝まで当たらない
	PairingCross Pairing = "cross"
	// PairingSameRank は同順位どうし。順位ごとにプール順(A_k, B_k, C_k, …)に並べ、隣どうしを当てる
	// (490 の A_k 対 B_k、525 の A2-B2、C2-D2)
	PairingSameRank Pairing = "same_rank"
	// PairingSeeded は帯の並び(順位、プール順。band_rank なら全体順位)をシードとして標準の位置に置く
	PairingSeeded Pairing = "seeded"
)

// Band は順位帯。Ranks のプール(ブロック)順位を集めて 1 つの順位決定グループにする。
// Pairing が空なら、1 回戦のみの型と同順位 1 つの帯は同順位どうし、2 つの順位を 2 のべきの数の
// プールから集める帯はたすき掛け、それ以外(T7、band_rank を使うとき)はシード順。
type Band struct {
	Ranks   []int
	Shape   Shape
	Pairing Pairing
	// Name はグループ名。空なら "5th placement tournament" のように作る
	Name string
}

// PoolBandsParams はテンプレート 1「プール → 順位帯」のパラメータ。
type PoolBandsParams struct {
	Teams int
	Pools int
	Bands []Band
}

// PoolBandsTemplate はテンプレート 1「プール → 順位帯の順位決定」(R3 1-1 の A1〜A6、71.4%)。
// シードを蛇行で Pools 個のプールへ配り、帯ごとに順位決定グループを作る。プールの大きさが揃って
// いれば group_rank、揃っていなければ段の全体順位 band_rank を出どころにする。
type PoolBandsTemplate struct{}

// Name はテンプレート名。
func (PoolBandsTemplate) Name() string { return TemplatePoolBands }

// Build は PoolBandsParams か *PoolBandsParams を受け取る。
func (PoolBandsTemplate) Build(b *Builder, params any) error {
	p, err := paramsAs[PoolBandsParams](params)
	if err != nil {
		return err
	}
	pools, err := addQualifyingPools(b, p.Teams, p.Pools, false)
	if err != nil {
		return err
	}
	if len(p.Bands) == 0 {
		return nil
	}
	stage := b.AddStage("順位決定")
	return addBands(b, stage, pools, p.Bands, defaultPoolPairing)
}

// BlocksParams はテンプレート 2「ノックアウトブロック → 順位ごとの集約」のパラメータ。
// Teams は 8 の倍数で、Teams/8 個のブロックを作る。Bands の Ranks はブロック内の順位(1〜8)。
type BlocksParams struct {
	Teams int
	Bands []Band
}

// BlocksTemplate はテンプレート 2(R3 1-1 の B、15.1%)。8 チームのトーナメント(T8 ブロック型)を
// 並べ、各ブロックの同じ順位を次の段で集める(525: 1 位 → T4、2・3 位 → T8 1 回戦のみ、…)。
type BlocksTemplate struct{}

// Name はテンプレート名。
func (BlocksTemplate) Name() string { return TemplateBlocks }

// Build は BlocksParams か *BlocksParams を受け取る。
func (BlocksTemplate) Build(b *Builder, params any) error {
	p, err := paramsAs[BlocksParams](params)
	if err != nil {
		return err
	}
	if p.Teams < 2*blockSize || p.Teams%blockSize != 0 {
		return domain.Invalid("teams", fmt.Sprintf("must be a multiple of %d and at least %d, got %d",
			blockSize, 2*blockSize, p.Teams))
	}
	b.SetTeams(p.Teams)
	stage := b.AddStage("ブロック")
	def, err := bracketDefFor(BracketT8Block)
	if err != nil {
		return err
	}
	var blocks []int
	for _, seeds := range snakeSeeds(p.Teams, p.Teams/blockSize) {
		sources := make([]Source, def.size)
		for i, seed := range def.slotSeeds {
			sources[i] = SeedSource(seeds[seed-1])
		}
		g, err := b.AddGroup(GroupSpec{
			Stage: stage, Name: "Tournament {L}", Bracket: BracketT8Block, Sources: sources,
		})
		if err != nil {
			return err
		}
		blocks = append(blocks, g)
	}
	if len(p.Bands) == 0 {
		return nil
	}
	next := b.AddStage("順位決定")
	// 参照サイトのブロック集約は同順位どうしを当てる(525 の 5th–9th は 5th - 7th の組に各ブロックの 2 位)
	return addBands(b, next, blocks, p.Bands, func(bandInfo) Pairing { return PairingSameRank })
}

// RoundRobinParams はテンプレート 3「総当たり 1 段」のパラメータ。Pools が 1 なら全チームの
// 総当たり 1 つ、2 なら 2 プールと、FinalPoolRanks の順位を集めた決勝プール(既定は 1・2 位)。
type RoundRobinParams struct {
	Teams          int
	Pools          int
	FinalPoolRanks []int
}

// RoundRobinTemplate はテンプレート 3(R3 1-1 の C〜E、11.4%)。順位表がそのまま最終順位になる。
type RoundRobinTemplate struct{}

// Name はテンプレート名。
func (RoundRobinTemplate) Name() string { return TemplateRoundRobin }

// Build は RoundRobinParams か *RoundRobinParams を受け取る。
func (RoundRobinTemplate) Build(b *Builder, params any) error {
	p, err := paramsAs[RoundRobinParams](params)
	if err != nil {
		return err
	}
	switch p.Pools {
	case 1:
		_, err := addQualifyingPools(b, p.Teams, 1, true)
		return err
	case 2:
		ranks := p.FinalPoolRanks
		if len(ranks) == 0 {
			ranks = []int{1, 2}
		}
		pools, err := addQualifyingPools(b, p.Teams, 2, false)
		if err != nil {
			return err
		}
		stage := b.AddStage("決勝")
		band := Band{Ranks: ranks, Shape: ShapePool, Name: "Championship pool"}
		return addBands(b, stage, pools, []Band{band}, defaultPoolPairing)
	}
	return domain.Invalid("pools", fmt.Sprintf("round robin supports 1 or 2 pools, got %d", p.Pools))
}

func paramsAs[T any](params any) (T, error) {
	switch p := params.(type) {
	case T:
		return p, nil
	case *T:
		if p != nil {
			return *p, nil
		}
	}
	var zero T
	return zero, domain.Invalid("params", fmt.Sprintf("want %T, got %T", zero, params))
}

// addQualifyingPools はシードを蛇行で配ったプールの段を足す。final はプールの順位がそのまま
// 最終順位になる(1 プールの総当たり)かどうか。
func addQualifyingPools(b *Builder, teams, pools int, final bool) ([]int, error) {
	if pools < 1 || teams < 2*pools {
		return nil, domain.Invalid("pools",
			fmt.Sprintf("%d teams cannot fill %d pools of 2 or more", teams, pools))
	}
	b.SetTeams(teams)
	stage := b.AddStage("予選")
	var groups []int
	for _, seeds := range snakeSeeds(teams, pools) {
		sources := make([]Source, len(seeds))
		for i, seed := range seeds {
			sources[i] = SeedSource(seed)
		}
		g, err := b.AddGroup(GroupSpec{
			Stage: stage, Name: "Qualifying pool {L}", Sources: sources, Final: final,
		})
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// bandInfo は帯の組み合わせの既定を決めるための情報。
type bandInfo struct {
	band   Band
	def    *bracketDef // プールなら nil
	pools  int
	ranks  int
	even   bool
	powOf2 bool
}

func defaultPoolPairing(info bandInfo) Pairing {
	switch {
	case info.def == nil || !info.even || !info.def.firstRoundPairs:
		return PairingSeeded
	case info.def.placeBlock > 0 || info.ranks == 1:
		return PairingSameRank
	case info.ranks == 2 && info.powOf2 && info.def.size == 2*info.pools:
		return PairingCross
	}
	return PairingSameRank
}

// addBands は前の段のグループ(prev、すべて同じ段)の順位を集めて順位帯のグループを足す。
func addBands(b *Builder, stage int, prev []int, bands []Band, defaultPairing func(bandInfo) Pairing) error {
	plan := b.Plan()
	sizes := make([]int, len(prev))
	for i, g := range prev {
		sizes[i] = plan.Groups[g].Size()
	}
	even := slices.Min(sizes) == slices.Max(sizes)
	prevStage := plan.Groups[prev[0]].Stage
	countAt := func(rank int) int {
		count := 0
		for _, size := range sizes {
			if size >= rank {
				count++
			}
		}
		return count
	}
	used := map[int]bool{}
	for _, band := range bands {
		if len(band.Ranks) == 0 {
			return domain.Invalid("bands", "band has no ranks")
		}
		var entrants []Source
		for _, rank := range band.Ranks {
			if rank < 1 || rank > slices.Max(sizes) || used[rank] {
				return domain.Invalid("bands", fmt.Sprintf("rank %d is out of range or used twice", rank))
			}
			used[rank] = true
			for k := range countAt(rank) {
				if even {
					entrants = append(entrants, GroupRankSource(prev[k], rank))
				} else {
					entrants = append(entrants, BandRankSource(prevStage, rank, k+1))
				}
			}
		}
		offset := 0
		for rank := 1; rank < slices.Min(band.Ranks); rank++ {
			offset += countAt(rank)
		}
		info := bandInfo{band: band, pools: len(prev), ranks: len(band.Ranks), even: even,
			powOf2: len(prev) >= 2 && len(prev)&(len(prev)-1) == 0}
		spec := GroupSpec{Stage: stage, Final: true, RankOffset: offset, Name: band.Name}
		if band.Shape == ShapePool {
			spec.Sources = entrants
			if spec.Name == "" {
				spec.Name = ordinal(offset+1) + " placement pool"
			}
		} else {
			def, err := bracketDefFor(BracketType(band.Shape))
			if err != nil {
				return err
			}
			info.def = &def
			pairing := band.Pairing
			if pairing == "" {
				pairing = defaultPairing(info)
			}
			sources, err := pairEntrants(info, def, pairing, entrants, prev)
			if err != nil {
				return err
			}
			spec.Sources = sources
			spec.Bracket = def.kind
			if spec.Name == "" {
				spec.Name = placementName(def, offset)
			}
		}
		if _, err := b.AddGroup(spec); err != nil {
			return err
		}
	}
	return nil
}

// placementName は順位決定トーナメントの名前。1 回戦のみの型は順位の組の範囲を出す
// (参照サイトの "5th-9th placement tournament"、"21st-29th placement tournament")。
func placementName(def bracketDef, offset int) string {
	if def.placeBlock > 0 && len(def.matches) > def.placeBlock {
		last := def.matches[len(def.matches)-1].winnerPlace
		return fmt.Sprintf("%s-%s placement tournament", ordinal(offset+1), ordinal(offset+last))
	}
	return ordinal(offset+1) + " placement tournament"
}

// pairEntrants は帯の参加者(順位ごとにプール順)を型の枠の順に並べる。
func pairEntrants(
	info bandInfo, def bracketDef, pairing Pairing, entrants []Source, prev []int,
) ([]Source, error) {
	if len(entrants) != def.size {
		return nil, domain.Invalid("bands", fmt.Sprintf("band %v has %d teams but %s needs %d",
			info.band.Ranks, len(entrants), def.kind, def.size))
	}
	switch pairing {
	case PairingSeeded:
		out := make([]Source, def.size)
		for i, seed := range def.slotSeeds {
			out[i] = entrants[seed-1]
		}
		return out, nil
	case PairingSameRank:
		if !info.even || !def.firstRoundPairs {
			return nil, domain.Invalid("bands",
				"same_rank pairing needs equal pools and a first round of pairs")
		}
		return entrants, nil
	case PairingCross:
		if !info.even || !def.firstRoundPairs || info.ranks != 2 || !info.powOf2 || def.size != 2*len(prev) {
			return nil, domain.Invalid("bands",
				"cross pairing needs 2 ranks from 2^n equal pools filling a bracket of twice that size")
		}
		return crossPairs(info.band.Ranks, prev), nil
	}
	return nil, domain.Invalid("bands", fmt.Sprintf("unknown pairing %q", pairing))
}

// crossPairs はたすき掛けの枠の並び。上位の順位のプールを標準のシード位置(4 プールなら A, D, B, C)に
// 置き、上の山の相手は下の山のプールを逆順に、下の山の相手は上の山のプールを逆順に取る。
// 4 プールで A1-C2, D1-B2, B1-D2, C1-A2(435 で確認)、2 プールで A1-B2, B1-A2(JFDA の Class 1)。
func crossPairs(ranks []int, pools []int) []Source {
	high, low := min(ranks[0], ranks[1]), max(ranks[0], ranks[1])
	order := standardSeedOrder(len(pools))
	half := len(order) / 2
	top, bottom := order[:half], order[half:]
	var out []Source
	for i, p := range order {
		other := top
		j := i
		if i < half {
			other = bottom
		} else {
			j = i - half
		}
		opp := other[len(other)-1-j]
		out = append(out, GroupRankSource(pools[p-1], high), GroupRankSource(pools[opp-1], low))
	}
	return out
}
