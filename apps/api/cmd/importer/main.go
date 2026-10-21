// Command importer は参照サイト score.clubjr.com から大会のページを取得してパースし、DB に書く。
//
//	cd apps/api && DATABASE_URL=postgres://... go run ./cmd/importer -tournament 525,440
//	cd apps/api && DATABASE_URL=postgres://... go run ./cmd/importer -representative
//	cd apps/api && go run ./cmd/importer -tournament 525 -dry-run > /tmp/525.json
//
// 取得したページは -cache-dir(既定はリポジトリ直下の .local/import-cache。git worktree の中で
// 実行しても本体のリポジトリの .local を使う)に保存し、キャッシュにあるページは取り直さない。
// DB には 1 大会を 1 トランザクションで書き、同じ大会 ID の既存データは消してから入れる。
// 取り込んだデータは実在の個人名を含むので、開発用 DB にだけ入れる。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"scoreview/api/internal/importer"
	"scoreview/api/internal/importer/fetch"
	"scoreview/api/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "importer:", err)
		os.Exit(1)
	}
}

type options struct {
	ids         []int64
	dryRun      bool
	cacheDir    string
	minInterval time.Duration
	contact     string
	players     bool
	offline     bool
	quiet       bool
}

func parseFlags() (options, error) {
	var (
		opts        options
		tournaments = flag.String("tournament", "",
			"取り込む大会の ID(参照サイトの /scview/{ID})。カンマ区切りで複数")
		representative = flag.Bool("representative", false,
			"docs/representative-tournaments.md の 16 大会を取り込む")
	)
	flag.BoolVar(&opts.dryRun, "dry-run", false, "DB に書かず、中間表現を JSON で標準出力に書く")
	flag.StringVar(&opts.cacheDir, "cache-dir", "",
		"キャッシュの場所(既定はリポジトリ直下の "+fetch.DefaultCacheSubdir+")")
	flag.DurationVar(&opts.minInterval, "min-interval", fetch.DefaultMinInterval,
		"リクエストの最小間隔(1s 以上)")
	flag.StringVar(&opts.contact, "contact", "", "User-Agent に付ける連絡先(既定は付けない)")
	flag.BoolVar(&opts.players, "players", false, "選手ページも取る(グレイテストと成績表の列構成用)")
	flag.BoolVar(&opts.offline, "offline", false, "キャッシュだけを読み、参照サイトに接続しない")
	flag.BoolVar(&opts.quiet, "quiet", false, "取得したページの URL を標準エラーに出さない")
	flag.Parse()

	if *representative {
		opts.ids = importer.RepresentativeTournaments()
	}
	ids, err := importer.ParseTournamentIDs(*tournaments)
	if err != nil {
		return opts, fmt.Errorf("-tournament: %w", err)
	}
	opts.ids = append(opts.ids, ids...)
	if len(opts.ids) == 0 {
		return opts, errors.New("-tournament or -representative is required")
	}
	return opts, nil
}

func run() error {
	opts, err := parseFlags()
	if err != nil {
		return err
	}
	dir := opts.cacheDir
	if dir == "" {
		if dir, err = fetch.DefaultCacheDir(); err != nil {
			return err
		}
	}
	cache, err := fetch.NewCache(dir)
	if err != nil {
		return err
	}
	var fetcher fetch.Fetcher = fetch.Offline{Cache: cache}
	var client *fetch.Client
	if !opts.offline {
		client, err = fetch.NewClient(cache,
			fetch.WithMinInterval(opts.minInterval), fetch.WithContact(opts.contact))
		if err != nil {
			return err
		}
		fetcher = client
	}
	imOpts := []importer.Option{importer.WithFetcher(fetcher), importer.WithPlayers(opts.players)}
	if !opts.quiet {
		imOpts = append(imOpts, importer.WithProgress(os.Stderr))
	}
	im, err := importer.New(imOpts...)
	if err != nil {
		return err
	}

	var sink importer.Sink = importer.JSONSink{W: os.Stdout}
	if !opts.dryRun {
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			return errors.New("DATABASE_URL is not set (use -dry-run to print JSON instead)")
		}
		db, err := store.Open(dsn)
		if err != nil {
			return err
		}
		sink = importer.DBSink{DB: db, Log: os.Stderr}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	requests := func() int {
		if client == nil {
			return 0
		}
		return client.Requests()
	}
	var failed []string
	started := time.Now()
	for _, id := range opts.ids {
		t0, r0 := time.Now(), requests()
		err := importOne(ctx, im, sink, id)
		fmt.Fprintf(os.Stderr,
			"importer: tournament %d done in %s, %d requests to the reference site\n",
			id, time.Since(t0).Round(time.Millisecond), requests()-r0)
		if err != nil {
			if ctx.Err() != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "importer: tournament %d failed: %v\n", id, err)
			failed = append(failed, fmt.Sprint(id))
		}
	}
	fmt.Fprintf(os.Stderr, "importer: %d tournaments in %s, %d requests to the reference site\n",
		len(opts.ids), time.Since(started).Round(time.Millisecond), requests())
	if len(failed) > 0 {
		return fmt.Errorf("failed tournaments: %s", strings.Join(failed, ","))
	}
	return nil
}

func importOne(ctx context.Context, im *importer.Importer, sink importer.Sink, id int64) error {
	bundle, err := im.Collect(ctx, id)
	if err != nil {
		return err
	}
	if len(bundle.Missing) > 0 {
		fmt.Fprintf(os.Stderr,
			"importer: tournament %d: %d pages are missing on the reference site\n",
			id, len(bundle.Missing))
	}
	return sink.Write(ctx, bundle)
}
