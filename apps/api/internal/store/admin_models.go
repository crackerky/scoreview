package store

import (
	"time"

	"scoreview/api/internal/domain"
)

// 管理画面と認証の表の GORM モデル(models.go と同じく store の外へ出さない)。
// スキーマの正本は migrations/00001_init.sql と 00003_admin_auth_bracket_slots.sql。

// User は users の行。PasswordHash は argon2id の PHC 文字列。
type User struct {
	ID           int64       `gorm:"column:id;primaryKey"`
	Email        string      `gorm:"column:email"`
	PasswordHash string      `gorm:"column:password_hash"`
	DisplayName  string      `gorm:"column:display_name"`
	Role         domain.Role `gorm:"column:role"`
	CreatedAt    time.Time   `gorm:"column:created_at"`
}

// TableName は GORM に表名を伝える。
func (User) TableName() string { return "users" }

// Session は sessions の行。TokenHash はトークンの SHA-256(平文は保存しない)。
type Session struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	TokenHash []byte    `gorm:"column:token_hash"`
	UserID    int64     `gorm:"column:user_id"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName は GORM に表名を伝える。
func (Session) TableName() string { return "sessions" }

// TournamentStaff は tournament_staff の行(記録係の担当大会)。
type TournamentStaff struct {
	UserID       int64 `gorm:"column:user_id;primaryKey"`
	TournamentID int64 `gorm:"column:tournament_id;primaryKey"`
}

// TableName は GORM に表名を伝える。
func (TournamentStaff) TableName() string { return "tournament_staff" }

// LoginFailure は login_failures の行。Email は小文字に正規化した値。
type LoginFailure struct {
	Email       string    `gorm:"column:email"`
	AttemptedAt time.Time `gorm:"column:attempted_at"`
}

// TableName は GORM に表名を伝える。
func (LoginFailure) TableName() string { return "login_failures" }

// BracketSlot は bracket_slots の行。格子の (RowIndex, ColIndex) のセルに、MatchID の試合の
// Role(left / right / winner / loser)のチームを出す。
type BracketSlot struct {
	GroupID  int64                  `gorm:"column:group_id;primaryKey"`
	RowIndex int                    `gorm:"column:row_index;primaryKey"`
	ColIndex int                    `gorm:"column:col_index;primaryKey"`
	MatchID  int64                  `gorm:"column:match_id"`
	Role     domain.BracketSlotRole `gorm:"column:role"`
}

// TableName は GORM に表名を伝える。
func (BracketSlot) TableName() string { return "bracket_slots" }

// AuditLog は audit_log の行(00004)。Before と After は変更前後の値の JSON。ClientID は端末が付けた
// 操作の ID で、同じ ID の操作を 2 回適用しないために引く。
type AuditLog struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	TournamentID int64     `gorm:"column:tournament_id"`
	MatchID      *int64    `gorm:"column:match_id"`
	TeamID       *int64    `gorm:"column:team_id"`
	UserID       *int64    `gorm:"column:user_id"`
	Action       string    `gorm:"column:action"`
	Before       []byte    `gorm:"column:before;type:jsonb"`
	After        []byte    `gorm:"column:after;type:jsonb"`
	ClientID     *string   `gorm:"column:client_id"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

// TableName は GORM に表名を伝える。
func (AuditLog) TableName() string { return "audit_log" }

// StaffAssignment は staff_assignments の行(00004)。StartTime と EndTime は "HH:MM"(書き込み)
// または "HH:MM:SS"(読み取り)。
type StaffAssignment struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	TournamentID int64      `gorm:"column:tournament_id"`
	UserID       int64      `gorm:"column:user_id"`
	MatchID      *int64     `gorm:"column:match_id"`
	Field        *string    `gorm:"column:field"`
	MatchDate    *time.Time `gorm:"column:match_date"`
	StartTime    *string    `gorm:"column:start_time"`
	EndTime      *string    `gorm:"column:end_time"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

// TableName は GORM に表名を伝える。
func (StaffAssignment) TableName() string { return "staff_assignments" }
