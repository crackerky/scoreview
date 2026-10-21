package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// 認証(internal/auth)が使う読み書き。パスワードの照合とトークンの生成は auth が行い、
// ここは行の出し入れだけをする。

const (
	sqlByEmail             = "email = ?"
	sqlByTokenHash         = "token_hash = ?" //nolint:gosec // SQL の条件で、資格情報ではない
	sqlLiveSession         = "token_hash = ? AND expires_at > ?"
	sqlExpiredOfUser       = "user_id = ? AND expires_at <= ?"
	sqlByUser              = "user_id = ?"
	sqlOtherSessionsOfUser = "user_id = ? AND token_hash <> ?"
	sqlFailuresSince       = "email = ? AND attempted_at > ?"
	sqlFailuresFromIPSince = "ip = ? AND attempted_at > ?"
	sqlFailuresOlderThan   = "attempted_at <= ?"
	sqlOrderAttemptedAt    = "attempted_at"
)

// Credential はパスワードでの本人確認に要るユーザーの値。
type Credential struct {
	UserID       int64
	PasswordHash string
}

// FindCredential は正規化済みのメールアドレスのユーザーの資格情報を返す。無ければ found=false
// (エラーにしない。呼び出し側がダミーの照合で処理時間を揃えるため)。
func FindCredential(ctx context.Context, db *gorm.DB, email string) (Credential, bool, error) {
	var users []User
	err := db.WithContext(ctx).
		Select("id", "password_hash").
		Where(sqlByEmail, email).
		Limit(1).
		Find(&users).
		Error
	if err != nil {
		return Credential{}, false, fmt.Errorf("find user by email: %w", err)
	}
	if len(users) == 0 {
		return Credential{}, false, nil
	}
	return Credential{UserID: users[0].ID, PasswordHash: users[0].PasswordHash}, true, nil
}

func toDomainUser(user User) domain.User {
	return domain.User{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}
}

// GetUser はユーザーを返す。無ければ ErrUnauthorized(ログイン中に削除されたユーザーは認証を失う)。
func GetUser(ctx context.Context, db *gorm.DB, id int64) (domain.User, error) {
	var users []User
	if err := db.WithContext(ctx).Where(sqlByID, id).Limit(1).Find(&users).Error; err != nil {
		return domain.User{}, fmt.Errorf("load user %d: %w", id, err)
	}
	if len(users) == 0 {
		return domain.User{}, fmt.Errorf("%w: user %d does not exist", domain.ErrUnauthorized, id)
	}
	return toDomainUser(users[0]), nil
}

// CreateSession はセッションを保存する。tokenHash はトークンの SHA-256。
func CreateSession(
	ctx context.Context,
	db *gorm.DB,
	userID int64,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	row := Session{TokenHash: tokenHash, UserID: userID, ExpiresAt: expiresAt}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// FindSessionUser は有効期限内のセッションのユーザーを返す。無い、期限切れ、ユーザーが削除済みなら
// ErrUnauthorized。
func FindSessionUser(
	ctx context.Context,
	db *gorm.DB,
	tokenHash []byte,
	now time.Time,
) (domain.User, error) {
	var sessions []Session
	err := db.WithContext(ctx).Where(sqlLiveSession, tokenHash, now).Limit(1).Find(&sessions).Error
	if err != nil {
		return domain.User{}, fmt.Errorf("find session: %w", err)
	}
	if len(sessions) == 0 {
		return domain.User{}, fmt.Errorf("%w: no live session", domain.ErrUnauthorized)
	}
	return GetUser(ctx, db, sessions[0].UserID)
}

// DeleteSession はセッションを消す。無くてもエラーにしない(ログアウトを何度押してもよい)。
func DeleteSession(ctx context.Context, db *gorm.DB, tokenHash []byte) error {
	if err := db.WithContext(ctx).
		Where(sqlByTokenHash, tokenHash).
		Delete(&Session{}).
		Error; err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions はユーザーの期限切れのセッションを消す(ログインのたびに掃除する)。
func DeleteExpiredSessions(ctx context.Context, db *gorm.DB, userID int64, now time.Time) error {
	err := db.WithContext(ctx).Where(sqlExpiredOfUser, userID, now).Delete(&Session{}).Error
	if err != nil {
		return fmt.Errorf("delete expired sessions of user %d: %w", userID, err)
	}
	return nil
}

// LoginFailureStats は、ある試行の前の時点の直近の失敗(窓の中だけ)。
type LoginFailureStats struct {
	// AccountFailures と AccountLast は同じメールアドレスへの失敗の数と、最後の失敗の時刻
	// (無ければゼロ値)
	AccountFailures int64
	AccountLast     time.Time
	// IPFailures と IPOldest は同じ IP からの(どのアドレスへの)失敗の数と、最も古い失敗の時刻。
	// IP が分からない(空文字)ときは数えない
	IPFailures int64
	IPOldest   time.Time
}

// loginLockClass は advisory lock の 2 引数形式の 1 つ目の鍵。ログイン試行の lock であることを表し、
// 他の用途の lock と鍵が重ならないようにする。
const loginLockClass = 1

// sqlLockLoginAttempts は同じメールアドレスの試行を、トランザクションの終わりまで直列にする。
const sqlLockLoginAttempts = "SELECT pg_advisory_xact_lock(?, hashtext(?))"

// BeginLoginAttempt はログイン試行の確認と記録を、同じメールアドレスについて直列に行う。
//
// トランザクションの中で email の advisory lock を取り、窓より古い記録を消してから直近の失敗を
// 数えて decide に渡す。decide がエラーを返したらそれを返し、何も記録しない。nil なら、この試行を
// 照合の前に失敗として記録する(照合に成功したら呼び出し側が ClearLoginFailures で消す)。
// 照合の前に記録するので、並行する試行が「数える → 照合 → 記録」の間をすり抜けられない
// (セキュリティ点検 M-1)。
func BeginLoginAttempt(
	ctx context.Context,
	db *gorm.DB,
	email, ip string,
	now time.Time,
	window time.Duration,
	decide func(LoginFailureStats) error,
) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockLoginAttempts(tx, email); err != nil {
			return err
		}
		since := now.Add(-window)
		if err := tx.Where(sqlFailuresOlderThan, since).Delete(&LoginFailure{}).Error; err != nil {
			return fmt.Errorf("prune login failures: %w", err)
		}
		stats, err := loginFailureStats(tx, email, ip, since)
		if err != nil {
			return err
		}
		if err := decide(stats); err != nil {
			return err
		}
		row := LoginFailure{Email: email, IP: ip, AttemptedAt: now}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("record login attempt: %w", err)
		}
		return nil
	})
}

func lockLoginAttempts(tx *gorm.DB, email string) error {
	// SQL は定数で、値はプレースホルダで渡す
	//nolint:forbidigo // pg_advisory_xact_lock は void を返し、クエリビルダ(Select + Scan)で呼べない
	err := tx.Exec(sqlLockLoginAttempts, loginLockClass, email).Error
	if err != nil {
		return fmt.Errorf("lock login attempts: %w", err)
	}
	return nil
}

func loginFailureStats(
	tx *gorm.DB,
	email, ip string,
	since time.Time,
) (LoginFailureStats, error) {
	var stats LoginFailureStats
	var account []LoginFailure
	err := tx.Where(sqlFailuresSince, email, since).Order(sqlOrderAttemptedAt).Find(&account).Error
	if err != nil {
		return LoginFailureStats{}, fmt.Errorf("load login failures: %w", err)
	}
	stats.AccountFailures = int64(len(account))
	if len(account) > 0 {
		stats.AccountLast = account[len(account)-1].AttemptedAt
	}
	if ip == "" {
		return stats, nil
	}
	var fromIP []LoginFailure
	err = tx.Select("attempted_at").
		Where(sqlFailuresFromIPSince, ip, since).
		Order(sqlOrderAttemptedAt).
		Find(&fromIP).
		Error
	if err != nil {
		return LoginFailureStats{}, fmt.Errorf("load login failures from ip: %w", err)
	}
	stats.IPFailures = int64(len(fromIP))
	if len(fromIP) > 0 {
		stats.IPOldest = fromIP[0].AttemptedAt
	}
	return stats, nil
}

// ClearLoginFailures はメールアドレスのログイン失敗の記録を消す(ログインに成功したとき)。
// 照合の前に記録したこの試行の分も消える。
func ClearLoginFailures(ctx context.Context, db *gorm.DB, email string) error {
	if err := db.WithContext(ctx).
		Where(sqlByEmail, email).
		Delete(&LoginFailure{}).
		Error; err != nil {
		return fmt.Errorf("clear login failures: %w", err)
	}
	return nil
}

// SetOwnPassword は本人が変えたパスワードのハッシュを保存し、keepTokenHash(変更に使ったセッション)
// 以外のそのユーザーのセッションを消す(古いパスワードで入った他の端末を締め出す)。
func SetOwnPassword(
	ctx context.Context,
	db *gorm.DB,
	userID int64,
	passwordHash string,
	keepTokenHash []byte,
) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&User{}).Where(sqlByID, userID).Update("password_hash", passwordHash)
		if result.Error != nil {
			return fmt.Errorf("set password of user %d: %w", userID, result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("%w: user %d does not exist", domain.ErrUnauthorized, userID)
		}
		err := tx.Where(sqlOtherSessionsOfUser, userID, keepTokenHash).Delete(&Session{}).Error
		if err != nil {
			return fmt.Errorf("revoke other sessions of user %d: %w", userID, err)
		}
		return nil
	})
}
