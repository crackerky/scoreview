package auth_test

import (
	"context"
	"crypto/sha256"
	"os"
	"strings"
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
	assert.NoError(t, auth.ValidatePassword(strings.Repeat("a", auth.MinPasswordLen)))
	// 文字数で数える(日本語のパスフレーズをバイト数で弾かない)
	assert.NoError(t, auth.ValidatePassword(strings.Repeat("あ", auth.MinPasswordLen)))
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
	session, err := auth.Login(ctx, db, "  Admin@Example.com ", password, baseTime)
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

	_, wrongPassword := auth.Login(ctx, db, "admin@example.com", "wrong password!!", baseTime)
	_, unknownEmail := auth.Login(ctx, db, "nobody@example.com", password, baseTime)
	require.ErrorIs(t, wrongPassword, domain.ErrUnauthorized)
	require.ErrorIs(t, unknownEmail, domain.ErrUnauthorized)
	assert.Equal(t, wrongPassword.Error(), unknownEmail.Error())
}

func TestSessionExpiry(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "staff@example.com", domain.RoleStaff)
	session, err := auth.Login(ctx, db, "staff@example.com", password, baseTime)
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
	session, err := auth.Login(ctx, db, "admin@example.com", password, baseTime)
	require.NoError(t, err)
	require.NoError(t, auth.Logout(ctx, db, session.Token))
	_, err = auth.Authenticate(ctx, db, session.Token, baseTime)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	// 2 回目のログアウトもエラーにしない
	require.NoError(t, auth.Logout(ctx, db, session.Token))
}

func TestLoginAttemptLimit(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	newUser(t, db, "admin@example.com", domain.RoleAdmin)

	for i := range auth.MaxLoginFailures {
		at := baseTime.Add(time.Duration(i) * time.Second)
		_, err := auth.Login(ctx, db, "admin@example.com", "wrong password!!", at)
		require.ErrorIs(t, err, domain.ErrUnauthorized, "attempt %d", i)
	}
	// 制限中は正しいパスワードでも断る
	_, err := auth.Login(ctx, db, "admin@example.com", password, baseTime.Add(time.Minute))
	require.ErrorIs(t, err, domain.ErrTooManyRequests)
	// 存在しないアドレスへの試行も同じように数える(他のアドレスには影響しない)
	_, err = auth.Login(ctx, db, "other@example.com", password, baseTime.Add(time.Minute))
	require.ErrorIs(t, err, domain.ErrUnauthorized)

	// 最初の失敗から窓が過ぎれば、古い失敗が外れてログインできる
	later := baseTime.Add(auth.LoginFailureWindow + auth.MaxLoginFailures*time.Second)
	_, err = auth.Login(ctx, db, "admin@example.com", password, later)
	require.NoError(t, err)

	// 成功すると失敗の記録が消える
	var count int64
	require.NoError(
		t,
		db.Model(&store.LoginFailure{}).Where("email = ?", "admin@example.com").Count(&count).Error,
	)
	assert.Zero(t, count)
}
