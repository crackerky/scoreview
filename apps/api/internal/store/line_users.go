package store

import "time"

// 公式アカウントの友だち(migrations/00041)。本人用のリンクと申込は、この行の ID で LINE の人を指す。

// LineUser は line_users の行。LineUserID は公式アカウントから見たユーザー ID。
type LineUser struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	LineUserID   string     `gorm:"column:line_user_id"`
	Friend       bool       `gorm:"column:friend"`
	FollowedAt   *time.Time `gorm:"column:followed_at"`
	UnfollowedAt *time.Time `gorm:"column:unfollowed_at"`
	UserID       *int64     `gorm:"column:user_id"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (LineUser) TableName() string { return "line_users" }
