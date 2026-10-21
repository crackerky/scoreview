package store

import (
	"time"

	"scoreview/api/internal/domain"
)

// 大会形式エンジンの表の GORM モデル(スキーマの正本は migrations/00005_division_formats.sql)。
// groups と matches に足した列は、既存のモデル(models.go)を変えずに、列を絞った別のモデルで読み書きする。

// DivisionFormat は division_formats の行。Params は format.EncodeParams の JSON、Schedule は
// scheduleJSON(未設定なら nil)。
type DivisionFormat struct {
	DivisionID         int64     `gorm:"column:division_id;primaryKey"`
	Template           string    `gorm:"column:template"`
	Params             []byte    `gorm:"column:params;type:jsonb"`
	StartLetter        string    `gorm:"column:start_letter"`
	Tiebreak           string    `gorm:"column:tiebreak"`
	BandRule           string    `gorm:"column:band_rule"`
	ForfeitWinnerScore int       `gorm:"column:forfeit_winner_score"`
	ForfeitLoserScore  int       `gorm:"column:forfeit_loser_score"`
	Schedule           []byte    `gorm:"column:schedule;type:jsonb"`
	Version            int       `gorm:"column:version"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (DivisionFormat) TableName() string { return "division_formats" }

// Stage は stages の行。
type Stage struct {
	ID                int64      `gorm:"column:id;primaryKey"`
	DivisionID        int64      `gorm:"column:division_id"`
	StageIndex        int        `gorm:"column:stage_index"`
	Name              string     `gorm:"column:name"`
	ConfirmedAt       *time.Time `gorm:"column:confirmed_at"`
	ConfirmedUnplayed bool       `gorm:"column:confirmed_unplayed"`
}

// TableName は GORM に表名を伝える。
func (Stage) TableName() string { return "stages" }

// GroupSlot は group_slots の行。Source は format.EncodeSource の JSON。
type GroupSlot struct {
	ID       int64  `gorm:"column:id;primaryKey"`
	GroupID  int64  `gorm:"column:group_id"`
	Position int    `gorm:"column:position"`
	Source   []byte `gorm:"column:source;type:jsonb"`
	TeamID   *int64 `gorm:"column:team_id"`
}

// TableName は GORM に表名を伝える。
func (GroupSlot) TableName() string { return "group_slots" }

// GroupRankDecision は group_rank_decisions の行。TeamIDs は上位から並べたチーム ID の JSON 配列、
// Basis は指定した時点の並んだチームの各基準の値(format.TeamValues の JSON 配列。00015)。
type GroupRankDecision struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	GroupID   *int64    `gorm:"column:group_id"`
	StageID   *int64    `gorm:"column:stage_id"`
	BandRank  *int      `gorm:"column:band_rank"`
	TeamIDs   []byte    `gorm:"column:team_ids;type:jsonb"`
	Basis     []byte    `gorm:"column:basis;type:jsonb"`
	Reason    string    `gorm:"column:reason"`
	DecidedBy *int64    `gorm:"column:decided_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName は GORM に表名を伝える。
func (GroupRankDecision) TableName() string { return "group_rank_decisions" }

// formatGroupRow は groups の行のうち、形と確定に関わる列だけ。
type formatGroupRow struct {
	ID                int64            `gorm:"column:id;primaryKey"`
	DivisionID        int64            `gorm:"column:division_id"`
	Name              string           `gorm:"column:name"`
	Kind              domain.GroupKind `gorm:"column:kind"`
	StageID           *int64           `gorm:"column:stage_id"`
	FormatIndex       *int             `gorm:"column:format_index"`
	ConfirmedAt       *time.Time       `gorm:"column:confirmed_at"`
	ConfirmedUnplayed bool             `gorm:"column:confirmed_unplayed"`
}

// TableName は GORM に表名を伝える。
func (formatGroupRow) TableName() string { return "groups" }

// matchSlotRow は matches の行のうち、左右の枠の列だけ。
type matchSlotRow struct {
	ID          int64  `gorm:"column:id;primaryKey"`
	LeftSlotID  *int64 `gorm:"column:left_slot_id"`
	RightSlotID *int64 `gorm:"column:right_slot_id"`
}

// TableName は GORM に表名を伝える。
func (matchSlotRow) TableName() string { return "matches" }
