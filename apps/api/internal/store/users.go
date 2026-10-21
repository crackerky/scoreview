package store

import (
	"context"
	"fmt"
	"net/mail"
	"slices"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const (
	sqlOrderUsers         = "role, email, id"
	sqlStaffOfTournament  = "user_id = ? AND tournament_id = ?"
	sqlOrderTournamentIDs = "tournament_id"
	// maxEmailLen は RFC 5321 4.5.3.1 の転送経路の上限(254 文字)
	maxEmailLen = 254
)

// UserFields は管理画面で編集するユーザーの値。Email は正規化済み(auth.NormalizeEmail)であること。
type UserFields struct {
	Email       string
	DisplayName string
	Role        domain.Role
}

// UserRecord はユーザーと担当大会(staff のとき意味を持つ。tournament_id 昇順)。
type UserRecord struct {
	domain.User
	TournamentIDs []int64
}

func (f UserFields) validate() error {
	if len(f.Email) > maxEmailLen {
		return domain.Invalid("email", "too long")
	}
	if addr, err := mail.ParseAddress(f.Email); err != nil || addr.Address != f.Email {
		return domain.Invalid("email", "must be a plain email address")
	}
	if !f.Role.Valid() {
		return domain.Invalid("role", "must be admin or staff")
	}
	return nil
}

// ListUsers は全ユーザーを役割、メールアドレスの順に返す。
func ListUsers(ctx context.Context, db *gorm.DB) ([]UserRecord, error) {
	var users []User
	if err := db.WithContext(ctx).Order(sqlOrderUsers).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	var staff []TournamentStaff
	if err := db.WithContext(ctx).Order(sqlOrderTournamentIDs).Find(&staff).Error; err != nil {
		return nil, fmt.Errorf("list tournament staff: %w", err)
	}
	byUser := map[int64][]int64{}
	for _, row := range staff {
		byUser[row.UserID] = append(byUser[row.UserID], row.TournamentID)
	}
	out := make([]UserRecord, 0, len(users))
	for _, user := range users {
		ids := byUser[user.ID]
		if ids == nil {
			ids = []int64{}
		}
		out = append(out, UserRecord{User: toDomainUser(user), TournamentIDs: ids})
	}
	return out, nil
}

// GetUserRecord はユーザーを返す。無ければ USER_ID の NotFound。
func GetUserRecord(ctx context.Context, db *gorm.DB, id int64) (UserRecord, error) {
	user, err := findByID[User](ctx, db, id, domain.ResourceUser)
	if err != nil {
		return UserRecord{}, err
	}
	ids, err := StaffTournamentIDs(ctx, db, id)
	if err != nil {
		return UserRecord{}, err
	}
	return UserRecord{User: toDomainUser(user), TournamentIDs: ids}, nil
}

// CreateUser はユーザーを作る。passwordHash は auth.HashPassword の結果。同じメールアドレスが
// あれば ErrConflict。
func CreateUser(
	ctx context.Context,
	db *gorm.DB,
	fields UserFields,
	passwordHash string,
) (UserRecord, error) {
	if err := fields.validate(); err != nil {
		return UserRecord{}, err
	}
	row := User{
		Email:        fields.Email,
		PasswordHash: passwordHash,
		DisplayName:  fields.DisplayName,
		Role:         fields.Role,
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return UserRecord{}, writeError("create user", err)
	}
	return UserRecord{User: toDomainUser(row), TournamentIDs: []int64{}}, nil
}

// UpdateUser はユーザーの値を変える。passwordHash が nil でなければパスワードも変え、そのユーザーの
// 全セッションを消す(古いパスワードで入った端末を締め出す)。actor 自身の役割は変えられない
// (最後の管理者が自分を staff にして管理者がいなくなるのを防ぐ)。
func UpdateUser(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	id int64,
	fields UserFields,
	passwordHash *string,
) (UserRecord, error) {
	if err := fields.validate(); err != nil {
		return UserRecord{}, err
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := findByID[User](ctx, tx, id, domain.ResourceUser)
		if err != nil {
			return err
		}
		if id == actor.UserID && fields.Role != user.Role {
			return fmt.Errorf("%w: cannot change your own role", domain.ErrConflict)
		}
		user.Email, user.DisplayName, user.Role = fields.Email, fields.DisplayName, fields.Role
		if passwordHash != nil {
			user.PasswordHash = *passwordHash
			if err := tx.Where(sqlByUser, id).Delete(&Session{}).Error; err != nil {
				return fmt.Errorf("revoke sessions of user %d: %w", id, err)
			}
		}
		if err := tx.Save(&user).Error; err != nil {
			return writeError("update user", err)
		}
		return nil
	})
	if err != nil {
		return UserRecord{}, err
	}
	return GetUserRecord(ctx, db, id)
}

// DeleteUser はユーザーを消す(セッションと担当大会は ON DELETE CASCADE)。自分自身は消せない。
func DeleteUser(ctx context.Context, db *gorm.DB, actor domain.Actor, id int64) error {
	if id == actor.UserID {
		return fmt.Errorf("%w: cannot delete yourself", domain.ErrConflict)
	}
	return deleteByID[User](ctx, db, id, domain.ResourceUser)
}

// SetUserTournaments はユーザーの担当大会を tournamentIDs で置き換える。
func SetUserTournaments(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	tournamentIDs []int64,
) (UserRecord, error) {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findByID[User](ctx, tx, id, domain.ResourceUser); err != nil {
			return err
		}
		if err := tx.Where(sqlByUser, id).Delete(&TournamentStaff{}).Error; err != nil {
			return fmt.Errorf("clear tournaments of user %d: %w", id, err)
		}
		ids := slices.Clone(tournamentIDs)
		slices.Sort(ids)
		ids = slices.Compact(ids)
		if len(ids) == 0 {
			return nil
		}
		rows := make([]TournamentStaff, len(ids))
		for i, tournamentID := range ids {
			rows[i] = TournamentStaff{UserID: id, TournamentID: tournamentID}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return writeError("set tournaments of user", err)
		}
		return nil
	})
	if err != nil {
		return UserRecord{}, err
	}
	return GetUserRecord(ctx, db, id)
}

// StaffTournamentIDs はユーザーの担当大会の ID を昇順で返す。
func StaffTournamentIDs(ctx context.Context, db *gorm.DB, userID int64) ([]int64, error) {
	ids := []int64{}
	err := db.WithContext(ctx).Model(&TournamentStaff{}).Where(sqlByUser, userID).
		Order(sqlOrderTournamentIDs).Pluck("tournament_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("load tournaments of user %d: %w", userID, err)
	}
	return ids, nil
}

// IsTournamentStaff はユーザーが大会の担当かどうかを返す。
func IsTournamentStaff(ctx context.Context, db *gorm.DB, userID, tournamentID int64) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&TournamentStaff{}).
		Where(sqlStaffOfTournament, userID, tournamentID).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check staff of tournament %d: %w", tournamentID, err)
	}
	return count > 0, nil
}

// deleteByID は id の行を消す。無ければ resource の NotFound。
func deleteByID[T any](ctx context.Context, db *gorm.DB, id int64, resource domain.Resource) error {
	var row T
	result := db.WithContext(ctx).Where(sqlByID, id).Delete(&row)
	if result.Error != nil {
		return writeError(fmt.Sprintf("delete %s [%d]", resource, id), result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.NotFound(resource, id)
	}
	return nil
}
