// Package httpapi は HTTP の薄いアダプタ層。リクエストの検証、store の呼び出し、DTO への変換だけを行う。
package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/bracket"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/notify"
	"scoreview/api/internal/sanitize"
	"scoreview/api/internal/store"
	"scoreview/api/internal/viewing"
)

// Server は api の HTTP ハンドラ一式。
type Server struct {
	db            *gorm.DB
	internalToken string
	log           *slog.Logger
	mux           *http.ServeMux
	sanitizer     *sanitize.Sanitizer
	brackets      *bracket.Registry
	passwords     *auth.PasswordGate
	now           func() time.Time
	// boards は公開のスコアサイトの掲示板の読み方。ハンドラはこの口だけを使い、DB を直接読まない
	boards viewing.Reader
	// entries は大会への申込の受付。決まりは entry にあり、ハンドラはこの口だけを使う
	entries entry.Desk
	// notices は管理画面から公式アカウントで送る口。決まりは notify にある
	notices notify.Desk
	// line は公式アカウントの Webhook の受け口。LINE の設定が無ければ nil で、受け口を作らない
	line LineReceiver
}

// Option は Server の構成を変える Functional Option。
type Option func(*Server)

// WithLogger はログの出力先を差し替える。
func WithLogger(log *slog.Logger) Option {
	return func(s *Server) { s.log = log }
}

// WithClock は現在時刻の取り方を差し替える(セッションの期限と試合の経過時間のテスト用)。
func WithClock(now func() time.Time) Option {
	return func(s *Server) { s.now = now }
}

// WithBracketTemplates はトーナメント表のテンプレートを差し替える(既定は bracket.Default)。
func WithBracketTemplates(registry *bracket.Registry) Option {
	return func(s *Server) { s.brackets = registry }
}

// WithBoards は掲示板の読み方を差し替える(本番は使い回しを挟む。既定は毎回 DB から読む)。
func WithBoards(boards viewing.Reader) Option {
	return func(s *Server) { s.boards = boards }
}

// WithEntryDesk は申込の受付を渡す。本番は招待リンクの作り方を持たせた entry.Service を渡す(渡さなければ
// DB で読み書きする entry.Service で、招待リンクは空になる)。
func WithEntryDesk(desk entry.Desk) Option {
	return func(s *Server) { s.entries = desk }
}

// WithNotifications は管理画面から送る口を渡す。本番は LINE の送り方を持たせた notify.Service を渡す
// (渡さなければ DB で宛先を読む notify.Service で、送れない)。
func WithNotifications(desk notify.Desk) Option {
	return func(s *Server) { s.notices = desk }
}

// WithPasswordGate はパスワード照合の同時実行数の制限を差し替える(混雑時の 503 のテスト用)。
func WithPasswordGate(gate *auth.PasswordGate) Option {
	return func(s *Server) { s.passwords = gate }
}

// NewServer はルートを登録した Server を返す。internalToken は web との共有秘密で、
// 全リクエストの X-Internal-Token と定数時間で比較する。
func NewServer(db *gorm.DB, internalToken string, opts ...Option) *Server {
	s := &Server{
		db: db, internalToken: internalToken, log: slog.Default(), mux: http.NewServeMux(),
		sanitizer: sanitize.New(), brackets: bracket.Default(), now: time.Now,
		passwords: auth.NewPasswordGate(auth.MaxConcurrentPasswordChecks, auth.PasswordCheckWait),
		boards:    viewing.NewService(store.NewViewingSource(db)),
		entries:   entry.NewService(store.NewEntryStore(db)),
		notices:   notify.NewService(store.NewNotifyStore(db)),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.browseRoutes()
	s.authRoutes()
	s.adminRoutes()
	s.formatRoutes()
	s.dayOfRoutes()
	s.liveTablesRoutes()
	s.boardRoutes()
	s.entryRoutes()
	s.entryLinkRoutes()
	s.notificationRoutes()
	s.lineRoutes()
}

// ServeHTTP はパニック回復と内部トークン検証を通してからルートへ渡す。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			s.log.Error("panic", "path", r.URL.Path, "recovered", rec)
			writeError(w, s.log, errInternal)
		}
	}()
	got := r.Header.Get("X-Internal-Token")
	if subtle.ConstantTimeCompare([]byte(got), []byte(s.internalToken)) != 1 {
		writeError(w, s.log, domain.ErrUnauthorized)
		return
	}
	s.mux.ServeHTTP(w, r)
}

var errInternal = errors.New("internal")

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// writeError はエラーを HTTP ステータスと {error:{code,message}} に変換する唯一の場所。
func writeError(w http.ResponseWriter, log *slog.Logger, err error) {
	var body errorBody
	status := http.StatusInternalServerError
	body.Error.Code = "internal"
	body.Error.Message = "internal error"

	var notFound *domain.NotFoundError
	var invalid *domain.ValidationError
	var mismatch *domain.ScoreMismatchError
	var retry *domain.RetryAfterError
	if errors.As(err, &retry) {
		w.Header().Set("Retry-After", retryAfterSeconds(retry.After))
	}
	switch {
	case errors.As(err, &mismatch):
		// 記録係の画面が「最新のスコアで確認し直す」を出せるよう、一般の conflict と分ける
		status, body.Error.Code, body.Error.Message = http.StatusConflict, "score_mismatch", mismatch.Error()
	case errors.As(err, &notFound):
		status, body.Error.Code, body.Error.Message = http.StatusNotFound, "not_found", notFound.Error()
	case errors.As(err, &invalid):
		status, body.Error.Code, body.Error.Message = http.StatusBadRequest, "bad_request", invalid.Error()
	case errors.Is(err, domain.ErrBadRequest):
		status, body.Error.Code, body.Error.Message = http.StatusBadRequest, "bad_request", "bad request"
	case errors.Is(err, domain.ErrUnauthorized):
		status, body.Error.Code, body.Error.Message = http.StatusUnauthorized, "unauthorized", "unauthorized"
	case errors.Is(err, domain.ErrForbidden):
		status, body.Error.Code, body.Error.Message = http.StatusForbidden, "forbidden", "forbidden"
	case errors.Is(err, domain.ErrConflict):
		status, body.Error.Code, body.Error.Message = http.StatusConflict, "conflict", err.Error()
	case errors.Is(err, domain.ErrTooManyRequests):
		status, body.Error.Code, body.Error.Message = http.StatusTooManyRequests, "too_many_requests",
			"too many login attempts; try again later"
	case errors.Is(err, domain.ErrUnavailable):
		status, body.Error.Code, body.Error.Message = http.StatusServiceUnavailable, "unavailable",
			"server is busy; try again shortly"
	default:
		// 内部エラーの詳細はログにだけ出し、応答には載せない
		log.Error("internal error", "err", err)
	}
	writeJSON(w, status, body)
}

// retryAfterSeconds は Retry-After の値(整数の秒。RFC 9110 10.2.3)。端数は切り上げ、1 秒以上にする。
func retryAfterSeconds(after time.Duration) string {
	return strconv.Itoa(max(1, int(math.Ceil(after.Seconds()))))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// エンコード失敗時はヘッダー送信後なので応答を直せない。接続断として扱われる
	_ = json.NewEncoder(w).Encode(v)
}
