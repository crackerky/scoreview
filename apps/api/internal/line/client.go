package line

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"scoreview/api/internal/notify"
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

	// friends は FriendCount の答えを friendsTTL の間使い回す(LINE の数の読み出しは 1 時間に 60 回まで)
	friendsMu sync.Mutex
	friends   friendCount
}

// friendCount は読んだ友だちの数の見込みと、読んだ時刻。
type friendCount struct {
	count int64
	known bool
	at    time.Time
}

// friendsTTL は友だちの数を使い回す時間(LINE は日ごとに数えるので、30 分で変わることはまず無い)。
const friendsTTL = 30 * time.Minute

var (
	_ talk.Messenger = (*Client)(nil)
	_ notify.Sender  = (*Client)(nil)
)

// NewClient はチャネル ID とシークレットで送る Client を返す。baseURL はテストで差し替える。
func NewClient(channelID, secret, baseURL string, now func() time.Time) *Client {
	return &Client{channelID: channelID, secret: secret, baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{Timeout: requestTimeout}, now: now}
}

// Reply は talk.Messenger.Reply。
// https://developers.line.biz/ja/reference/messaging-api/#send-reply-message
func (c *Client) Reply(ctx context.Context, replyToken string, messages []talk.Message) error {
	body := map[string]any{"replyToken": replyToken, "messages": toMessages(messages)}
	return c.post(ctx, "/v2/bot/message/reply", body, "")
}

// Push は talk.Messenger.Push。返信のトークンが無いとき(管理画面の操作など)に、公式アカウントから
// 送る。送った数は公式アカウントの月の配信数に数えられる。
// https://developers.line.biz/ja/reference/messaging-api/#send-push-message
func (c *Client) Push(ctx context.Context, to string, messages []talk.Message) error {
	body := map[string]any{"to": to, "messages": toMessages(messages)}
	return c.post(ctx, "/v2/bot/message/push", body, "")
}

// Multicast は notify.Sender.Multicast。同じ retryKey の要求を LINE がもう受け付けていれば、LINE は
// 送らずに 409 を返すので、成功として扱う。LINE の 5xx と、応答が無い失敗(通信の切断、時間切れ)は、
// LINE が受け付けたか分からないので notify.ErrTemporary を包む(LINE が同じキーで送り直してよいと
// している場合)。4xx は LINE が受け付けなかった失敗。
// https://developers.line.biz/ja/reference/messaging-api/#send-multicast-message
// https://developers.line.biz/ja/docs/messaging-api/retrying-api-request/
func (c *Client) Multicast(ctx context.Context, to []string, message notify.Message, retryKey string) error {
	body := map[string]any{"to": to, "messages": []any{toNotifyMessage(message)}}
	return sendResult(c.post(ctx, "/v2/bot/message/multicast", body, retryKey))
}

// Broadcast は notify.Sender.Broadcast。送り直しのキーと失敗の扱いは Multicast と同じ。
// https://developers.line.biz/ja/reference/messaging-api/#send-broadcast-message
func (c *Client) Broadcast(ctx context.Context, message notify.Message, retryKey string) error {
	body := map[string]any{"messages": []any{toNotifyMessage(message)}}
	return sendResult(c.post(ctx, "/v2/bot/message/broadcast", body, retryKey))
}

// friendDays は友だちの数を読む日の数(前日から。LINE は日ごとの数を後から数えるので、前日の数が
// まだ無いことがある)。
const friendDays = 3

// FriendCount は notify.Sender.FriendCount。前日から順に、LINE が数え終えた最初の日の友だちの数(友だち
// になった人の累計)からブロックしている人の数を引く(日付は日本時間)。どの日もまだ数えていなければ
// known は偽。答えは friendsTTL の間使い回す。
// https://developers.line.biz/ja/reference/messaging-api/#get-number-of-followers
func (c *Client) FriendCount(ctx context.Context) (int64, bool, error) {
	c.friendsMu.Lock()
	defer c.friendsMu.Unlock()
	now := c.now()
	if !c.friends.at.IsZero() && now.Before(c.friends.at.Add(friendsTTL)) {
		return c.friends.count, c.friends.known, nil
	}
	count, known, err := c.readFriendCount(ctx, now)
	if err != nil {
		return 0, false, err
	}
	c.friends = friendCount{count: count, known: known, at: now}
	return count, known, nil
}

func (c *Client) readFriendCount(ctx context.Context, now time.Time) (int64, bool, error) {
	day := now.In(japan())
	for range friendDays {
		day = day.AddDate(0, 0, -1)
		var got struct {
			Status    string `json:"status"`
			Followers int64  `json:"followers"`
			Blocks    int64  `json:"blocks"`
		}
		if err := c.get(ctx, "/v2/bot/insight/followers?date="+day.Format("20060102"), &got); err != nil {
			return 0, false, err
		}
		if got.Status == "ready" {
			return max(got.Followers-got.Blocks, 0), true, nil
		}
	}
	return 0, false, nil
}

// japan は日本時間(LINE の数の日付の区切り)。
func japan() *time.Location { return time.FixedZone("JST", 9*60*60) }

// sendResult は送る要求の結果を notify の決まりにする。同じ retryKey の要求を LINE がもう受け付けて
// いれば、LINE は送らずに 409 を返すので成功として扱う。LINE の 5xx と応答が無い失敗は
// notify.ErrTemporary を包み、4xx は LINE が受け付けなかった失敗のまま返す。
func sendResult(err error) error {
	var failed *apiError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &failed) && failed.status == http.StatusConflict:
		return nil
	case errors.As(err, &failed) && failed.status < http.StatusInternalServerError:
		return err
	default:
		return fmt.Errorf("%w: %w", notify.ErrTemporary, err)
	}
}

// Quota は notify.Sender.Quota。上限の種類が none(追加メッセージのあるプラン)なら上限は無い。
// https://developers.line.biz/ja/reference/messaging-api/#get-quota
// https://developers.line.biz/ja/reference/messaging-api/#get-consumption
func (c *Client) Quota(ctx context.Context) (notify.Quota, error) {
	var limit struct {
		Type  string `json:"type"`
		Value int64  `json:"value"`
	}
	if err := c.get(ctx, "/v2/bot/message/quota", &limit); err != nil {
		return notify.Quota{}, err
	}
	var used struct {
		TotalUsage int64 `json:"totalUsage"`
	}
	if err := c.get(ctx, "/v2/bot/message/quota/consumption", &used); err != nil {
		return notify.Quota{}, err
	}
	quota := notify.Quota{Used: used.TotalUsage}
	if limit.Type == "limited" {
		quota.Limit = &limit.Value
	}
	return quota, nil
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

// maxAltText は LINE のカードの代わりの文(通知やトークの一覧に出る)の上限。
// https://developers.line.biz/ja/reference/messaging-api/#flex-message
const maxAltText = 1500

// toNotifyMessage は運営が送るものを LINE のメッセージにする。ボタンが無ければ文だけ、あれば文の下に
// ボタンを縦に並べたカード(Flex Message の bubble)にする。申込のボタンは公式アカウントに「エントリー」を
// 送り(押すと本人用の申込のリンクが返る)、ほかのボタンは URL を開く。
// https://developers.line.biz/ja/reference/messaging-api/#message-action
// https://developers.line.biz/ja/reference/messaging-api/#uri-action
func toNotifyMessage(message notify.Message) any {
	if len(message.Buttons) == 0 {
		return textMessage{Type: "text", Text: message.Text}
	}
	buttons := make([]map[string]any, len(message.Buttons))
	for i, button := range message.Buttons {
		action := map[string]any{"type": "uri", "label": button.Label, "uri": button.URL}
		if button.Kind == notify.ButtonEntry {
			action = map[string]any{"type": "message", "label": button.Label, "text": "エントリー"}
		}
		style := "secondary"
		if i == 0 {
			style = "primary"
		}
		buttons[i] = map[string]any{"type": "button", "style": style, "height": "sm", "action": action}
	}
	alt := []rune(message.Text)
	if len(alt) > maxAltText {
		alt = alt[:maxAltText]
	}
	return map[string]any{
		"type":    "flex",
		"altText": string(alt),
		"contents": map[string]any{
			"type": "bubble",
			"body": map[string]any{"type": "box", "layout": "vertical", "contents": []any{
				map[string]any{"type": "text", "text": message.Text, "wrap": true, "size": "md"},
			}},
			"footer": map[string]any{"type": "box", "layout": "vertical", "spacing": "sm", "contents": buttons},
		},
	}
}

// post は body を JSON で送る。retryKey が空でなければ X-Line-Retry-Key に付ける。
func (c *Client) post(ctx context.Context, path string, body any, retryKey string) error {
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
	if retryKey != "" {
		req.Header.Set("X-Line-Retry-Key", retryKey)
	}
	return c.do(req, nil)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build LINE request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return c.do(req, out)
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
		return &apiError{path: req.URL.Path, status: res.StatusCode, body: body}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode LINE %s: %w", req.URL.Path, err)
	}
	return nil
}

// apiError は Messaging API が 2xx 以外を返したこと。本文は LINE のエラーの説明(秘密は含まない)。
type apiError struct {
	path   string
	status int
	body   []byte
}

func (e *apiError) Error() string {
	return fmt.Sprintf("LINE %s returned %d: %s", e.path, e.status, e.body)
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
