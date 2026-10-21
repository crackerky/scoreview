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
	sqlByEmail           = "email = ?"
	sqlByTokenHash       = "token_hash = ?" //nolint:gosec // SQL の条件で、資格情報ではない
	sqlLiveSession       = "token_hash = ? AND expires_at > ?"
	sqlExpiredOfUser     = "user_id = ? AND expires_at <= ?"
	sqlByUser            = "user_id = ?"
	sqlFailuresSince     = "email = ? AND attempted_at > ?"
	sqlFailuresOlderThan = "email = ? AND attempted_at <= ?"
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

// CountLoginFailures は since より後の、メールアドレスのログイン失敗の数を返す。
func CountLoginFailures(
	ctx context.Context,
	db *gorm.DB,
	email string,
	since time.Time,
) (int64, error) {
	var count int64
	err := db.WithContext(ctx).
		Model(&LoginFailure{}).
		Where(sqlFailuresSince, email, since).
		Count(&count).
		Error
	if err != nil {
		return 0, fmt.Errorf("count login failures: %w", err)
	}
	return count, nil
}

// RecordLoginFailure はログイン失敗を 1 件記録し、window より古い同じアドレスの記録を消す
// (表が際限なく伸びないように)。
func RecordLoginFailure(
	ctx context.Context,
	db *gorm.DB,
	email string,
	at time.Time,
	window time.Duration,
) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(sqlFailuresOlderThan, email, at.Add(-window)).
			Delete(&LoginFailure{}).
			Error; err != nil {
			return fmt.Errorf("prune login failures: %w", err)
		}
		if err := tx.Create(&LoginFailure{Email: email, AttemptedAt: at}).Error; err != nil {
			return fmt.Errorf("record login failure: %w", err)
		}
		return nil
	})
}

// ClearLoginFailures はメールアドレスのログイン失敗の記録を消す(ログインに成功したとき)。
func ClearLoginFailures(ctx context.Context, db *gorm.DB, email string) error {
	if err := db.WithContext(ctx).
		Where(sqlByEmail, email).
		Delete(&LoginFailure{}).
		Error; err != nil {
		return fmt.Errorf("clear login failures: %w", err)
	}
	return nil
}
