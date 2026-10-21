// Command importcheck は取り込んだ大会を api から読み、参照サイトの HTML(取込キャッシュ)と
// 突き合わせる。テストではなく、開発用 DB に実データを取り込んだ後に手元で実行する確認用の道具。
//
//	cd apps/api && INTERNAL_API_TOKEN=... go run ./cmd/importcheck -representative
//	cd apps/api && INTERNAL_API_TOKEN=... go run ./cmd/importcheck -tournament 525 \
//	  -api http://localhost:8081
//
// 閲覧用の全エンドポイント(大会トップ、チーム一覧、スケジュール、ディビジョン成績、グループ、
// チーム、試合、選手、ギャラリー)を叩いて 200 で返るかを数え、大会名、一覧の区分、グループ数、
// チーム数、試合数とスコア、順位表の 1〜3 位、プールの順位、試合ページのスコアと得点経過を
// キャッシュの HTML と比べる。参照サイトには接続しない(キャッシュにないページがあれば失敗する)。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"scoreview/api/internal/importer"
	"scoreview/api/internal/importer/fetch"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "importcheck:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		tournaments    = flag.String("tournament", "", "確かめる大会の ID。カンマ区切りで複数")
		representative = flag.Bool("representative", false,
			"docs/representative-tournaments.md の 16 大会を確かめる")
		apiBase  = flag.String("api", "http://localhost:8080", "api のベース URL")
		cacheDir = flag.String("cache-dir", "",
			"取込キャッシュの場所(既定はリポジトリ直下の "+fetch.DefaultCacheSubdir+")")
		players = flag.Bool("players", true, "選手ページのエンドポイントも叩く")
	)
	flag.Parse()
	token := os.Getenv("INTERNAL_API_TOKEN")
	if token == "" {
		return errors.New("INTERNAL_API_TOKEN is not set")
	}
	var ids []int64
	if *representative {
		ids = importer.RepresentativeTournaments()
	}
	more, err := importer.ParseTournamentIDs(*tournaments)
	if err != nil {
		return fmt.Errorf("-tournament: %w", err)
	}
	ids = append(ids, more...)
	if len(ids) == 0 {
		return errors.New("-tournament or -representative is required")
	}
	dir := *cacheDir
	if dir == "" {
		if dir, err = fetch.DefaultCacheDir(); err != nil {
			return err
		}
	}
	cache, err := fetch.NewCache(dir)
	if err != nil {
		return err
	}
	im, err := importer.New(importer.WithFetcher(fetch.Offline{Cache: cache}))
	if err != nil {
		return err
	}
	client := newAPIClient(*apiBase, token)
	ctx := context.Background()
	list, err := client.tournamentList(ctx)
	if err != nil {
		return err
	}

	failures := 0
	for _, id := range ids {
		failures += checkTournament(ctx, im, client, list, id, *players)
	}
	if failures > 0 {
		return fmt.Errorf("%d mismatches or failed requests", failures)
	}
	fmt.Println("all checks passed")
	return nil
}

// checkTournament は 1 大会を確かめて結果を標準出力に書き、不一致と失敗したリクエストの数を返す。
func checkTournament(
	ctx context.Context, im *importer.Importer, client *apiClient, list *listDTO,
	id int64, players bool,
) int {
	started := time.Now()
	bundle, err := im.Collect(ctx, id)
	if err != nil {
		fmt.Printf("== %d: キャッシュから読めない: %v\n", id, err)
		return 1
	}
	c := &checker{api: client, b: bundle, listed: list, players: players}
	c.run(ctx)
	fmt.Printf("== %d %s (%s)\n",
		id, bundle.Top.Name.JA, time.Since(started).Round(time.Millisecond))
	fmt.Printf("   endpoints: %d requests, %d not 200\n", c.requests, len(c.non200))
	failures := len(c.non200)
	for _, s := range c.non200 {
		fmt.Printf("   NG http %s\n", s)
	}
	for _, r := range c.results {
		mark := "OK"
		if !r.ok {
			mark = "NG"
			failures++
		}
		fmt.Printf("   %s %s: %s\n", mark, r.name, r.detail)
	}
	if len(c.siteInconsistent) > 0 {
		fmt.Printf("   info 参照サイトでも得点経過とスコアが食い違う試合: %s\n",
			strings.Join(c.siteInconsistent, ", "))
	}
	return failures
}
