package store

import (
	"context"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// regulationForfeitScore は division の試合の不戦勝の規定スコア。大会の設定があればそれ、無ければ
// 大会の形の設定(division_formats。形の適用で運営者が決める)、どちらも無ければ nil。
//
// 記録(forfeitScoreOf)と順位計算(loadFormatState)は同じ値を使う。規定スコアがあれば記録する
// スコアは必ずそれになり、順位計算は記録したスコアで数えるので、閲覧ページの成績表と順位が
// 食い違わない(docs/format-engine.md「試合の結果の種類」)。
func regulationForfeitScore(
	ctx context.Context,
	db *gorm.DB,
	division Division,
) (*domain.ForfeitScore, error) {
	tournament, err := findInternal[Tournament](ctx, db, division.TournamentID, "tournament")
	if err != nil {
		return nil, err
	}
	if setting := tournamentForfeitScore(tournament); setting != nil {
		return setting, nil
	}
	row, ok, err := findFormat(ctx, db, division.ID)
	if err != nil || !ok {
		return nil, err
	}
	return &domain.ForfeitScore{Winner: row.ForfeitWinnerScore, Loser: row.ForfeitLoserScore}, nil
}
