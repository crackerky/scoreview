package store

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/notify"
)

// 管理画面から公式アカウントで送る宛先と、送った記録(migrations/00043)。条件の決まりは notify にあり、
// ここは条件ごとに決まった問い合わせで宛先を読み、記録を書く。

const (
	sqlTableTeamEntries    = "team_entries AS te"
	sqlTableMemberEntries  = "member_entries AS me"
	sqlSelectRepRecipients = "lu.id, lu.line_user_id, lu.friend, lu.name AS profile_name, " +
		"te.rep_name AS entry_name, te.team_name_full AS team_name, te.id AS entry_id, te.invite_code, TRUE AS rep"
	sqlSelectMemberRecipients = "lu.id, lu.line_user_id, lu.friend, lu.name AS profile_name, " +
		"me.name AS entry_name, te.team_name_full AS team_name, te.id AS entry_id, '' AS invite_code, FALSE AS rep"
	// 代表が書いたメンバーのうち、まだ自分の LINE を結んでいない人({まだ登録していない人} の差し込み)
	sqlSelectUnregistered = "me.team_entry_id, me.name"
	sqlUnregisteredOf     = "me.team_entry_id IN ? AND me.line_user_id IS NULL"
	sqlOrderUnregistered  = "me.team_entry_id, me.sort_order, me.id"
	sqlJoinRepLineUser    = "JOIN line_users lu ON lu.id = te.rep_line_user_id"
	sqlJoinMemberLineUser = "JOIN line_users lu ON lu.id = me.line_user_id"
	sqlJoinMemberEntry    = "JOIN team_entries te ON te.id = me.team_entry_id"
	sqlEntriesOf          = "te.tournament_id = ?"
	sqlEntriesNotOf       = "te.tournament_id <> ?"
	sqlEntryStatusIs      = "te.status = ?"
	sqlEntryStatusIsNot   = "te.status <> ?"
	sqlEntryDivisionIs    = "te.division_id = ?"
	sqlEntryNoScorer      = "te.scorer_member_id IS NULL"
	// 代表が書いたメンバーの行のうち、まだ誰も自分の LINE を結んでいない行がある
	sqlEntryIncomplete = "EXISTS (SELECT 1 FROM member_entries pending " +
		"WHERE pending.team_entry_id = te.id AND pending.line_user_id IS NULL)"
	// この大会に、取り消していない申込をまだ出していない
	sqlNotAppliedTo = "NOT EXISTS (SELECT 1 FROM team_entries mine " +
		"WHERE mine.rep_line_user_id = te.rep_line_user_id AND mine.tournament_id = ? AND mine.status <> ?)"
	sqlOrderEntriesSubmitted = "te.submitted_at, te.id"
	sqlOrderEntriesNewest    = "te.submitted_at DESC, te.id DESC"
	sqlOrderMembers          = "te.submitted_at, te.id, me.sort_order, me.id"
	sqlByRequestKey          = "request_key = ?"
	sqlDueBatches            = "state = ? AND send_at <= ?"
	sqlOrderDue              = "send_at, id"
	sqlScheduledBatch        = "id = ? AND state = ?"
	sqlByBatch               = "batch_id = ?"
	sqlByBatches             = "batch_id IN ?"
	sqlJobsOfBatch           = "batch_id = ? AND line_user_id IN ?"
	sqlEveryoneJobOfBatch    = "batch_id = ? AND line_user_id IS NULL"
	// 送れなかった宛先と、持ち時間の切れた途中の宛先(送ってみる宛先が無い skipped の行は除く)
	sqlClaimable = "batch_id = ? AND retry_key <> '' AND " +
		"(status = ? OR (status = ? AND (claimed_until IS NULL OR claimed_until < ?)))"
	sqlSelectLineIDs = "id, line_user_id"
	// 結果ごとの人数と、そのうち持ち時間の切れた人数(途中の人のうち送り直せる人)
	sqlSelectJobCountsAt = "batch_id, status, count(*) AS count, " +
		"count(*) FILTER (WHERE claimed_until IS NULL OR claimed_until < ?) AS expired"
	sqlOrderBatchesNewest     = "created_at DESC, id DESC"
	sqlGroupBatchStatus       = "batch_id, status"
	sqlSelectUserNames        = "id, display_name, email"
	maxTeamsPerRecipient      = 3
	notificationJobsBatchSize = 500
)

// NotificationBatch は notification_batches の行。Audience は宛先の条件の JSON。
type NotificationBatch struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	TournamentID int64      `gorm:"column:tournament_id"`
	RequestKey   string     `gorm:"column:request_key"`
	Audience     []byte     `gorm:"column:audience;type:jsonb"`
	Body         string     `gorm:"column:body"`
	Automatic    bool       `gorm:"column:automatic"`
	SentBy       *int64     `gorm:"column:sent_by"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	SendAt       *time.Time `gorm:"column:send_at"`
	State        string     `gorm:"column:state"`
	Note         string     `gorm:"column:note"`
	// Buttons は届く文の下のボタン(migrations/00048)。[{kind, label, url}]
	Buttons []byte `gorm:"column:buttons;type:jsonb"`
}

// TableName は GORM に表名を伝える。
func (NotificationBatch) TableName() string { return "notification_batches" }

// NotificationJob は notification_jobs の行。
type NotificationJob struct {
	ID      int64 `gorm:"column:id;primaryKey"`
	BatchID int64 `gorm:"column:batch_id"`
	// LineUserID は宛先の line_users の行(友だち全員の行は nil。migrations/00050)
	LineUserID   *int64     `gorm:"column:line_user_id"`
	Name         string     `gorm:"column:name"`
	Teams        string     `gorm:"column:teams"`
	Status       string     `gorm:"column:status"`
	Error        string     `gorm:"column:error"`
	RetryKey     string     `gorm:"column:retry_key"`
	ClaimedUntil *time.Time `gorm:"column:claimed_until"`
	SentAt       *time.Time `gorm:"column:sent_at"`
	// Body はその人に差し込んで送った文(migrations/00049。差し込みの無い送信は空)
	Body string `gorm:"column:body"`
}

// TableName は GORM に表名を伝える。
func (NotificationJob) TableName() string { return "notification_jobs" }

// audienceJSON は notification_batches.audience の形。
type audienceJSON struct {
	Kind       string `json:"kind"`
	Status     string `json:"status,omitempty"`
	DivisionID int64  `json:"divisionId,omitempty"`
}

// buttonJSON は notification_batches.buttons の 1 つ。
type buttonJSON struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	URL   string `json:"url,omitempty"`
}

func encodeButtons(buttons []notify.Button) ([]byte, error) {
	out := make([]buttonJSON, len(buttons))
	for i, button := range buttons {
		out[i] = buttonJSON{Kind: string(button.Kind), Label: button.Label, URL: button.URL}
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("encode notification buttons: %w", err)
	}
	return data, nil
}

func decodeButtons(id int64, data []byte) ([]notify.Button, error) {
	if len(data) == 0 {
		return []notify.Button{}, nil
	}
	var raw []buttonJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode buttons of notification %d: %w", id, err)
	}
	out := make([]notify.Button, len(raw))
	for i, button := range raw {
		out[i] = notify.Button{Kind: notify.ButtonKind(button.Kind), Label: button.Label, URL: button.URL}
	}
	return out, nil
}

// NotifyStore は notify.Store を GORM で実装する。
type NotifyStore struct {
	db      *gorm.DB
	entries *EntryStore
}

var _ notify.Store = (*NotifyStore)(nil)

// NewNotifyStore は db で読み書きする NotifyStore を返す。
func NewNotifyStore(db *gorm.DB) *NotifyStore {
	return &NotifyStore{db: db, entries: NewEntryStore(db)}
}

// Tournament は notify.Store.Tournament。申込の受付と同じ読み方をする。
func (s *NotifyStore) Tournament(ctx context.Context, id int64) (entry.Tournament, error) {
	return s.entries.Tournament(ctx, id)
}

// recipientRow は宛先を読む問い合わせの 1 行(申込 1 件、またはメンバーの行 1 つ)。
type recipientRow struct {
	ID          int64  `gorm:"column:id"`
	LineUserID  string `gorm:"column:line_user_id"`
	Friend      bool   `gorm:"column:friend"`
	ProfileName string `gorm:"column:profile_name"`
	EntryName   string `gorm:"column:entry_name"`
	TeamName    string `gorm:"column:team_name"`
	EntryID     int64  `gorm:"column:entry_id"`
	InviteCode  string `gorm:"column:invite_code"`
	// Rep はこの行が代表として(メンバーではなく)の行か
	Rep bool `gorm:"column:rep"`
}

// unregisteredRow はまだ登録していないメンバー 1 人。
type unregisteredRow struct {
	TeamEntryID int64  `gorm:"column:team_entry_id"`
	Name        string `gorm:"column:name"`
}

// Recipients は notify.Store.Recipients。問い合わせは条件ごとに 1 回(参加者は代表とメンバーの 2 回)。
func (s *NotifyStore) Recipients(
	ctx context.Context,
	tournamentID int64,
	audience notify.Audience,
) ([]notify.Recipient, error) {
	db := s.db.WithContext(ctx)
	reps := db.Table(sqlTableTeamEntries).Select(sqlSelectRepRecipients).Joins(sqlJoinRepLineUser)
	if audience.DivisionID != 0 {
		reps = reps.Where(sqlEntryDivisionIs, audience.DivisionID)
	}
	cancelled := string(entry.StatusCancelled)
	var queries []*gorm.DB
	switch audience.Kind {
	case notify.AudienceReps:
		reps = reps.Where(sqlEntriesOf, tournamentID)
		if audience.Status != "" {
			reps = reps.Where(sqlEntryStatusIs, string(audience.Status))
		} else {
			reps = reps.Where(sqlEntryStatusIsNot, cancelled)
		}
		queries = append(queries, reps.Order(sqlOrderEntriesSubmitted))
	case notify.AudienceIncomplete:
		queries = append(queries, reps.Where(sqlEntriesOf, tournamentID).Where(sqlEntryStatusIsNot, cancelled).
			Where(sqlEntryIncomplete).Order(sqlOrderEntriesSubmitted))
	case notify.AudienceParticipants:
		approved := string(entry.StatusApproved)
		members := db.Table(sqlTableMemberEntries).Select(sqlSelectMemberRecipients).
			Joins(sqlJoinMemberEntry).Joins(sqlJoinMemberLineUser).
			Where(sqlEntriesOf, tournamentID).Where(sqlEntryStatusIs, approved)
		if audience.DivisionID != 0 {
			members = members.Where(sqlEntryDivisionIs, audience.DivisionID)
		}
		queries = append(queries,
			reps.Where(sqlEntriesOf, tournamentID).Where(sqlEntryStatusIs, approved).Order(sqlOrderEntriesSubmitted),
			members.Order(sqlOrderMembers))
	case notify.AudienceNoScorer:
		queries = append(queries, reps.Where(sqlEntriesOf, tournamentID).
			Where(sqlEntryStatusIs, string(entry.StatusApproved)).Where(sqlEntryNoScorer).Order(sqlOrderEntriesSubmitted))
	case notify.AudiencePastReps:
		queries = append(queries, reps.Where(sqlEntriesNotOf, tournamentID).Where(sqlEntryStatusIsNot, cancelled).
			Where(sqlNotAppliedTo, tournamentID, cancelled).Order(sqlOrderEntriesNewest))
	case notify.AudienceFriends:
		// 友だち全員は LINE の一斉送信で送り、宛先の人を選ばない
		return []notify.Recipient{}, nil
	default:
		return nil, domain.Invalid("kind", "is unknown")
	}
	var rows []recipientRow
	for _, query := range queries {
		var found []recipientRow
		if err := query.Find(&found).Error; err != nil {
			return nil, fmt.Errorf("list recipients (%s) of tournament %d: %w", audience.Kind, tournamentID, err)
		}
		rows = append(rows, found...)
	}
	unregistered, err := s.unregistered(ctx, rows)
	if err != nil {
		return nil, err
	}
	return collectRecipients(rows, unregistered), nil
}

// unregistered は代表の行の申込ごとの、まだ登録していないメンバーの名前(書いた順)。
func (s *NotifyStore) unregistered(ctx context.Context, rows []recipientRow) (map[int64][]string, error) {
	var ids []int64
	for _, row := range rows {
		if row.Rep {
			ids = append(ids, row.EntryID)
		}
	}
	out := map[int64][]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var found []unregisteredRow
	err := s.db.WithContext(ctx).Table(sqlTableMemberEntries).Select(sqlSelectUnregistered).
		Where(sqlUnregisteredOf, ids).Order(sqlOrderUnregistered).Find(&found).Error
	if err != nil {
		return nil, fmt.Errorf("list unregistered members: %w", err)
	}
	for _, row := range found {
		out[row.TeamEntryID] = append(out[row.TeamEntryID], row.Name)
	}
	return out, nil
}

// collectRecipients は行を人ごとにまとめる。並びは最初に出てきた順で、チーム名は 1 人につき
// maxTeamsPerRecipient まで(画面で宛先を見分けるためのもので、全部は要らない)。
func collectRecipients(rows []recipientRow, unregistered map[int64][]string) []notify.Recipient {
	out := []notify.Recipient{}
	index := make(map[int64]int, len(rows))
	// entries は人ごとの、申込の id から Entries の位置(代表とメンバーの両方で同じ申込に出てくる人は 1 つに)
	entries := make(map[int64]map[int64]int, len(rows))
	for _, row := range rows {
		at, ok := index[row.ID]
		if !ok {
			name := row.ProfileName
			if name == "" {
				name = row.EntryName
			}
			out = append(out, notify.Recipient{LineUserID: row.ID, UserID: row.LineUserID, Friend: row.Friend,
				Name: name, Teams: []string{}})
			at = len(out) - 1
			index[row.ID] = at
		}
		recipient := &out[at]
		if len(recipient.Teams) < maxTeamsPerRecipient && !slices.Contains(recipient.Teams, row.TeamName) {
			recipient.Teams = append(recipient.Teams, row.TeamName)
		}
		item := notify.RecipientEntry{Team: row.TeamName}
		if row.Rep {
			item.InviteCode, item.Unregistered = row.InviteCode, unregistered[row.EntryID]
		}
		if entries[row.ID] == nil {
			entries[row.ID] = map[int64]int{}
		}
		if pos, seen := entries[row.ID][row.EntryID]; !seen {
			entries[row.ID][row.EntryID] = len(recipient.Entries)
			recipient.Entries = append(recipient.Entries, item)
		} else if row.Rep {
			recipient.Entries[pos] = item
		}
	}
	return out
}

// FindBatch は notify.Store.FindBatch。
func (s *NotifyStore) FindBatch(ctx context.Context, requestKey string) (int64, bool, error) {
	var rows []NotificationBatch
	err := s.db.WithContext(ctx).Select("id").Where(sqlByRequestKey, requestKey).Limit(1).Find(&rows).Error
	if err != nil {
		return 0, false, fmt.Errorf("find notification by request key: %w", err)
	}
	if len(rows) == 0 {
		return 0, false, nil
	}
	return rows[0].ID, true, nil
}

// Claim は notify.Store.Claim。持つ宛先を 1 回の UPDATE で選んで書き換え、書き換えた行だけを返す
// (同時に呼ばれても、行のロックで後の UPDATE は書き換え済みの行を条件から外すので、同じまとまりを
// 2 つの呼び出しが持つことはない)。まとまりの宛先は同じ状態と期限で書かれるので、まとめて持たれる。
func (s *NotifyStore) Claim(ctx context.Context, batchID int64, now, until time.Time) ([]notify.Group, error) {
	db := s.db.WithContext(ctx)
	var claimed []NotificationJob
	pending, failed := string(notify.JobPending), string(notify.JobFailed)
	err := db.Model(&claimed).Clauses(clause.Returning{}).Where(sqlClaimable, batchID, failed, pending, now).
		Updates(map[string]any{"status": pending, "claimed_until": until}).Error
	if err != nil {
		return nil, writeError("claim notification recipients", err)
	}
	if len(claimed) == 0 {
		return nil, nil
	}
	slices.SortFunc(claimed, func(a, b NotificationJob) int { return cmp.Compare(a.ID, b.ID) })
	ids := make([]int64, 0, len(claimed))
	for _, job := range claimed {
		if job.LineUserID != nil {
			ids = append(ids, *job.LineUserID)
		}
	}
	var users []LineUser
	if err := db.Select(sqlSelectLineIDs).Where(sqlByIDs, ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("load LINE ids of notification %d: %w", batchID, err)
	}
	lineIDs := make(map[int64]string, len(users))
	for _, user := range users {
		lineIDs[user.ID] = user.LineUserID
	}
	var groups []notify.Group
	index := map[string]int{}
	for _, job := range claimed {
		at, ok := index[job.RetryKey]
		if !ok {
			// 同じまとまりの人には同じ文を送ったので、最初の人の文をまとまりの文にする
			groups = append(groups, notify.Group{RetryKey: job.RetryKey, Text: job.Body,
				Everyone: job.LineUserID == nil})
			at = len(groups) - 1
			index[job.RetryKey] = at
		}
		if job.LineUserID != nil {
			groups[at].LineUserIDs = append(groups[at].LineUserIDs, *job.LineUserID)
			groups[at].UserIDs = append(groups[at].UserIDs, lineIDs[*job.LineUserID])
		}
	}
	return groups, nil
}

// CreateBatch は notify.Store.CreateBatch。同じ request_key の行があれば、一意の制約で 1 回だけ作る。
func (s *NotifyStore) CreateBatch(ctx context.Context, batch notify.NewBatch) (int64, bool, error) {
	audience, err := json.Marshal(audienceJSON{Kind: string(batch.Audience.Kind),
		Status: string(batch.Audience.Status), DivisionID: batch.Audience.DivisionID})
	if err != nil {
		return 0, false, fmt.Errorf("encode notification audience: %w", err)
	}
	buttons, err := encodeButtons(batch.Buttons)
	if err != nil {
		return 0, false, err
	}
	row := NotificationBatch{TournamentID: batch.TournamentID, RequestKey: batch.RequestKey, Audience: audience,
		Body: batch.Body, Buttons: buttons, CreatedAt: batch.At, SendAt: batch.SendAt, State: string(notify.BatchSent)}
	if batch.SendAt != nil {
		row.State = string(notify.BatchScheduled)
	}
	// 送った運営がいない送信(0)は NULL にする(users に 0 の行は無い)
	if batch.SentBy != 0 {
		row.SentBy = &batch.SentBy
	}
	created := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "request_key"}}, DoNothing: true}).
			Create(&row)
		if result.Error != nil {
			return writeError("notification", result.Error)
		}
		if result.RowsAffected == 0 {
			var existing NotificationBatch
			if err := tx.Select("id").Where(sqlByRequestKey, batch.RequestKey).Take(&existing).Error; err != nil {
				return fmt.Errorf("find notification by request key: %w", err)
			}
			row.ID = existing.ID
			return nil
		}
		created = true
		return insertJobs(tx, row.ID, batch.Jobs, batch.ClaimedUntil)
	})
	if err != nil {
		return 0, false, err
	}
	return row.ID, created, nil
}

// insertJobs は送信 batchID の宛先の行を書く。送る宛先(pending)は claimedUntil までこの送信が持つ。
func insertJobs(tx *gorm.DB, batchID int64, newJobs []notify.NewJob, claimedUntil time.Time) error {
	if len(newJobs) == 0 {
		return nil
	}
	jobs := make([]NotificationJob, len(newJobs))
	for i, job := range newJobs {
		jobs[i] = NotificationJob{BatchID: batchID, Name: job.Name, Teams: job.Teams,
			Status: string(job.Status), RetryKey: job.RetryKey, Body: job.Body}
		// 友だち全員の行(LineUserID が 0)は line_users の行を持たない
		if job.LineUserID != 0 {
			jobs[i].LineUserID = &newJobs[i].LineUserID
		}
		if job.Status == notify.JobPending {
			jobs[i].ClaimedUntil = &claimedUntil
		}
	}
	if err := tx.CreateInBatches(jobs, notificationJobsBatchSize).Error; err != nil {
		return writeError("notification recipients", err)
	}
	return nil
}

// DueBatches は notify.Store.DueBatches。
func (s *NotifyStore) DueBatches(ctx context.Context, now time.Time, limit int) ([]notify.Due, error) {
	var rows []NotificationBatch
	err := s.db.WithContext(ctx).Where(sqlDueBatches, string(notify.BatchScheduled), now).Order(sqlOrderDue).
		Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list due notifications: %w", err)
	}
	out := make([]notify.Due, len(rows))
	for i, row := range rows {
		batch, err := toBatch(row, nil)
		if err != nil {
			return nil, err
		}
		out[i] = notify.Due{ID: row.ID, TournamentID: row.TournamentID, Audience: batch.Audience, Body: row.Body,
			Buttons: batch.Buttons}
	}
	return out, nil
}

// StartScheduled は notify.Store.StartScheduled。予約から送ったことへの書き換えは、予約のままのとき
// だけ効く 1 回の UPDATE で行うので、同時に送ろうとしても宛先を書くのは 1 つだけ。
func (s *NotifyStore) StartScheduled(
	ctx context.Context,
	id int64,
	jobs []notify.NewJob,
	claimedUntil time.Time,
) (bool, error) {
	started := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		started, err = moveScheduled(tx, id, map[string]any{"state": string(notify.BatchSent)})
		if err != nil || !started {
			return err
		}
		return insertJobs(tx, id, jobs, claimedUntil)
	})
	if err != nil {
		return false, err
	}
	return started, nil
}

// SkipScheduled は notify.Store.SkipScheduled。
func (s *NotifyStore) SkipScheduled(ctx context.Context, id int64, reason notify.SkipReason) (bool, error) {
	return moveScheduled(s.db.WithContext(ctx), id,
		map[string]any{"state": string(notify.BatchSkipped), "note": string(reason)})
}

// CancelScheduled は notify.Store.CancelScheduled。
func (s *NotifyStore) CancelScheduled(ctx context.Context, id int64) (bool, error) {
	return moveScheduled(s.db.WithContext(ctx), id, map[string]any{"state": string(notify.BatchCancelled)})
}

// moveScheduled は予約のままの送信 id だけを updates で書き換え、書き換えたかを返す。
func moveScheduled(db *gorm.DB, id int64, updates map[string]any) (bool, error) {
	result := db.Model(&NotificationBatch{}).Where(sqlScheduledBatch, id, string(notify.BatchScheduled)).
		Updates(updates)
	if result.Error != nil {
		return false, writeError("notification schedule", result.Error)
	}
	return result.RowsAffected == 1, nil
}

// FinishJobs は notify.Store.FinishJobs。
func (s *NotifyStore) FinishJobs(
	ctx context.Context,
	batchID int64,
	lineUserIDs []int64,
	result notify.JobResult,
) error {
	return s.finish(s.db.WithContext(ctx).Model(&NotificationJob{}).Where(sqlJobsOfBatch, batchID, lineUserIDs), result)
}

// FinishEveryone は notify.Store.FinishEveryone。
func (s *NotifyStore) FinishEveryone(ctx context.Context, batchID int64, result notify.JobResult) error {
	return s.finish(s.db.WithContext(ctx).Model(&NotificationJob{}).Where(sqlEveryoneJobOfBatch, batchID), result)
}

// finish は rows(宛先の行の問い合わせ)に結果を書く。sent_at は LINE が受け付けた時刻なので、送れな
// かった行には書かない(送り直しで送れたら書く)。
func (*NotifyStore) finish(rows *gorm.DB, result notify.JobResult) error {
	var sentAt *time.Time
	if result.Status == notify.JobSent {
		sentAt = &result.At
	}
	updates := map[string]any{"status": string(result.Status), "error": result.Error, "sent_at": sentAt}
	if err := rows.Updates(updates).Error; err != nil {
		return writeError("notification result", err)
	}
	return nil
}

// Batches は notify.Store.Batches。問い合わせは送信の数によらず 3 回(送信、結果の人数、送った人)。
func (s *NotifyStore) Batches(ctx context.Context, tournamentID int64, now time.Time) ([]notify.Batch, error) {
	db := s.db.WithContext(ctx)
	var rows []NotificationBatch
	err := db.Where(sqlByTournament, tournamentID).Order(sqlOrderBatchesNewest).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list notifications of tournament %d: %w", tournamentID, err)
	}
	out := make([]notify.Batch, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]int64, len(rows))
	index := make(map[int64]int, len(rows))
	for i, row := range rows {
		ids[i], index[row.ID] = row.ID, i
	}
	var counts []struct {
		BatchID int64  `gorm:"column:batch_id"`
		Status  string `gorm:"column:status"`
		Count   int    `gorm:"column:count"`
		Expired int    `gorm:"column:expired"`
	}
	err = db.Model(&NotificationJob{}).Select(sqlSelectJobCountsAt, now).Where(sqlByBatches, ids).
		Group(sqlGroupBatchStatus).Find(&counts).Error
	if err != nil {
		return nil, fmt.Errorf("count notification results: %w", err)
	}
	names, err := s.senderNames(ctx, rows)
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		if out[i], err = toBatch(row, names); err != nil {
			return nil, err
		}
	}
	for _, count := range counts {
		addCount(&out[index[count.BatchID]].Counts, notify.JobStatus(count.Status), count.Count, count.Expired)
	}
	return out, nil
}

// Batch は notify.Store.Batch。
func (s *NotifyStore) Batch(ctx context.Context, id int64, now time.Time) (notify.Batch, error) {
	row, err := findByID[NotificationBatch](ctx, s.db, id, domain.ResourceNotification)
	if err != nil {
		return notify.Batch{}, err
	}
	var jobs []NotificationJob
	if err := s.db.WithContext(ctx).Where(sqlByBatch, id).Order(sqlOrderID).Find(&jobs).Error; err != nil {
		return notify.Batch{}, fmt.Errorf("load recipients of notification %d: %w", id, err)
	}
	names, err := s.senderNames(ctx, []NotificationBatch{row})
	if err != nil {
		return notify.Batch{}, err
	}
	out, err := toBatch(row, names)
	if err != nil {
		return notify.Batch{}, err
	}
	out.Jobs = make([]notify.Job, len(jobs))
	for i, job := range jobs {
		status := notify.JobStatus(job.Status)
		out.Jobs[i] = notify.Job{Name: job.Name, Teams: job.Teams, Status: status, Error: job.Error,
			SentAt: job.SentAt}
		expired := 0
		if job.ClaimedUntil == nil || job.ClaimedUntil.Before(now) {
			expired = 1
		}
		addCount(&out.Counts, status, 1, expired)
	}
	return out, nil
}

// senderNames は送った運営の表示名(無ければメールアドレス)を、ユーザー ID ごとに読む。
func (s *NotifyStore) senderNames(ctx context.Context, rows []NotificationBatch) (map[int64]string, error) {
	var ids []int64
	for _, row := range rows {
		if row.SentBy != nil && !slices.Contains(ids, *row.SentBy) {
			ids = append(ids, *row.SentBy)
		}
	}
	names := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	var users []User
	if err := s.db.WithContext(ctx).Select(sqlSelectUserNames).Where(sqlByIDs, ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("load senders of notifications: %w", err)
	}
	for _, user := range users {
		names[user.ID] = cmp.Or(user.DisplayName, user.Email)
	}
	return names, nil
}

func toBatch(row NotificationBatch, names map[int64]string) (notify.Batch, error) {
	var audience audienceJSON
	if err := json.Unmarshal(row.Audience, &audience); err != nil {
		return notify.Batch{}, fmt.Errorf("decode audience of notification %d: %w", row.ID, err)
	}
	out := notify.Batch{ID: row.ID, TournamentID: row.TournamentID, Body: row.Body, Automatic: row.Automatic,
		CreatedAt: row.CreatedAt, SendAt: row.SendAt, State: notify.BatchState(row.State),
		Note: notify.SkipReason(row.Note), Audience: notify.Audience{Kind: notify.AudienceKind(audience.Kind),
			Status: entry.Status(audience.Status), DivisionID: audience.DivisionID}}
	if row.SentBy != nil {
		out.SentByName = names[*row.SentBy]
	}
	buttons, err := decodeButtons(row.ID, row.Buttons)
	if err != nil {
		return notify.Batch{}, err
	}
	out.Buttons = buttons
	return out, nil
}

// addCount は結果 status の count 人を足す。expired はそのうち持ち時間の切れた人数(途中の人のうち
// 送り直せる人を数えるのに使う)。
func addCount(counts *notify.Counts, status notify.JobStatus, count, expired int) {
	switch status {
	case notify.JobPending:
		counts.Pending += count
		counts.Resendable += expired
	case notify.JobSent:
		counts.Sent += count
	case notify.JobSkipped:
		counts.Skipped += count
	case notify.JobFailed:
		counts.Failed += count
		counts.Resendable += count
	}
}
