package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// 大会の準備の操作: 大会のコピー(運営者レビュー H9)と、試合の無い部門のチームの一括削除(H4)。

// divisionCopyRow は divisions の行のうち、コピー元(00030)だけ。Division のモデルに列を足さずに読み書きする。
type divisionCopyRow struct {
	ID           int64  `gorm:"column:id;primaryKey"`
	CopiedFromID *int64 `gorm:"column:copied_from_id"`
}

// TableName は GORM に表名を伝える。
func (divisionCopyRow) TableName() string { return "divisions" }

// CopyTournament は大会 sourceID を fields(名前、開催日など。TopHTML は無害化済み)の新しい大会として
// コピーする。引き継ぐのはバナーと部門(コピー元を覚え、形はその部門で決めるときの初期値になる)、
// withTeams ならチームと選手(最終順位は消す)。試合、組、結果、ギャラリー、記録係の割り当ては
// 引き継がない。
func CopyTournament(
	ctx context.Context,
	db *gorm.DB,
	sourceID int64,
	fields TournamentFields,
	withTeams bool,
) (TournamentRecord, error) {
	if err := fields.validate(); err != nil {
		return TournamentRecord{}, err
	}
	var row Tournament
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findByID[Tournament](ctx, tx, sourceID, domain.ResourceTournament); err != nil {
			return err
		}
		if err := fields.apply(&row); err != nil {
			return err
		}
		if err := tx.Create(&row).Error; err != nil {
			return writeError("create tournament copy", err)
		}
		if err := copyBanners(ctx, tx, sourceID, row.ID); err != nil {
			return err
		}
		return copyDivisions(ctx, tx, sourceID, row.ID, withTeams)
	})
	if err != nil {
		return TournamentRecord{}, err
	}
	return toTournamentRecord(row), nil
}

func copyBanners(ctx context.Context, tx *gorm.DB, sourceID, targetID int64) error {
	var banners []TournamentBanner
	if err := tx.WithContext(ctx).Where(sqlByTournament, sourceID).Order(sqlOrderSortID).
		Find(&banners).Error; err != nil {
		return fmt.Errorf("list banners of tournament %d: %w", sourceID, err)
	}
	for _, banner := range banners {
		banner.ID, banner.TournamentID = 0, targetID
		if err := tx.WithContext(ctx).Create(&banner).Error; err != nil {
			return writeError("copy banner", err)
		}
	}
	return nil
}

func copyDivisions(ctx context.Context, tx *gorm.DB, sourceID, targetID int64, withTeams bool) error {
	db := tx.WithContext(ctx)
	var divisions []Division
	if err := db.Where(sqlByTournament, sourceID).Order(sqlOrderSortID).Find(&divisions).Error; err != nil {
		return fmt.Errorf("list divisions of tournament %d: %w", sourceID, err)
	}
	now := time.Now()
	for _, division := range divisions {
		from := division.ID
		division.ID, division.TournamentID = 0, targetID
		// 新しい大会はまだ結果が無いので、最終順位の公開は外す
		division.StandingsPublished = false
		division.CreatedAt, division.UpdatedAt = now, now
		if err := db.Create(&division).Error; err != nil {
			return writeError("copy division", err)
		}
		if err := db.Model(&divisionCopyRow{}).Where(sqlByID, division.ID).
			Update("copied_from_id", from).Error; err != nil {
			return writeError("remember copied division", err)
		}
		if withTeams {
			if err := copyTeams(ctx, tx, from, division); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyTeams(ctx context.Context, tx *gorm.DB, sourceDivision int64, target Division) error {
	db := tx.WithContext(ctx)
	var teams []Team
	if err := db.Where(sqlByDivision, sourceDivision).Order(sqlOrderSortID).Find(&teams).Error; err != nil {
		return fmt.Errorf("list teams of division %d: %w", sourceDivision, err)
	}
	now := time.Now()
	for _, team := range teams {
		from := team.ID
		team.ID, team.TournamentID, team.DivisionID = 0, target.TournamentID, target.ID
		team.FinalRank = nil
		team.CreatedAt, team.UpdatedAt = now, now
		if err := db.Create(&team).Error; err != nil {
			return writeError("copy team", err)
		}
		var players []Player
		if err := db.Where(sqlByTeam, from).Order(sqlOrderID).Find(&players).Error; err != nil {
			return fmt.Errorf("list players of team %d: %w", from, err)
		}
		for _, player := range players {
			player.ID, player.TeamID = 0, team.ID
			player.CreatedAt, player.UpdatedAt = now, now
			if err := db.Create(&player).Error; err != nil {
				return writeError("copy player", err)
			}
		}
	}
	return nil
}

// GetInheritedFormat は、大会のコピーで作った部門(まだ形が無いもの)のコピー元の形を返す。シードは
// コピー元のシード順のチームを、この部門の同じ正式名のチームに置き換えたもの(見つからないチームは
// 除く)。コピーで作った部門でない、形が既にある、コピー元に形が無いなら ok=false。
func GetInheritedFormat(ctx context.Context, db *gorm.DB, divisionID int64) (SavedFormat, bool, error) {
	link, err := findByID[divisionCopyRow](ctx, db, divisionID, domain.ResourceDivision)
	if err != nil {
		return SavedFormat{}, false, err
	}
	if link.CopiedFromID == nil {
		return SavedFormat{}, false, nil
	}
	if _, own, err := loadFormatState(ctx, db, divisionID); err != nil || own {
		return SavedFormat{}, false, err
	}
	saved, ok, err := GetSavedFormat(ctx, db, *link.CopiedFromID)
	if err != nil || !ok {
		return SavedFormat{}, false, err
	}
	var source, mine []Team
	if err := db.WithContext(ctx).Where(sqlByIDs, saved.Seeds).Find(&source).Error; err != nil {
		return SavedFormat{}, false, fmt.Errorf("load seeds of division %d: %w", *link.CopiedFromID, err)
	}
	if err := db.WithContext(ctx).Where(sqlByDivision, divisionID).Find(&mine).Error; err != nil {
		return SavedFormat{}, false, fmt.Errorf("load teams of division %d: %w", divisionID, err)
	}
	nameOf := map[int64]string{}
	for _, team := range source {
		nameOf[team.ID] = team.NameFullJa
	}
	byName := map[string]int64{}
	for _, team := range mine {
		byName[team.NameFullJa] = team.ID
	}
	var seeds []int64
	for _, old := range saved.Seeds {
		if id, ok := byName[nameOf[old]]; ok {
			seeds = append(seeds, id)
		}
	}
	saved.Seeds = seeds
	return saved, true, nil
}

// DeleteDivisionTeams は部門のチーム(と選手)をまとめて消し、消した数を返す。試合がある部門は
// ErrConflict(試合の対戦チームが消えるのを防ぐ。名簿の貼り間違いを直すのは試合を作る前)。
func DeleteDivisionTeams(ctx context.Context, db *gorm.DB, divisionID int64) (int, error) {
	var deleted int64
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findByID[Division](ctx, tx, divisionID, domain.ResourceDivision); err != nil {
			return err
		}
		if err := lockDivision(ctx, tx, divisionID); err != nil {
			return err
		}
		var matches int64
		if err := tx.Model(&Match{}).Where(sqlByDivision, divisionID).Count(&matches).Error; err != nil {
			return fmt.Errorf("count matches of division %d: %w", divisionID, err)
		}
		if matches > 0 {
			return fmt.Errorf("%w: この部門には試合が %d 件あるので、チームをまとめて削除できません", domain.ErrConflict,
				matches)
		}
		result := tx.Where(sqlByDivision, divisionID).Delete(&Team{})
		if result.Error != nil {
			return writeError("delete teams of division", result.Error)
		}
		deleted = result.RowsAffected
		return nil
	})
	return int(deleted), err
}
