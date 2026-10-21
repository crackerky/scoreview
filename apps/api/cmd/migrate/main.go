// Command migrate は DATABASE_URL の DB にマイグレーションを最新まで適用する。
package main

import (
	"context"
	"fmt"
	"os"

	"scoreview/api/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	db, err := store.Open(dsn)
	if err != nil {
		return err
	}
	return store.Migrate(context.Background(), db)
}
