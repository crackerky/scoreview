package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/talk"
)

// 公式アカウントの友だちと、届いたものの記録(migrations/00041)。決まりは talk にある。

const (
	sqlByLineUser     = "line_user_id = ?"
	sqlByWebhookEvent = "webhook_event_id = ?"
	// 使い方の案内をまだ返していないか、前に返してから間があいた人
	sqlHelpDue = "line_user_id = ? AND (helped_at IS NULL OR helped_at <= ?)"
)

// LineUser は line_users の行(参加者の台帳)。LineUserID は公式アカウントから見たユーザー ID。Name、Phone、
// Email は前回の入力(migrations/00042)で、ProfileUpdatedAt が nil ならまだ何も入れていない。
type LineUser struct {
	ID               int64      `gorm:"column:id;primaryKey"`
	LineUserID       string     `gorm:"column:line_user_id"`
	Friend           bool       `gorm:"column:friend"`
	FollowedAt       *time.Time `gorm:"column:followed_at"`
	UnfollowedAt     *time.Time `gorm:"column:unfollowed_at"`
	UserID           *int64     `gorm:"column:user_id"`
	Name             string     `gorm:"column:name"`
	Phone            string     `gorm:"column:phone"`
	Email            string     `gorm:"column:email"`
	ProfileUpdatedAt *time.Time `gorm:"column:profile_updated_at"`
	// Gender と Birthdate は前に答えた性別と生年月日(migrations/00054)
	Gender    *string    `gorm:"column:gender"`
	Birthdate *time.Time `gorm:"column:birthdate"`
	// HelpedAt は使い方の案内を最後に返した時刻(migrations/00047)
	HelpedAt  *time.Time `gorm:"column:helped_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (LineUser) TableName() string { return "line_users" }

// LineWebhookEvent は line_webhook_events の行。
type LineWebhookEvent struct {
	WebhookEventID string    `gorm:"column:webhook_event_id;primaryKey"`
	ReceivedAt     time.Time `gorm:"column:received_at"`
}

// TableName は GORM に表名を伝える。
func (LineWebhookEvent) TableName() string { return "line_webhook_events" }

// LineStore は talk.Store を GORM で実装する。
type LineStore struct {
	db *gorm.DB
}

var _ talk.Store = (*LineStore)(nil)

// NewLineStore は db で読み書きする LineStore を返す。
func NewLineStore(db *gorm.DB) *LineStore {
	return &LineStore{db: db}
}

// FirstSeen は talk.Store.FirstSeen。同じ ID の行があれば何もしない(一意の主キーで、同時に届いても
// 1 回だけ真になる)。
func (s *LineStore) FirstSeen(ctx context.Context, eventID string, at time.Time) (bool, error) {
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&LineWebhookEvent{WebhookEventID: eventID, ReceivedAt: at})
	if result.Error != nil {
		return false, fmt.Errorf("record webhook event %s: %w", eventID, result.Error)
	}
	return result.RowsAffected == 1, nil
}

// Forget は talk.Store.Forget。
func (s *LineStore) Forget(ctx context.Context, eventID string) error {
	err := s.db.WithContext(ctx).Where(sqlByWebhookEvent, eventID).Delete(&LineWebhookEvent{}).Error
	if err != nil {
		return fmt.Errorf("forget webhook event %s: %w", eventID, err)
	}
	return nil
}

// Follow は talk.Store.Follow。
func (s *LineStore) Follow(ctx context.Context, userID string, at time.Time) error {
	return s.upsert(ctx, LineUser{LineUserID: userID, Friend: true, FollowedAt: &at},
		map[string]any{"friend": true, "followed_at": at, "updated_at": at})
}

// Unfollow は talk.Store.Unfollow。
func (s *LineStore) Unfollow(ctx context.Context, userID string, at time.Time) error {
	return s.upsert(ctx, LineUser{LineUserID: userID, Friend: false, UnfollowedAt: &at},
		map[string]any{"friend": false, "unfollowed_at": at, "updated_at": at})
}

func (s *LineStore) upsert(ctx context.Context, row LineUser, update map[string]any) error {
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "line_user_id"}},
		DoUpdates: clause.Assignments(update),
	}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("record line user: %w", err)
	}
	return nil
}

// HelpDue は talk.Store.HelpDue。案内を返す時刻を書く更新を 1 回だけ行い、書けたら真にする(同じ人から
// 同時に届いても、案内は 1 回だけになる)。
func (s *LineStore) HelpDue(ctx context.Context, userID string, at time.Time, every time.Duration) (bool, error) {
	if _, err := s.LineUser(ctx, userID, at); err != nil {
		return false, err
	}
	result := s.db.WithContext(ctx).Model(&LineUser{}).Where(sqlHelpDue, userID, at.Add(-every)).
		Updates(map[string]any{"helped_at": at})
	if result.Error != nil {
		return false, fmt.Errorf("record help of line user: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

// Recipient は talk.Store.Recipient。
func (s *LineStore) Recipient(ctx context.Context, id int64) (talk.Recipient, error) {
	var rows []LineUser
	if err := s.db.WithContext(ctx).Where(sqlByID, id).Limit(1).Find(&rows).Error; err != nil {
		return talk.Recipient{}, fmt.Errorf("find line user %d: %w", id, err)
	}
	if len(rows) == 0 {
		return talk.Recipient{}, domain.NotFound(domain.ResourceLineUser, id)
	}
	return talk.Recipient{UserID: rows[0].LineUserID, Friend: rows[0].Friend}, nil
}

// LineUser は talk.Store.LineUser。友だちの記録が無い人(Webhook をこちらで受ける前からの友だち)は、
// メッセージを送ってきたので友だちとして作る。
func (s *LineStore) LineUser(ctx context.Context, userID string, at time.Time) (int64, error) {
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&LineUser{LineUserID: userID, Friend: true, CreatedAt: at, UpdatedAt: at}).Error
	if err != nil {
		return 0, fmt.Errorf("record line user: %w", err)
	}
	var row LineUser
	if err := s.db.WithContext(ctx).Where(sqlByLineUser, userID).Take(&row).Error; err != nil {
		return 0, fmt.Errorf("find line user: %w", err)
	}
	return row.ID, nil
}
