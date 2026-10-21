package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// PasswordChange は本人によるパスワードの変更。Token は変更に使うセッションのトークン(このセッション
// だけを残し、他の端末のセッションを消す)。ClientIP は web が渡した利用者の IP。
type PasswordChange struct {
	Current  string
	New      string
	Token    string
	ClientIP string
}

// ChangePassword は user 本人のパスワードを変える(セキュリティ点検 L-8)。新しいパスワードは
// ValidatePassword を満たし、今のパスワードと違うこと(違反は newPassword の検証エラー)。
// 今のパスワードの照合はログインと同じ試行の制限と同時実行数の制限に掛ける(セッションを奪った人に
// 今のパスワードを総当たりさせない)。違えば currentPassword の検証エラーにする(セッションは
// 有効なので 401 にしない)。
func ChangePassword(
	ctx context.Context,
	db *gorm.DB,
	gate *PasswordGate,
	user domain.User,
	change PasswordChange,
	now time.Time,
) error {
	if err := ValidatePassword(change.New); err != nil {
		var invalid *domain.ValidationError
		if errors.As(err, &invalid) {
			return domain.Invalid("newPassword", invalid.Reason)
		}
		return err
	}
	if change.New == change.Current {
		return domain.Invalid("newPassword", "must differ from the current password")
	}
	leave, err := gate.Enter(ctx)
	if err != nil {
		return err
	}
	defer leave()

	key := NormalizeEmail(user.Email)
	err = store.BeginLoginAttempt(ctx, db, key, change.ClientIP, now, LoginFailureWindow,
		func(stats store.LoginFailureStats) error { return checkLoginAttempt(stats, now) })
	if err != nil {
		return err
	}
	if _, err := verifyPassword(ctx, db, key, change.Current); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return domain.Invalid("currentPassword", "is incorrect")
		}
		return err
	}
	hash, err := HashPassword(change.New)
	if err != nil {
		return err
	}
	if err := store.SetOwnPassword(ctx, db, user.ID, hash, HashToken(change.Token)); err != nil {
		return err
	}
	return store.ClearLoginFailures(ctx, db, key)
}
