// Package config は環境変数から実行時の設定を読む。
package config

import (
	"errors"
	"fmt"
	"os"
)

// minInternalTokenLen は内部トークンの最小長(バイト)。推測で突破されない長さとして
// 32 文字(16 進で 128 bit 相当)を下限にする。
const minInternalTokenLen = 32

// API は api サーバーの設定。
type API struct {
	DatabaseURL   string
	InternalToken string
	Port          string
}

// ErrMissingEnv は必須の環境変数がないことを表す。
var ErrMissingEnv = errors.New("missing environment variable")

// LoadAPI は環境変数から API の設定を読む。必須の値がなければエラーを返し、既定値で起動しない。
func LoadAPI() (API, error) {
	cfg := API{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		InternalToken: os.Getenv("INTERNAL_API_TOKEN"),
		Port:          os.Getenv("PORT"),
	}
	if cfg.DatabaseURL == "" {
		return API{}, fmt.Errorf("%w: DATABASE_URL", ErrMissingEnv)
	}
	if len(cfg.InternalToken) < minInternalTokenLen {
		return API{}, fmt.Errorf(
			"%w: INTERNAL_API_TOKEN must be at least %d characters",
			ErrMissingEnv, minInternalTokenLen,
		)
	}
	if cfg.Port == "" {
		// Vercel は PORT を渡す。手元の開発では web の .env.example と揃えて 8080 を使う
		cfg.Port = "8080"
	}
	return cfg, nil
}
