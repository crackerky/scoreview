// Package testdb はテスト用に embedded-postgres を起動し、マイグレーション済みの空 DB を配る。
//
// 使い方:
//
//	func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
//	func TestX(t *testing.T) { db := testdb.New(t) ... }
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql の "pgx" ドライバを登録する
	"gorm.io/gorm"

	"scoreview/api/internal/store"
)

const templateDB = "scoreview_template"

var (
	adminDSN string
	counter  atomic.Int64
)

// Run は PostgreSQL を起動してテンプレート DB にマイグレーションを当て、m.Run() の結果を返す。
func Run(m *testing.M) int {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "testdb:", err)
		return 1
	}
	return code
}

func run(m *testing.M) (int, error) {
	port, err := freePort()
	if err != nil {
		return 0, err
	}
	base, err := os.MkdirTemp("", "scoreview-testdb-")
	if err != nil {
		return 0, fmt.Errorf("make temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(base) }()

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return 0, fmt.Errorf("user cache dir: %w", err)
	}
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V17).
		Port(uint32(port)). //nolint:gosec // freePort は 1〜65535 の値しか返さない
		RuntimePath(filepath.Join(base, "runtime")).
		DataPath(filepath.Join(base, "data")).
		// バイナリのダウンロードはテストのたびに繰り返さない
		CachePath(filepath.Join(cacheDir, "scoreview-embedded-postgres")).
		Logger(nil))
	if err := pg.Start(); err != nil {
		return 0, fmt.Errorf("start postgres: %w", err)
	}
	defer func() { _ = pg.Stop() }()

	adminDSN = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", port)
	if err := createDatabase(templateDB, ""); err != nil {
		return 0, err
	}
	db, err := store.Open(dsnFor(templateDB))
	if err != nil {
		return 0, err
	}
	if err := store.Migrate(context.Background(), db); err != nil {
		return 0, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return 0, fmt.Errorf("get sql.DB: %w", err)
	}
	// テンプレートへの接続が残っていると CREATE DATABASE ... TEMPLATE が失敗する
	if err := sqlDB.Close(); err != nil {
		return 0, fmt.Errorf("close template connection: %w", err)
	}
	return m.Run(), nil
}

// New はマイグレーション済みの空 DB をテンプレートから複製して返す。テストごとに独立している。
func New(t *testing.T) *gorm.DB {
	t.Helper()
	if adminDSN == "" {
		t.Fatal("testdb: TestMain で testdb.Run を呼んでいない")
	}
	name := fmt.Sprintf("scoreview_test_%d", counter.Add(1))
	if err := createDatabase(name, templateDB); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(dsnFor(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func dsnFor(dbName string) string {
	cfg, err := pgx.ParseConfig(adminDSN)
	if err != nil {
		panic(err) // adminDSN はこのパッケージ内で組み立てた固定形式なので失敗しない
	}
	return fmt.Sprintf("postgres://postgres:postgres@%s:%d/%s?sslmode=disable", cfg.Host, cfg.Port, dbName)
}

// createDatabase は CREATE DATABASE を発行する。DB 名はプレースホルダで渡せないため、
// このパッケージ内で生成した名前だけを pgx.Identifier で引用して埋め込む(外部入力は入らない)。
func createDatabase(name, template string) error {
	conn, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return fmt.Errorf("connect admin db: %w", err)
	}
	defer func() { _ = conn.Close() }()
	//nolint:gosec // DB 名はこのパッケージで生成し、pgx.Identifier で引用済み
	stmt := "CREATE DATABASE " + pgx.Identifier{name}.Sanitize()
	if template != "" {
		stmt += " TEMPLATE " + pgx.Identifier{template}.Sanitize()
	}
	if _, err := conn.ExecContext(context.Background(), stmt); err != nil {
		return fmt.Errorf("create database %s: %w", name, err)
	}
	return nil
}

func freePort() (int, error) {
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("find free port: %w", err)
	}
	defer func() { _ = l.Close() }()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address %T", l.Addr())
	}
	return addr.Port, nil
}
