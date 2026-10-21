package store

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 保存済みの形の指定を読む(ウィザードの初期値と「前回の大会の形をコピー」)。

const sqlOrderFormatsRecent = "updated_at DESC, division_id DESC"

// maxFormatCopies はコピー元として並べる形の数(直近のものから)。
const maxFormatCopies = 30

// SavedFormat は保存済みの形の指定。Seeds はシード順のチーム(枠に入っている分だけ)。
type SavedFormat struct {
	DivisionID     int64
	TournamentID   int64
	TournamentName string
	DivisionName   string
	Title          string
	Version        int
	Template       string
	Params         any
	Tiebreak       format.TiebreakPreset
	Forfeit        format.ForfeitScore
	Schedule       *ScheduleSetting
	Seeds          []int64
}

func savedFormatOf(ctx context.Context, db *gorm.DB, row DivisionFormat) (SavedFormat, error) {
	spec, err := specOf(row)
	if err != nil {
		return SavedFormat{}, err
	}
	plan, err := format.NewEngine().Generate(spec)
	if err != nil {
		return SavedFormat{}, fmt.Errorf("regenerate format of division %d: %w", row.DivisionID, err)
	}
	division, err := findInternal[Division](ctx, db, row.DivisionID, "division")
	if err != nil {
		return SavedFormat{}, err
	}
	tournament, err := findInternal[Tournament](ctx, db, division.TournamentID, "tournament")
	if err != nil {
		return SavedFormat{}, err
	}
	out := SavedFormat{DivisionID: row.DivisionID, TournamentID: tournament.ID, TournamentName: tournament.NameJa,
		DivisionName: division.NameJa, Title: format.Title(plan), Version: row.Version, Template: row.Template,
		Params: spec.Params, Tiebreak: format.TiebreakPreset(row.Tiebreak),
		Forfeit: format.ForfeitScore{Winner: row.ForfeitWinnerScore, Loser: row.ForfeitLoserScore}}
	if row.Schedule != nil {
		var setting ScheduleSetting
		if err := json.Unmarshal(row.Schedule, &setting); err != nil {
			return SavedFormat{}, fmt.Errorf("division %d schedule: %w", row.DivisionID, err)
		}
		out.Schedule = &setting
	}
	return out, nil
}

// GetSavedFormat はディビジョンの形の指定とシードを返す。形が無ければ ok=false。
func GetSavedFormat(ctx context.Context, db *gorm.DB, divisionID int64) (SavedFormat, bool, error) {
	if _, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision); err != nil {
		return SavedFormat{}, false, err
	}
	fs, ok, err := loadFormatState(ctx, db, divisionID)
	if err != nil || !ok {
		return SavedFormat{}, ok, err
	}
	out, err := savedFormatOf(ctx, db, fs.row)
	if err != nil {
		return SavedFormat{}, false, err
	}
	for _, team := range fs.state.Seeds {
		if team != 0 {
			out.Seeds = append(out.Seeds, team)
		}
	}
	return out, true, nil
}

// ListSavedFormats は保存済みの形の指定を新しい順に返す(シードは含めない)。exclude の
// ディビジョンは除く。
func ListSavedFormats(ctx context.Context, db *gorm.DB, exclude int64) ([]SavedFormat, error) {
	var rows []DivisionFormat
	if err := db.WithContext(ctx).Order(sqlOrderFormatsRecent).Limit(maxFormatCopies + 1).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list formats: %w", err)
	}
	rows = slices.DeleteFunc(rows, func(r DivisionFormat) bool { return r.DivisionID == exclude })
	if len(rows) > maxFormatCopies {
		rows = rows[:maxFormatCopies]
	}
	out := make([]SavedFormat, 0, len(rows))
	for _, row := range rows {
		saved, err := savedFormatOf(ctx, db, row)
		if err != nil {
			return nil, err
		}
		out = append(out, saved)
	}
	return out, nil
}
