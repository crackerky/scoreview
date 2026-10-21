// Command seed は架空の大会データを DATABASE_URL の DB に投入する。実在の人名、チーム名は使わない。
// 投入する内容は決定的(乱数を使わない)なので、空の DB に入れれば ID も毎回同じになる。
//
//	cd apps/api && DATABASE_URL=postgres://... go run ./cmd/seed [-replace]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"scoreview/api/internal/store"
)

func main() {
	replace := flag.Bool("replace", false, "既存の大会(と配下のデータ)を削除してから投入する")
	flag.Parse()
	if err := run(context.Background(), *replace); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, replace bool) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	db, err := store.Open(dsn)
	if err != nil {
		return err
	}
	summary, err := seed(ctx, db, replace)
	if err != nil {
		return err
	}
	for _, line := range summary {
		fmt.Println(line)
	}
	return nil
}
