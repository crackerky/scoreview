package format

import (
	"fmt"
	"slices"

	"scoreview/api/internal/domain"
)

// Grid はトーナメント表の格子と、格子のセルが何を指すかの対応。格子の TeamID と MatchID は
// 保存するまで分からないので空にし、store が Refs を見て埋める(domain.BracketPlan の Games と
// Slots と同じ考え方。BracketPlanFor で変換できる)。
type Grid struct {
	Layout domain.BracketLayout
	Refs   []GridRef
}

// GridRef は格子のセル (Row, Col) が指すもの。Game なら試合 Match の game セル(スコアとリンク)、
// そうでなければ試合 Match の Role(左、右、勝者、敗者)のチームを出すセル。前の試合の勝者のセルは
// 次の試合の左右を兼ねるので、次の試合の left / right として 1 つだけ持つ(internal/bracket と同じ)。
type GridRef struct {
	Row   int
	Col   int
	Match string
	Game  bool
	Role  domain.BracketSlotRole
}

// gridSectionGap はトーナメント表の山(決勝の山、3 位決定、5 位決定…)の間に空ける行数。
// 勝者の上に順位の文言を最大 2 行、敗者の下に文言 2 行と敗者のセルを置くので、3 行空ければ
// 隣の山と重ならない。
const gridSectionGap = 3

type gridBuilder struct {
	plan    *Plan
	group   Group
	def     bracketDef
	matches map[string]Match
	cols    map[string]int
	rows    [][]domain.BracketCell
	refs    map[[2]int]GridRef
	order   [][2]int
	columns int
	cursor  int
}

// buildGrid は型からトーナメント表の格子を描く。描き方は参照サイトの罫線表現に合わせる
// (clone-spec C 2.3): 上下 2 行の対戦は、上の行に下罫線、上の行の次から下の行までに右罫線、
// 下の行に下罫線を引き、中央の行に game セル、次の列の中央に勝者(win)を置く。敗者は下の山に
// lose で再掲する。最後の列は Placement で、決まる順位を txt_sharp で書く。
func buildGrid(plan *Plan, index int) (*Grid, error) {
	group := plan.Groups[index]
	def, err := bracketDefFor(group.Bracket)
	if err != nil {
		return nil, err
	}
	gb := &gridBuilder{plan: plan, group: group, def: def, matches: map[string]Match{},
		cols: map[string]int{}, refs: map[[2]int]GridRef{}}
	var ordered []Match
	for _, m := range plan.GroupMatches(index) {
		gb.matches[m.Code] = m
		ordered = append(ordered, m)
	}
	maxCol := 0
	for _, m := range ordered {
		maxCol = max(maxCol, gb.column(m.Code))
	}
	gb.columns = maxCol + 2
	gb.cursor = 2 // 0 行目は見出し、1 行目は空ける
	gb.header()
	for _, root := range gb.roots(ordered) {
		gb.section(root)
	}
	gb.ensureRows(gb.cursor)
	grid := &Grid{Layout: domain.BracketLayout{Columns: gb.columns, Rows: gb.rows}}
	for _, key := range gb.order {
		grid.Refs = append(grid.Refs, gb.refs[key])
	}
	if err := grid.Layout.Validate(); err != nil {
		return nil, fmt.Errorf("grid for group %s: %w", group.Letter, err)
	}
	return grid, nil
}

// column は試合を描く列。両側の入力が出そろう列(1 回戦のチームは 0 列、前の試合の結果は
// その試合の次の列)のうち右の方。
func (gb *gridBuilder) column(code string) int {
	if col, ok := gb.cols[code]; ok {
		return col
	}
	m := gb.matches[code]
	col := max(gb.available(m.Left), gb.available(m.Right))
	gb.cols[code] = col
	return col
}

func (gb *gridBuilder) available(side SideRef) int {
	if side.From == nil {
		return 0
	}
	return gb.column(side.From.Code) + 1
}

// roots は勝者が同じグループの次の試合へ進まない試合(決勝、3 位決定、…)を番号順に返す。
func (gb *gridBuilder) roots(ordered []Match) []Match {
	fedWinner := map[string]bool{}
	for _, m := range ordered {
		for _, side := range []SideRef{m.Left, m.Right} {
			if side.From != nil && side.From.Outcome == domain.OutcomeWin {
				fedWinner[side.From.Code] = true
			}
		}
	}
	var roots []Match
	for _, m := range ordered {
		if !fedWinner[m.Code] {
			roots = append(roots, m)
		}
	}
	return roots
}

// ref はセルの対応を記録する。同じセルに 2 度記録したら後のものが勝つ。
func (gb *gridBuilder) ref(r GridRef) {
	key := [2]int{r.Row, r.Col}
	if _, ok := gb.refs[key]; !ok {
		gb.order = append(gb.order, key)
	}
	gb.refs[key] = r
}

func (gb *gridBuilder) ensureRows(n int) {
	for len(gb.rows) < n {
		gb.rows = append(gb.rows, make([]domain.BracketCell, gb.columns))
	}
}

func (gb *gridBuilder) cell(row, col int) *domain.BracketCell {
	gb.ensureRows(row + 1)
	return &gb.rows[row][col]
}

func (gb *gridBuilder) header() {
	last := gb.columns - 1
	for col := range last {
		name := ordinal(col+1) + " Round"
		// 見出しは internal/bracket の single4/8/16 と同じ(1 列目は常に 1st Round)
		switch {
		case gb.def.placeBlock > 0 || col == 0 && last > 2:
		case col == last-1:
			name = "FINAL"
		case col == last-2:
			name = "Semifinals"
		case col == last-3:
			name = "Quarterfinals"
		}
		*gb.cell(0, col) = domain.BracketCell{Kind: domain.BracketTxtYen, Text: name}
	}
	*gb.cell(0, last) = domain.BracketCell{Kind: domain.BracketTxtSharp, Text: "Placement"}
}

// section は 1 つの山を描き、順位の文言を置く。
func (gb *gridBuilder) section(root Match) {
	if gb.def.placeBlock > 0 && (root.Number-1)%gb.def.placeBlock == 0 {
		first := gb.group.RankOffset + root.WinnerPlace
		*gb.cell(gb.cursor-1, 0) = domain.BracketCell{Kind: domain.BracketTxtAnd,
			Text: fmt.Sprintf("%s - %s", ordinal(first), ordinal(first+2))}
	}
	mid := gb.draw(root)
	last := gb.columns - 1
	if mid+1 > gb.cursor-1 {
		gb.cursor = mid + 2
	}
	winner := gb.placeLabels(root.WinnerPlace)
	for i, label := range winner {
		gb.label(mid-len(winner)+i, last, label)
	}
	gb.ref(GridRef{Row: mid, Col: gb.column(root.Code) + 1, Match: root.Code, Role: domain.SlotWinner})
	gb.loser(root, mid)
	gb.cursor += gridSectionGap
}

// draw は試合 m とその前の試合を描き、勝者を書く行を返す。
func (gb *gridBuilder) draw(m Match) int {
	col := gb.column(m.Code)
	top := gb.input(m.Left, col)
	bottom := gb.input(m.Right, col)
	gb.cell(top, col).Borders.Bottom = true
	for row := top + 1; row <= bottom; row++ {
		gb.cell(row, col).Borders.Right = true
	}
	gb.cell(bottom, col).Borders.Bottom = true
	mid := (top + bottom) / 2
	game := gb.cell(mid, col)
	game.Kind = domain.BracketGame
	gb.ref(GridRef{Row: mid, Col: col, Match: m.Code, Game: true})
	gb.ref(GridRef{Row: top, Col: col, Match: m.Code, Role: domain.SlotLeft})
	gb.ref(GridRef{Row: bottom, Col: col, Match: m.Code, Role: domain.SlotRight})
	win := gb.cell(mid, col+1)
	win.Kind = domain.BracketWin
	win.Borders.Bottom = true
	if m.LoserPlace > 0 && gb.isInner(m) {
		gb.loser(m, mid)
	}
	return mid
}

// loser は試合 m の敗者の順位を、最後の列の勝者の行の下に文言と敗者のセル(lose)で書く。
func (gb *gridBuilder) loser(m Match, mid int) {
	labels := gb.placeLabels(m.LoserPlace)
	if len(labels) == 0 {
		return
	}
	last := gb.columns - 1
	for i, label := range labels {
		gb.label(mid+1+i, last, label)
	}
	row := mid + 1 + len(labels)
	if target := gb.cell(row, last); target.Kind == domain.BracketEmpty {
		target.Kind = domain.BracketLose
		target.Borders.Bottom = true
		gb.ref(GridRef{Row: row, Col: last, Match: m.Code, Role: domain.SlotLoser})
	}
	if row+1 > gb.cursor-1 {
		gb.cursor = row + 2
	}
}

// isInner は勝者が同じグループの次の試合へ進む試合か(敗者の順位をここで書く必要がある)。
func (gb *gridBuilder) isInner(m Match) bool {
	for _, other := range gb.matches {
		for _, side := range []SideRef{other.Left, other.Right} {
			if side.From != nil && side.From.Code == m.Code && side.From.Outcome == domain.OutcomeWin {
				return true
			}
		}
	}
	return false
}

// input は試合の片側を描き、その行を返す。前の試合の勝者なら山をたどり、1 回戦のチームと
// 別の山から来た敗者は新しい行に置く。
func (gb *gridBuilder) input(side SideRef, col int) int {
	if side.From != nil && side.From.Outcome == domain.OutcomeWin {
		row := gb.draw(gb.matches[side.From.Code])
		for c := gb.column(side.From.Code) + 1; c < col; c++ {
			gb.cell(row, c+1).Borders.Bottom = true
		}
		return row
	}
	row := gb.cursor
	gb.cursor += 2
	c := gb.cell(row, col)
	if side.From != nil {
		c.Kind = domain.BracketLose
		c.Text = side.From.Label()
		return row
	}
	c.Kind = domain.BracketTeam
	c.Text = gb.group.Slots[side.Slot-1].Source.Label(gb.plan)
	return row
}

func (gb *gridBuilder) label(row, col int, cell domain.BracketCell) {
	if row < 1 {
		return
	}
	target := gb.cell(row, col)
	if target.Kind != domain.BracketEmpty {
		return // 勝者のセルなどと重なるなら文言を省く
	}
	target.Kind, target.Text = cell.Kind, cell.Text
}

// placeLabels はグループ内の順位 place の文言。最終順位のグループは "1st"、"5th(tied)"、
// ブロックは "A rank 1" と行き先 "1st placement T →"(clone-spec C 2.3)。
func (gb *gridBuilder) placeLabels(place int) []domain.BracketCell {
	if place == 0 {
		return nil
	}
	if !gb.group.Final {
		labels := []domain.BracketCell{{Kind: domain.BracketTxtAnd,
			Text: fmt.Sprintf("%s rank %d", gb.group.Letter, place)}}
		if dest, ok := gb.destination(place); ok {
			labels = append(labels, domain.BracketCell{Kind: domain.BracketTxtSharp,
				Text: ordinal(dest) + " placement T →"})
		}
		return labels
	}
	text := ordinal(gb.group.RankOffset + place)
	if gb.def.placeTied(place) {
		text += "(tied)"
	}
	return []domain.BracketCell{{Kind: domain.BracketTxtSharp, Text: text}}
}

// destination はこのグループの place 位が次に入るグループで取りうる最上位の最終順位。
func (gb *gridBuilder) destination(place int) (int, bool) {
	for _, g := range gb.plan.Groups {
		for _, slot := range g.Slots {
			src := slot.Source
			if src.Kind != SourceGroupRank || src.Group != gb.group.Index || src.Rank != place {
				continue
			}
			best := 1
			for _, m := range gb.plan.GroupMatches(g.Index) {
				if slices.Contains([]int{m.Left.Slot, m.Right.Slot}, slot.Position) && m.WinnerPlace > 0 {
					best = m.WinnerPlace
				}
			}
			return g.RankOffset + best, true
		}
	}
	return 0, false
}
