package store

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// 時間割の貼り付け(管理画面「試合を組む」の時間割)。運営が Excel で決めた日付、時刻、コートを、部門の
// 今ある試合にまとめて書く。どの行がどの試合かは web が試合番号かチームの組で当てはめて、試合の id で
// 渡す。書いた後の時間割の確かめ(コートの重なり、休みなど)は CheckFormatSchedule が行う。

const (
	sqlMatchesOfDivisionIn = "division_id = ? AND id IN ?"
	// maxFieldLength はコートの名前の長さ(「コート 12」「A」など。表の 1 つの枠に収まる長さ)
	maxFieldLength = 20
	// maxScheduleItems は 1 回に書ける試合の数(参照サイトの最大の部門の試合数を超える)
	maxScheduleItems = 1000
)

// ScheduleItem は試合 1 つの日付、時刻("HH:MM")、コート。
type ScheduleItem struct {
	MatchID int64
	Date    time.Time
	Time    string
	Field   string
}

// CheckScheduleItems は貼り付けた時間割の行を確かめる。同じ試合は 1 回だけ、時刻は HH:MM、コートは 1〜
// maxFieldLength 字。
func CheckScheduleItems(items []ScheduleItem) error {
	if err := domain.CheckRange("items", len(items), 1, maxScheduleItems); err != nil {
		return err
	}
	seen := map[int64]bool{}
	for i, item := range items {
		field := fmt.Sprintf("items[%d]", i)
		if seen[item.MatchID] {
			return domain.Invalid(field+".matchId", "is written twice")
		}
		seen[item.MatchID] = true
		if err := domain.CheckClock(field+".time", item.Time); err != nil {
			return err
		}
		court := strings.TrimSpace(item.Field)
		if court == "" || utf8.RuneCountInString(court) > maxFieldLength {
			return domain.Invalid(field+".field", fmt.Sprintf("must be 1-%d characters", maxFieldLength))
		}
	}
	return nil
}

// ImportSchedule は部門 divisionID の試合に、貼り付けた日付、時刻、コートを書く。部門の行をロックして、
// どの試合もこの部門の始まっていない試合であることを確かめてから、全部を 1 つのトランザクションで書く
// (一部だけ書かない)。始まった試合があれば、その試合番号を付けて domain.ErrConflict。
func ImportSchedule(ctx context.Context, db *gorm.DB, divisionID int64, items []ScheduleItem) error {
	if err := CheckScheduleItems(items); err != nil {
		return err
	}
	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.MatchID
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockDivision(ctx, tx, divisionID); err != nil {
			return err
		}
		var matches []Match
		if err := tx.Where(sqlMatchesOfDivisionIn, divisionID, ids).Find(&matches).Error; err != nil {
			return fmt.Errorf("load matches to schedule: %w", err)
		}
		if len(matches) != len(ids) {
			return domain.Invalid("items", "has a match that is not in this division")
		}
		var started []string
		for _, match := range matches {
			if match.Status != domain.MatchScheduled {
				started = append(started, match.Code)
			}
		}
		if len(started) > 0 {
			return fmt.Errorf("%w: 始まった試合の時刻は変えられません(%s)", domain.ErrConflict, strings.Join(started, "、"))
		}
		now := time.Now()
		for _, item := range items {
			err := tx.Model(&Match{}).Where(sqlByID, item.MatchID).Updates(map[string]any{
				"match_date": item.Date, "start_time": item.Time, "field": strings.TrimSpace(item.Field),
				"updated_at": now,
			}).Error
			if err != nil {
				return writeError("write match schedule", err)
			}
		}
		return nil
	})
}
