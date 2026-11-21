package line

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"scoreview/api/internal/talk"
)

// DefaultBaseURL は Messaging API の URL。
const DefaultBaseURL = "https://api.line.me"

// requestTimeout は Messaging API への 1 回の要求の上限。返信のトークンは届いてから 1 分で使えなく
// なるので、それより十分短くする。
const requestTimeout = 10 * time.Second

// tokenMargin はステートレスのトークンを使い終える余裕。15 分の期限の 1 分前に新しく発行する。
const tokenMargin = time.Minute

// Client は公式アカウントとして送る。送るたびに要るチャネルアクセストークンは、チャネル ID と
// シークレットから 15 分だけ有効なステートレスのトークンを発行して使う。長期のトークンは 1 つの
// チャネルに 1 つしか持てず、発行し直すと前のものが無効になる(移行の前の Lイベントが止まる)ため。
// https://developers.line.biz/en/docs/basics/channel-access-token/
type Client struct {
	channelID string
	secret    string
	baseURL   string
	http      *http.Client
	now       func() time.Time

	mu      sync.Mutex
	token   string
	expires time.Time
}

var _ talk.Messenger = (*Client)(nil)

// NewClient はチャネル ID とシークレットで送る Client を返す。baseURL はテストで差し替える。
func NewClient(channelID, secret, baseURL string, now func() time.Time) *Client {
	return &Client{channelID: channelID, secret: secret, baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{Timeout: requestTimeout}, now: now}
}

// Reply は talk.Messenger.Reply。
// https://developers.line.biz/ja/reference/messaging-api/#send-reply-message
func (c *Client) Reply(ctx context.Context, replyToken string, messages []talk.Message) error {
	body := map[string]any{"replyToken": replyToken, "messages": toMessages(messages)}
	return c.post(ctx, "/v2/bot/message/reply", body)
}

// Push は talk.Messenger.Push。返信のトークンが無いとき(管理画面の操作など)に、公式アカウントから
// 送る。送った数は公式アカウントの月の配信数に数えられる。
// https://developers.line.biz/ja/reference/messaging-api/#send-push-message
func (c *Client) Push(ctx context.Context, to string, messages []talk.Message) error {
	body := map[string]any{"to": to, "messages": toMessages(messages)}
	return c.post(ctx, "/v2/bot/message/push", body)
}

type textMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func toMessages(messages []talk.Message) []textMessage {
	out := make([]textMessage, len(messages))
	for i, message := range messages {
		out[i] = textMessage{Type: "text", Text: message.Text}
	}
	return out
}

func (c *Client) post(ctx context.Context, path string, body any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode LINE request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build LINE request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return c.do(req, nil)
}

// accessToken は使えるトークンを返す。期限が近ければ発行し直す。
// https://developers.line.biz/ja/reference/messaging-api/#issue-stateless-channel-access-token
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && c.now().Before(c.expires.Add(-tokenMargin)) {
		return c.token, nil
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {c.channelID},
		"client_secret": {c.secret}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/oauth2/v3/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build LINE token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var issued struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := c.do(req, &issued); err != nil {
		return "", fmt.Errorf("issue LINE channel access token: %w", err)
	}
	c.token, c.expires = issued.AccessToken, c.now().Add(time.Duration(issued.ExpiresIn)*time.Second)
	return c.token, nil
}

// do は要求を送り、2xx でなければ応答の本文をエラーに含める(秘密は含まない)。
func (c *Client) do(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call LINE %s: %w", req.URL.Path, err)
	}
	defer func() { _ = res.Body.Close() }() // 読み終えた本文を閉じる失敗は、結果に影響しない
	body, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("read LINE %s: %w", req.URL.Path, err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("LINE %s returned %d: %s", req.URL.Path, res.StatusCode, body)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode LINE %s: %w", req.URL.Path, err)
	}
	return nil
}

// InviteURL は、公式アカウントのトークを「参加 招待コード」の入った状態で開く URL を作る関数を返す。
// basicID は公式アカウントのベーシック ID(@ で始まる)。空なら空の URL を返す。
// https://developers.line.biz/ja/docs/messaging-api/using-line-url-scheme/
func InviteURL(basicID string) func(code string) string {
	return func(code string) string {
		if basicID == "" {
			return ""
		}
		return "https://line.me/R/oaMessage/" + percentEncode(basicID) + "/?" + percentEncode("参加 "+code)
	}
}

// percentEncode は英数字と -_.~ 以外をすべて UTF-8 の %XX にする。LINE の URL スキームは、ベーシック ID の
// @ も %40 にするよう求めている(url.PathEscape は @ を残す)。空白は + ではなく %20 にする。
func percentEncode(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}
