package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// 英語名の補い(domain/english.go)。管理画面と管理 API で作るチーム、選手、組は作るときに補う。
// ここは補う前に作った部門のための一括の操作。取込ツールの書き込み(importwrite.go)はこの経路を通らない。

const (
	sqlTeamsWithoutEnglish   = "division_id = ? AND (name_full_en = '' OR name_short_en = '')"
	sqlGroupsWithoutEnglish  = "division_id = ? AND name_en = ''"
	sqlPlayersWithoutEnglish = "team_id IN (?) AND name_en = ''"
	sqlColumnID              = "id"
)

// fillTeamEnglish はチームの行の英語の正式名と短縮名を補う(domain.TeamNamesWithEnglish)。
func fillTeamEnglish(row *Team) {
	full, short := domain.TeamNamesWithEnglish(
		i18n(row.NameFullJa, row.NameFullEn), i18n(row.NameShortJa, row.NameShortEn))
	row.NameFullEn, row.NameShortEn = full.En, short.En
}

// FilledEnglishNames は FillDivisionEnglishNames で英語名を入れた数。
type FilledEnglishNames struct {
	Teams   int
	Players int
	Groups  int
}

// FillDivisionEnglishNames は部門のチーム(正式名と短縮名)、選手、組のうち英語名が空のものに日本語名を
// 入れ、入れた数を返す。英語名が既にあるものは変えない。部門が無ければ NotFound。
func FillDivisionEnglishNames(
	ctx context.Context,
	db *gorm.DB,
	divisionID int64,
) (FilledEnglishNames, error) {
	var filled FilledEnglishNames
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findByID[Division](ctx, tx, divisionID, domain.ResourceDivision); err != nil {
			return err
		}
		var err error
		if filled.Teams, err = fillTeamsEnglish(tx, divisionID); err != nil {
			return err
		}
		if filled.Players, err = fillPlayersEnglish(tx, divisionID); err != nil {
			return err
		}
		filled.Groups, err = fillGroupsEnglish(tx, divisionID)
		return err
	})
	if err != nil {
		return FilledEnglishNames{}, err
	}
	return filled, nil
}

func fillTeamsEnglish(tx *gorm.DB, divisionID int64) (int, error) {
	var teams []Team
	if err := tx.Where(sqlTeamsWithoutEnglish, divisionID).Order(sqlOrderID).
		Find(&teams).Error; err != nil {
		return 0, fmt.Errorf("list teams without english names of division %d: %w", divisionID, err)
	}
	for _, team := range teams {
		fillTeamEnglish(&team)
		if err := tx.Model(&Team{}).Where(sqlByID, team.ID).Updates(map[string]any{
			"name_full_en": team.NameFullEn, "name_short_en": team.NameShortEn,
		}).Error; err != nil {
			return 0, writeError("fill english team names", err)
		}
	}
	return len(teams), nil
}

func fillPlayersEnglish(tx *gorm.DB, divisionID int64) (int, error) {
	teamIDs := tx.Model(&Team{}).Select(sqlColumnID).Where(sqlByDivision, divisionID)
	var players []Player
	if err := tx.Where(sqlPlayersWithoutEnglish, teamIDs).Order(sqlOrderID).
		Find(&players).Error; err != nil {
		return 0, fmt.Errorf("list players without english names of division %d: %w", divisionID, err)
	}
	for _, player := range players {
		if err := tx.Model(&Player{}).Where(sqlByID, player.ID).
			Update("name_en", player.NameJa).Error; err != nil {
			return 0, writeError("fill english player name", err)
		}
	}
	return len(players), nil
}

func fillGroupsEnglish(tx *gorm.DB, divisionID int64) (int, error) {
	var groups []Group
	if err := tx.Where(sqlGroupsWithoutEnglish, divisionID).Order(sqlOrderID).
		Find(&groups).Error; err != nil {
		return 0, fmt.Errorf("list groups without english names of division %d: %w", divisionID, err)
	}
	for _, group := range groups {
		if err := tx.Model(&Group{}).Where(sqlByID, group.ID).
			Update("name_en", group.Name).Error; err != nil {
			return 0, writeError("fill english group name", err)
		}
	}
	return len(groups), nil
}
