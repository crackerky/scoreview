package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

const password = "correct horse battery staple"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	// OWASP の推奨構成(m=19456, t=2, p=1)で PHC 文字列にする
	assert.True(t, strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$"), hash)

	ok, err := auth.VerifyPassword(hash, password)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = auth.VerifyPassword(hash, password+"x")
	require.NoError(t, err)
	assert.False(t, ok)

	// 同じパスワードでも塩が違うので別のハッシュになる
	again, err := auth.HashPassword(password)
	require.NoError(t, err)
	assert.NotEqual(t, hash, again)
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	for _, encoded := range []string{
		"", "plain", "$argon2i$v=19$m=19456,t=2,p=1$c2FsdA$a2V5",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$a2V5", "$argon2id$v=19$m=0,t=2,p=1$c2FsdA$a2V5",
		"$argon2id$v=19$m=19456,t=2,p=1$!!$a2V5",
	} {
		_, err := auth.VerifyPassword(encoded, password)
		assert.ErrorIs(t, err, auth.ErrMalformedHash, encoded)
	}
}

func TestValidatePassword(t *testing.T) {
	assert.ErrorIs(t, auth.ValidatePassword("short"), domain.ErrBadRequest)
	assert.ErrorIs(
		t,
		auth.ValidatePassword(strings.Repeat("a", auth.MaxPasswordLen+1)),
		domain.ErrBadRequest,
	)
	assert.NoError(t, auth.ValidatePassword("correct horse battery staple"))
	// 文字数で数える(日本語のパスフレーズをバイト数で弾かない)
	assert.NoError(t, auth.ValidatePassword("あいうえおかきくけこさし"))
}

// 同じ文字の繰り返しと、よく使われるパスワードを拒否する(セキュリティ点検 L-8)。
func TestValidatePasswordRejectsWeakPasswords(t *testing.T) {
	for _, weak := range []string{
		strings.Repeat("a", auth.MinPasswordLen),
		strings.Repeat("あ", auth.MinPasswordLen),
		strings.Repeat("ab", 6),
		strings.Repeat("abc", 5),
		strings.Repeat("1234", 3) + "1",
		// NCSC の上位 10 万件にある 12 文字以上のもの。大文字小文字は区別しない
		"1qaz2wsx3edc", "Q1W2E3R4T5Y6", "123456789012", "qwertyuiopasdfghjkl",
	} {
		assert.ErrorIs(t, auth.ValidatePassword(weak), domain.ErrBadRequest, weak)
	}
	for _, ok := range []string{"abcdeabcdeab", "kumo no ue no ultimate", "a long admin password"} {
		assert.NoError(t, auth.ValidatePassword(ok), ok)
	}
}

func newUser(t *testing.T, db *gorm.DB, email string, role domain.Role) store.UserRecord {
	t.Helper()
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	user, err := store.CreateUser(context.Background(), db,
		store.UserFields{Email: email, DisplayName: "Test", Role: role}, hash)
	require.NoError(t, err)
	return user
}

var baseTime = time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)

func TestLoginIssuesHashedSession(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	user := newUser(t, db, "admin@example.com", domain.RoleAdmin)

	// メールアドレスは前後の空白と大文字小文字を区別しない
	session, err := login(ctx, db, "  Admin@Example.com ", password, baseTime)
	require.NoError(t, err)
	assert.Equal(t, user.ID, session.User.ID)
	assert.Equal(t, baseTime.Add(auth.SessionTTL), session.ExpiresAt)
	assert.Len(t, session.Token, 43) // 32 バイトの base64url(パディングなし)

	// DB には平文ではなく SHA-256 だけがある
	var stored []store.Session
	require.NoError(t, db.Find(&stored).Error)
	require.Len(t, stored, 1)
	sum := sha256.Sum256([]byte(session.Token))
	assert.Equal(t, sum[:], stored[0].TokenHash)
	assert.NotContains(t, string(stored[0].TokenHash), session.Token)

	got, err := auth.Authenticate(ctx, db, session.Token, baseTime.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, domain.RoleAdmin, got.Role)
}

func TestLoginFailureDoesNotRevealEmail(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "admin@example.com", domain.RoleAdmin)

	_, wrongPassword := login(ctx, db, "admin@example.com", "wrong password!!", baseTime)
	_, unknownEmail := login(ctx, db, "nobody@example.com", password, baseTime)
	require.ErrorIs(t, wrongPassword, domain.ErrUnauthorized)
	require.ErrorIs(t, unknownEmail, domain.ErrUnauthorized)
	assert.Equal(t, wrongPassword.Error(), unknownEmail.Error())
}

func TestSessionExpiry(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "staff@example.com", domain.RoleStaff)
	session, err := login(ctx, db, "staff@example.com", password, baseTime)
	require.NoError(t, err)

	_, err = auth.Authenticate(ctx, db, session.Token, baseTime.Add(auth.SessionTTL-time.Second))
	require.NoError(t, err)
	_, err = auth.Authenticate(ctx, db, session.Token, baseTime.Add(auth.SessionTTL))
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	_, err = auth.Authenticate(ctx, db, "", baseTime)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	_, err = auth.Authenticate(ctx, db, "unknown-token", baseTime)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestLogout(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "admin@example.com", domain.RoleAdmin)
	session, err := login(ctx, db, "admin@example.com", password, baseTime)
	require.NoError(t, err)
	require.NoError(t, auth.Logout(ctx, db, session.Token))
	_, err = auth.Authenticate(ctx, db, session.Token, baseTime)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	// 2 回目のログアウトもエラーにしない
	require.NoError(t, auth.Logout(ctx, db, session.Token))
}

// login は制限の掛からない大きさのゲートでログインする(IP は分からない扱い)。
func login(
	ctx context.Context,
	db *gorm.DB,
	email, pass string,
	at time.Time,
) (auth.Session, error) {
	gate := auth.NewPasswordGate(auth.MaxConcurrentPasswordChecks, time.Minute)
	return auth.Login(ctx, db, gate, auth.Attempt{Email: email, Password: pass}, at)
}

func TestLoginDelay(t *testing.T) {
	for failures, want := range map[int64]time.Duration{
		0: 0, auth.FreeLoginFailures - 1: 0,
		auth.FreeLoginFailures:     time.Second,
		auth.FreeLoginFailures + 1: 2 * time.Second,
		auth.FreeLoginFailures + 5: 32 * time.Second,
		auth.FreeLoginFailures + 6: auth.MaxLoginDelay,
		1000:                       auth.MaxLoginDelay,
	} {
		assert.Equal(t, want, auth.LoginDelay(failures), "failures %d", failures)
	}
}

// retryAfter は err が RetryAfterError なら待つ時間を返す。
func retryAfter(t *testing.T, err error) time.Duration {
	t.Helper()
	var retry *domain.RetryAfterError
	require.ErrorAs(t, err, &retry)
	return retry.After
}

func TestLoginAttemptLimit(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "admin@example.com", domain.RoleAdmin)

	// FreeLoginFailures 回までは間を空けずに試せる
	for i := range auth.FreeLoginFailures {
		_, err := login(ctx, db, "admin@example.com", "wrong password!!", baseTime)
		require.ErrorIs(t, err, domain.ErrUnauthorized, "attempt %d", i)
	}
	// その後は最後の失敗から 1 秒空けるまで、正しいパスワードでも照合しない
	_, err := login(ctx, db, "admin@example.com", password, baseTime)
	require.ErrorIs(t, err, domain.ErrTooManyRequests)
	assert.Equal(t, time.Second, retryAfter(t, err))
	// 断った試行は数えない(待てば次の間隔は延びない)
	at := baseTime.Add(time.Second)
	_, err = login(ctx, db, "admin@example.com", "wrong password!!", at)
	require.ErrorIs(t, err, domain.ErrUnauthorized)
	// 失敗のたびに間隔が倍になる
	_, err = login(ctx, db, "admin@example.com", password, at.Add(time.Second))
	require.ErrorIs(t, err, domain.ErrTooManyRequests)
	assert.Equal(t, time.Second, retryAfter(t, err))
	// 他のアドレスには影響しない
	_, err = login(ctx, db, "other@example.com", password, at)
	require.ErrorIs(t, err, domain.ErrUnauthorized)

	// 正しい利用者は間隔を待てばログインでき、成功すると失敗の記録が消える
	_, err = login(ctx, db, "admin@example.com", password, at.Add(2*time.Second))
	require.NoError(t, err)
	var count int64
	require.NoError(
		t,
		db.Model(&store.LoginFailure{}).Where("email = ?", "admin@example.com").Count(&count).Error,
	)
	assert.Zero(t, count)
}

func TestLoginLimitWaitsAtMostMaxDelay(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "admin@example.com", domain.RoleAdmin)
	// 他人が窓の中で何度間違えても、正しい利用者が待つのは最後の失敗から MaxLoginDelay まで
	at := baseTime
	for range 20 {
		_, err := login(ctx, db, "admin@example.com", "wrong password!!", at)
		require.ErrorIs(t, err, domain.ErrUnauthorized)
		at = at.Add(auth.LoginDelay(int64(auth.FreeLoginFailures + 20)))
	}
	_, err := login(ctx, db, "admin@example.com", password, at)
	require.NoError(t, err)
}

func TestLoginLimitPerIP(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "admin@example.com", domain.RoleAdmin)
	gate := auth.NewPasswordGate(auth.MaxConcurrentPasswordChecks, time.Minute)
	attempt := func(email, pass, ip string, at time.Time) error {
		_, err := auth.Login(ctx, db, gate, auth.Attempt{Email: email, Password: pass, ClientIP: ip}, at)
		return err
	}
	// 同じ IP から多くのアドレスへ 1 回ずつ試す(アカウント単位の制限には掛からない)
	for i := range auth.MaxLoginFailuresPerIP {
		email := fmt.Sprintf("user%d@example.com", i)
		require.ErrorIs(t, attempt(email, password, "203.0.113.7", baseTime), domain.ErrUnauthorized)
	}
	err := attempt("admin@example.com", password, "203.0.113.7", baseTime.Add(time.Minute))
	require.ErrorIs(t, err, domain.ErrTooManyRequests)
	assert.Equal(t, auth.LoginFailureWindow-time.Minute, retryAfter(t, err))
	// 別の IP と、IP の分からない試行には掛からない
	require.NoError(t, attempt("admin@example.com", password, "198.51.100.1", baseTime.Add(time.Minute)))
	require.NoError(t, attempt("admin@example.com", password, "", baseTime.Add(time.Minute)))
	// 窓が過ぎれば同じ IP からも試せる
	later := baseTime.Add(auth.LoginFailureWindow)
	require.NoError(t, attempt("admin@example.com", password, "203.0.113.7", later))
}

// 同じアドレスへ並行に送っても、照合するのは間隔なしで許す回数まで(セキュリティ点検 M-1。
// 以前は確認と記録の間をすり抜けて 60 並列で 58 回照合できた)。
func TestConcurrentLoginAttemptsAreSerialized(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "admin@example.com", domain.RoleAdmin)
	gate := auth.NewPasswordGate(auth.MaxConcurrentPasswordChecks, time.Minute)
	const parallel = 60
	errs := make([]error, parallel)
	var wg sync.WaitGroup
	for i := range parallel {
		wg.Go(func() {
			attempt := auth.Attempt{Email: "admin@example.com", Password: fmt.Sprintf("bad guess %04d", i)}
			_, errs[i] = auth.Login(ctx, db, gate, attempt, baseTime)
		})
	}
	wg.Wait()
	checked, limited := 0, 0
	for _, err := range errs {
		switch {
		case errors.Is(err, domain.ErrUnauthorized):
			checked++
		case errors.Is(err, domain.ErrTooManyRequests):
			limited++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	assert.Equal(t, auth.FreeLoginFailures, checked)
	assert.Equal(t, parallel-auth.FreeLoginFailures, limited)
	var recorded int64
	require.NoError(t, db.Model(&store.LoginFailure{}).Count(&recorded).Error)
	assert.EqualValues(t, auth.FreeLoginFailures, recorded)
}

func TestPasswordGateRejectsWhenBusy(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "admin@example.com", domain.RoleAdmin)
	gate := auth.NewPasswordGate(1, 0)
	attempt := auth.Attempt{Email: "admin@example.com", Password: password}
	// 照合中の席がふさがっているうちに次を送る
	leave, err := gate.Enter(ctx)
	require.NoError(t, err)
	_, err = auth.Login(ctx, db, gate, attempt, baseTime)
	require.ErrorIs(t, err, domain.ErrUnavailable)
	assert.Equal(t, time.Second, retryAfter(t, err))
	leave()
	_, err = auth.Login(ctx, db, gate, attempt, baseTime)
	require.NoError(t, err)
}
