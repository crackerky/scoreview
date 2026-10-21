package bracket

import (
	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// FourTeams は 4 チームのトーナメント(準決勝 2、決勝、3 位決定戦の 4 試合)。大会形式エンジンの
// T4 で作り、試合番号は参照サイトと同じく決勝が 1、準決勝が 2 と 3、3 位決定戦が 4(R3 4 章、5-5)。
func FourTeams() Template {
	return mustFormatTemplate("single4", format.BracketT4, domain.I18nText{
		Ja: "4 チーム(準決勝、決勝、3 位決定戦。1〜4 位)",
		En: "4 teams (semifinals, final, 3rd place)",
	})
}
