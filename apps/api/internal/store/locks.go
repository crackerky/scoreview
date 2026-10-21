package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
)

// 試合の結果と大会形式の進行を変えるトランザクションの直列化。
//
// 結果の変更(matchupdate.go)、形の操作(確定、同順位の指定、差し替え)、形の適用は、どれも
// ディビジョンの全体(枠、下流の試合、group_teams、最終順位)を読み直して書く。同じディビジョンで
// 並行に走ると、片方が古い順位で下流を埋めたまま残る(確定と最後の試合の同時実行)ので、
// 先頭でディビジョンの行をロックして直列にする。ロックの順序は常に
// 「大会(形の適用だけ)→ ディビジョン → 試合」で、逆順に取る経路を作らない(デッドロックを防ぐ)。

// lockStrengthNoKeyUpdate は FOR NO KEY UPDATE。チームや試合の INSERT が外部キーの検査で
// divisions の行に取る FOR KEY SHARE とは衝突させず、同じディビジョンの操作どうしだけを待たせる。
const lockStrengthNoKeyUpdate = "NO KEY UPDATE"

// lockDivision はディビジョンの行をトランザクションの終わりまでロックする。無ければ NotFound。
func lockDivision(ctx context.Context, tx *gorm.DB, divisionID int64) error {
	var ids []int64
	err := tx.WithContext(ctx).Model(&Division{}).
		Clauses(clause.Locking{Strength: lockStrengthNoKeyUpdate}).
		Where(sqlByID, divisionID).
		Pluck("id", &ids).
		Error
	if err != nil {
		return fmt.Errorf("lock division %d: %w", divisionID, err)
	}
	if len(ids) == 0 {
		return domain.NotFound(domain.ResourceDivision, divisionID)
	}
	return nil
}

// lockTournament は大会の行をロックする。形の適用(グループ文字の通し番号と、他のディビジョンの
// 試合を避けた時間割を大会単位で決める)を同じ大会の中で直列にする。
func lockTournament(ctx context.Context, tx *gorm.DB, tournamentID int64) error {
	var ids []int64
	err := tx.WithContext(ctx).Model(&Tournament{}).
		Clauses(clause.Locking{Strength: lockStrengthNoKeyUpdate}).
		Where(sqlByID, tournamentID).
		Pluck("id", &ids).
		Error
	if err != nil {
		return fmt.Errorf("lock tournament %d: %w", tournamentID, err)
	}
	if len(ids) == 0 {
		return domain.NotFound(domain.ResourceTournament, tournamentID)
	}
	return nil
}

// lockMatchInDivision は試合のディビジョンをロックしてから試合の行をロックする(順序は
// 「ディビジョン → 試合」)。
func lockMatchInDivision(ctx context.Context, tx *gorm.DB, matchID int64) (Match, error) {
	match, err := findByID[Match](ctx, tx, matchID, domain.ResourceMatch)
	if err != nil {
		return Match{}, err
	}
	if err := lockDivision(ctx, tx, match.DivisionID); err != nil {
		return Match{}, err
	}
	locked, err := lockMatch(ctx, tx, matchID)
	if err != nil {
		return Match{}, err
	}
	if locked.DivisionID != match.DivisionID {
		// 試合のディビジョンは作った後に変わらない(管理画面の試合の更新も division_id を書かない)
		return Match{}, fmt.Errorf("%w: 試合 %d のディビジョンが変わりました", domain.ErrConflict, matchID)
	}
	return locked, nil
}
