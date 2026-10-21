// Command devdb は開発用の PostgreSQL を起動し、マイグレーションを当てて Ctrl+C まで待つ。
// データはリポジトリ直下の .local/devdb に残るので、再起動しても取込済みのデータは消えない。
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql の "pgx" ドライバを登録する

	"scoreview/api/internal/store"
)

// devPort は開発用 DB のポート。.env.example の DATABASE_URL と揃える。
// 既定の 5432 を避けるのは、手元に別の PostgreSQL がある場合に衝突しないため。
const devPort = 54329

const devDBName = "scoreview"

func main() {
	dataDir := flag.String("data", "../../.local/devdb", "PostgreSQL のデータディレクトリ(apps/api からの相対パス)")
	flag.Parse()
	if err := run(*dataDir); err != nil {
		fmt.Fprintln(os.Stderr, "devdb:", err)
		os.Exit(1)
	}
}

func run(dataDir string) error {
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("user cache dir: %w", err)
	}
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(devPort).
		DataPath(filepath.Join(abs, "data")).
		RuntimePath(filepath.Join(abs, "runtime")).
		CachePath(filepath.Join(cacheDir, "scoreview-embedded-postgres")))
	if err := pg.Start(); err != nil {
		return fmt.Errorf("start postgres: %w", err)
	}
	defer func() { _ = pg.Stop() }()

	if err := ensureDatabase(); err != nil {
		return err
	}
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", devPort, devDBName)
	db, err := store.Open(dsn)
	if err != nil {
		return err
	}
	if err := store.Migrate(context.Background(), db); err != nil {
		return err
	}
	fmt.Println("devdb ready:", dsn)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	return nil
}

// ensureDatabase は開発用 DB がなければ作る。DB 名は定数で外部入力は入らない。
func ensureDatabase() error {
	conn, err := sql.Open("pgx", fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", devPort))
	if err != nil {
		return fmt.Errorf("connect admin db: %w", err)
	}
	defer func() { _ = conn.Close() }()
	var exists bool
	err = conn.QueryRowContext(context.Background(),
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", devDBName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	if exists {
		return nil
	}
	_, err = conn.ExecContext(context.Background(), "CREATE DATABASE "+pgx.Identifier{devDBName}.Sanitize())
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("create database: %w", err)
	}
	return nil
}
