package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// 監査ログ(docs/format-engine.md「監査ログ」)。スコア、得点経過、試合の状態、結果の種類、最終順位の
// 変更を、変更と同じトランザクションで 1 操作 1 行に記録する。本部と記録係の食い違いを事実で
// 解決するため(2018 年の USAU の処分の事例。運営実務の調査 5.3)。

const (
	sqlByClientID   = "client_id = ?"
	sqlOrderAuditID = "id DESC"
)

// 監査ログの操作の種類。
const (
	AuditEventAdd    = "event.add"
	AuditEventRemove = "event.remove"
	AuditEventUpdate = "event.update"
	AuditMatchStatus = "match.status"
	AuditMatchResult = "match.result"
	AuditFinalRank   = "team.final_rank"
)

// AuditMatchState は監査ログに残す試合の結果の値。
type AuditMatchState struct {
	Status     domain.MatchStatus `json:"status"`
	ResultType domain.ResultType  `json:"resultType"`
	LeftScore  *int               `json:"leftScore"`
	RightScore *int               `json:"rightScore"`
	WinnerSide *domain.Side       `json:"winnerSide"`
}

// AuditPlayer は監査ログに残す選手(記録した時点の背番号と名前)。
type AuditPlayer struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
	Name   string `json:"name"`
}

// AuditEvent は監査ログに残す得点経過の 1 行。
type AuditEvent struct {
	ClientID   string           `json:"clientId"`
	Kind       domain.EventKind `json:"kind"`
	Side       *domain.Side     `json:"side"`
	ElapsedSec int              `json:"elapsedSec"`
	Scorer     *AuditPlayer     `json:"scorer"`
	Assist     *AuditPlayer     `json:"assist"`
	Callahan   bool             `json:"callahan"`
}

// AuditSnapshot は変更の前または後の値。操作の種類に応じて使う欄だけを持つ。
type AuditSnapshot struct {
	Match     *AuditMatchState `json:"match,omitempty"`
	Event     *AuditEvent      `json:"event,omitempty"`
	FinalRank *int             `json:"finalRank,omitempty"`
	Format    *AuditFormat     `json:"format,omitempty"`
	// Note は操作の補足(「得点経過(9-8)と違う最終スコアで確定」「棄権の一括処理」)。無ければ空
	Note string `json:"note,omitempty"`
}

// AuditRecord は監査ログの 1 行。UserName と UserRole は操作した利用者(消された利用者なら空)。
type AuditRecord struct {
	ID       int64
	At       time.Time
	UserID   *int64
	UserName string
	MatchID  *int64
	TeamID   *int64
	Action   string
	Before   AuditSnapshot
	After    AuditSnapshot
	UserRole domain.Role
}

// auditEntry は監査ログに書く 1 行。
type auditEntry struct {
	tournamentID int64
	matchID      *int64
	teamID       *int64
	actor        domain.Actor
	action       string
	before       AuditSnapshot
	after        AuditSnapshot
	clientID     *string
}

func writeAudit(ctx context.Context, tx *gorm.DB, entry auditEntry) error {
	before, err := json.Marshal(entry.before)
	if err != nil {
		return fmt.Errorf("encode audit before: %w", err)
	}
	after, err := json.Marshal(entry.after)
	if err != nil {
		return fmt.Errorf("encode audit after: %w", err)
	}
	row := AuditLog{
		TournamentID: entry.tournamentID,
		MatchID:      entry.matchID,
		TeamID:       entry.teamID,
		Action:       entry.action,
		Before:       before,
		After:        after,
		ClientID:     entry.clientID,
	}
	if entry.actor.UserID != 0 {
		userID := entry.actor.UserID
		row.UserID = &userID
	}
	if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
		return writeError("write audit log", err)
	}
	return nil
}

// opApplied は端末の操作 ID の操作が、この試合に既に適用済み(監査ログにある)かどうか。nil なら false。
// 同じ操作 ID が別の試合の操作に使われていれば ErrConflict にする(操作 ID は端末が作る UUID で、
// 衝突は端末の不具合か書き換え。適用済みとして黙って捨てると、記録が消えたことに誰も気づかない)。
func opApplied(ctx context.Context, tx *gorm.DB, matchID int64, clientID *string) (bool, error) {
	if clientID == nil {
		return false, nil
	}
	var rows []AuditLog
	err := tx.WithContext(ctx).
		Select("match_id").
		Where(sqlByClientID, *clientID).
		Limit(1).
		Find(&rows).
		Error
	if err != nil {
		return false, fmt.Errorf("find operation %s: %w", *clientID, err)
	}
	if len(rows) == 0 {
		return false, nil
	}
	if rows[0].MatchID == nil || *rows[0].MatchID != matchID {
		return false, fmt.Errorf("%w: clientId is already used by another operation", domain.ErrConflict)
	}
	return true, nil
}

func auditMatchState(match Match) *AuditMatchState {
	return &AuditMatchState{
		Status: match.Status, ResultType: match.ResultType,
		LeftScore: match.LeftScore, RightScore: match.RightScore, WinnerSide: match.WinnerSide,
	}
}

// auditEventOf は得点経過の行を監査ログの形にする。選手は記録した時点の背番号と名前を残す。
func auditEventOf(ctx context.Context, tx *gorm.DB, event MatchEvent) (*AuditEvent, error) {
	var ids idSet
	ids.addPtr(event.ScorerID)
	ids.addPtr(event.AssistID)
	players, err := loadPlayers(ctx, tx, ids.list)
	if err != nil {
		return nil, err
	}
	player := func(id *int64) *AuditPlayer {
		if id == nil {
			return nil
		}
		row, ok := players[*id]
		if !ok {
			return &AuditPlayer{ID: *id}
		}
		return &AuditPlayer{ID: row.ID, Number: row.Number, Name: row.NameJa}
	}
	return &AuditEvent{
		ClientID: event.ClientID, Kind: event.Kind, Side: event.Side, ElapsedSec: event.ElapsedSec,
		Scorer: player(event.ScorerID), Assist: player(event.AssistID), Callahan: event.Callahan,
	}, nil
}

// ListMatchAudit は試合の監査ログを新しい順で返す。無ければ MATCH_ID の NotFound。
func ListMatchAudit(ctx context.Context, db *gorm.DB, matchID int64) ([]AuditRecord, error) {
	if _, err := findByID[Match](ctx, db, matchID, domain.ResourceMatch); err != nil {
		return nil, err
	}
	var rows []AuditLog
	if err := db.WithContext(ctx).
		Where(sqlByMatch, matchID).
		Order(sqlOrderAuditID).
		Find(&rows).
		Error; err != nil {
		return nil, fmt.Errorf("list audit log of match %d: %w", matchID, err)
	}
	return toAuditRecords(ctx, db, rows)
}

func toAuditRecords(ctx context.Context, db *gorm.DB, rows []AuditLog) ([]AuditRecord, error) {
	var userIDs idSet
	for _, row := range rows {
		userIDs.addPtr(row.UserID)
	}
	users := map[int64]User{}
	if len(userIDs.list) > 0 {
		var found []User
		if err := db.WithContext(ctx).Where(sqlByIDs, userIDs.list).Find(&found).Error; err != nil {
			return nil, fmt.Errorf("load users of audit log: %w", err)
		}
		for _, user := range found {
			users[user.ID] = user
		}
	}
	out := make([]AuditRecord, 0, len(rows))
	for _, row := range rows {
		record := AuditRecord{
			ID: row.ID, At: row.CreatedAt, UserID: row.UserID, MatchID: row.MatchID,
			TeamID: row.TeamID, Action: row.Action,
		}
		if row.UserID != nil {
			if user, ok := users[*row.UserID]; ok {
				record.UserName = user.DisplayName
				if record.UserName == "" {
					record.UserName = user.Email
				}
				record.UserRole = user.Role
			}
		}
		if err := decodeSnapshot(row.Before, &record.Before); err != nil {
			return nil, fmt.Errorf("audit log %d before: %w", row.ID, err)
		}
		if err := decodeSnapshot(row.After, &record.After); err != nil {
			return nil, fmt.Errorf("audit log %d after: %w", row.ID, err)
		}
		out = append(out, record)
	}
	return out, nil
}

func decodeSnapshot(data []byte, dst *AuditSnapshot) error {
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("decode audit snapshot: %w", err)
	}
	return nil
}
