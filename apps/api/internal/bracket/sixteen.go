package bracket

import (
	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// SixteenTeams は 16 チームで 1〜8 位を決めるトーナメント(大会形式エンジンの T16、20 試合)。
// 1 回戦の敗者には順位を付けない(別の順位決定戦へ送る型。R3 4 章)。試合番号は決勝が 1。
func SixteenTeams() Template {
	return mustFormatTemplate("single16", format.BracketT16, domain.I18nText{
		Ja: "16 チーム(1〜8 位。1 回戦の敗者は順位なし)",
		En: "16 teams (1st-8th)",
	})
}
