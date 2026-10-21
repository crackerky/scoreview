package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// SessionTTL はセッションの有効期間(ログインからの絶対時間)。大会は 1 日単位で運営され、
// 記録係が試合の途中で切れないよう 1 日の運営時間より長く、端末の置き忘れで翌日まで
// 残らないよう 24 時間にする。延長はしない(期限が来たらログインし直す)。
const SessionTTL = 24 * time.Hour

// sessionTokenBytes はセッショントークンの乱数のバイト数。256 bit は総当たりで当てられない長さ
// (OWASP Session Management Cheat Sheet の下限 64 bit を十分に超える)。
const sessionTokenBytes = 32

// ログイン試行の制限。同じメールアドレスで LoginFailureWindow の間に MaxLoginFailures 回失敗したら、
// 窓から古い失敗が外れるまで照合せずに断る(パスワードの総当たり対策。OWASP Authentication
// Cheat Sheet の「アカウント単位のロックアウト」。窓が過ぎれば自動で解ける)。
const (
	MaxLoginFailures   = 10
	LoginFailureWindow = 15 * time.Minute
)

// ErrInvalidCredentials はメールアドレスかパスワードが違うことを表す。どちらが違うかは区別しない。
var ErrInvalidCredentials = fmt.Errorf("%w: invalid email or password", domain.ErrUnauthorized)

// ErrTooManyAttempts はログイン試行の制限を超えたことを表す。
var ErrTooManyAttempts = fmt.Errorf("%w: too many failed login attempts", domain.ErrTooManyRequests)

// Session はログインで発行したセッション。Token は平文で、呼び出し元に返すだけで保存しない。
type Session struct {
	Token     string
	ExpiresAt time.Time
	User      domain.User
}

// NormalizeEmail はメールアドレスを比較用の形(前後の空白を除いた小文字)にする。
// users.email にはこの形で保存する。
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// newSessionToken は推測できないトークン(crypto/rand の 32 バイトを base64url)と、その SHA-256 を返す。
func newSessionToken() (string, []byte, error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("read session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, HashToken(token), nil
}

// HashToken はトークンを保存と照合に使う SHA-256 にする。トークンは 256 bit の乱数なので、
// 塩も遅いハッシュも要らない(辞書攻撃が成り立たない)。
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// Login はメールアドレスとパスワードでログインし、セッションを発行する。
//
// 手順は「本人確認 → ユーザーの特定 → セッション発行」の 3 段(docs/line-integration.md)。
// LINE ログインを足すときは本人確認の段に LINE の認可コードの検証を並べ、後の 2 段を共有する。
// 失敗はメールアドレスの有無を区別せず ErrInvalidCredentials を返し、存在しないアドレスでも
// ダミーのハッシュを照合して処理時間を揃える。制限を超えたら ErrTooManyAttempts。
func Login(
	ctx context.Context,
	db *gorm.DB,
	email, password string,
	now time.Time,
) (Session, error) {
	key := NormalizeEmail(email)
	failures, err := store.CountLoginFailures(ctx, db, key, now.Add(-LoginFailureWindow))
	if err != nil {
		return Session{}, err
	}
	if failures >= MaxLoginFailures {
		return Session{}, ErrTooManyAttempts
	}

	// 1. 本人確認(パスワードの照合)
	userID, err := verifyPassword(ctx, db, key, password)
	if errors.Is(err, ErrInvalidCredentials) {
		if recErr := store.RecordLoginFailure(
			ctx,
			db,
			key,
			now,
			LoginFailureWindow,
		); recErr != nil {
			return Session{}, recErr
		}
		return Session{}, err
	}
	if err != nil {
		return Session{}, err
	}

	// 2. ユーザーの特定
	user, err := store.GetUser(ctx, db, userID)
	if err != nil {
		return Session{}, err
	}

	// 3. セッション発行
	return issueSession(ctx, db, user, key, now)
}

// verifyPassword はパスワードを照合し、一致したユーザーの ID を返す。
func verifyPassword(ctx context.Context, db *gorm.DB, email, password string) (int64, error) {
	cred, found, err := store.FindCredential(ctx, db, email)
	if err != nil {
		return 0, err
	}
	if !found {
		verifyDummy(password)
		return 0, ErrInvalidCredentials
	}
	ok, err := VerifyPassword(cred.PasswordHash, password)
	if err != nil {
		return 0, fmt.Errorf("verify password of user %d: %w", cred.UserID, err)
	}
	if !ok {
		return 0, ErrInvalidCredentials
	}
	return cred.UserID, nil
}

func issueSession(
	ctx context.Context,
	db *gorm.DB,
	user domain.User,
	email string,
	now time.Time,
) (Session, error) {
	token, hash, err := newSessionToken()
	if err != nil {
		return Session{}, err
	}
	expiresAt := now.Add(SessionTTL)
	if err := store.DeleteExpiredSessions(ctx, db, user.ID, now); err != nil {
		return Session{}, err
	}
	if err := store.CreateSession(ctx, db, user.ID, hash, expiresAt); err != nil {
		return Session{}, err
	}
	if err := store.ClearLoginFailures(ctx, db, email); err != nil {
		return Session{}, err
	}
	return Session{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

// Authenticate はセッショントークンのユーザーを返す。トークンが空、未知、期限切れなら ErrUnauthorized。
func Authenticate(
	ctx context.Context,
	db *gorm.DB,
	token string,
	now time.Time,
) (domain.User, error) {
	if token == "" {
		return domain.User{}, fmt.Errorf("%w: no session token", domain.ErrUnauthorized)
	}
	return store.FindSessionUser(ctx, db, HashToken(token), now)
}

// Logout はセッションを消す。
func Logout(ctx context.Context, db *gorm.DB, token string) error {
	return store.DeleteSession(ctx, db, HashToken(token))
}
