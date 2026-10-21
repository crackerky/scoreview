package bracket

import "scoreview/api/internal/domain"

// FourTeams は 4 チームのトーナメント(準決勝 2、決勝、3 位決定戦の 4 試合)。参照サイトの
// 1st placement tournament(T4E。Semifinals / FINAL / Placement の 3 列、1st〜4th)に当たる形。
func FourTeams() Template {
	spec := singleSpec{
		pattern:    "T4",
		name:       "4 Teams",
		teams:      4,
		roundNames: []string{"Semifinals", "FINAL"},
		thirdPlace: true,
	}
	return Template{
		ID: "single4",
		Name: domain.I18nText{
			Ja: "4 チーム(準決勝、決勝、3 位決定戦)",
			En: "4 teams (semifinals, final, 3rd place)",
		},
		Teams: spec.teams,
		plan:  func() domain.BracketPlan { return singleElimination(spec) },
	}
}
