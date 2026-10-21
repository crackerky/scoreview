package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

// ログイン試行の制限(docs/api.md「認証」)。
//
// アカウント単位: 同じメールアドレスで LoginFailureWindow の間に FreeLoginFailures 回失敗したら、
// 以後は最後の失敗から LoginDelay の間を空けないと照合しない。間隔は失敗のたびに倍になり、
// MaxLoginDelay で頭打ちになる。一定時間締め出す方式(以前は 15 分に 10 回で 15 分)だと、他人が
// わざと間違えるだけで正しい利用者を締め出せる(セキュリティ点検 M-5)。間隔を延ばす方式なら
// 正しい利用者は最長でも MaxLoginDelay 待てば試せ、総当たりは 1 分に 1 回程度に抑えられる
// (OWASP Authentication Cheat Sheet の Account Lockout: 失敗ごとに指数的に延ばす)。
//
// IP 単位: 同じ IP から LoginFailureWindow の間に MaxLoginFailuresPerIP 回失敗したら、窓から古い
// 失敗が外れるまで照合しない(多くのアドレスへ同じパスワードを試す攻撃の対策)。IP は web が渡す。
// 会場の Wi-Fi で記録係が同じ IP を共有しても、30 回の失敗には届かない。
const (
	LoginFailureWindow    = 15 * time.Minute
	FreeLoginFailures     = 5
	LoginBaseDelay        = time.Second
	MaxLoginDelay         = time.Minute
	MaxLoginFailuresPerIP = 30
)

// LoginDelay は窓の中の失敗が failures 回のとき、最後の失敗から次の試行まで空ける時間。
// FreeLoginFailures 回未満なら 0、そこから LoginBaseDelay を起点に倍々で、MaxLoginDelay まで。
func LoginDelay(failures int64) time.Duration {
	if failures < FreeLoginFailures {
		return 0
	}
	delay := LoginBaseDelay
	for range failures - FreeLoginFailures {
		delay *= 2
		if delay >= MaxLoginDelay {
			return MaxLoginDelay
		}
	}
	return delay
}

// Attempt はログインの試行。ClientIP は web が渡した利用者の IP(分からなければ空文字)。
type Attempt struct {
	Email    string
	Password string
	ClientIP string
}

// ErrInvalidCredentials はメールアドレスかパスワードが違うことを表す。どちらが違うかは区別しない。
var ErrInvalidCredentials = fmt.Errorf("%w: invalid email or password", domain.ErrUnauthorized)

// ErrTooManyAttempts はログイン試行の制限を超えたことを表す。Login は待つ時間を付けた
// domain.RetryAfterError に包んで返す。
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
// ダミーのハッシュを照合して処理時間を揃える。試行の制限に掛かったら ErrTooManyAttempts、
// gate が混んでいれば domain.ErrUnavailable(どちらも domain.RetryAfterError に包む)。
func Login(
	ctx context.Context,
	db *gorm.DB,
	gate *PasswordGate,
	attempt Attempt,
	now time.Time,
) (Session, error) {
	leave, err := gate.Enter(ctx)
	if err != nil {
		return Session{}, err
	}
	defer leave()

	key := NormalizeEmail(attempt.Email)
	err = store.BeginLoginAttempt(ctx, db, key, attempt.ClientIP, now, LoginFailureWindow,
		func(stats store.LoginFailureStats) error { return checkLoginAttempt(stats, now) })
	if err != nil {
		return Session{}, err
	}

	// 1. 本人確認(パスワードの照合)。失敗の記録は BeginLoginAttempt が照合の前に済ませた
	userID, err := verifyPassword(ctx, db, key, attempt.Password)
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

// checkLoginAttempt は直近の失敗から、今の試行を照合してよいかを決める。
func checkLoginAttempt(stats store.LoginFailureStats, now time.Time) error {
	if stats.IPFailures >= MaxLoginFailuresPerIP {
		return domain.RetryAfter(ErrTooManyAttempts, stats.IPOldest.Add(LoginFailureWindow).Sub(now))
	}
	if wait := stats.AccountLast.Add(LoginDelay(stats.AccountFailures)).Sub(now); wait > 0 {
		return domain.RetryAfter(ErrTooManyAttempts, wait)
	}
	return nil
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
