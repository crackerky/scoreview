// Command importer は参照サイト score.clubjr.com から 1 大会分のページを取得してパースする。
//
//	cd apps/api && go run ./cmd/importer -tournament 525 -dry-run > /tmp/525.json
//
// 取得したページは -cache-dir(既定はリポジトリ直下の .local/import-cache)に保存し、次回からは
// 取り直さない。今は -dry-run(中間表現を JSON で標準出力に書く)だけを実装している。
// DB への書き込みは importer.Sink を store で実装して差し込む。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"scoreview/api/internal/importer"
	"scoreview/api/internal/importer/fetch"
)

// repoMarker はリポジトリ直下にあるファイル。既定のキャッシュ置き場を決めるために親をたどって探す。
const repoMarker = "pnpm-workspace.yaml"

// defaultCacheSubdir はリポジトリ直下からのキャッシュの場所(CLAUDE.md gotcha)。
const defaultCacheSubdir = ".local/import-cache"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "importer:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		tournament  = flag.Int64("tournament", 0, "取り込む大会の ID(参照サイトの /scview/{ID})")
		dryRun      = flag.Bool("dry-run", false, "DB に書かず、中間表現を JSON で標準出力に書く")
		cacheDir    = flag.String("cache-dir", "", "キャッシュの場所(既定はリポジトリ直下の "+defaultCacheSubdir+")")
		minInterval = flag.Duration("min-interval", fetch.DefaultMinInterval, "リクエストの最小間隔(1s 以上)")
		contact     = flag.String("contact", "", "User-Agent に付ける連絡先(既定は付けない)")
		players     = flag.Bool("players", false, "選手ページも取る(キャラハンとグレイテスト用)")
		offline     = flag.Bool("offline", false, "キャッシュだけを読み、参照サイトに接続しない")
		quiet       = flag.Bool("quiet", false, "取得したページの URL を標準エラーに出さない")
	)
	flag.Parse()
	if *tournament <= 0 {
		return errors.New("-tournament is required")
	}
	if !*dryRun {
		return errors.New("DB への書き込みはまだない。-dry-run を付けて実行する")
	}

	dir := *cacheDir
	if dir == "" {
		root, err := findRepoRoot()
		if err != nil {
			return err
		}
		dir = filepath.Join(root, defaultCacheSubdir)
	}
	cache, err := fetch.NewCache(dir)
	if err != nil {
		return err
	}
	var fetcher fetch.Fetcher = fetch.Offline{Cache: cache}
	if !*offline {
		fetcher, err = fetch.NewClient(cache, fetch.WithMinInterval(*minInterval), fetch.WithContact(*contact))
		if err != nil {
			return err
		}
	}
	opts := []importer.Option{importer.WithFetcher(fetcher), importer.WithPlayers(*players)}
	if !*quiet {
		opts = append(opts, importer.WithProgress(os.Stderr))
	}
	im, err := importer.New(opts...)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	started := time.Now()
	bundle, err := im.Collect(ctx, *tournament)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "importer: tournament %d collected in %s (missing pages: %d)\n",
		*tournament, time.Since(started).Round(time.Millisecond), len(bundle.Missing))
	var sink importer.Sink = importer.JSONSink{W: os.Stdout}
	return sink.Write(ctx, bundle)
}

// findRepoRoot はカレントから親をたどって repoMarker のあるディレクトリを返す。
func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, repoMarker)); err == nil {
			return dir, nil
		}
		if dir == filepath.Dir(dir) {
			return "", fmt.Errorf("%s not found above %s; pass -cache-dir", repoMarker, wd)
		}
	}
}
