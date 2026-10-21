// Package store は GORM のモデルとリポジトリを持ち、domain の型へ変換して返す。
// GORM の Raw と Exec は使わない(golangci-lint の forbidigo で禁止)。条件は必ずプレースホルダで渡す。
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scoreview/api/migrations"
)

// Open は PostgreSQL へ接続する。
func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		// 明示しない限り関連を書き込まない。意図しない INSERT/UPDATE を防ぐ
		FullSaveAssociations: false,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// Migrate は埋め込んだ goose の SQL を最新まで適用する。
func Migrate(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// IsNotFound は GORM の「行がない」エラーかどうかを返す。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
