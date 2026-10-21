package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClock は実時間を待たずに待機を記録する時計。
type fakeClock struct {
	mu     sync.Mutex
	cur    time.Time
	sleeps []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{cur: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)}
}

func (f *fakeClock) now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cur
}

func (f *fakeClock) sleep(_ context.Context, d time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sleeps = append(f.sleeps, d)
	f.cur = f.cur.Add(d)
	return nil
}

// server は path ごとに決めたステータスの列を順に返す。列が尽きたら最後の値を返し続ける。
type server struct {
	mu       sync.Mutex
	statuses map[string][]int
	hits     map[string]int
	agents   []string
}

func newServer(t *testing.T, statuses map[string][]int) (*server, *httptest.Server) {
	t.Helper()
	s := &server{statuses: statuses, hits: map[string]int{}}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.agents = append(s.agents, r.UserAgent())
		seq := s.statuses[r.URL.Path]
		n := s.hits[r.URL.Path]
		s.hits[r.URL.Path] = n + 1
		status := http.StatusOK
		if len(seq) > 0 {
			status = seq[min(n, len(seq)-1)]
		}
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte("<html>" + r.URL.Path + "</html>")) // テスト用サーバーの書き込み失敗は検証対象外
	}))
	t.Cleanup(ts.Close)
	return s, ts
}

func newTestClient(t *testing.T, ts *httptest.Server, clock *fakeClock, opts ...Option) *Client {
	t.Helper()
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)
	all := append([]Option{
		WithHTTPClient(ts.Client()),
		WithClock(clock.now, clock.sleep),
	}, opts...)
	c, err := NewClient(cache, all...)
	require.NoError(t, err)
	return c
}

func TestFetchUsesCache(t *testing.T) {
	srv, ts := newServer(t, nil)
	c := newTestClient(t, ts, newFakeClock())

	for range 2 {
		body, err := c.Fetch(context.Background(), ts.URL+"/scview/525?lang=jp")
		require.NoError(t, err)
		assert.Equal(t, "<html>/scview/525</html>", string(body))
	}
	assert.Equal(t, 1, srv.hits["/scview/525"], "2 回目はキャッシュから返す")
	assert.Equal(t, 1, c.Requests(), "キャッシュから返したページは数えない")
}

func TestFetchNotFoundIsTypedAndCached(t *testing.T) {
	srv, ts := newServer(t, map[string][]int{"/scview/999999": {http.StatusInternalServerError}})
	c := newTestClient(t, ts, newFakeClock())

	for range 2 {
		_, err := c.Fetch(context.Background(), ts.URL+"/scview/999999?lang=jp")
		require.ErrorIs(t, err, ErrNotFound)
		var nf *NotFoundError
		require.ErrorAs(t, err, &nf)
		assert.Equal(t, http.StatusInternalServerError, nf.Status)
	}
	assert.Equal(t, 1, srv.hits["/scview/999999"], "500 は再試行せず、2 回目は記録から返す")
}

func TestFetchRetriesWithExponentialBackoff(t *testing.T) {
	path := "/pscore/scv_team/open/1"
	srv, ts := newServer(t, map[string][]int{path: {503, 503, 200}})
	clock := newFakeClock()
	c := newTestClient(t, ts, clock, WithRetry(2, 2*time.Second))

	_, err := c.Fetch(context.Background(), ts.URL+path)
	require.NoError(t, err)
	assert.Equal(t, 3, srv.hits[path])
	assert.Equal(t, 3, c.Requests(), "再試行も数える")
	// 再試行の前に 2 秒、4 秒待つ。その時点で前回から 1.2 秒以上経っているので間隔待ちは起きない
	assert.Equal(t, []time.Duration{2 * time.Second, 4 * time.Second}, clock.sleeps)
}

func TestFetchGivesUpAfterMaxRetries(t *testing.T) {
	path := "/pscore/scv_team/open/2"
	srv, ts := newServer(t, map[string][]int{path: {503}})
	c := newTestClient(t, ts, newFakeClock())

	_, err := c.Fetch(context.Background(), ts.URL+path)
	var se *StatusError
	require.ErrorAs(t, err, &se)
	assert.Equal(t, http.StatusServiceUnavailable, se.Status)
	assert.False(t, errors.Is(err, ErrNotFound))
	assert.Equal(t, 1+DefaultMaxRetries, srv.hits[path])
}

func TestFetchKeepsMinInterval(t *testing.T) {
	_, ts := newServer(t, nil)
	clock := newFakeClock()
	c := newTestClient(t, ts, clock)

	_, err := c.Fetch(context.Background(), ts.URL+"/a")
	require.NoError(t, err)
	_, err = c.Fetch(context.Background(), ts.URL+"/b")
	require.NoError(t, err)
	_, err = c.Fetch(context.Background(), ts.URL+"/a") // キャッシュから。待たない
	require.NoError(t, err)
	assert.Equal(t, []time.Duration{DefaultMinInterval}, clock.sleeps)
}

func TestFetchUserAgent(t *testing.T) {
	srv, ts := newServer(t, nil)
	c := newTestClient(t, ts, newFakeClock(), WithContact("ops@example.com"))
	_, err := c.Fetch(context.Background(), ts.URL+"/a")
	require.NoError(t, err)
	assert.Equal(t, []string{"scoreview-importer/0.1 (+contact: ops@example.com)"}, srv.agents)

	srv2, ts2 := newServer(t, nil)
	c2 := newTestClient(t, ts2, newFakeClock())
	_, err = c2.Fetch(context.Background(), ts2.URL+"/a")
	require.NoError(t, err)
	assert.Equal(t, []string{UserAgent}, srv2.agents)
}

func TestNewClientRejectsShortInterval(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)
	_, err = NewClient(cache, WithMinInterval(500*time.Millisecond))
	assert.Error(t, err)
}

func TestFileName(t *testing.T) {
	a := FileName("https://score.clubjr.com/pscore/scv_team/open/17322?lang=jp")
	b := FileName("https://score.clubjr.com/pscore/scv_team/open/17322?lang=en")
	assert.Regexp(t, `^score\.clubjr\.com_pscore_scv_team_open_17322_lang_jp-[0-9a-f]{8}\.html$`, a)
	assert.NotEqual(t, a, b)
}

func TestOfflineFetcher(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, cache.Put("https://example.test/a", []byte("A")))
	off := Offline{Cache: cache}

	body, err := off.Fetch(context.Background(), "https://example.test/a")
	require.NoError(t, err)
	assert.Equal(t, "A", string(body))
	_, err = off.Fetch(context.Background(), "https://example.test/b")
	assert.ErrorIs(t, err, ErrCacheMiss)
}
