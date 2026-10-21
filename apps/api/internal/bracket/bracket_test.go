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

// TestTemplateShapes は格子の大きさ、試合と game セルの数、見出し行を確かめる。
func TestTemplateShapes(t *testing.T) {
	cases := []struct {
		template bracket.Template
		columns  int
		rows     int
		matches  int
		header   []string
	}{
		// 2 試合 × 4 行 - 1 + 見出し 1 行 = 8 行、3 位決定戦 5 行
		{bracket.FourTeams(), 3, 13, 4, []string{"Semifinals", "FINAL", "Placement"}},
		{bracket.EightTeams(), 4, 21, 8, []string{"1st Round", "Semifinals", "FINAL", "Placement"}},
		{
			bracket.SixteenTeams(), 5, 32, 15,
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
			// game セルの位置は game の種類、slot の位置はチームを出す種類
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
	top, game, bottom := rows[1][0], rows[2][0], rows[3][0]
	assert.Equal(t, domain.BracketTeam, top.Kind)
	assert.Equal(t, domain.BracketBorders{Bottom: true}, top.Borders)
	assert.Equal(t, domain.BracketGame, game.Kind)
	assert.Equal(t, domain.BracketBorders{Right: true}, game.Borders)
	assert.Equal(t, domain.BracketTeam, bottom.Kind)
	assert.Equal(t, domain.BracketBorders{Right: true, Bottom: true}, bottom.Borders)
	winner := rows[2][1]
	assert.Equal(t, domain.BracketWin, winner.Kind)
	assert.True(t, winner.Borders.Bottom)
	// 次の試合との間は 1 行空く
	assert.Equal(t, domain.BracketEmpty, rows[4][0].Kind)
	assert.Equal(t, domain.BracketTeam, rows[5][0].Kind)
}

// TestEightTeamsReferences は後続試合の未確定チームの参照と、格子のチーム名のセルの役割を確かめる。
func TestEightTeamsReferences(t *testing.T) {
	plan := bracket.EightTeams().Plan()
	ref := func(match int, outcome domain.Outcome) domain.PlannedEntrant {
		return domain.PlannedEntrant{From: &domain.PlannedRef{Match: match, Outcome: outcome}}
	}
	// 試合の番号: 1〜4 が 1 回戦、5 と 6 が準決勝、7 が決勝、8 が 3 位決定戦
	for i, match := range plan.Matches {
		assert.Equal(t, i+1, match.Number)
	}
	assert.Equal(t, ref(0, domain.OutcomeWin), plan.Matches[4].Left)
	assert.Equal(t, ref(1, domain.OutcomeWin), plan.Matches[4].Right)
	assert.Equal(t, ref(4, domain.OutcomeWin), plan.Matches[6].Left)
	assert.Equal(t, ref(5, domain.OutcomeWin), plan.Matches[6].Right)
	assert.Equal(t, ref(4, domain.OutcomeLose), plan.Matches[7].Left)
	assert.Equal(t, ref(5, domain.OutcomeLose), plan.Matches[7].Right)

	roles := map[[2]int]domain.PlannedSlot{}
	for _, slot := range plan.Slots {
		key := [2]int{slot.Row, slot.Col}
		_, dup := roles[key]
		require.False(t, dup, "slot (%d,%d) が重複", slot.Row, slot.Col)
		roles[key] = slot
	}
	// 1 回戦の勝者のセルは準決勝の参加チームのセルを兼ねる(役割は準決勝の left)
	semiTop := roles[[2]int{2, 1}]
	assert.Equal(t, 4, semiTop.Match)
	assert.Equal(t, domain.SlotLeft, semiTop.Role)
	// 優勝は決勝の勝者、準優勝は決勝の敗者(Placement 列)
	var champion, runnerUp domain.PlannedSlot
	for _, slot := range plan.Slots {
		if slot.Col == 3 && slot.Match == 6 {
			switch slot.Role {
			case domain.SlotWinner:
				champion = slot
			case domain.SlotLoser:
				runnerUp = slot
			}
		}
	}
	assert.Equal(t, 8, champion.Row)
	assert.Equal(t, "1st", plan.Layout.Rows[champion.Row-1][3].Text)
	assert.Equal(t, "2nd", plan.Layout.Rows[runnerUp.Row-1][3].Text)
	// 3 位決定戦は決勝の列の下に、小見出しつきで描く
	assert.Equal(t, "3rd - 4th", plan.Layout.Rows[16][2].Text)
	assert.Equal(t, domain.BracketTxtAnd, plan.Layout.Rows[16][2].Kind)
	assert.Equal(t, domain.BracketLose, plan.Layout.Rows[17][2].Kind)
}

// TestPlanIsFresh は Plan が呼ぶたびに別の格子を返す(呼び出し側の書き換えが次に漏れない)ことを確かめる。
func TestPlanIsFresh(t *testing.T) {
	tmpl := bracket.FourTeams()
	first := tmpl.Plan()
	first.Layout.Rows[1][0].Text = "changed"
	assert.Empty(t, tmpl.Plan().Layout.Rows[1][0].Text)
}
