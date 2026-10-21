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

// 記録係の画面を、同じ試合を開いた複数の端末で使っても食い違わないようにするための関数群
// (docs/operations/operator-experience.md「記録係:間違えにくく、直しやすい入力」)。
// 試合の記録担当(いま記録している端末)、ほぼ同時に入った同じ種類の記録の検出、得点のチームの付け替え、
// 記録画面に出す試合時間。試合のスコアと状態を変えるのは得点のチームの付け替えだけで、
// matchupdate.go の runMatchOp を通す(scoreview-design 8)。

const (
	// 期限切れか、同じ端末なら上書きしてよい(記録担当の延長と、空いた試合の引き受け)
	sqlScorerFree         = "match_scorers.expires_at <= ? OR (match_scorers.user_id = ? AND match_scorers.device_id = ?)"
	sqlActiveScorer       = "match_id = ? AND expires_at > ?"
	sqlScorerOfDevice     = "match_id = ? AND user_id = ? AND device_id = ?"
	sqlAddOpsOfMatch      = "match_id = ? AND action = ? AND client_id IS NOT NULL"
	sqlDivisionFormatByID = "division_id = ?"
)

// AuditEventSide は得点のチームの付け替え(監査ログの操作の種類)。取り消しと追加の 2 操作に
// 分けず 1 行に残し、本部が「どの得点をどちらのチームへ直したか」を 1 つの訂正として読めるようにする。
const AuditEventSide = "event.side"

// MatchScorer は match_scorers の行(00012)。
type MatchScorer struct {
	MatchID   int64     `gorm:"column:match_id;primaryKey"`
	UserID    int64     `gorm:"column:user_id"`
	DeviceID  string    `gorm:"column:device_id"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (MatchScorer) TableName() string { return "match_scorers" }

// ScorerClaim は試合の記録担当(期限内のもの)。
type ScorerClaim struct {
	UserID    int64
	UserName  string
	DeviceID  string
	ExpiresAt time.Time
}

// ClaimResult は記録担当の引き受けの結果。Mine はこの端末が記録担当になったか。
type ClaimResult struct {
	Scorer ScorerClaim
	Mine   bool
}

// ClaimScorer は deviceID の端末を試合の記録担当にする(期限を now + domain.ScorerClaimTTL に延ばす)。
// 他の端末が期限内の記録担当なら、takeover が false のときは変えずにその記録担当を返し(Mine=false)、
// true のときは引き継ぐ。試合が無ければ MATCH_ID の NotFound、deviceID が UUID でなければ検証エラー。
func ClaimScorer(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	matchID int64,
	deviceID string,
	takeover bool,
	now time.Time,
) (ClaimResult, error) {
	if !domain.IsClientID(deviceID) {
		return ClaimResult{}, domain.Invalid("deviceId", "must be a UUID")
	}
	if _, err := findByID[Match](ctx, db, matchID, domain.ResourceMatch); err != nil {
		return ClaimResult{}, err
	}
	row := MatchScorer{
		MatchID: matchID, UserID: actor.UserID, DeviceID: deviceID,
		ExpiresAt: now.Add(domain.ScorerClaimTTL), UpdatedAt: now,
	}
	upsert := clause.OnConflict{
		Columns:   []clause.Column{{Name: "match_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "device_id", "expires_at", "updated_at"}),
	}
	if !takeover {
		upsert.Where = clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: sqlScorerFree, Vars: []any{now, actor.UserID, deviceID}},
		}}
	}
	if err := db.WithContext(ctx).Clauses(upsert).Create(&row).Error; err != nil {
		return ClaimResult{}, writeError("claim match scorer", err)
	}
	scorer, err := activeScorer(ctx, db, matchID, now)
	if err != nil {
		return ClaimResult{}, err
	}
	if scorer == nil {
		// 上の upsert で期限内の行が必ずある。無ければ同時に解放された(次の延長で取り直す)
		return ClaimResult{}, fmt.Errorf("match %d: %w: scorer was released concurrently",
			matchID, domain.ErrConflict)
	}
	mine := scorer.UserID == actor.UserID && scorer.DeviceID == deviceID
	return ClaimResult{Scorer: *scorer, Mine: mine}, nil
}

// ReleaseScorer は deviceID の端末が記録担当なら外す(記録画面を閉じたとき)。他の端末が記録担当なら
// 何もしない。
func ReleaseScorer(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	matchID int64,
	deviceID string,
) error {
	if !domain.IsClientID(deviceID) {
		return domain.Invalid("deviceId", "must be a UUID")
	}
	err := db.WithContext(ctx).
		Where(sqlScorerOfDevice, matchID, actor.UserID, deviceID).
		Delete(&MatchScorer{}).
		Error
	if err != nil {
		return fmt.Errorf("release scorer of match %d: %w", matchID, err)
	}
	return nil
}

// activeScorer は試合の、now の時点で期限内の記録担当。いなければ nil。
func activeScorer(ctx context.Context, db *gorm.DB, matchID int64, now time.Time) (*ScorerClaim, error) {
	var rows []MatchScorer
	if err := db.WithContext(ctx).
		Where(sqlActiveScorer, matchID, now).
		Limit(1).
		Find(&rows).
		Error; err != nil {
		return nil, fmt.Errorf("find scorer of match %d: %w", matchID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	names, err := userNames(ctx, db, []int64{rows[0].UserID})
	if err != nil {
		return nil, err
	}
	return &ScorerClaim{
		UserID: rows[0].UserID, UserName: names[rows[0].UserID], DeviceID: rows[0].DeviceID,
		ExpiresAt: rows[0].ExpiresAt,
	}, nil
}

func userNames(ctx context.Context, db *gorm.DB, ids []int64) (map[int64]string, error) {
	var users []User
	if err := db.WithContext(ctx).Where(sqlByIDs, ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("load users: %w", err)
	}
	names := make(map[int64]string, len(users))
	for _, user := range users {
		// 表示名が未登録ならメールアドレス(監査ログと同じ)
		names[user.ID] = user.DisplayName
		if user.DisplayName == "" {
			names[user.ID] = user.Email
		}
	}
	return names, nil
}

// ConcurrentRecordView は、別の記録係がほぼ同時に入れた同じ種類の 2 件の記録(二重記録の疑い)。
type ConcurrentRecordView struct {
	Kind           domain.EventKind
	FirstClientID  string
	SecondClientID string
	FirstUser      string
	SecondUser     string
	SecondsApart   int
}

// concurrentRecords は試合の得点経過のうち、別の利用者がほぼ同時に入れた同じ種類の記録の組
// (domain.FindConcurrentRecords)。記録した利用者は得点経過の追加の監査ログから引く。
func concurrentRecords(
	ctx context.Context,
	db *gorm.DB,
	matchID int64,
	events []MatchEvent,
) ([]ConcurrentRecordView, error) {
	var adds []AuditLog
	if err := db.WithContext(ctx).
		Select("client_id", "user_id").
		Where(sqlAddOpsOfMatch, matchID, AuditEventAdd).
		Find(&adds).
		Error; err != nil {
		return nil, fmt.Errorf("load recorders of match %d: %w", matchID, err)
	}
	recorder := make(map[string]int64, len(adds))
	for _, add := range adds {
		if add.ClientID != nil && add.UserID != nil {
			recorder[*add.ClientID] = *add.UserID
		}
	}
	recorded := make([]domain.RecordedEvent, 0, len(events))
	for _, event := range events {
		recorded = append(recorded, domain.RecordedEvent{
			ClientID: event.ClientID, Kind: event.Kind, UserID: recorder[event.ClientID],
			RecordedAt: event.CreatedAt,
		})
	}
	found := domain.FindConcurrentRecords(recorded)
	views := make([]ConcurrentRecordView, 0, len(found))
	if len(found) == 0 {
		return views, nil
	}
	var ids idSet
	for _, record := range found {
		ids.add(record.First.UserID)
		ids.add(record.Second.UserID)
	}
	names, err := userNames(ctx, db, ids.list)
	if err != nil {
		return nil, err
	}
	for _, record := range found {
		views = append(views, ConcurrentRecordView{
			Kind: record.Kind, FirstClientID: record.First.ClientID,
			SecondClientID: record.Second.ClientID,
			FirstUser:      names[record.First.UserID], SecondUser: names[record.Second.UserID],
			SecondsApart: int(record.Second.RecordedAt.Sub(record.First.RecordedAt) / time.Second),
		})
	}
	return views, nil
}

// checkConfirmedScore は、記録係が確認シートで見たスコア expected が、lock した試合の行の現在の
// スコアと同じかを確かめる(確認した後に別の端末で得点が入ったまま終了しないため)。スコアの無い試合は
// 0-0 として比べる。
func checkConfirmedScore(match Match, expected *domain.Score) error {
	var current domain.Score
	if match.LeftScore != nil && match.RightScore != nil {
		current = domain.Score{Left: *match.LeftScore, Right: *match.RightScore}
	}
	return domain.CheckExpectedScore(expected, current)
}

// fillScoringState は記録画面のための値(記録担当、二重記録の疑い、試合時間)を detail に入れる。
func fillScoringState(
	ctx context.Context,
	db *gorm.DB,
	match Match,
	events []MatchEvent,
	detail *MatchDetail,
) error {
	var err error
	if detail.Scorer, err = activeScorer(ctx, db, match.ID, time.Now()); err != nil {
		return err
	}
	if detail.ConcurrentRecords, err = concurrentRecords(ctx, db, match.ID, events); err != nil {
		return err
	}
	detail.MatchMinutes, err = divisionMatchMinutes(ctx, db, match.DivisionID)
	return err
}

// divisionMatchMinutes はディビジョンの時間割の 1 試合の枠の長さ(分)。大会の形を適用していない
// ディビジョンなど、設定が無ければ nil(記録画面は既定の長さで経過時間の長さを確かめる)。
func divisionMatchMinutes(ctx context.Context, db *gorm.DB, divisionID int64) (*int, error) {
	var formats []DivisionFormat
	if err := db.WithContext(ctx).
		Select("schedule").
		Where(sqlDivisionFormatByID, divisionID).
		Limit(1).
		Find(&formats).
		Error; err != nil {
		return nil, fmt.Errorf("load format of division %d: %w", divisionID, err)
	}
	if len(formats) == 0 || len(formats[0].Schedule) == 0 {
		return nil, nil
	}
	var setting ScheduleSetting
	if err := json.Unmarshal(formats[0].Schedule, &setting); err != nil {
		return nil, fmt.Errorf("decode schedule of division %d: %w", divisionID, err)
	}
	if setting.SlotMinutes <= 0 {
		return nil, nil
	}
	minutes := setting.SlotMinutes
	return &minutes, nil
}

// MoveGoalToSide は得点を相手チームの得点に付け替える(チームを押し間違えた得点の訂正)。得点者、
// アシスト、キャラハンは元のチームの選手なので外す。スコアは得点経過から計算し直す。
// 得点でなければ検証エラー、既に side の得点か、終了した試合なら ErrConflict(先に試合中へ戻す)。
// 監査ログには AuditEventSide の 1 行で、変更前と後の得点経過を残す。
func MoveGoalToSide(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	matchID int64,
	ref EventRef,
	side domain.Side,
	clientID *string,
) (MatchDetail, error) {
	if side != domain.SideLeft && side != domain.SideRight {
		return MatchDetail{}, domain.Invalid("side", "must be left or right")
	}
	return runMatchOp(ctx, db, actor, matchID, clientID, AuditEventSide,
		func(tx *gorm.DB, match Match) (matchOpChange, error) {
			if match.Status == domain.MatchFinished {
				return matchOpChange{}, fmt.Errorf(
					"%w: match is finished; reopen it before correcting events", domain.ErrConflict)
			}
			event, err := findEvent(ctx, tx, matchID, ref)
			if err != nil {
				return matchOpChange{}, err
			}
			if event.Kind != domain.EventGoal || event.Side == nil {
				return matchOpChange{}, domain.Invalid("eventId", "only a goal can change its team")
			}
			if *event.Side == side {
				return matchOpChange{}, fmt.Errorf(
					"%w: the goal is already for that team", domain.ErrConflict)
			}
			before, err := auditEventOf(ctx, tx, event)
			if err != nil {
				return matchOpChange{}, err
			}
			event.Side, event.ScorerID, event.AssistID, event.Callahan = &side, nil, nil, false
			err = tx.WithContext(ctx).Model(&MatchEvent{}).Where(sqlByID, event.ID).Updates(map[string]any{
				"side": side, "scorer_id": nil, "assist_id": nil, "callahan": false,
			}).Error
			if err != nil {
				return matchOpChange{}, writeError("move goal to the other team", err)
			}
			if err := recomputeMatch(ctx, tx, match, match.Status, match.WinnerSide); err != nil {
				return matchOpChange{}, err
			}
			after, err := auditEventOf(ctx, tx, event)
			return matchOpChange{before: before, after: after}, err
		})
}
