package bracket

import "scoreview/api/internal/domain"

// EightTeams は 8 チームのトーナメント(1 回戦 4、準決勝 2、決勝、3 位決定戦の 8 試合)。
// 列は参照サイトの T8A と同じ 4 列(1 回戦 / 準決勝 / 決勝 / Placement。clone-spec C 2.3)。
func EightTeams() Template {
	spec := singleSpec{
		pattern:    "T8",
		name:       "8 Teams",
		teams:      8,
		roundNames: []string{"1st Round", "Semifinals", "FINAL"},
		thirdPlace: true,
	}
	return Template{
		ID: "single8",
		Name: domain.I18nText{
			Ja: "8 チーム(1 回戦、準決勝、決勝、3 位決定戦)",
			En: "8 teams (1st round, semifinals, final, 3rd place)",
		},
		Teams: spec.teams,
		plan:  func() domain.BracketPlan { return singleElimination(spec) },
	}
}
