package bracket

import (
	"fmt"

	"scoreview/api/internal/domain"
)

// singleSpec はシングルエリミネーションの表の形。teams は 2 のべき乗。
type singleSpec struct {
	pattern    string
	name       string
	teams      int
	roundNames []string // 列見出し。len = ラウンド数
	thirdPlace bool     // 準決勝の敗者による 3 位決定戦を付けるか
}

// 格子の行の割り付け。1 回戦の 1 試合は参照サイトと同じく 3 行(上のチーム、スコア、下のチーム)で、
// 試合の間に 1 行空ける(clone-spec C 2.3)。
const (
	headerRows     = 1
	rowsPerOpening = 4
)

// grid は設計図を組み立てる途中の状態。
type grid struct {
	cells   [][]domain.BracketCell
	matches []domain.PlannedMatch
	games   []domain.PlannedCell
	slots   map[[2]int]domain.PlannedSlot
	order   [][2]int // slots を作った順(出力を決定的にする)
}

func newGrid(rows, columns int) *grid {
	cells := make([][]domain.BracketCell, rows)
	for r := range cells {
		cells[r] = make([]domain.BracketCell, columns)
	}
	return &grid{cells: cells, slots: map[[2]int]domain.PlannedSlot{}}
}

func (g *grid) slot(row, col, match int, role domain.BracketSlotRole) {
	key := [2]int{row, col}
	if _, ok := g.slots[key]; !ok {
		g.order = append(g.order, key)
	}
	g.slots[key] = domain.PlannedSlot{
		PlannedCell: domain.PlannedCell{Row: row, Col: col, Match: match},
		Role:        role,
	}
}

// bout は col 列の top 行と bottom 行のチームの対戦を、参照サイトの罫線の組み合わせで描く
// (clone-spec C 2.3「1 試合の描き方」)。上のチームのセルに下罫線、その次の行から下のチームの行まで
// 右罫線、下のチームのセルに下罫線を引き、中央の行に game セル、右隣の列の中央の行に勝者(win)を置く。
// 上と下のチームのセルはこの試合の左と右(スコアは「上-下」の順)。戻り値は game セルの行。
func (g *grid) bout(col, top, bottom, match int) int {
	g.cells[top][col].Borders.Bottom = true
	for r := top + 1; r <= bottom; r++ {
		g.cells[r][col].Borders.Right = true
	}
	g.cells[bottom][col].Borders.Bottom = true
	mid := (top + bottom) / 2
	g.cells[mid][col].Kind = domain.BracketGame
	g.games = append(g.games, domain.PlannedCell{Row: mid, Col: col, Match: match})

	winner := &g.cells[mid][col+1]
	winner.Kind = domain.BracketWin
	winner.Borders.Bottom = true
	winner.Borders.Left = true
	g.slot(mid, col+1, match, domain.SlotWinner)
	// 前のラウンドの勝者のセルは、この試合の参加チームのセルを兼ねる。実際の参加チームを出すため
	// 役割を left / right で上書きする(管理者が手で入れ替えた場合もそれが出る)
	g.slot(top, col, match, domain.SlotLeft)
	g.slot(bottom, col, match, domain.SlotRight)
	return mid
}

func (g *grid) addMatch(left, right domain.PlannedEntrant) int {
	g.matches = append(
		g.matches,
		domain.PlannedMatch{Number: len(g.matches) + 1, Left: left, Right: right},
	)
	return len(g.matches) - 1
}

func seed(index int) domain.PlannedEntrant { return domain.PlannedEntrant{Seed: &index} }

func from(match int, outcome domain.Outcome) domain.PlannedEntrant {
	return domain.PlannedEntrant{From: &domain.PlannedRef{Match: match, Outcome: outcome}}
}

// placement は順位列に「見出し(txt_sharp)」と、その下の行にチーム(win または lose)を置く。
func (g *grid) placement(row, col int, label string, kind domain.BracketCellKind, match int,
	role domain.BracketSlotRole,
) {
	g.cells[row-1][col] = domain.BracketCell{Kind: domain.BracketTxtSharp, Text: label}
	cell := &g.cells[row][col]
	cell.Kind = kind
	cell.Borders.Bottom = true
	g.slot(row, col, match, role)
}

// singleElimination はシングルエリミネーションの設計図を作る。
//
// 列は ラウンド数 + 1(最後が Placement 列。参照サイトの T8A は 4 列、T4E は 3 列。clone-spec C 2.3)。
// 行 0 は見出し(ラウンド名は txt_yen、Placement は txt_sharp)。試合の番号は 1 回戦の上から順、
// 次のラウンドの上から順、…、決勝、3 位決定戦の順。
func singleElimination(spec singleSpec) domain.BracketPlan {
	rounds := len(spec.roundNames)
	if spec.teams != 1<<rounds {
		// テンプレートの定義(このパッケージ内の定数)の誤りなので、起動時のテストで必ず見つかる
		panic(fmt.Sprintf("bracket %s: %d teams need %d rounds", spec.pattern, spec.teams, rounds))
	}
	placementCol := rounds
	mainRows := headerRows + rowsPerOpening*(spec.teams/2) - 1
	rows := mainRows
	if spec.thirdPlace {
		// 見出し 1 行、試合 3 行、4 位のチーム 1 行
		rows += 5
	}
	g := newGrid(rows, rounds+1)
	for col, name := range spec.roundNames {
		g.cells[0][col] = domain.BracketCell{Kind: domain.BracketTxtYen, Text: name}
	}
	g.cells[0][placementCol] = domain.BracketCell{Kind: domain.BracketTxtSharp, Text: "Placement"}

	// 1 回戦: シード 2i と 2i+1
	type drawn struct{ match, row int }
	var current []drawn
	for i := range spec.teams / 2 {
		top := headerRows + rowsPerOpening*i
		match := g.addMatch(seed(2*i), seed(2*i+1))
		g.cells[top][0].Kind = domain.BracketTeam
		g.cells[top+2][0].Kind = domain.BracketTeam
		current = append(current, drawn{match: match, row: g.bout(0, top, top+2, match)})
	}
	// 2 回戦以降: 前のラウンドの隣り合う 2 試合の勝者
	var semis []drawn
	for col := 1; col < rounds; col++ {
		if len(current) == 2 {
			semis = current
		}
		next := make([]drawn, 0, len(current)/2)
		for i := 0; i < len(current); i += 2 {
			upper, lower := current[i], current[i+1]
			match := g.addMatch(
				from(upper.match, domain.OutcomeWin),
				from(lower.match, domain.OutcomeWin),
			)
			next = append(next, drawn{match: match, row: g.bout(col, upper.row, lower.row, match)})
		}
		current = next
	}
	final := current[0]
	// 決勝の勝者のセル(bout が置いた win)に見出しを付け、その下に準優勝を置く
	g.cells[final.row-1][placementCol] = domain.BracketCell{
		Kind: domain.BracketTxtSharp,
		Text: "1st",
	}
	g.placement(final.row+2, placementCol, "2nd", domain.BracketLose, final.match, domain.SlotLoser)

	if spec.thirdPlace && len(semis) == 2 {
		thirdPlace(g, mainRows, rounds-1, semis[0].match, semis[1].match)
	}

	plan := domain.BracketPlan{
		Pattern: spec.pattern,
		Name:    spec.name,
		Teams:   spec.teams,
		Layout:  domain.BracketLayout{Columns: rounds + 1, Rows: g.cells},
		Matches: g.matches,
		Games:   g.games,
		Slots:   make([]domain.PlannedSlot, 0, len(g.order)),
	}
	for _, key := range g.order {
		plan.Slots = append(plan.Slots, g.slots[key])
	}
	return plan
}

// thirdPlace は準決勝の敗者どうしの 3 位決定戦を、決勝の列(col)の head 行から下に描く。
// 見出し(txt_and)は参照サイトの順位決定戦の小見出し "3rd - 4th" と同じ表記(clone-spec C 2.3)。
func thirdPlace(g *grid, head, col, semiUpper, semiLower int) {
	g.cells[head][col] = domain.BracketCell{Kind: domain.BracketTxtAnd, Text: "3rd - 4th"}
	top, bottom := head+1, head+3
	match := g.addMatch(from(semiUpper, domain.OutcomeLose), from(semiLower, domain.OutcomeLose))
	g.cells[top][col].Kind = domain.BracketLose
	g.cells[bottom][col].Kind = domain.BracketLose
	row := g.bout(col, top, bottom, match)
	placementCol := col + 1
	g.cells[row-1][placementCol] = domain.BracketCell{Kind: domain.BracketTxtSharp, Text: "3rd"}
	g.placement(row+2, placementCol, "4th", domain.BracketLose, match, domain.SlotLoser)
}
