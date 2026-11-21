// Package config は環境変数から実行時の設定を読む。
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// minInternalTokenLen は内部トークンの最小長(バイト)。推測で突破されない長さとして
// 32 文字(16 進で 128 bit 相当)を下限にする。
const minInternalTokenLen = 32

// placeholderMarker は以前の .env.example にあった内部トークンの例の目印。リポジトリを読んだ人が
// 知っている値なので、手元の開発でも使わせない(.env.example は空にし、生成の手順を書いた)。
const placeholderMarker = "change-me"

// API は api サーバーの設定。
type API struct {
	DatabaseURL   string
	InternalToken string
	Port          string
	// SiteURL は利用者が開くサイトの URL(公式 LINE の返信に入れる本人用のリンクの前半)
	SiteURL string
	LINE    LINE
	// LINEProblem は LINE の設定が足りないときの理由。そのとき LINE は空にして、Webhook を受けずに起動する
	// (LINE の入れ忘れで、観戦と記録係まで止めないため。呼び出し側がエラーとしてログに残す)
	LINEProblem error
}

// LINE は公式アカウントの Messaging API の設定(docs/line-integration.md)。ChannelID と ChannelSecret の
// 両方があるときだけ、Webhook の受け口を作る。BasicID は公式アカウントのベーシック ID(@ で始まる)で、
// メンバーの招待リンクに使う(Webhook を受けるなら必須。無いとメンバーを招待する手段が無くなる)。
type LINE struct {
	ChannelID     string
	ChannelSecret string
	BasicID       string
	// APIBaseURL は Messaging API の URL。手元で偽の LINE を立てて確かめるときだけ設定する(空なら本物)
	APIBaseURL string
}

// Enabled は公式アカウントの Webhook を受けるか。
func (l LINE) Enabled() bool {
	return l.ChannelID != "" && l.ChannelSecret != ""
}

// ErrMissingEnv は必須の環境変数がないことを表す。
var ErrMissingEnv = errors.New("missing environment variable")

// LoadAPI は環境変数から API の設定を読む。必須の値がなければエラーを返し、既定値で起動しない。
func LoadAPI() (API, error) {
	cfg := API{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		InternalToken: os.Getenv("INTERNAL_API_TOKEN"),
		Port:          os.Getenv("PORT"),
		SiteURL:       os.Getenv("PUBLIC_SITE_URL"),
		LINE: LINE{
			ChannelID:     os.Getenv("LINE_MESSAGING_CHANNEL_ID"),
			ChannelSecret: os.Getenv("LINE_MESSAGING_CHANNEL_SECRET"),
			BasicID:       os.Getenv("LINE_BASIC_ID"),
			APIBaseURL:    os.Getenv("LINE_API_BASE_URL"),
		},
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
	if strings.Contains(strings.ToLower(cfg.InternalToken), placeholderMarker) {
		return API{}, fmt.Errorf(
			"%w: INTERNAL_API_TOKEN is the placeholder from .env.example; generate a random value",
			ErrMissingEnv,
		)
	}
	if err := cfg.LINE.check(cfg.SiteURL); err != nil {
		cfg.LINE, cfg.LINEProblem = LINE{}, err
	}
	if cfg.Port == "" {
		// Vercel は PORT を渡す。手元の開発では web の .env.example と揃えて 8080 を使う
		cfg.Port = "8080"
	}
	return cfg, nil
}

// check は LINE の設定の組み合わせを確かめる。足りない設定は入れ忘れなので、半端なまま動かさない。
func (l LINE) check(siteURL string) error {
	if (l.ChannelID == "") != (l.ChannelSecret == "") {
		return fmt.Errorf("%w: set both LINE_MESSAGING_CHANNEL_ID and LINE_MESSAGING_CHANNEL_SECRET",
			ErrMissingEnv)
	}
	if l.Enabled() && !strings.HasPrefix(l.BasicID, "@") {
		return fmt.Errorf("%w: LINE_BASIC_ID must be the basic ID of the official account (starts with @)",
			ErrMissingEnv)
	}
	if l.Enabled() && !strings.HasPrefix(siteURL, "https://") && !strings.HasPrefix(siteURL, "http://") {
		return fmt.Errorf("%w: PUBLIC_SITE_URL must be an http(s) URL to send entry links", ErrMissingEnv)
	}
	return nil
}
