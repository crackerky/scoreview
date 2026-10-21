package format

import (
	"fmt"

	"scoreview/api/internal/domain"
)

// BracketPlanFor はトーナメントの型を、管理画面のトーナメント表の設計図(domain.BracketPlan)にする。
// internal/bracket の Registry に登録して、単独のトーナメント表を作るときにも参照サイトの型
// (試合番号は決勝が 1、3 位決定・5 位決定などの行き先つき)を使えるようにするため。
// 枠は表の上からの順がシード(BracketPlan の約束)で、格子のチーム名のセルは空にする。
func BracketPlanFor(kind BracketType) (domain.BracketPlan, error) {
	def, err := bracketDefFor(kind)
	if err != nil {
		return domain.BracketPlan{}, err
	}
	b := &Builder{start: "A"}
	b.SetTeams(def.size)
	stage := b.AddStage("")
	sources := make([]Source, def.size)
	for i := range sources {
		sources[i] = SeedSource(i + 1)
	}
	spec := GroupSpec{Stage: stage, Bracket: kind, Sources: sources, Final: true}
	if _, err := b.AddGroup(spec); err != nil {
		return domain.BracketPlan{}, err
	}
	plan := b.plan
	grid, err := buildGrid(&plan, 0)
	if err != nil {
		return domain.BracketPlan{}, err
	}
	matches := plan.GroupMatches(0)
	index := map[string]int{}
	for i, m := range matches {
		index[m.Code] = i
	}
	entrant := func(side SideRef) domain.PlannedEntrant {
		if side.From != nil {
			return domain.PlannedEntrant{From: &domain.PlannedRef{Match: index[side.From.Code],
				Outcome: side.From.Outcome}}
		}
		seed := side.Slot - 1
		return domain.PlannedEntrant{Seed: &seed}
	}
	out := domain.BracketPlan{
		Pattern: fmt.Sprintf("T%d", def.size),
		Name:    string(kind),
		Teams:   def.size,
		Layout:  grid.Layout,
	}
	for _, row := range out.Layout.Rows {
		for c := range row {
			if row[c].Kind == domain.BracketTeam || row[c].Kind == domain.BracketLose {
				row[c].Text = ""
			}
		}
	}
	for _, m := range matches {
		out.Matches = append(out.Matches, domain.PlannedMatch{Number: m.Number,
			Left: entrant(m.Left), Right: entrant(m.Right)})
	}
	for _, ref := range grid.Refs {
		cell := domain.PlannedCell{Row: ref.Row, Col: ref.Col, Match: index[ref.Match]}
		if ref.Game {
			out.Games = append(out.Games, cell)
			continue
		}
		out.Slots = append(out.Slots, domain.PlannedSlot{PlannedCell: cell, Role: ref.Role})
	}
	return out, nil
}
