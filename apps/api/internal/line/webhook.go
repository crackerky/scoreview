// Package line は LINE の Messaging API の HTTP の形を知る層。Webhook の署名の確かめと読み取り、
// 返信の送信、LINE の URL の組み立てを持つ(docs/line-integration.md)。受け答えの決まりは talk にある。
package line

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/talk"
)

// Verify は Webhook の本文の署名(x-line-signature)を確かめる。署名は、チャネルシークレットを鍵にした
// 本文の HMAC-SHA256 を base64 にしたもの。本文は届いたままのバイト列で確かめる(読み直すと合わない)。
// https://developers.line.biz/ja/docs/messaging-api/verify-webhook-signature/
func Verify(secret string, body []byte, signature string) bool {
	given, err := base64.StdEncoding.DecodeString(signature)
	if err != nil || secret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body) // hash.Hash の Write はエラーを返さない
	return hmac.Equal(given, mac.Sum(nil))
}

type webhookBody struct {
	Events []webhookEvent `json:"events"`
}

type webhookEvent struct {
	Type           string `json:"type"`
	Mode           string `json:"mode"`
	Timestamp      int64  `json:"timestamp"`
	WebhookEventID string `json:"webhookEventId"`
	ReplyToken     string `json:"replyToken"`
	Source         struct {
		Type   string `json:"type"`
		UserID string `json:"userId"`
	} `json:"source"`
	Message struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"message"`
	Postback struct {
		Data string `json:"data"`
	} `json:"postback"`
}

// ParseEvents は Webhook の本文を talk の Event にする。1 対 1 のトーク以外(グループ、トークルーム)
// からのものは、UserID を空にして受け答えの対象から外す。
// https://developers.line.biz/ja/reference/messaging-api/#webhook-event-objects
func ParseEvents(body []byte) ([]talk.Event, error) {
	var parsed webhookBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, domain.Invalid("body", "is not a LINE webhook")
	}
	out := make([]talk.Event, 0, len(parsed.Events))
	for _, raw := range parsed.Events {
		event := talk.Event{ID: raw.WebhookEventID, Standby: raw.Mode == "standby",
			ReplyToken: raw.ReplyToken, At: time.UnixMilli(raw.Timestamp)}
		if raw.Source.Type == "user" {
			event.UserID = raw.Source.UserID
		}
		switch raw.Type {
		case "follow":
			event.Type = talk.EventFollow
		case "unfollow":
			event.Type = talk.EventUnfollow
		case "postback":
			event.Type, event.PostbackData = talk.EventPostback, raw.Postback.Data
		case "message":
			event.Type = talk.EventOther
			if raw.Message.Type == "text" {
				event.Type, event.Text = talk.EventMessage, raw.Message.Text
			}
		default:
			event.Type = talk.EventOther
		}
		out = append(out, event)
	}
	return out, nil
}

// Handler は届いたものを処理する。talk.Bot が実装する。
type Handler interface {
	Handle(ctx context.Context, events []talk.Event) error
}

// Webhook は公式アカウントの Webhook の受け口。
type Webhook struct {
	secret  string
	handler Handler
}

// NewWebhook は secret(チャネルシークレット)で署名を確かめ、handler に渡す Webhook を返す。
func NewWebhook(secret string, handler Handler) *Webhook {
	return &Webhook{secret: secret, handler: handler}
}

// Receive は届いた本文を確かめて処理する。署名が合わなければ domain.ErrUnauthorized。
func (w *Webhook) Receive(ctx context.Context, body []byte, signature string) error {
	if !Verify(w.secret, body, signature) {
		return fmt.Errorf("%w: the LINE signature does not match", domain.ErrUnauthorized)
	}
	events, err := ParseEvents(body)
	if err != nil {
		return err
	}
	return w.handler.Handle(ctx, events)
}
