// Command api は Vercel の Go サービスのエントリポイント。PORT で HTTP を待ち受ける。
package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"scoreview/api/internal/config"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/httpapi"
	"scoreview/api/internal/line"
	"scoreview/api/internal/notify"
	"scoreview/api/internal/store"
	"scoreview/api/internal/talk"
	"scoreview/api/internal/viewing"
)

// readHeaderTimeout はヘッダー受信の上限。Slowloris 型の接続占有を防ぐ。
const readHeaderTimeout = 10 * time.Second

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.LoadAPI()
	if err != nil {
		return err
	}
	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	// 観客の取り直しが重なる掲示板は、大会ごとに短いあいだ使い回す(viewing.Cache)
	boards := viewing.NewCache(viewing.NewService(store.NewViewingSource(db)), viewing.DefaultCacheTTL, time.Now)
	entryOpts := []entry.Option{entry.WithInviteURL(line.InviteURL(cfg.LINE.BasicID))}
	var notifyOpts []notify.Option
	var client *line.Client
	lines := store.NewLineStore(db)
	if cfg.LINE.Enabled() {
		base := cfg.LINE.APIBaseURL
		if base == "" {
			base = line.DefaultBaseURL
		}
		client = line.NewClient(cfg.LINE.ChannelID, cfg.LINE.ChannelSecret, base, time.Now)
		// 代表への知らせは申込の操作とは別なので、送れなくても操作は止めず、エラーとしてログに残す
		report := func(err error) { log.Error("line notice failed", "err", err) }
		entryOpts = append(entryOpts, entry.WithNotifier(talk.NewNotifier(lines, client, cfg.SiteURL), report))
		// 管理画面から送った結果は記録と画面に残るので、ここではログに残すだけ
		reportSend := func(err error) { log.Error("line notification failed", "err", err) }
		notifyOpts = append(notifyOpts, notify.WithSender(client, reportSend))
	}
	entries := entry.NewService(store.NewEntryStore(db), entryOpts...)
	notifyOpts = append(notifyOpts, notify.WithSiteURL(cfg.SiteURL),
		notify.WithInviteURL(line.InviteURL(cfg.LINE.BasicID)))
	notices := notify.NewService(store.NewNotifyStore(db), notifyOpts...)
	opts := []httpapi.Option{httpapi.WithLogger(log), httpapi.WithBoards(boards), httpapi.WithEntryDesk(entries),
		httpapi.WithNotifications(notices)}
	if client != nil {
		bot := talk.NewBot(lines, entries, client, cfg.SiteURL)
		opts = append(opts, httpapi.WithLineWebhook(line.NewWebhook(cfg.LINE.ChannelSecret, bot)))
	}
	if cfg.LINEProblem != nil {
		log.Error("line webhook disabled: the LINE settings are incomplete", "err", cfg.LINEProblem)
	}
	log.Info("line webhook", "enabled", cfg.LINE.Enabled())
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewServer(db, cfg.InternalToken, opts...),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Info("api listening", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
