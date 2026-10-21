// Package fetch は参照サイトのページを取得する。取得結果は URL ごとのファイルとして
// キャッシュに保存し、同じページを取り直さない(CLAUDE.md gotcha)。
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// DefaultMinInterval はリクエストの最小間隔。参照サイトへの自動アクセスは 1 秒以上空ける規約
// (CLAUDE.md)に、調査時と同じ 0.2 秒の余裕を足した値(clone-spec A の調査は 1.2 秒間隔)。
const DefaultMinInterval = 1200 * time.Millisecond

// UserAgent は取込ツールの User-Agent。連絡先は WithContact で設定値から付ける。
const UserAgent = "scoreview-importer/0.1"

// DefaultMaxRetries は一時的な失敗(通信エラー、404/500 以外の 4xx/5xx)を再試行する最大回数。
const DefaultMaxRetries = 2

// DefaultBackoff は最初の再試行までの待ち時間。再試行のたびに 2 倍にする(指数バックオフ)。
const DefaultBackoff = 2 * time.Second

// maxBodyBytes は 1 ページの最大サイズ。参照サイトで最大のギャラリーページが約 66KB、
// 大会トップが約 40KB(clone-spec A 6、E 10)なので、十分に大きい 16MiB で打ち切る。
const maxBodyBytes = 16 << 20

// ErrNotFound はページが存在しないことを表す。参照サイトは存在しない ID に 500、
// 数値でない ID に 404 を返す(clone-spec A 7)。
var ErrNotFound = errors.New("page not found")

// NotFoundError は存在しないページの URL と HTTP ステータスを持つ。errors.Is(err, ErrNotFound) が真になる。
type NotFoundError struct {
	URL    string
	Status int
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s: HTTP %d", e.URL, e.Status)
}

func (e *NotFoundError) Unwrap() error { return ErrNotFound }

// Fetcher はページの HTML を返す。存在しないページは *NotFoundError を返す。
type Fetcher interface {
	Fetch(ctx context.Context, pageURL string) ([]byte, error)
}

// Client は参照サイトから HTTP で取得し、キャッシュに保存する Fetcher。
// キャッシュにあるページはネットワークに出ない。並行に呼んでも最小間隔は守られる。
type Client struct {
	cache       *Cache
	httpClient  *http.Client
	minInterval time.Duration
	maxRetries  int
	backoff     time.Duration
	contact     string
	now         func() time.Time
	sleep       func(ctx context.Context, d time.Duration) error

	mu      sync.Mutex
	lastReq time.Time
}

// Option は Client の設定を変える。
type Option func(*Client)

// WithMinInterval はリクエストの最小間隔を変える。1 秒未満は規約違反なので NewClient がエラーにする。
func WithMinInterval(d time.Duration) Option { return func(c *Client) { c.minInterval = d } }

// WithContact は User-Agent に付ける連絡先を設定する。空なら付けない。
func WithContact(contact string) Option { return func(c *Client) { c.contact = contact } }

// WithHTTPClient は HTTP クライアントを差し替える(テストで httptest のサーバーに向ける)。
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.httpClient = hc } }

// WithRetry は再試行の回数と最初の待ち時間を変える。
func WithRetry(maxRetries int, backoff time.Duration) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.backoff = backoff
	}
}

// WithClock は現在時刻と待機の関数を差し替える。テストで実時間を待たずに間隔を検証するため。
func WithClock(now func() time.Time, sleep func(ctx context.Context, d time.Duration) error) Option {
	return func(c *Client) {
		c.now = now
		c.sleep = sleep
	}
}

// minAllowedInterval は WithMinInterval で許す下限。参照サイトへの自動アクセスは 1 秒以上空ける(CLAUDE.md)。
const minAllowedInterval = time.Second

// httpTimeout は 1 リクエストのタイムアウト。参照サイトの応答は通常 1 秒以内なので、余裕を見た値。
const httpTimeout = 30 * time.Second

// NewClient は cache を使う Client を作る。
func NewClient(cache *Cache, opts ...Option) (*Client, error) {
	c := &Client{
		cache:       cache,
		httpClient:  &http.Client{Timeout: httpTimeout},
		minInterval: DefaultMinInterval,
		maxRetries:  DefaultMaxRetries,
		backoff:     DefaultBackoff,
		now:         time.Now,
		sleep:       sleepContext,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.minInterval < minAllowedInterval {
		return nil, fmt.Errorf("min interval %s is shorter than %s", c.minInterval, minAllowedInterval)
	}
	if c.maxRetries < 0 {
		return nil, fmt.Errorf("max retries must not be negative: %d", c.maxRetries)
	}
	return c, nil
}

func (c *Client) userAgent() string {
	if c.contact == "" {
		return UserAgent
	}
	return UserAgent + " (+contact: " + c.contact + ")"
}

// Fetch はキャッシュにあればそれを返し、なければ取得してキャッシュに保存する。
// 存在しないページ(404/500)もキャッシュに記録し、次回からは取得せずに *NotFoundError を返す。
func (c *Client) Fetch(ctx context.Context, pageURL string) ([]byte, error) {
	if body, err := c.cache.Get(pageURL); err == nil {
		return body, nil
	} else if !errors.Is(err, ErrCacheMiss) {
		return nil, err
	}

	body, err := c.fetchWithRetry(ctx, pageURL)
	var nf *NotFoundError
	if errors.As(err, &nf) {
		if perr := c.cache.PutNotFound(pageURL, nf.Status); perr != nil {
			return nil, perr
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if err := c.cache.Put(pageURL, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Client) fetchWithRetry(ctx context.Context, pageURL string) ([]byte, error) {
	var lastErr error
	wait := c.backoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := c.sleep(ctx, wait); err != nil {
				return nil, fmt.Errorf("fetch %s: %w", pageURL, err)
			}
			wait *= 2
		}
		body, err := c.fetchOnce(ctx, pageURL)
		if err == nil {
			return body, nil
		}
		if errors.Is(err, ErrNotFound) || ctx.Err() != nil {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("fetch %s: gave up after %d retries: %w", pageURL, c.maxRetries, lastErr)
}

// waitTurn は前回のリクエストから最小間隔が経つまで待ち、今回のリクエストの時刻を記録する。
func (c *Client) waitTurn(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.lastReq.IsZero() {
		if d := c.lastReq.Add(c.minInterval).Sub(c.now()); d > 0 {
			if err := c.sleep(ctx, d); err != nil {
				return err
			}
		}
	}
	c.lastReq = c.now()
	return nil
}

func (c *Client) fetchOnce(ctx context.Context, pageURL string) ([]byte, error) {
	if err := c.waitTurn(ctx); err != nil {
		return nil, fmt.Errorf("wait for interval: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request %s: %w", pageURL, err)
	}
	req.Header.Set("User-Agent", c.userAgent())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", pageURL, err)
	}
	defer func() {
		// 本文は読み終えているか、読む必要がない。Close の失敗で取得結果は変わらない
		_ = resp.Body.Close()
	}()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusInternalServerError:
		return nil, &NotFoundError{URL: pageURL, Status: resp.StatusCode}
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("get %s: %w", pageURL, &StatusError{Status: resp.StatusCode})
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", pageURL, err)
	}
	if len(body) > maxBodyBytes {
		return nil, fmt.Errorf("read %s: body exceeds %d bytes", pageURL, maxBodyBytes)
	}
	return body, nil
}

// StatusError は 200 でも 404/500 でもない HTTP ステータス。再試行の対象になる。
type StatusError struct {
	Status int
}

func (e *StatusError) Error() string { return fmt.Sprintf("unexpected HTTP status %d", e.Status) }

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
