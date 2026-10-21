package format

import (
	"fmt"

	"scoreview/api/internal/domain"
)

// bracketSide は型の定義の中の試合の片側。slot(1 から)か、同じ型の中の試合番号 from の勝者・敗者。
type bracketSide struct {
	slot    int
	from    int
	outcome domain.Outcome
}

func slotSide(n int) bracketSide { return bracketSide{slot: n} }
func winOf(n int) bracketSide    { return bracketSide{from: n, outcome: domain.OutcomeWin} }
func loseOf(n int) bracketSide   { return bracketSide{from: n, outcome: domain.OutcomeLose} }

type bracketMatchDef struct {
	number      int
	round       int
	left, right bracketSide
	winnerPlace int
	loserPlace  int
}

// bracketDef はトーナメントの型。slotSeeds[i] は上から i+1 番目の枠に入るシード(1 から)。
type bracketDef struct {
	kind      BracketType
	size      int
	slotSeeds []int
	matches   []bracketMatchDef
	// firstRoundPairs は 1 回戦が枠 (1,2)(3,4)… の順に組まれるか。たすき掛けと同順位どうしの
	// 組み合わせは、この並びを前提に枠を埋める
	firstRoundPairs bool
	// placeBlock は「1 回戦のみ」型で同順位を分ける試合の数(R3 4 章、clone-spec C 2.3 の 6628)。
	// 0 は通常の型
	placeBlock int
}

// standardSeedOrder は n(2 のべき)チームの単純勝ち抜きで、上から並べたシード順。
// 1 と 2 が決勝まで当たらない標準の配置(8 なら 1,8,4,5,2,7,3,6)。
func standardSeedOrder(n int) []int {
	order := []int{1}
	for size := 2; size <= n; size *= 2 {
		next := make([]int, 0, size)
		for _, s := range order {
			next = append(next, s, size+1-s)
		}
		order = next
	}
	return order
}

// bracketDefFor は型の定義を返す。試合番号は決勝を 1 として 1 回戦へ向かって振り、
// その後に 3 位決定、5 位決定、敗者戦、7 位決定を続ける(R3 4 章の T8 全順位型。
// T7 と T16 の番号は同じ考え方で作った推測)。
func bracketDefFor(kind BracketType) (bracketDef, error) {
	switch kind {
	case BracketT2:
		return bracketDef{
			kind: kind, size: 2, slotSeeds: []int{1, 2}, firstRoundPairs: true,
			matches: []bracketMatchDef{
				{number: 1, round: 1, left: slotSide(1), right: slotSide(2), winnerPlace: 1, loserPlace: 2},
			},
		}, nil
	case BracketT4:
		return bracketDef{
			kind: kind, size: 4, slotSeeds: standardSeedOrder(4), firstRoundPairs: true,
			matches: []bracketMatchDef{
				{number: 1, round: 2, left: winOf(2), right: winOf(3), winnerPlace: 1, loserPlace: 2},
				{number: 2, round: 1, left: slotSide(1), right: slotSide(2)},
				{number: 3, round: 1, left: slotSide(3), right: slotSide(4)},
				{number: 4, round: 2, left: loseOf(2), right: loseOf(3), winnerPlace: 3, loserPlace: 4},
			},
		}, nil
	case BracketT7:
		// 最上位シードは 1 回戦免除で準決勝から入る。5〜7 位は、準決勝で 1 位シードに当たる側の
		// 1 回戦の敗者を 5 位決定戦へ直接送り、残りの 2 敗者の勝者と当てる(参照サイトの 9 試合に
		// 合う形として推測)
		return bracketDef{
			kind: kind, size: 7, slotSeeds: []int{1, 4, 5, 2, 7, 3, 6},
			matches: []bracketMatchDef{
				{number: 1, round: 3, left: winOf(2), right: winOf(3), winnerPlace: 1, loserPlace: 2},
				{number: 2, round: 2, left: slotSide(1), right: winOf(4)},
				{number: 3, round: 2, left: winOf(5), right: winOf(6)},
				{number: 4, round: 1, left: slotSide(2), right: slotSide(3)},
				{number: 5, round: 1, left: slotSide(4), right: slotSide(5)},
				{number: 6, round: 1, left: slotSide(6), right: slotSide(7)},
				{number: 7, round: 3, left: loseOf(2), right: loseOf(3), winnerPlace: 3, loserPlace: 4},
				{number: 8, round: 3, left: loseOf(4), right: winOf(9), winnerPlace: 5, loserPlace: 6},
				{number: 9, round: 2, left: loseOf(5), right: loseOf(6), loserPlace: 7},
			},
		}, nil
	case BracketT8, BracketT8Block:
		// R3 6 章で確認した構造: 1=[2勝]vs[3勝]、8=[2負]vs[3負]、10=[4負]vs[5負]、11=[6負]vs[7負]、
		// 9=[10勝]vs[11勝]、12=[10負]vs[11負]
		return bracketDef{
			kind: kind, size: 8, slotSeeds: standardSeedOrder(8), firstRoundPairs: true,
			matches: []bracketMatchDef{
				{number: 1, round: 3, left: winOf(2), right: winOf(3), winnerPlace: 1, loserPlace: 2},
				{number: 2, round: 2, left: winOf(4), right: winOf(5)},
				{number: 3, round: 2, left: winOf(6), right: winOf(7)},
				{number: 4, round: 1, left: slotSide(1), right: slotSide(2)},
				{number: 5, round: 1, left: slotSide(3), right: slotSide(4)},
				{number: 6, round: 1, left: slotSide(5), right: slotSide(6)},
				{number: 7, round: 1, left: slotSide(7), right: slotSide(8)},
				{number: 8, round: 3, left: loseOf(2), right: loseOf(3), winnerPlace: 3, loserPlace: 4},
				{number: 9, round: 3, left: winOf(10), right: winOf(11), winnerPlace: 5, loserPlace: 6},
				{number: 10, round: 2, left: loseOf(4), right: loseOf(5)},
				{number: 11, round: 2, left: loseOf(6), right: loseOf(7)},
				{number: 12, round: 3, left: loseOf(10), right: loseOf(11), winnerPlace: 7, loserPlace: 8},
			},
		}, nil
	case BracketT8R1:
		return firstRoundOnlyDef(kind, 8), nil
	case BracketT12:
		return firstRoundOnlyDef(kind, 12), nil
	case BracketT16:
		return t16Def(), nil
	}
	return bracketDef{}, domain.Invalid("bracket", fmt.Sprintf("unknown bracket type %q", kind))
}

// firstRoundOnlyDef は 1 回戦だけの型。2 試合ずつを 1 つの順位の組にし、組 b の勝者は 4b+1 位、
// 敗者は 4b+3 位の同順位(clone-spec C 2.3 の 6628: 5th(tied), 7th(tied), 9th(tied), 11th(tied))。
func firstRoundOnlyDef(kind BracketType, size int) bracketDef {
	def := bracketDef{kind: kind, size: size, firstRoundPairs: true, placeBlock: 2}
	for i := range size {
		def.slotSeeds = append(def.slotSeeds, i+1)
	}
	for i := range size / 2 {
		block := i / def.placeBlock
		def.matches = append(def.matches, bracketMatchDef{
			number: i + 1, round: 1, left: slotSide(2*i + 1), right: slotSide(2*i + 2),
			winnerPlace: 4*block + 1, loserPlace: 4*block + 3,
		})
	}
	return def
}

// t16Def は 16 チームで 1〜8 位を決める型(R3 4 章、342 の 4992)。1 回戦の敗者は順位を
// 決めず、別の段の枠から match(m, lose) で参照する。
func t16Def() bracketDef {
	def := bracketDef{kind: BracketT16, size: 16, slotSeeds: standardSeedOrder(16), firstRoundPairs: true}
	add := func(number, round int, left, right bracketSide, winnerPlace, loserPlace int) {
		def.matches = append(def.matches, bracketMatchDef{
			number: number, round: round, left: left, right: right,
			winnerPlace: winnerPlace, loserPlace: loserPlace,
		})
	}
	add(1, 4, winOf(2), winOf(3), 1, 2)
	add(2, 3, winOf(4), winOf(5), 0, 0)
	add(3, 3, winOf(6), winOf(7), 0, 0)
	for i := range 4 {
		add(4+i, 2, winOf(8+2*i), winOf(9+2*i), 0, 0)
	}
	for i := range 8 {
		add(8+i, 1, slotSide(2*i+1), slotSide(2*i+2), 0, 0)
	}
	add(16, 4, loseOf(2), loseOf(3), 3, 4)
	add(17, 4, winOf(18), winOf(19), 5, 6)
	add(18, 3, loseOf(4), loseOf(5), 0, 0)
	add(19, 3, loseOf(6), loseOf(7), 0, 0)
	add(20, 4, loseOf(18), loseOf(19), 7, 8)
	return def
}

// placeTied はグループ内の順位 place を複数の試合が決める(同順位)かどうか。
func (d bracketDef) placeTied(place int) bool {
	count := 0
	for _, m := range d.matches {
		if m.winnerPlace == place {
			count++
		}
		if m.loserPlace == place {
			count++
		}
	}
	return count > 1
}
