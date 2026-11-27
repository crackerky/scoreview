package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
)

// 部門の組分けの下書き(migrations/00052)。管理画面「試合を組む」で、運営がチームを組に入れた途中や、
// Excel の表を貼って決めた組を、形を作る前に残す。形を作るときは、この組をそのまま予選の組にする
// (format.PoolBandsParams.PoolSeeds)。

const (
	// maxDraftPools は下書きの組の数の上限(試合を組む画面で 1 部門に作る組。参照サイトの最大を超える)
	maxDraftPools = 16
)

// PoolDraft は pool_drafts の行。
type PoolDraft struct {
	DivisionID int64     `gorm:"column:division_id;primaryKey"`
	Pools      []byte    `gorm:"column:pools;type:jsonb"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (PoolDraft) TableName() string { return "pool_drafts" }

// GetPoolDraft は部門の組分けの下書きを読む。無ければ found は偽。
func GetPoolDraft(ctx context.Context, db *gorm.DB, divisionID int64) ([][]int64, time.Time, bool, error) {
	var rows []PoolDraft
	if err := db.WithContext(ctx).Where(sqlByDivision, divisionID).Limit(1).Find(&rows).Error; err != nil {
		return nil, time.Time{}, false, fmt.Errorf("load pool draft of division %d: %w", divisionID, err)
	}
	if len(rows) == 0 {
		return nil, time.Time{}, false, nil
	}
	var pools [][]int64
	if err := json.Unmarshal(rows[0].Pools, &pools); err != nil {
		return nil, time.Time{}, false, fmt.Errorf("decode pool draft of division %d: %w", divisionID, err)
	}
	return pools, rows[0].UpdatedAt, true, nil
}

// SavePoolDraft は部門の組分けの下書きを置き換える。組は 1〜maxDraftPools 個(空の組も残せる。途中の
// 下書きのため)、チームはこの部門のチームで、同じチームは 1 回だけ。違えば domain.ValidationError。
func SavePoolDraft(ctx context.Context, db *gorm.DB, divisionID int64, pools [][]int64, at time.Time) error {
	if err := domain.CheckRange("pools", len(pools), 1, maxDraftPools); err != nil {
		return err
	}
	var ids []int64
	seen := map[int64]bool{}
	for i, pool := range pools {
		for _, id := range pool {
			if seen[id] {
				return domain.Invalid(fmt.Sprintf("pools[%d]", i), fmt.Sprintf("team %d is in two pools", id))
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) > 0 {
		var count int64
		if err := db.WithContext(ctx).Model(&Team{}).Where(sqlTeamsInDivision, ids, divisionID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("check teams of pool draft: %w", err)
		}
		if int(count) != len(ids) {
			return domain.Invalid("pools", "has a team that is not in this division")
		}
	}
	data, err := json.Marshal(pools)
	if err != nil {
		return fmt.Errorf("encode pool draft: %w", err)
	}
	row := PoolDraft{DivisionID: divisionID, Pools: data, UpdatedAt: at}
	err = db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "division_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"pools", "updated_at"})}).Create(&row).Error
	if err != nil {
		return writeError("save pool draft", err)
	}
	return nil
}
