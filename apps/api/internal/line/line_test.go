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
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/line"
	"scoreview/api/internal/notify"
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

func TestClientPushes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/v3/token":
			respond(w, `{"access_token":"tok","expires_in":900}`)
		case "/v2/bot/message/push":
			assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
			var body struct {
				To       string `json:"to"`
				Messages []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"messages"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "Urep", body.To)
			require.Len(t, body.Messages, 2)
			assert.Equal(t, "受け付けました", body.Messages[0].Text)
			respond(w, `{"sentMessages":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := line.NewClient("1656", "secret", server.URL, time.Now)
	require.NoError(t, client.Push(context.Background(), "Urep",
		[]talk.Message{{Text: "受け付けました"}, {Text: "招待"}}))
}

func TestClientMulticastsWithRetryKey(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/v3/token":
			respond(w, `{"access_token":"tok","expires_in":900}`)
		case "/v2/bot/message/multicast":
			var body struct {
				To       []string `json:"to"`
				Messages []struct {
					Text string `json:"text"`
				} `json:"messages"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, []string{"U1", "U2"}, body.To)
			assert.Equal(t, "大会\n\n明日は 8 時", body.Messages[0].Text)
			key := r.Header.Get("X-Line-Retry-Key")
			if slices.Contains(keys, key) {
				// 同じキーの 2 回目は LINE が受け付け済みとして 409 を返す
				w.WriteHeader(http.StatusConflict)
				respond(w, `{"message":"The retry key is already accepted"}`)
				return
			}
			keys = append(keys, key)
			respond(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := line.NewClient("1656", "secret", server.URL, time.Now)
	ctx := context.Background()
	require.NoError(t, client.Multicast(ctx, []string{"U1", "U2"}, notify.Message{Text: "大会\n\n明日は 8 時"}, "key-1"))
	require.NoError(t, client.Multicast(ctx, []string{"U1", "U2"}, notify.Message{Text: "大会\n\n明日は 8 時"}, "key-1"),
		"a retry LINE already accepted is a success")
	assert.Equal(t, []string{"key-1"}, keys)
}

// LINE が受け付けたか分からない失敗(5xx、応答が無い)だけを、同じキーで送り直してよい失敗にする。
func TestClientMarksTemporaryFailures(t *testing.T) {
	status := http.StatusInternalServerError
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v3/token" {
			respond(w, `{"access_token":"tok","expires_in":900}`)
			return
		}
		w.WriteHeader(status)
		respond(w, `{"message":"x"}`)
	}))
	client := line.NewClient("1656", "secret", server.URL, time.Now)
	ctx := context.Background()
	require.ErrorIs(t, client.Multicast(ctx, []string{"U1"}, notify.Message{Text: "x"}, "key-1"), notify.ErrTemporary)
	status = http.StatusTooManyRequests
	err := client.Multicast(ctx, []string{"U1"}, notify.Message{Text: "x"}, "key-2")
	require.Error(t, err)
	assert.NotErrorIs(t, err, notify.ErrTemporary, "LINE refused it, so the result is known")
	server.Close()
	require.ErrorIs(t, client.Multicast(ctx, []string{"U1"}, notify.Message{Text: "x"}, "key-3"), notify.ErrTemporary,
		"no answer from LINE")
}

func TestClientReadsQuota(t *testing.T) {
	limited := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/v3/token":
			respond(w, `{"access_token":"tok","expires_in":900}`)
		case "/v2/bot/message/quota":
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
			if limited {
				respond(w, `{"type":"limited","value":200}`)
			} else {
				respond(w, `{"type":"none"}`)
			}
		case "/v2/bot/message/quota/consumption":
			respond(w, `{"totalUsage":37}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := line.NewClient("1656", "secret", server.URL, time.Now)
	quota, err := client.Quota(context.Background())
	require.NoError(t, err)
	require.NotNil(t, quota.Limit)
	assert.Equal(t, int64(200), *quota.Limit)
	assert.Equal(t, int64(37), quota.Used)
	limited = false
	quota, err = client.Quota(context.Background())
	require.NoError(t, err)
	assert.Nil(t, quota.Limit, "a plan with additional messages has no limit")
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

// ボタンのある文は、文の下にボタンを並べたカードにする。申込のボタンは「エントリー」を送り、ほかは URL を開く。
func TestClientSendsButtonsAsACard(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v3/token" {
			respond(w, `{"access_token":"tok","expires_in":900}`)
			return
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		respond(w, `{}`)
	}))
	defer server.Close()
	client := line.NewClient("1656", "secret", server.URL, time.Now)
	message := notify.Message{Text: "大会\n\n明日は 8 時", Buttons: []notify.Button{
		{Kind: notify.ButtonEntry, Label: "申込を開く"},
		{Kind: notify.ButtonSite, Label: "スコアサイトを見る", URL: "https://score.example/tournaments/7"}}}
	require.NoError(t, client.Multicast(context.Background(), []string{"U1"}, message, "key-1"))
	sent := got["messages"].([]any)[0].(map[string]any) //nolint:forcetypeassert // テストで JSON の形を読む
	assert.Equal(t, "flex", sent["type"])
	assert.Equal(t, "大会\n\n明日は 8 時", sent["altText"])
	data, err := json.Marshal(sent["contents"])
	require.NoError(t, err)
	assert.Contains(t, string(data), `"action":{"label":"申込を開く","text":"エントリー","type":"message"}`)
	assert.Contains(t, string(data), `"uri":"https://score.example/tournaments/7"`)
}

// 友だち全員には一斉送信で送り、送り直しのキーを付ける。人数の見込みは、前日から順に LINE が数え終えた
// 最初の日の友だちの数からブロックの数を引いたもの(日付は日本時間)。
func TestClientBroadcastsAndCountsFriends(t *testing.T) {
	var dates []string
	var broadcast map[string]any
	var key string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/v3/token":
			respond(w, `{"access_token":"tok","expires_in":900}`)
		case "/v2/bot/message/broadcast":
			key = r.Header.Get("X-Line-Retry-Key")
			require.NoError(t, json.NewDecoder(r.Body).Decode(&broadcast))
			respond(w, `{}`)
		case "/v2/bot/insight/followers":
			date := r.URL.Query().Get("date")
			dates = append(dates, date)
			if date == "20261126" {
				respond(w, `{"status":"unready"}`)
				return
			}
			respond(w, `{"status":"ready","followers":130,"targetedReaches":100,"blocks":10}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	// 日本時間では 11 月 27 日の朝(UTC ではまだ 26 日)
	now := func() time.Time { return time.Date(2026, 11, 26, 23, 0, 0, 0, time.UTC) }
	client := line.NewClient("1656", "secret", server.URL, now)
	require.NoError(t, client.Broadcast(context.Background(), notify.Message{Text: "大会\n\nお知らせ"}, "key-1"))
	assert.Equal(t, "key-1", key)
	assert.NotContains(t, broadcast, "to")
	count, known, err := client.FriendCount(context.Background())
	require.NoError(t, err)
	assert.True(t, known)
	assert.Equal(t, int64(120), count)
	assert.Equal(t, []string{"20261126", "20261125"}, dates)
	again, _, err := client.FriendCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, count, again)
	assert.Len(t, dates, 2, "the count is reused for a while (LINE allows 60 reads an hour)")
}
