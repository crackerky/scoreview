// Package store は GORM のモデルとリポジトリを持ち、domain の型へ変換して返す。
// GORM の Raw と Exec は使わない(golangci-lint の forbidigo で禁止)。条件は必ずプレースホルダで渡す。
package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scoreview/api/migrations"
)

// Open は PostgreSQL へ接続する。
func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: NewLogger(os.Stdout),
		// 明示しない限り関連を書き込まない。意図しない INSERT/UPDATE を防ぐ
		FullSaveAssociations: false,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// slowQueryThreshold は遅い問い合わせとして警告を出す時間。GORM の既定(logger.Default)と同じ値。
const slowQueryThreshold = 200 * time.Millisecond

// NewLogger は GORM のログを w に書くロガーを返す。警告(遅い問い合わせ)とエラーだけを出し、
// SQL の値はプレースホルダのまま出す(ParameterizedQueries)。値にはパスワードのハッシュ、
// メールアドレス、選手の実名が入るので、ログを読める人に見せない。
func NewLogger(w io.Writer) logger.Interface {
	return logger.New(log.New(w, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:        slowQueryThreshold,
		LogLevel:             logger.Warn,
		ParameterizedQueries: true,
		Colorful:             false,
	})
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
	// 並行する作業がマイグレーションの番号を帯で分けて使う(00003〜 と 00010〜)ので、後から入った
	// 小さい番号も当てる。既定の goose は適用済みより小さい未適用の番号があるとエラーにする
	if err := goose.UpContext(ctx, sqlDB, ".", goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// IsNotFound は GORM の「行がない」エラーかどうかを返す。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
