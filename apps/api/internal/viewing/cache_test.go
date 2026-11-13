package viewing_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/viewing"
)

// countingReader は呼ばれた回数を数え、大会 ID を名前に入れた掲示板を返す Reader。
type countingReader struct {
	calls atomic.Int64
	err   error
	// gate が nil でなければ、閉じるまで返さない(読み込みを重ねるため)
	gate chan struct{}
}

func (r *countingReader) Board(ctx context.Context, tournamentID int64) (viewing.Board, error) {
	r.calls.Add(1)
	if r.gate != nil {
		<-r.gate
	}
	if r.err != nil {
		return viewing.Board{}, r.err
	}
	if err := ctx.Err(); err != nil {
		return viewing.Board{}, err
	}
	return viewing.Board{Tournament: viewing.Tournament{ID: tournamentID}}, nil
}

var _ viewing.Reader = (*countingReader)(nil)

// clock はテストが進める時計。
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestCacheReusesBoardUntilTTL(t *testing.T) {
	next := &countingReader{}
	at := &clock{now: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	cache := viewing.NewCache(next, 3*time.Second, at.Now)
	ctx := context.Background()

	for range 5 {
		board, err := cache.Board(ctx, 7)
		require.NoError(t, err)
		assert.Equal(t, int64(7), board.Tournament.ID)
	}
	assert.Equal(t, int64(1), next.calls.Load())

	_, err := cache.Board(ctx, 8)
	require.NoError(t, err)
	assert.Equal(t, int64(2), next.calls.Load(), "each tournament is read on its own")

	at.advance(3 * time.Second)
	_, err = cache.Board(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, int64(3), next.calls.Load(), "an expired board is read again")
}

func TestCacheDoesNotKeepErrors(t *testing.T) {
	next := &countingReader{err: errors.New("db is down")}
	at := &clock{now: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	cache := viewing.NewCache(next, time.Minute, at.Now)

	_, err := cache.Board(context.Background(), 7)
	require.ErrorIs(t, err, next.err)
	next.err = nil
	_, err = cache.Board(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, int64(2), next.calls.Load())
}

func TestCacheMergesConcurrentReads(t *testing.T) {
	next := &countingReader{gate: make(chan struct{})}
	at := &clock{now: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	cache := viewing.NewCache(next, time.Minute, at.Now)

	const viewers = 20
	var wg sync.WaitGroup
	errs := make(chan error, viewers)
	for range viewers {
		wg.Go(func() {
			_, err := cache.Board(context.Background(), 7)
			errs <- err
		})
	}
	// 最初の読み込みが始まってから門を開ける。残りは読み込みを待つか、使い回しを受け取る
	require.Eventually(t, func() bool { return next.calls.Load() == 1 }, time.Second, time.Millisecond)
	close(next.gate)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, int64(1), next.calls.Load())
}

func TestCacheLoadSurvivesTheFirstViewerLeaving(t *testing.T) {
	next := &countingReader{}
	at := &clock{now: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	cache := viewing.NewCache(next, time.Minute, at.Now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	board, err := cache.Board(ctx, 7)
	require.NoError(t, err, "the shared load is not tied to one viewer's connection")
	assert.Equal(t, int64(7), board.Tournament.ID)
}
