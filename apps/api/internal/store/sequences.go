package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// GeneratedIDFloor は自動採番する ID の下限。取込ツールは参照サイトの ID(2026 年時点で最大は
// 選手 ID の 6 桁台)をそのまま主キーに入れ、参照サイトに ID がない行(未試合の試合、ダミーの
// ディビジョン、個人戦のエントリー)と管理画面からの登録は自動採番する。両者が衝突しないように、
// 自動採番はこの値以上から始める。
const GeneratedIDFloor = 100_000_000

// sqlReserveSequence はシーケンスを「表の最大 ID」「シーケンスの現在値」「下限 - 1」の最大値に
// 進める(is_called=true なので次の nextval はその値 + 1)。現在値の方が大きければそのまま。
// 表名は ? で渡すので SQL に連結しない。
const sqlReserveSequence = "setval(pg_get_serial_sequence(?, 'id'), GREATEST(" +
	"COALESCE(MAX(id), 0), " +
	"COALESCE(pg_sequence_last_value(pg_get_serial_sequence(?, 'id')::regclass), 0), " +
	"?), true)"

// sequenceTables は BIGSERIAL の id を持つ表。migrations の CREATE TABLE と揃える。
func sequenceTables() []string {
	return []string{
		"tournaments", "tournament_banners", "divisions", "groups", "teams", "players",
		"matches", "match_events", "galleries", "gallery_photos", "users", "sessions",
	}
}

// ReserveGeneratedIDRange は全表の id のシーケンスを、次の自動採番が GeneratedIDFloor 以上で、
// かつ既存の最大 ID より大きくなるように進める。何度呼んでもよい(値を戻さない)。
// 取込ツールが参照サイトの ID で INSERT した後と、空の DB を作った後に呼ぶ。
func ReserveGeneratedIDRange(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, table := range sequenceTables() {
			var value int64
			err := tx.Table(table).
				Select(sqlReserveSequence, table, table, GeneratedIDFloor-1).
				Scan(&value).
				Error
			if err != nil {
				return fmt.Errorf("reserve id range of %s: %w", table, err)
			}
		}
		return nil
	})
}
