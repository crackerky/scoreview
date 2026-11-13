package viewing

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultCacheTTL は掲示板を使い回す時間。観客の画面は 15 秒おきに取り直すので、それより十分短く、
// 点が入ってから観客に見えるまでの遅れ(web の配信の使い回しと合わせて 10 秒ほど)が気にならない長さ。
const DefaultCacheTTL = 3 * time.Second

// maxCacheEntries は使い回す大会の数の上限。同時に開かれる大会は数個なので、これを超えるのは
// 古い大会を順に開かれたときだけ。超えたら古いものを捨てる。
const maxCacheEntries = 256

// Cache は Reader の結果を大会ごとに ttl のあいだ使い回す Reader。同じ大会の読み込みが重なったら
// 1 回にまとめる。観客の数だけ同じ大会が数秒おきに読まれるので、DB を読む回数を観客の数から切り離す。
// エラーは使い回さない(大会が無い、DB が一時的に落ちた、のどちらも次の要求で読み直す)。
//
// 使い回しはプロセスの中だけ。複数のインスタンスのあいだは web の配信の使い回し(Cache-Control)が受け持つ。
type Cache struct {
	next    Reader
	ttl     time.Duration
	now     func() time.Time
	flights singleflight.Group

	mu      sync.Mutex
	entries map[int64]cacheEntry
}

type cacheEntry struct {
	board Board
	at    time.Time
}

// NewCache は next の結果を ttl のあいだ使い回す Cache を返す。now は時刻の取り方(テスト用)。
func NewCache(next Reader, ttl time.Duration, now func() time.Time) *Cache {
	return &Cache{next: next, ttl: ttl, now: now, entries: map[int64]cacheEntry{}}
}

// Board は Reader.Board。
func (c *Cache) Board(ctx context.Context, tournamentID int64) (Board, error) {
	if board, ok := c.fresh(tournamentID); ok {
		return board, nil
	}
	value, err, _ := c.flights.Do(strconv.FormatInt(tournamentID, 10), func() (any, error) {
		// 待っているあいだに別の読み込みが入れていれば、それを使う
		if board, ok := c.fresh(tournamentID); ok {
			return board, nil
		}
		// 読み込みは待っている全員のもの。最初に来た人が接続を切っても止めない
		board, err := c.next.Board(context.WithoutCancel(ctx), tournamentID)
		if err != nil {
			return nil, err
		}
		c.keep(tournamentID, board)
		return board, nil
	})
	if err != nil {
		return Board{}, err
	}
	board, ok := value.(Board)
	if !ok {
		return Board{}, fmt.Errorf("board cache: unexpected value %T", value)
	}
	return board, nil
}

func (c *Cache) fresh(tournamentID int64) (Board, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[tournamentID]
	if !ok || c.now().Sub(entry.at) >= c.ttl {
		return Board{}, false
	}
	return entry.board, true
}

func (c *Cache) keep(tournamentID int64, board Board) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if len(c.entries) >= maxCacheEntries {
		for id, entry := range c.entries {
			if now.Sub(entry.at) >= c.ttl {
				delete(c.entries, id)
			}
		}
		// 期限内のものだけで上限に届いていれば、全部捨てて読み直させる(上限に届くのは異常な使われ方のとき)
		if len(c.entries) >= maxCacheEntries {
			clear(c.entries)
		}
	}
	c.entries[tournamentID] = cacheEntry{board: board, at: now}
}
