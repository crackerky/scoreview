package bracket_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/bracket"
	"scoreview/api/internal/domain"
)

func TestDefaultRegistry(t *testing.T) {
	registry := bracket.Default()
	ids := []string{}
	for _, tmpl := range registry.List() {
		ids = append(ids, tmpl.ID)
	}
	assert.Equal(t, []string{"single4", "single8", "single16"}, ids)
	_, ok := registry.Get("single8")
	assert.True(t, ok)
	_, ok = registry.Get("missing")
	assert.False(t, ok)
}

func TestWithTemplateReplacesSameID(t *testing.T) {
	custom := bracket.FourTeams()
	custom.Name = domain.I18nText{Ja: "差し替え", En: "Replaced"}
	registry := bracket.NewRegistry(
		bracket.WithTemplate(bracket.FourTeams()),
		bracket.WithTemplate(custom),
	)
	require.Len(t, registry.List(), 1)
	got, _ := registry.Get("single4")
	assert.Equal(t, "Replaced", got.Name.En)
}

// TestTemplateShapes は格子の大きさ、試合と game セルの数、見出し行を確かめる。single4/8/16 は
// 大会形式エンジンの T4、T8 全順位型、T16 で作る。
func TestTemplateShapes(t *testing.T) {
	cases := []struct {
		template bracket.Template
		columns  int
		rows     int
		matches  int
		header   []string
	}{
		{bracket.FourTeams(), 3, 21, 4, []string{"Semifinals", "FINAL", "Placement"}},
		{bracket.EightTeams(), 4, 48, 12, []string{"1st Round", "Semifinals", "FINAL", "Placement"}},
		{
			bracket.SixteenTeams(), 5, 64, 20,
			[]string{"1st Round", "Quarterfinals", "Semifinals", "FINAL", "Placement"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.template.ID, func(t *testing.T) {
			plan := tc.template.Plan()
			require.NoError(t, plan.Layout.Validate())
			assert.Equal(t, tc.template.Teams, plan.Teams)
			assert.Equal(t, tc.columns, plan.Layout.Columns)
			assert.Len(t, plan.Layout.Rows, tc.rows)
			assert.Len(t, plan.Matches, tc.matches)
			assert.Len(t, plan.Games, tc.matches, "game セルは試合ごとに 1 つ")

			header := make([]string, 0, tc.columns)
			for _, cell := range plan.Layout.Rows[0] {
				header = append(header, cell.Text)
			}
			assert.Equal(t, tc.header, header)
			last := plan.Layout.Rows[0][tc.columns-1]
			assert.Equal(t, domain.BracketTxtSharp, last.Kind)

			// 全シードがちょうど 1 回ずつ 1 回戦に出る
			seen := map[int]int{}
			for _, match := range plan.Matches {
				for _, side := range []domain.PlannedEntrant{match.Left, match.Right} {
					if side.Seed != nil {
						seen[*side.Seed]++
					}
				}
			}
			assert.Len(t, seen, tc.template.Teams)
			for seedIndex, count := range seen {
				assert.Equal(t, 1, count, "seed %d", seedIndex)
			}
			// 試合番号は決勝が 1(参照サイトと同じ。R3 5-5)
			assert.Equal(t, 1, plan.Matches[0].Number)
			assert.NotNil(t, plan.Matches[0].Left.From, "決勝は準決勝の勝者どうし")
			for _, game := range plan.Games {
				assert.Equal(t, domain.BracketGame, plan.Layout.Rows[game.Row][game.Col].Kind)
			}
			for _, slot := range plan.Slots {
				kind := plan.Layout.Rows[slot.Row][slot.Col].Kind
				assert.Contains(
					t,
					[]domain.BracketCellKind{
						domain.BracketTeam,
						domain.BracketWin,
						domain.BracketLose,
					},
					kind,
					"slot (%d,%d)",
					slot.Row,
					slot.Col,
				)
			}
		})
	}
}

// TestOpeningMatchUsesThreeRows は 1 回戦の 1 試合が参照サイトと同じ 3 行の罫線で描かれることを確かめる
// (clone-spec C 2.3: 上のチームに下罫線、スコアの行と下のチームの行に右罫線、下のチームに下罫線、
// 右隣の列のスコアの行に勝者)。
func TestOpeningMatchUsesThreeRows(t *testing.T) {
	plan := bracket.EightTeams().Plan()
	rows := plan.Layout.Rows
	top, game, bottom := rows[2][0], rows[3][0], rows[4][0]
	assert.Equal(t, domain.BracketTeam, top.Kind)
	assert.Equal(t, domain.BracketBorders{Bottom: true}, top.Borders)
	assert.Equal(t, domain.BracketGame, game.Kind)
	assert.Equal(t, domain.BracketBorders{Right: true}, game.Borders)
	assert.Equal(t, domain.BracketTeam, bottom.Kind)
	assert.Equal(t, domain.BracketBorders{Right: true, Bottom: true}, bottom.Borders)
	winner := rows[3][1]
	assert.Equal(t, domain.BracketWin, winner.Kind)
	assert.True(t, winner.Borders.Bottom)
	// 次の試合との間は 1 行空く
	assert.Equal(t, domain.BracketEmpty, rows[5][0].Kind)
	assert.Equal(t, domain.BracketTeam, rows[6][0].Kind)
}

// TestEightTeamsReferences は後続試合の未確定チームの参照と、格子のチーム名のセルの役割を確かめる
// (参照サイトの T8 全順位型。R3 6 章)。
func TestEightTeamsReferences(t *testing.T) {
	plan := bracket.EightTeams().Plan()
	ref := func(number int, outcome domain.Outcome) domain.PlannedEntrant {
		return domain.PlannedEntrant{From: &domain.PlannedRef{Match: number - 1, Outcome: outcome}}
	}
	for i, match := range plan.Matches {
		assert.Equal(t, i+1, match.Number)
	}
	byNumber := func(number int) domain.PlannedMatch { return plan.Matches[number-1] }
	// 1 = [2勝] 対 [3勝]、2 = [4勝] 対 [5勝]、8 = [2負] 対 [3負]、9 = [10勝] 対 [11勝]、12 = [10負] 対 [11負]
	assert.Equal(t, ref(2, domain.OutcomeWin), byNumber(1).Left)
	assert.Equal(t, ref(3, domain.OutcomeWin), byNumber(1).Right)
	assert.Equal(t, ref(4, domain.OutcomeWin), byNumber(2).Left)
	assert.Equal(t, ref(5, domain.OutcomeWin), byNumber(2).Right)
	assert.Equal(t, ref(2, domain.OutcomeLose), byNumber(8).Left)
	assert.Equal(t, ref(10, domain.OutcomeWin), byNumber(9).Left)
	assert.Equal(t, ref(11, domain.OutcomeLose), byNumber(12).Right)
	require.NotNil(t, byNumber(4).Left.Seed)
	assert.Equal(t, 0, *byNumber(4).Left.Seed, "1 回戦の 4 は表の一番上の枠")

	roles := map[[2]int]domain.PlannedSlot{}
	for _, slot := range plan.Slots {
		key := [2]int{slot.Row, slot.Col}
		_, dup := roles[key]
		require.False(t, dup, "slot (%d,%d) が重複", slot.Row, slot.Col)
		roles[key] = slot
	}
	// 1 回戦の勝者のセルは準決勝の参加チームのセルを兼ねる(役割は準決勝 2 の left)
	semiTop := roles[[2]int{3, 1}]
	assert.Equal(t, 1, semiTop.Match)
	assert.Equal(t, domain.SlotLeft, semiTop.Role)
	// 優勝は決勝の勝者、準優勝は決勝の敗者(Placement 列)
	champion, runnerUp := roles[[2]int{9, 3}], roles[[2]int{11, 3}]
	assert.Equal(t, domain.SlotWinner, champion.Role)
	assert.Equal(t, 0, champion.Match)
	assert.Equal(t, domain.SlotLoser, runnerUp.Role)
	assert.Equal(t, "1st", plan.Layout.Rows[8][3].Text)
	assert.Equal(t, "2nd", plan.Layout.Rows[10][3].Text)
	assert.Equal(t, "3rd", plan.Layout.Rows[21][3].Text)
	assert.Equal(t, domain.BracketLose, plan.Layout.Rows[21][2].Kind, "3 位決定戦は準決勝の敗者")
}

// TestPlanIsFresh は Plan が呼ぶたびに別の格子を返す(呼び出し側の書き換えが次に漏れない)ことを確かめる。
func TestPlanIsFresh(t *testing.T) {
	tmpl := bracket.FourTeams()
	first := tmpl.Plan()
	first.Layout.Rows[1][0].Text = "changed"
	assert.Empty(t, tmpl.Plan().Layout.Rows[1][0].Text)
}
