// Package httpapi は HTTP の薄いアダプタ層。リクエストの検証、store の呼び出し、DTO への変換だけを行う。
package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/bracket"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/sanitize"
)

// Server は api の HTTP ハンドラ一式。
type Server struct {
	db            *gorm.DB
	internalToken string
	log           *slog.Logger
	mux           *http.ServeMux
	sanitizer     *sanitize.Sanitizer
	brackets      *bracket.Registry
	now           func() time.Time
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

// NewServer はルートを登録した Server を返す。internalToken は web との共有秘密で、
// 全リクエストの X-Internal-Token と定数時間で比較する。
func NewServer(db *gorm.DB, internalToken string, opts ...Option) *Server {
	s := &Server{
		db: db, internalToken: internalToken, log: slog.Default(), mux: http.NewServeMux(),
		sanitizer: sanitize.New(), brackets: bracket.Default(), now: time.Now,
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
	default:
		// 内部エラーの詳細はログにだけ出し、応答には載せない
		log.Error("internal error", "err", err)
	}
	writeJSON(w, status, body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// エンコード失敗時はヘッダー送信後なので応答を直せない。接続断として扱われる
	_ = json.NewEncoder(w).Encode(v)
}
