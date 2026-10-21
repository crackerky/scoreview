package auth

import (
	"context"
	"fmt"
	"time"

	"scoreview/api/internal/domain"
)

// MaxConcurrentPasswordChecks はパスワード照合(argon2id)を同時に行う数の上限。1 回の照合は
// argonMemoryKiB(19 MiB)を確保するので、上限まで並んでも 8 × 19 MiB = 152 MiB で、Vercel の関数の
// 既定のメモリ 2 GB(Hobby は 2 GB 固定。https://vercel.com/docs/functions/limitations の
// Memory size limits)の 1 割に収まる。既定の 1 vCPU では、これより並べても照合の速さは上がらない。
// 上限が無いと、認証の要らないログインを並行に送るだけでメモリを使い切らせられる(セキュリティ点検 M-2)。
const MaxConcurrentPasswordChecks = 8

// PasswordCheckWait は照合の順番を待つ時間の上限。1 回の照合は数十ミリ秒なので、ふつうの混み方なら
// 待てば入れる。待っても空かなければ ErrUnavailable(HTTP 503)にする。
const PasswordCheckWait = 2 * time.Second

// passwordCheckRetryAfter は混雑で断ったときに、次に試すまで待たせる時間(Retry-After)。
const passwordCheckRetryAfter = time.Second

// PasswordGate はパスワード照合の同時実行数を制限するセマフォ。並行に使ってよい。
type PasswordGate struct {
	slots chan struct{}
	wait  time.Duration
}

// NewPasswordGate は同時に size 個まで照合させ、空きを wait まで待つゲートを返す。
// size が 1 未満なら 1 にする。
func NewPasswordGate(size int, wait time.Duration) *PasswordGate {
	return &PasswordGate{slots: make(chan struct{}, max(size, 1)), wait: wait}
}

// Enter は空きを待って入り、出るための関数を返す。wait の間に空かなければ、Retry-After つきの
// ErrUnavailable を返す。ctx が終われば ctx のエラーを返す。
func (g *PasswordGate) Enter(ctx context.Context) (func(), error) {
	leave := func() { <-g.slots }
	select {
	case g.slots <- struct{}{}:
		return leave, nil
	default:
	}
	timer := time.NewTimer(g.wait)
	defer timer.Stop()
	select {
	case g.slots <- struct{}{}:
		return leave, nil
	case <-timer.C:
		return nil, domain.RetryAfter(
			fmt.Errorf("%w: too many password checks in progress", domain.ErrUnavailable),
			passwordCheckRetryAfter,
		)
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for password check: %w", ctx.Err())
	}
}
