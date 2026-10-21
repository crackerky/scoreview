// Command api は Vercel の Go サービスのエントリポイント。PORT で HTTP を待ち受ける。
package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"scoreview/api/internal/config"
	"scoreview/api/internal/httpapi"
	"scoreview/api/internal/store"
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
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewServer(db, cfg.InternalToken, httpapi.WithLogger(log)),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Info("api listening", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
