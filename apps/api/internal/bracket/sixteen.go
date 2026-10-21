package bracket

import "scoreview/api/internal/domain"

// SixteenTeams は 16 チームのシングルエリミネーション(1 回戦 8、準々決勝 4、準決勝 2、決勝の 15 試合)。
// 列は参照サイトの T16A と同じ 5 列(clone-spec C 2.3)。3 位決定戦は付けない。
func SixteenTeams() Template {
	spec := singleSpec{
		pattern:    "T16",
		name:       "16 Teams",
		teams:      16,
		roundNames: []string{"1st Round", "Quarterfinals", "Semifinals", "FINAL"},
	}
	return Template{
		ID:    "single16",
		Name:  domain.I18nText{Ja: "16 チーム(シングルエリミネーション)", En: "16 teams (single elimination)"},
		Teams: spec.teams,
		plan:  func() domain.BracketPlan { return singleElimination(spec) },
	}
}
