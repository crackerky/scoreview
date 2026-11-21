package line_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/line"
	"scoreview/api/internal/talk"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerify(t *testing.T) {
	body := []byte(`{"destination":"U0","events":[]}`)
	assert.True(t, line.Verify("secret", body, sign("secret", body)))
	assert.False(t, line.Verify("other", body, sign("secret", body)))
	assert.False(t, line.Verify("secret", append(body, ' '), sign("secret", body)), "the body must not change")
	assert.False(t, line.Verify("secret", body, "not base64!"))
	assert.False(t, line.Verify("", body, sign("", body)), "an empty secret never verifies")
}

const webhookBody = `{"destination":"Uxxx","events":[
 {"type":"follow","mode":"active","timestamp":1763600000000,"webhookEventId":"01A","replyToken":"r1",
  "source":{"type":"user","userId":"U1"},"deliveryContext":{"isRedelivery":false},"follow":{"isUnblocked":false}},
 {"type":"message","mode":"active","timestamp":1763600001000,"webhookEventId":"01B","replyToken":"r2",
  "source":{"type":"user","userId":"U1"},"message":{"id":"1","type":"text","text":"エントリー"}},
 {"type":"postback","mode":"standby","timestamp":1763600002000,"webhookEventId":"01C",
  "source":{"type":"user","userId":"U2"},"postback":{"data":"action=entry"}},
 {"type":"message","mode":"active","timestamp":1763600003000,"webhookEventId":"01D","replyToken":"r4",
  "source":{"type":"group","groupId":"G1","userId":"U3"},"message":{"id":"2","type":"text","text":"エントリー"}},
 {"type":"message","mode":"active","timestamp":1763600004000,"webhookEventId":"01E","replyToken":"r5",
  "source":{"type":"user","userId":"U1"},"message":{"id":"3","type":"sticker"}},
 {"type":"unfollow","mode":"active","timestamp":1763600005000,"webhookEventId":"01F",
  "source":{"type":"user","userId":"U4"}}
]}`

func TestParseEvents(t *testing.T) {
	events, err := line.ParseEvents([]byte(webhookBody))
	require.NoError(t, err)
	require.Len(t, events, 6)
	assert.Equal(t, talk.Event{ID: "01A", Type: talk.EventFollow, UserID: "U1", ReplyToken: "r1",
		At: time.UnixMilli(1763600000000)}, events[0])
	assert.Equal(t, talk.EventMessage, events[1].Type)
	assert.Equal(t, "エントリー", events[1].Text)
	assert.Equal(t, talk.EventPostback, events[2].Type)
	assert.Equal(t, "action=entry", events[2].PostbackData)
	assert.True(t, events[2].Standby)
	assert.Empty(t, events[3].UserID, "messages in a group are not answered")
	assert.Equal(t, talk.EventOther, events[4].Type, "stickers are not answered")
	assert.Equal(t, talk.EventUnfollow, events[5].Type)

	_, err = line.ParseEvents([]byte(`not json`))
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

type recorder struct{ events []talk.Event }

func (r *recorder) Handle(_ context.Context, events []talk.Event) error {
	r.events = append(r.events, events...)
	return nil
}

func TestWebhookRejectsWrongSignature(t *testing.T) {
	handler := &recorder{}
	webhook := line.NewWebhook("secret", handler)
	body := []byte(webhookBody)
	require.ErrorIs(t, webhook.Receive(context.Background(), body, sign("wrong", body)), domain.ErrUnauthorized)
	assert.Empty(t, handler.events)
	require.NoError(t, webhook.Receive(context.Background(), body, sign("secret", body)))
	assert.Len(t, handler.events, 6)
}

func TestClientIssuesTokenOnceAndReplies(t *testing.T) {
	var issued, replied atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/v3/token":
			issued.Add(1)
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "client_credentials", r.PostForm.Get("grant_type"))
			assert.Equal(t, "1656", r.PostForm.Get("client_id"))
			assert.Equal(t, "secret", r.PostForm.Get("client_secret"))
			respond(w, `{"token_type":"Bearer","access_token":"tok","expires_in":900}`)
		case "/v2/bot/message/reply":
			replied.Add(1)
			assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
			var body struct {
				ReplyToken string `json:"replyToken"`
				Messages   []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"messages"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "r1", body.ReplyToken)
			assert.Equal(t, "text", body.Messages[0].Type)
			assert.Equal(t, "こんにちは", body.Messages[0].Text)
			respond(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	client := line.NewClient("1656", "secret", server.URL, func() time.Time { return now })
	ctx := context.Background()
	require.NoError(t, client.Reply(ctx, "r1", []talk.Message{{Text: "こんにちは"}}))
	require.NoError(t, client.Reply(ctx, "r1", []talk.Message{{Text: "こんにちは"}}))
	assert.Equal(t, int32(1), issued.Load(), "the token is reused while valid")
	assert.Equal(t, int32(2), replied.Load())

	// 期限の 1 分前を過ぎたら発行し直す
	now = now.Add(14*time.Minute + time.Second)
	require.NoError(t, client.Reply(ctx, "r1", []talk.Message{{Text: "こんにちは"}}))
	assert.Equal(t, int32(2), issued.Load())
}

func TestClientReportsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v3/token" {
			respond(w, `{"access_token":"tok","expires_in":900}`)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		respond(w, `{"message":"Invalid reply token"}`)
	}))
	defer server.Close()
	client := line.NewClient("1656", "secret", server.URL, time.Now)
	err := client.Reply(context.Background(), "old", []talk.Message{{Text: "x"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid reply token")
	assert.NotContains(t, err.Error(), "secret")
}

func TestInviteURL(t *testing.T) {
	assert.Equal(t, "https://line.me/R/oaMessage/%40abc1234/?%E5%8F%82%E5%8A%A0%20aaaa",
		line.InviteURL("@abc1234")("aaaa"))
	assert.Empty(t, line.InviteURL("")("aaaa"))
}

// respond は偽の LINE の応答を書く。書けなかったときは、受け取った Client の側の検査が落ちるので、
// ここではエラーを捨てる。
func respond(w io.Writer, body string) {
	_, _ = io.WriteString(w, body)
}
