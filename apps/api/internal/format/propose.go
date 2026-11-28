package format

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"scoreview/api/internal/domain"
)

// Sport は形の提案で使う競技の区分(R3 1-3、5 章)。
type Sport string

// Sport の値。
const (
	SportSeven Sport = "seven" // 7 人制
	SportFour  Sport = "four"  // 4on4
	SportBeach Sport = "beach" // ビーチ
)

// ProposalInput は形の提案の入力。
type ProposalInput struct {
	Teams       int
	Courts      int
	SlotMinutes int
	Start       time.Time
	Sport       Sport
	// Schedule は時間割の条件(日数、各日の時間、昼休み、休み、段の日分け)。nil なら Courts、
	// SlotMinutes、Start と競技の既定の休みで終わりのない 1 日として組む
	Schedule *ScheduleConfig
	// PoolSeeds は運営が決めた組分け(組ごとのシード番号)。あれば、その組のあとに続く順位決定戦の候補
	// だけを返す(組分けは変えない)
	PoolSeeds [][]int
}

// Proposal は形の候補 1 つ。Matches と End は Spec から作った時間割(BuildSchedule)の値。
type Proposal struct {
	ID          string
	Title       string
	Spec        Spec
	Recommended bool
	// Note は候補の根拠(参照サイトの運営会社での使われ方)
	Note       string
	Matches    int
	End        time.Time
	Violations int
	// EndDay は最後の試合が何日目か(0 から)
	EndDay int
	// Overflow は会場の時間に収まらない試合の数、FitNote はその説明(収まれば空)
	Overflow int
	FitNote  string
	// Ranked は最終順位が付くチーム数、Unique はそのうち同順位でないチーム数
	Ranked int
	Unique int
}

// RestSlotsFor は競技ごとの同じチームの試合の間の休みのスロット数の既定(R3 5-3: 4on4 は連戦を
// 許し、7 人制とビーチは 1 スロット以上休む)。
func RestSlotsFor(sport Sport) int {
	if sport == SportFour {
		return 0
	}
	return 1
}

// 提案するプールの大きさの範囲。参照サイトの予選プールは 3〜6 チーム(R3 1-2)。
const (
	minPoolSize = 3
	maxPoolSize = 6
	maxPools    = 8
	// maxRoundRobin は 1 プールの総当たりを提案するチーム数の上限(R3 1-2 の 224 は 5 チーム)
	maxRoundRobin = 8
)

// 提案の入力の上限。コート数の分だけ配列を作り、チーム数の分だけ試合を作るので、上限が無いと
// 巨大な値で計算とメモリを使い切らせられる(セキュリティ点検 L-3)。どれも実際の大会を十分に超える値。
const (
	// maxProposalTeams はチーム数の上限。参照サイトの最大のディビジョンでも数十チーム
	maxProposalTeams = 128
	// maxProposalCourts はコート数の上限。大きな会場でも十数面
	maxProposalCourts = 64
	// maxProposalSlotMinutes は 1 試合の枠(分)の上限。1 日
	maxProposalSlotMinutes = 24 * 60
)

func checkProposalSize(in ProposalInput) error {
	if err := domain.CheckRange("teams", in.Teams, 2, maxProposalTeams); err != nil {
		return err
	}
	if err := domain.CheckRange("courts", in.Courts, 1, maxProposalCourts); err != nil {
		return err
	}
	return domain.CheckRange("slotMinutes", in.SlotMinutes, 1, maxProposalSlotMinutes)
}

// Propose はチーム数、コート数、競技から形の候補を返す。運営会社の頻出形(R3 8 節)を先に、
// 会場の時間に収まる候補を収まらない候補より前に並べ、収まる最初の候補を Recommended にする。
// 作れない組み合わせは候補にしない。
func Propose(in ProposalInput) ([]Proposal, error) {
	if in.Schedule != nil {
		in.Courts, in.SlotMinutes = len(in.Schedule.Courts), in.Schedule.SlotMinutes
	}
	if in.Teams < 2 || in.Courts < 1 || in.SlotMinutes <= 0 {
		return nil, domain.Invalid("proposal", "teams (2+), courts (1+) and slot minutes are required")
	}
	if err := checkProposalSize(in); err != nil {
		return nil, err
	}
	switch in.Sport {
	case SportSeven, SportFour, SportBeach:
	default:
		return nil, domain.Invalid("sport", fmt.Sprintf("unknown sport %q", in.Sport))
	}
	engine := NewEngine()
	courts := make([]string, in.Courts)
	for i := range courts {
		courts[i] = fmt.Sprint(i + 1)
	}
	cfg := ScheduleConfig{Start: in.Start, SlotMinutes: in.SlotMinutes, Courts: courts,
		TeamRestSlots: RestSlotsFor(in.Sport)}
	if in.Schedule != nil {
		cfg = *in.Schedule
	}

	var out []Proposal
	seen := map[string]bool{}
	add := func(id, note string, spec Spec) {
		if seen[id] {
			return
		}
		plan, err := engine.Generate(spec)
		if err != nil {
			return // この組み合わせでは作れない(帯の型が合わない等)
		}
		sched, err := BuildSchedule(plan, cfg)
		if err != nil {
			return
		}
		seen[id] = true
		p := Proposal{ID: id, Title: Title(plan), Spec: spec, Note: note, Matches: len(plan.Matches),
			Violations: len(sched.Violations), Overflow: sched.Overflow, FitNote: OverflowNote(cfg, sched)}
		p.End, p.EndDay = EndOf(cfg, sched)
		p.Ranked, p.Unique = FinalPlaces(plan)
		out = append(out, p)
	}

	if len(in.PoolSeeds) > 0 {
		if err := CheckPoolSeeds(in.PoolSeeds, in.Teams, len(in.PoolSeeds)); err != nil {
			return nil, err
		}
		for _, c := range followUps(in.Teams, in.PoolSeeds) {
			add(c.id, c.note, c.spec)
		}
		return sortProposals(out), nil
	}
	for _, c := range frequentShapes(in.Teams, in.Sport) {
		add(c.id, c.note, c.spec)
	}
	for pools := 1; pools <= maxPools; pools++ {
		size := in.Teams / pools
		if pools == 1 {
			if in.Teams <= maxRoundRobin {
				add("rr1", "全チームの総当たり。順位表がそのまま最終順位になります",
					Spec{Template: TemplateRoundRobin, Params: RoundRobinParams{Teams: in.Teams, Pools: 1}})
			}
			continue
		}
		if size < minPoolSize || (in.Teams+pools-1)/pools > maxPoolSize {
			continue
		}
		if bands, ok := perRankBands(in.Teams, pools); ok {
			add(fmt.Sprintf("pb%d", pools), "各プールの同じ順位を集めて順位決定戦を行う形です",
				Spec{Template: TemplatePoolBands, Params: PoolBandsParams{Teams: in.Teams, Pools: pools, Bands: bands}})
		}
		if pools == 2 {
			add("rr2", "2 プールの上位 2 チームずつで決勝プールを行います(7 人制の 446 と同じ形)",
				Spec{Template: TemplateRoundRobin, Params: RoundRobinParams{Teams: in.Teams, Pools: 2}})
		}
	}
	if in.Teams >= 3 && in.Teams <= maxKnockoutTeams {
		add("ko", "全チームで 1 つのトーナメントを組みます。上位のシードは 1 回戦を免除し、負けたチームも順位決定戦で順位を決めます",
			Spec{Template: TemplateKnockout, Params: KnockoutParams{Teams: in.Teams}})
	}
	if blocks := blockCount(in.Teams); blocks >= 3 {
		if cuts, ok := overallCuts(in.Teams); ok {
			add(fmt.Sprintf("blocks%d-cuts", blocks),
				"8 チーム(足りなければ 7 チーム)のトーナメントのブロックのあと、ブロックの順位を並べて 1〜16 位、"+
					"17〜24 位…のトーナメントに分けます(参照サイトの 2026 GAIA ULTIMATE CUP と同じ形)",
				Spec{Template: TemplateBlocks, Params: BlocksParams{Teams: in.Teams, Cuts: cuts}})
		}
	}
	if in.Teams >= 2*blockSize && in.Teams%blockSize == 0 {
		blocks := in.Teams / blockSize
		if shape, ok := shapeForCount(blocks); ok && shape != ShapePool {
			var bands []Band
			for rank := 1; rank <= blockSize; rank++ {
				bands = append(bands, Band{Ranks: []int{rank}, Shape: shape})
			}
			add(fmt.Sprintf("blocks%d", blocks), "8 チームのトーナメント(ブロック)の同じ順位を集めます",
				Spec{Template: TemplateBlocks, Params: BlocksParams{Teams: in.Teams, Bands: bands}})
		}
	}
	return sortProposals(out), nil
}

// sortProposals は収まる候補を先に並べ(元の順は保つ)、収まる候補の先頭を勧める。
func sortProposals(out []Proposal) []Proposal {
	slices.SortStableFunc(out, func(a, b Proposal) int {
		return cmp.Compare(min(a.Overflow, 1), min(b.Overflow, 1))
	})
	if len(out) > 0 && out[0].Overflow == 0 {
		out[0].Recommended = true
	}
	return out
}

// followUps は運営が決めた組分け(poolSeeds)のあとに続く順位決定戦の候補。組の同じ順位を集める形、
// 2 つの順位ずつ集める形(1・2 位、3・4 位…)、順位決定戦なし(組の順位まで)の順。作れない形は除く。
func followUps(teams int, poolSeeds [][]int) []shapeCandidate {
	sizes := make([]int, len(poolSeeds))
	for i, seeds := range poolSeeds {
		sizes[i] = len(seeds)
	}
	spec := func(bands []Band) Spec {
		return Spec{Template: TemplatePoolBands, Params: PoolBandsParams{Teams: teams, Pools: len(poolSeeds),
			Bands: bands, PoolSeeds: poolSeeds}}
	}
	var out []shapeCandidate
	if bands, ok := rankBands(sizes, 1); ok {
		out = append(out, shapeCandidate{"fixed-rank", "各組の同じ順位を集めて順位決定戦を行います", spec(bands)})
	}
	if bands, ok := rankBands(sizes, 2); ok && len(poolSeeds) > 1 {
		out = append(out, shapeCandidate{"fixed-pairs",
			"各組の 1・2 位、3・4 位…を集めて順位決定戦を行います(1 回戦は A組1位 対 B組2位 などのたすき掛け)",
			spec(bands)})
	}
	out = append(out, shapeCandidate{"fixed-none", "組の中の総当たりだけで、順位決定戦は行いません", spec(nil)})
	return out
}

// overallCuts はブロックの順位の全体の並びを分ける大きさ。1〜16 位のトーナメントのあとを 8 チームずつ
// にし、残りを最後のグループにする(GAIA の 95 チームは 16, 8 × 9, 7)。残りが 1 チームなら、最後の 8 と
// 合わせて 4 と 5 に分ける。17 チームより少なければ作らない。
func overallCuts(teams int) ([]int, bool) {
	rest := teams - 16
	if rest < 2 {
		return nil, false
	}
	cuts := []int{16}
	for ; rest >= blockSize; rest -= blockSize {
		cuts = append(cuts, blockSize)
	}
	switch {
	case rest == 1 && cuts[len(cuts)-1] == blockSize:
		cuts = append(cuts[:len(cuts)-1], 4, 5)
	case rest == 1:
		return nil, false
	case rest > 1:
		cuts = append(cuts, rest)
	}
	return cuts, true
}

type shapeCandidate struct {
	id   string
	note string
	spec Spec
}

// frequentShapes は運営会社の頻出形(R3 1-2、8 節)のうち、チーム数と競技に合うもの。
func frequentShapes(teams int, sport Sport) []shapeCandidate {
	var out []shapeCandidate
	pool := func(pools int, bands ...Band) Spec {
		return Spec{Template: TemplatePoolBands, Params: PoolBandsParams{Teams: teams, Pools: pools, Bands: bands}}
	}
	one := func(rank int, shape Shape) Band { return Band{Ranks: []int{rank}, Shape: shape} }
	switch {
	case sport == SportFour && teams == 12:
		out = append(out, shapeCandidate{"pb4", "4on4 で最も多い形(直近の大会の 22%)。プール 12 試合 + 4 チームのトーナメント 3 つで 12 試合",
			pool(4, one(1, ShapeT4), one(2, ShapeT4), one(3, ShapeT4))})
	case sport == SportBeach && teams == 32:
		out = append(out, shapeCandidate{"blocks4-525", "ビーチの主流(4 ブロック → 1〜4 位のトーナメント、5〜12 位と 13〜20 位は 1 回戦のみ、21 位以下は 12 チーム)",
			Spec{Template: TemplateBlocks, Params: BlocksParams{Teams: 32, Bands: []Band{
				{Ranks: []int{1}, Shape: ShapeT4}, {Ranks: []int{2, 3}, Shape: ShapeT8R1},
				{Ranks: []int{4, 5}, Shape: ShapeT8R1}, {Ranks: []int{6, 7, 8}, Shape: ShapeT12},
			}}}})
	case teams == 16:
		out = append(out, shapeCandidate{"pb4-t8", "4 プール(4 チーム)→ 各組の 1・2 位で 1〜8 位、3・4 位で 9〜16 位を決めるトーナメント(1 回戦は A組1位 対 C組2位 のたすき掛け)",
			pool(4, Band{Ranks: []int{1, 2}, Shape: ShapeT8}, Band{Ranks: []int{3, 4}, Shape: ShapeT8})})
	case sport == SportSeven && teams == 8:
		out = append(out, shapeCandidate{"rr2", "2 プール + 決勝プール(7 人制で多い形)",
			Spec{Template: TemplateRoundRobin, Params: RoundRobinParams{Teams: 8, Pools: 2}}})
	}
	return out
}

// perRankBands はプールの各順位を 1 つの帯にする。帯のチーム数が 1 になる順位は 1 つ上の帯に
// まとめる。作れない帯の大きさがあれば ok=false。
func perRankBands(teams, pools int) ([]Band, bool) {
	sizes := make([]int, pools)
	for i, seeds := range snakeSeeds(teams, pools) {
		sizes[i] = len(seeds)
	}
	return rankBands(sizes, 1)
}

// rankBands はプールの大きさ sizes から、順位を per 個ずつ(1 なら 1 位、2 位…、2 なら 1・2 位、3・4 位…)
// 1 つの帯にする。帯のチーム数が 1 になる順位は 1 つ上の帯にまとめる。作れない帯の大きさがあれば ok=false。
func rankBands(sizes []int, per int) ([]Band, bool) {
	countAt := func(rank int) int {
		count := 0
		for _, s := range sizes {
			if s >= rank {
				count++
			}
		}
		return count
	}
	var groups [][]int
	for rank := 1; rank <= slices.Max(sizes); rank++ {
		last := len(groups) - 1
		if last >= 0 && (countAt(rank) == 1 || len(groups[last]) < per) {
			groups[last] = append(groups[last], rank)
			continue
		}
		groups = append(groups, []int{rank})
	}
	var bands []Band
	for _, ranks := range groups {
		count := 0
		for _, r := range ranks {
			count += countAt(r)
		}
		shape, ok := shapeForCount(count)
		if !ok {
			return nil, false
		}
		bands = append(bands, Band{Ranks: ranks, Shape: shape})
	}
	return bands, true
}

// shapeForCount は帯のチーム数に合う型。2・4・7・8 はトーナメント、3・5・6 は総当たり。
func shapeForCount(count int) (Shape, bool) {
	switch count {
	case 2:
		return ShapeT2, true
	case 4:
		return ShapeT4, true
	case 7:
		return ShapeT7, true
	case 8:
		return ShapeT8, true
	case 3, 5, 6:
		return ShapePool, true
	}
	return "", false
}

// Title は形を一行で表す(「4 プール(各 3 チーム) → 4 チームのトーナメント × 3(1〜4 位、5〜8 位、9〜12 位)」)。
func Title(plan *Plan) string {
	if len(plan.Stages) == 0 {
		return ""
	}
	first := plan.StageGroups(0)
	sizes := map[int]int{}
	var order []int
	for _, g := range first {
		size := plan.Groups[g].Size()
		if sizes[size] == 0 {
			order = append(order, size)
		}
		sizes[size]++
	}
	unit := "プール"
	if plan.Groups[first[0]].Kind == domain.GroupBracket {
		unit = "ブロック"
	}
	var head string
	if len(first) == 1 && plan.Groups[first[0]].Kind == domain.GroupBracket {
		head = fmt.Sprintf("%d チームのトーナメント", plan.Groups[first[0]].Size())
	} else if len(first) == 1 {
		head = fmt.Sprintf("%d チームの総当たり", plan.Groups[first[0]].Size())
	} else if len(order) == 1 {
		head = fmt.Sprintf("%d %s(各 %d チーム)", len(first), unit, order[0])
	} else {
		var parts []string
		for _, size := range order {
			parts = append(parts, fmt.Sprintf("%d チーム×%d", size, sizes[size]))
		}
		head = fmt.Sprintf("%d %s(%s)", len(first), unit, strings.Join(parts, "、"))
	}
	if len(plan.Stages) < 2 {
		return head
	}
	// 後の段は「8 チームのトーナメント × 2(1〜8 位、9〜16 位)」のように、型の記号(T8、1st)ではなく
	// 運営の言葉で書く(運営者レビュー M1)
	counts := map[string]int{}
	var kinds []string
	var places []string
	for _, g := range plan.StageGroups(1) {
		group := plan.Groups[g]
		kind := fmt.Sprintf("%d チームのトーナメント", group.Size())
		if group.Kind == domain.GroupPool {
			kind = fmt.Sprintf("%d チームの総当たり", group.Size())
		}
		if counts[kind] == 0 {
			kinds = append(kinds, kind)
		}
		counts[kind]++
		places = append(places, fmt.Sprintf("%d〜%d 位", group.RankOffset+1, group.RankOffset+group.Size()))
	}
	var tail []string
	for _, kind := range kinds {
		if counts[kind] > 1 {
			tail = append(tail, fmt.Sprintf("%s × %d", kind, counts[kind]))
		} else {
			tail = append(tail, kind)
		}
	}
	return fmt.Sprintf("%s → %s(%s)", head, strings.Join(tail, "、"), strings.Join(places, "、"))
}

// FinalPlaces は全試合を終えたときに最終順位が付くチーム数と、そのうち同順位でないチーム数を返す。
func FinalPlaces(plan *Plan) (int, int) {
	ranked, unique := 0, 0
	for _, g := range plan.Groups {
		if !g.Final {
			continue
		}
		if g.Kind == domain.GroupPool {
			ranked += g.Size()
			unique += g.Size()
			continue
		}
		def, err := bracketDefFor(g.Bracket)
		if err != nil {
			continue
		}
		for _, m := range def.matches {
			for _, place := range []int{m.winnerPlace, m.loserPlace} {
				if place == 0 {
					continue
				}
				ranked++
				if !def.placeTied(place) {
					unique++
				}
			}
		}
	}
	return ranked, unique
}
