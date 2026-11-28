package format

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"scoreview/api/internal/domain"
)

// TemplateKnockout は「トーナメントだけ」の形。全チームで 1 つの勝ち抜きを組み、負けたチームも順位決定戦で
// 順位を決める(運営者の要望 2026-11-27: 最初からトーナメントで組みたい)。
const TemplateKnockout = "knockout"

// knockoutPrefix は勝ち抜きの型の名前の頭。"KO6" は 6 チームの勝ち抜き。
const knockoutPrefix = "KO"

// 勝ち抜きだけの形のチーム数の範囲。上限は、基にする全順位の型(T16Full)が 16 チームまでのため。
const (
	minKnockoutTeams = 2
	maxKnockoutTeams = 16
)

// KnockoutBracket は n チームの勝ち抜きの型の名前。
func KnockoutBracket(n int) BracketType {
	return BracketType(knockoutPrefix + strconv.Itoa(n))
}

// knockoutTeams は勝ち抜きの型の名前からチーム数を読む。勝ち抜きの型でなければ ok=false。
func knockoutTeams(kind BracketType) (int, bool) {
	rest, found := strings.CutPrefix(string(kind), knockoutPrefix)
	if !found {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil
}

// KnockoutParams は「トーナメントだけ」のパラメータ。
type KnockoutParams struct {
	Teams int
}

// KnockoutTemplate は全チームの勝ち抜き 1 つの形。上位のシードは 1 回戦を免除する。
type KnockoutTemplate struct{}

// Name はテンプレート名。
func (KnockoutTemplate) Name() string { return TemplateKnockout }

// Build は KnockoutParams か *KnockoutParams を受け取る。
func (KnockoutTemplate) Build(b *Builder, params any) error {
	p, err := paramsAs[KnockoutParams](params)
	if err != nil {
		return err
	}
	kind := KnockoutBracket(p.Teams)
	def, err := bracketDefFor(kind)
	if err != nil {
		return err
	}
	b.SetTeams(p.Teams)
	stage := b.AddStage("トーナメント")
	sources := make([]Source, def.size)
	for i, seed := range def.slotSeeds {
		sources[i] = SeedSource(seed)
	}
	_, err = b.AddGroup(GroupSpec{
		Stage: stage, Name: placementName(def, 0), Bracket: kind, Sources: sources, Final: true,
	})
	return err
}

// knockoutDef は n チームで順位を決める勝ち抜きの型。全順位を決める基の型(2、4、8、16 チーム)から、
// n より大きいシードの枠を空けて作る(pruneBracket)。上位のシードほど 1 回戦を免除される。
func knockoutDef(kind BracketType, n int) (bracketDef, error) {
	if n < minKnockoutTeams || n > maxKnockoutTeams {
		return bracketDef{}, domain.Invalid("teams",
			fmt.Sprintf("a knockout takes %d to %d teams, got %d", minKnockoutTeams, maxKnockoutTeams, n))
	}
	var base bracketDef
	var err error
	switch {
	case n == 2:
		base, err = bracketDefFor(BracketT2)
	case n <= 4:
		base, err = bracketDefFor(BracketT4)
	case n <= 8:
		base, err = bracketDefFor(BracketT8)
	default:
		base = t16FullDef()
	}
	if err != nil {
		return bracketDef{}, err
	}
	return pruneBracket(base, kind, n)
}

// outcomeRef は基の型の試合の勝者か敗者が、空けた枠を詰めたあとに誰になるか。ok が偽なら誰もいない。
// side は基の型の枠か、残った試合の結果(基の型の番号)。
type outcomeRef struct {
	ok   bool
	side bracketSide
}

// pruneBracket は基の型の n より大きいシードの枠を空け、片側しかいない試合を取り除く。取り除いた試合の
// 勝者は残った側がそのまま進み、敗者はいない。取り除いた試合が順位を決めていたなら、その順位は残った側を
// 出した試合の結果に付け替える(4 チームの型で 3 チームなら、3 位決定戦の代わりに 1 回戦の敗者が 3 位)。
// 残った試合は基の番号の順に 1 から振り直し、回戦も 1 から詰める。
func pruneBracket(base bracketDef, kind BracketType, n int) (bracketDef, error) {
	present := func(slot int) bool { return base.slotSeeds[slot-1] <= n }
	done := map[int][2]outcomeRef{} // 基の番号 → [勝者, 敗者]
	resolve := func(s bracketSide) (outcomeRef, bool) {
		if s.slot > 0 {
			return outcomeRef{ok: present(s.slot), side: s}, true
		}
		r, found := done[s.from]
		if !found {
			return outcomeRef{}, false
		}
		if s.outcome == domain.OutcomeWin {
			return r[0], true
		}
		return r[1], true
	}
	places := map[bracketSide]int{}
	var kept []bracketMatchDef
	remaining := slices.Clone(base.matches)
	for len(remaining) > 0 {
		var next []bracketMatchDef
		for _, m := range remaining {
			left, leftReady := resolve(m.left)
			right, rightReady := resolve(m.right)
			if !leftReady || !rightReady {
				next = append(next, m)
				continue
			}
			switch {
			case left.ok && right.ok:
				m.left, m.right = left.side, right.side
				kept = append(kept, m)
				done[m.number] = [2]outcomeRef{{ok: true, side: winOf(m.number)}, {ok: true, side: loseOf(m.number)}}
			case left.ok || right.ok:
				only := left
				if !left.ok {
					only = right
				}
				if m.winnerPlace > 0 {
					if only.side.slot > 0 {
						return bracketDef{}, domain.Invalid("bracket",
							fmt.Sprintf("%s: a team would get place %d without playing", kind, m.winnerPlace))
					}
					places[only.side] = m.winnerPlace
				}
				done[m.number] = [2]outcomeRef{only, {}}
			default:
				done[m.number] = [2]outcomeRef{{}, {}}
			}
		}
		if len(next) == len(remaining) {
			return bracketDef{}, fmt.Errorf("prune %s: matches refer to each other", kind)
		}
		remaining = next
	}
	return renumberBracket(base, kind, n, kept, places)
}

// renumberBracket は残った試合と枠を詰めた型にする。places は取り除いた試合から付け替えた順位。
func renumberBracket(
	base bracketDef, kind BracketType, n int, kept []bracketMatchDef, places map[bracketSide]int,
) (bracketDef, error) {
	slices.SortFunc(kept, func(a, c bracketMatchDef) int { return a.number - c.number })
	numbers := map[int]int{}
	var rounds []int
	for i, m := range kept {
		numbers[m.number] = i + 1
		if !slices.Contains(rounds, m.round) {
			rounds = append(rounds, m.round)
		}
	}
	slices.Sort(rounds)
	slots := map[int]int{}
	def := bracketDef{kind: kind, size: n, firstRoundPairs: n == base.size && base.firstRoundPairs}
	for slot, seed := range base.slotSeeds {
		if seed <= n {
			def.slotSeeds = append(def.slotSeeds, seed)
			slots[slot+1] = len(def.slotSeeds)
		}
	}
	side := func(s bracketSide) bracketSide {
		if s.slot > 0 {
			return slotSide(slots[s.slot])
		}
		return bracketSide{from: numbers[s.from], outcome: s.outcome}
	}
	for _, m := range kept {
		for _, moved := range []struct {
			place *int
			from  bracketSide
		}{{&m.winnerPlace, winOf(m.number)}, {&m.loserPlace, loseOf(m.number)}} {
			if place, ok := places[moved.from]; ok {
				if *moved.place != 0 {
					return bracketDef{}, fmt.Errorf("prune %s: match %d already decides place %d", kind, m.number, *moved.place)
				}
				*moved.place = place
			}
		}
		def.matches = append(def.matches, bracketMatchDef{
			number: numbers[m.number], round: slices.Index(rounds, m.round) + 1,
			left: side(m.left), right: side(m.right),
			winnerPlace: m.winnerPlace, loserPlace: m.loserPlace,
		})
	}
	return def, nil
}
