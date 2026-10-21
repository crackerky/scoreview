package bracket

import (
	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// EightTeams は 8 チームで 1〜8 位を決めるトーナメント(参照サイトの T8 全順位型、12 試合)。
// 試合番号は決勝が 1、準決勝が 2 と 3、1 回戦が 4〜7、3 位決定戦が 8、5 位決定戦が 9、
// 敗者戦が 10 と 11、7 位決定戦が 12(R3 6 章)。
func EightTeams() Template {
	return mustFormatTemplate("single8", format.BracketT8, domain.I18nText{
		Ja: "8 チーム全順位(1 回戦、準決勝、決勝、3・5・7 位決定戦。1〜8 位)",
		En: "8 teams (1st-8th)",
	})
}
