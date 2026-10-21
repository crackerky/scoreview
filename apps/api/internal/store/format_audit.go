package store

import (
	"context"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// 大会形式の操作の監査ログ(形の適用、プールと段の確定、同順位の指定、対戦チームの差し替え)。

// 大会形式の操作の種類。
const (
	AuditFormatApply       = "format.apply"
	AuditGroupConfirm      = "group.confirm"
	AuditStageConfirm      = "stage.confirm"
	AuditGroupRankDecision = "group.rank_decision"
	AuditStageRankDecision = "stage.rank_decision"
	AuditMatchAlignTeams   = "match.align_teams"
)

// AuditFormat は大会形式の操作で監査ログに残す値。操作の種類に応じて使う欄だけを持つ。
type AuditFormat struct {
	Template string  `json:"template,omitempty"`
	Version  int     `json:"version,omitempty"`
	Group    string  `json:"group,omitempty"`
	Stage    string  `json:"stage,omitempty"`
	Rank     int     `json:"rank,omitempty"`
	Teams    []int64 `json:"teams,omitempty"`
	Reason   string  `json:"reason,omitempty"`
	Unplayed bool    `json:"unplayed,omitempty"`
	// Published は大会の締めで順位を公開した、Minutes は中断の再開でずらした分数
	Published bool `json:"published,omitempty"`
	Minutes   int  `json:"minutes,omitempty"`
}

func auditFormatOp(
	ctx context.Context,
	tx *gorm.DB,
	tournamentID int64,
	actor domain.Actor,
	action string,
	matchID *int64,
	after AuditFormat,
) error {
	return writeAudit(ctx, tx, auditEntry{tournamentID: tournamentID, matchID: matchID, actor: actor,
		action: action, after: AuditSnapshot{Format: &after}})
}
