package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
)

// 雷・雨・暑さによる大会の中断と再開(レビュー H2。docs/format-engine.md「日程とコートの自動割り当て」
// の中断)。中断中は閲覧ページの大会の上部に帯を出し、再開すると中断した分だけ未開始の試合を一括で
// ずらす。中断は会場(大会)単位で、全部門の試合に及ぶ。

// 監査ログの操作の種類。
const (
	AuditTournamentSuspend = "tournament.suspend"
	AuditTournamentResume  = "tournament.resume"
)

// shiftStepMinutes は再開でずらす分数の刻み。時刻を 5 分刻みで組む運用に合わせて切り上げる
// (13:07 再開を 13:10 にそろえる)。
const shiftStepMinutes = 5

const sqlScheduledTimedMatches = "division_id IN ? AND status = ? AND match_date IS NOT NULL" +
	" AND start_time IS NOT NULL"

// TournamentSuspension は tournament_suspensions の行。
type TournamentSuspension struct {
	TournamentID int64     `gorm:"column:tournament_id;primaryKey"`
	Reason       string    `gorm:"column:reason"`
	Note         string    `gorm:"column:note"`
	ResumeAt     *string   `gorm:"column:resume_at"`
	SuspendedAt  time.Time `gorm:"column:suspended_at"`
	SuspendedBy  *int64    `gorm:"column:suspended_by"`
}

// TableName は GORM に表名を伝える。
func (TournamentSuspension) TableName() string { return "tournament_suspensions" }

// Suspension は大会の中断。Reason は thunder / rain / heat / other、Note は「その他」の説明、
// ResumeAt は再開予定 "HH:MM"(未定なら nil)、Since は中断した時刻。
type Suspension struct {
	Reason   string
	Note     string
	ResumeAt *string
	Since    time.Time
}

// SuspensionInput は中断の入力。
type SuspensionInput struct {
	Reason   string
	Note     string
	ResumeAt *string
}

func (in SuspensionInput) validate() error {
	// 値は tournament_suspensions.reason の CHECK と同じ
	switch in.Reason {
	case "thunder", "rain", "heat", "other":
	default:
		return domain.Invalid("reason", "must be thunder, rain, heat or other")
	}
	if in.ResumeAt != nil {
		if err := domain.CheckClock("resumeAt", *in.ResumeAt); err != nil {
			return err
		}
	}
	if len([]rune(in.Note)) > 100 {
		return domain.Invalid("note", "must be 100 characters or less")
	}
	return nil
}

// GetSuspension は大会の中断を返す。中断していなければ nil。
func GetSuspension(ctx context.Context, db *gorm.DB, tournamentID int64) (*Suspension, error) {
	var rows []TournamentSuspension
	if err := db.WithContext(ctx).Where(sqlByTournament, tournamentID).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load suspension of tournament %d: %w", tournamentID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	row := rows[0]
	return &Suspension{Reason: row.Reason, Note: row.Note, ResumeAt: row.ResumeAt, Since: row.SuspendedAt}, nil
}

// SuspendTournament は大会を中断する。既に中断中なら理由と再開予定だけを書き直す(中断した時刻は
// 変えない。再開でずらす分数は最初の中断からの時間)。
func SuspendTournament(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	tournamentID int64,
	input SuspensionInput,
	now time.Time,
) (Suspension, error) {
	input.Note = strings.TrimSpace(input.Note)
	if err := input.validate(); err != nil {
		return Suspension{}, err
	}
	if _, err := findByID[Tournament](ctx, db, tournamentID, domain.ResourceTournament); err != nil {
		return Suspension{}, err
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		by := actor.UserID
		row := TournamentSuspension{TournamentID: tournamentID, Reason: input.Reason, Note: input.Note,
			ResumeAt: input.ResumeAt, SuspendedAt: now, SuspendedBy: &by}
		err := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tournament_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"reason", "note", "resume_at"}),
		}).Create(&row).Error
		if err != nil {
			return writeError("save suspension", err)
		}
		return auditFormatOp(ctx, tx, tournamentID, actor, AuditTournamentSuspend, nil,
			AuditFormat{Reason: input.Reason})
	})
	if err != nil {
		return Suspension{}, err
	}
	suspension, err := GetSuspension(ctx, db, tournamentID)
	if err != nil {
		return Suspension{}, err
	}
	return *suspension, nil
}

// OverEnd は再開でずらした結果、部門の時間割の終了時刻を過ぎる試合。
type OverEnd struct {
	MatchID     int64
	Code        string
	Division    string
	Time        string
	OverMinutes int
}

// ResumeResult は再開の結果。Minutes はずらした分数、Shifted はずらした試合の数。
type ResumeResult struct {
	Minutes int
	Shifted int
	OverEnd []OverEnd
}

// ResumeTournament は中断を終え、大会の全部門のまだ始まっていない試合を、中断した分(minutes が
// nil なら中断からの経過を 5 分単位に切り上げた分)だけ遅らせる。試合の順序とコートは変えない。
// ずらした結果、部門の時間割の終了時刻を過ぎる試合を返す(本部が打ち切りや短縮を決める)。
func ResumeTournament(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	tournamentID int64,
	minutes *int,
	now time.Time,
) (ResumeResult, error) {
	if minutes != nil && (*minutes < 0 || *minutes > maxShiftMinutes) {
		return ResumeResult{}, domain.Invalid("minutes", fmt.Sprintf("must be between 0 and %d", maxShiftMinutes))
	}
	var out ResumeResult
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockTournament(ctx, tx, tournamentID); err != nil {
			return err
		}
		suspension, err := GetSuspension(ctx, tx, tournamentID)
		if err != nil {
			return err
		}
		if suspension == nil {
			return fmt.Errorf("%w: この大会は中断していません", domain.ErrConflict)
		}
		out.Minutes = elapsedStep(suspension.Since, now)
		if minutes != nil {
			out.Minutes = *minutes
		}
		if out.Shifted, out.OverEnd, err = shiftTournament(ctx, tx, tournamentID, out.Minutes); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where(sqlByTournament, tournamentID).
			Delete(&TournamentSuspension{}).Error; err != nil {
			return fmt.Errorf("remove suspension of tournament %d: %w", tournamentID, err)
		}
		return auditFormatOp(ctx, tx, tournamentID, actor, AuditTournamentResume, nil,
			AuditFormat{Reason: suspension.Reason, Minutes: out.Minutes})
	})
	return out, err
}

// elapsedStep は since から now までの分数を shiftStepMinutes 単位に切り上げる(負なら 0)。
func elapsedStep(since, now time.Time) int {
	elapsed := now.Sub(since).Minutes()
	if elapsed <= 0 {
		return 0
	}
	return int(math.Ceil(elapsed/shiftStepMinutes)) * shiftStepMinutes
}

// shiftTournament は大会の全部門の未試合の試合を minutes 分遅らせ、ずらした数と終了時刻を過ぎる
// 試合を返す。部門の行を id の順にロックする(locks.go の順序: 大会 → ディビジョン)。
func shiftTournament(
	ctx context.Context,
	tx *gorm.DB,
	tournamentID int64,
	minutes int,
) (int, []OverEnd, error) {
	var divisions []Division
	if err := tx.WithContext(ctx).Where(sqlByTournament, tournamentID).Order(sqlOrderID).
		Find(&divisions).Error; err != nil {
		return 0, nil, fmt.Errorf("list divisions of tournament %d: %w", tournamentID, err)
	}
	if len(divisions) == 0 {
		return 0, nil, nil
	}
	ids := make([]int64, len(divisions))
	names := map[int64]string{}
	for i, d := range divisions {
		if err := lockDivision(ctx, tx, d.ID); err != nil {
			return 0, nil, err
		}
		ids[i] = d.ID
		names[d.ID] = d.NameJa
	}
	var matches []Match
	if err := tx.WithContext(ctx).Where(sqlScheduledTimedMatches, ids, domain.MatchScheduled).
		Order(sqlOrderID).Find(&matches).Error; err != nil {
		return 0, nil, fmt.Errorf("list scheduled matches of tournament %d: %w", tournamentID, err)
	}
	ends, err := scheduleEnds(ctx, tx, ids)
	if err != nil {
		return 0, nil, err
	}
	shifted := 0
	var over []OverEnd
	for _, m := range matches {
		start, ok := matchStart(m)
		if !ok {
			continue
		}
		naive := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), 0, 0, time.UTC)
		moved := naive.Add(time.Duration(minutes) * time.Minute)
		if minutes > 0 {
			if err := writeMatchTime(ctx, tx, m.ID, moved, nil); err != nil {
				return 0, nil, err
			}
			shifted++
		}
		end, ok := ends[m.DivisionID]
		if !ok {
			continue
		}
		finish := moved.Add(time.Duration(end.slotMinutes) * time.Minute)
		limit := time.Date(moved.Year(), moved.Month(), moved.Day(), end.end.Hour(), end.end.Minute(), 0, 0,
			time.UTC)
		if late := int(finish.Sub(limit).Minutes()); late > 0 {
			over = append(over, OverEnd{MatchID: m.ID, Code: m.Code, Division: names[m.DivisionID],
				Time: moved.Format("15:04"), OverMinutes: late})
		}
	}
	return shifted, over, nil
}

type scheduleEnd struct {
	end         time.Time
	slotMinutes int
}

// scheduleEnds は部門の時間割の終了時刻と 1 試合の枠の長さ(終了時刻の設定がある部門だけ)。
func scheduleEnds(ctx context.Context, tx *gorm.DB, divisionIDs []int64) (map[int64]scheduleEnd, error) {
	out := map[int64]scheduleEnd{}
	for _, id := range divisionIDs {
		row, ok, err := findFormat(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if !ok || row.Schedule == nil {
			continue
		}
		var setting ScheduleSetting
		if err := json.Unmarshal(row.Schedule, &setting); err != nil {
			return nil, fmt.Errorf("division %d schedule: %w", id, err)
		}
		if setting.End == nil {
			continue
		}
		end, err := time.Parse("15:04", *setting.End)
		if err != nil {
			return nil, fmt.Errorf("division %d schedule end: %w", id, err)
		}
		out[id] = scheduleEnd{end: end, slotMinutes: setting.SlotMinutes}
	}
	return out, nil
}
