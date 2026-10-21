package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 当日の時間割の操作(遅延の一括シフト、試合の時刻とコートの入れ替え)と制約の再検査。

// maxShiftMinutes は一度に遅らせる(早める)分数の上限。1 日の大会で半日を超えるずらしは入力の誤り。
const maxShiftMinutes = 6 * 60

func (fs *formatState) scheduleConfig() (format.ScheduleConfig, error) {
	if fs.schedule == nil {
		return format.ScheduleConfig{}, fmt.Errorf("%w: 時間割の設定がありません。形を適用するときに時間割を作ってください",
			domain.ErrConflict)
	}
	return fs.schedule.config()
}

// assignments は保存済みの試合の時刻とコート。時刻が無い試合は含めない(検査で「時刻なし」になる)。
func (fs *formatState) assignments() []format.Assignment {
	var out []format.Assignment
	for code, m := range fs.matches {
		start, ok := matchStart(m)
		if !ok || m.Field == nil {
			continue
		}
		naive := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), 0, 0, time.UTC)
		out = append(out, format.Assignment{Code: code, Start: naive, Court: *m.Field})
	}
	return out
}

// CheckFormatSchedule は保存済みの時間割を再検査し、違反を返す。
func CheckFormatSchedule(ctx context.Context, db *gorm.DB, divisionID int64) ([]ScheduleViolation, error) {
	fs, ok, err := loadFormatState(ctx, db, divisionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: この部門には大会の形が設定されていません", domain.ErrConflict)
	}
	cfg, err := fs.scheduleConfig()
	if err != nil {
		return nil, err
	}
	// 終了時刻は当日の遅れで超えるのが普通なので、再検査では見ない(生成時だけ見る)
	cfg.End = time.Time{}
	return toViolations(format.CheckSchedule(fs.plan, cfg, fs.assignments())), nil
}

// ShiftFormatSchedule はまだ始まっていない試合を minutes 分遅らせる(負なら早める)。court が nil なら
// 全コート、あればそのコートだけ。起点はまだ始まっていない最初の試合の予定時刻で、試合の順序は
// 変えない(format.ShiftCourt / ShiftAll)。戻り値は動かした後の違反。
func ShiftFormatSchedule(
	ctx context.Context,
	db *gorm.DB,
	divisionID int64,
	court *string,
	minutes int,
) ([]ScheduleViolation, error) {
	if minutes == 0 || minutes > maxShiftMinutes || minutes < -maxShiftMinutes {
		return nil, domain.Invalid("minutes", fmt.Sprintf("must be between -%d and %d, not 0",
			maxShiftMinutes, maxShiftMinutes))
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockDivision(ctx, tx, divisionID); err != nil {
			return err
		}
		fs, ok, err := loadFormatState(ctx, tx, divisionID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: この部門には大会の形が設定されていません", domain.ErrConflict)
		}
		current := fs.assignments()
		var from *time.Time
		for _, a := range current {
			m := fs.matches[a.Code]
			if m.Status != domain.MatchScheduled || (court != nil && a.Court != *court) {
				continue
			}
			if from == nil || a.Start.Before(*from) {
				start := a.Start
				from = &start
			}
		}
		if from == nil {
			return fmt.Errorf("%w: 遅らせる試合がありません", domain.ErrConflict)
		}
		var shifted []format.Assignment
		if court != nil {
			shifted = format.ShiftCourt(current, *court, *from, minutes)
		} else {
			shifted = format.ShiftAll(current, *from, minutes)
		}
		for i, a := range shifted {
			m := fs.matches[a.Code]
			// 始まった試合の予定時刻は記録として残す(起点より後に始まった試合中の試合など)
			if a.Start.Equal(current[i].Start) || m.Status != domain.MatchScheduled {
				continue
			}
			if err := writeMatchTime(ctx, tx, m.ID, a.Start, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return CheckFormatSchedule(ctx, db, divisionID)
}

func writeMatchTime(ctx context.Context, tx *gorm.DB, matchID int64, start time.Time, field *string) error {
	day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	updates := map[string]any{"match_date": day, "start_time": start.Format("15:04"), "updated_at": time.Now()}
	if field != nil {
		updates["field"] = *field
	}
	if err := tx.WithContext(ctx).Model(&Match{}).Where(sqlByID, matchID).Updates(updates).Error; err != nil {
		return writeError("move match", err)
	}
	return nil
}

// SwapMatchSchedule は 2 つの未試合の試合の日付、時刻、コートを入れ替え、入れ替えた後の違反を返す。
// 2 つは同じディビジョンの試合であること。
func SwapMatchSchedule(ctx context.Context, db *gorm.DB, matchID, otherID int64) ([]ScheduleViolation, error) {
	if matchID == otherID {
		return nil, domain.Invalid("otherMatchId", "must be another match")
	}
	var divisionID int64
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a, err := lockMatchInDivision(ctx, tx, matchID)
		if err != nil {
			return err
		}
		other, err := findByID[Match](ctx, tx, otherID, domain.ResourceMatch)
		if err != nil {
			return err
		}
		if a.DivisionID != other.DivisionID {
			return domain.Invalid("otherMatchId", "must be a match of the same division")
		}
		// 同じディビジョンなのでロックは取得済み。試合の行は id の順でなくてよい(ディビジョンの
		// ロックで同じディビジョンの操作は直列になっている)
		b, err := lockMatch(ctx, tx, otherID)
		if err != nil {
			return err
		}
		if a.Status != domain.MatchScheduled || b.Status != domain.MatchScheduled {
			return fmt.Errorf("%w: 始まった試合の時刻は入れ替えられません", domain.ErrConflict)
		}
		divisionID = a.DivisionID
		for _, pair := range [][2]Match{{a, b}, {b, a}} {
			target, source := pair[0], pair[1]
			err := tx.WithContext(ctx).Model(&Match{}).Where(sqlByID, target.ID).Updates(map[string]any{
				"match_date": source.MatchDate, "start_time": source.StartTime, "field": source.Field,
				"updated_at": time.Now(),
			}).Error
			if err != nil {
				return writeError("swap match times", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return CheckFormatSchedule(ctx, db, divisionID)
}
