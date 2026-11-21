package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"scoreview/api/internal/domain"
)

// 公式 LINE の Webhook の受け口(docs/api.md「LINE からの申込」)。ログインは無く、LINE の署名で LINE から
// 届いたことを確かめる(line.Webhook)。web からの内部トークンの確認は ServeHTTP が行う。

// maxWebhookBody は Webhook の本文の上限。LINE は 1 回に数件のイベントをまとめて送るだけなので、
// 十分に大きく、それ以上は読まない。
const maxWebhookBody = 1 << 20

// LineReceiver は公式アカウントの Webhook の本文を受け取る。line.Webhook が実装する。
type LineReceiver interface {
	// Receive は本文の署名を確かめて処理する。署名が合わなければ domain.ErrUnauthorized
	Receive(ctx context.Context, body []byte, signature string) error
}

// WithLineWebhook は公式アカウントの Webhook の受け口を有効にする(LINE の設定が無ければ渡さない。
// そのとき /api/v1/line/webhook は 404 になる)。
func WithLineWebhook(receiver LineReceiver) Option {
	return func(s *Server) { s.line = receiver }
}

func (s *Server) lineRoutes() {
	if s.line != nil {
		s.handle("POST /api/v1/line/webhook", s.receiveLineWebhook)
	}
}

func (s *Server) receiveLineWebhook(r *http.Request) (any, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody+1))
	if err != nil {
		return nil, fmt.Errorf("read LINE webhook: %w", err)
	}
	if len(body) > maxWebhookBody {
		return nil, domain.Invalid("body", "is too large")
	}
	if err := s.line.Receive(r.Context(), body, r.Header.Get("X-Line-Signature")); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}
