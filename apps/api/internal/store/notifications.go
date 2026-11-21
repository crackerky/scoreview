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
		"te.rep_name AS entry_name, te.team_name_full AS team_name"
	sqlSelectMemberRecipients = "lu.id, lu.line_user_id, lu.friend, lu.name AS profile_name, " +
		"me.name AS entry_name, te.team_name_full AS team_name"
	sqlJoinRepLineUser    = "JOIN line_users lu ON lu.id = te.rep_line_user_id"
	sqlJoinMemberLineUser = "JOIN line_users lu ON lu.id = me.line_user_id"
	sqlJoinMemberEntry    = "JOIN team_entries te ON te.id = me.team_entry_id"
	sqlEntriesOf          = "te.tournament_id = ?"
	sqlEntriesNotOf       = "te.tournament_id <> ?"
	sqlEntryStatusIs      = "te.status = ?"
	sqlEntryStatusIsNot   = "te.status <> ?"
	sqlEntryDivisionIs    = "te.division_id = ?"
	// 代表が書いたメンバーの行のうち、まだ誰も自分の LINE を結んでいない行がある
	sqlEntryIncomplete = "EXISTS (SELECT 1 FROM member_entries pending " +
		"WHERE pending.team_entry_id = te.id AND pending.line_user_id IS NULL)"
	// この大会に、取り消していない申込をまだ出していない
	sqlNotAppliedTo = "NOT EXISTS (SELECT 1 FROM team_entries mine " +
		"WHERE mine.rep_line_user_id = te.rep_line_user_id AND mine.tournament_id = ? AND mine.status <> ?)"
	sqlOrderEntriesSubmitted  = "te.submitted_at, te.id"
	sqlOrderEntriesNewest     = "te.submitted_at DESC, te.id DESC"
	sqlOrderMembers           = "te.submitted_at, te.id, me.sort_order, me.id"
	sqlByRequestKey           = "request_key = ?"
	sqlByBatch                = "batch_id = ?"
	sqlByBatches              = "batch_id IN ?"
	sqlJobsOfBatch            = "batch_id = ? AND line_user_id IN ?"
	sqlOrderBatchesNewest     = "created_at DESC, id DESC"
	sqlSelectJobCounts        = "batch_id, status, count(*) AS count"
	sqlGroupBatchStatus       = "batch_id, status"
	sqlSelectUserNames        = "id, display_name, email"
	maxTeamsPerRecipient      = 3
	notificationJobsBatchSize = 500
)

// NotificationBatch は notification_batches の行。Audience は宛先の条件の JSON。
type NotificationBatch struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	TournamentID int64     `gorm:"column:tournament_id"`
	RequestKey   string    `gorm:"column:request_key"`
	Audience     []byte    `gorm:"column:audience;type:jsonb"`
	Body         string    `gorm:"column:body"`
	Automatic    bool      `gorm:"column:automatic"`
	SentBy       *int64    `gorm:"column:sent_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

// TableName は GORM に表名を伝える。
func (NotificationBatch) TableName() string { return "notification_batches" }

// NotificationJob は notification_jobs の行。
type NotificationJob struct {
	ID         int64      `gorm:"column:id;primaryKey"`
	BatchID    int64      `gorm:"column:batch_id"`
	LineUserID int64      `gorm:"column:line_user_id"`
	Name       string     `gorm:"column:name"`
	Teams      string     `gorm:"column:teams"`
	Status     string     `gorm:"column:status"`
	Error      string     `gorm:"column:error"`
	SentAt     *time.Time `gorm:"column:sent_at"`
}

// TableName は GORM に表名を伝える。
func (NotificationJob) TableName() string { return "notification_jobs" }

// audienceJSON は notification_batches.audience の形。
type audienceJSON struct {
	Kind       string `json:"kind"`
	Status     string `json:"status,omitempty"`
	DivisionID int64  `json:"divisionId,omitempty"`
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
	case notify.AudiencePastReps:
		queries = append(queries, reps.Where(sqlEntriesNotOf, tournamentID).Where(sqlEntryStatusIsNot, cancelled).
			Where(sqlNotAppliedTo, tournamentID, cancelled).Order(sqlOrderEntriesNewest))
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
	return collectRecipients(rows), nil
}

// collectRecipients は行を人ごとにまとめる。並びは最初に出てきた順で、チーム名は 1 人につき
// maxTeamsPerRecipient まで(画面で宛先を見分けるためのもので、全部は要らない)。
func collectRecipients(rows []recipientRow) []notify.Recipient {
	out := []notify.Recipient{}
	index := make(map[int64]int, len(rows))
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
	}
	return out
}

// CreateBatch は notify.Store.CreateBatch。同じ request_key の行があれば、一意の制約で 1 回だけ作る。
func (s *NotifyStore) CreateBatch(ctx context.Context, batch notify.NewBatch) (int64, bool, error) {
	audience, err := json.Marshal(audienceJSON{Kind: string(batch.Audience.Kind),
		Status: string(batch.Audience.Status), DivisionID: batch.Audience.DivisionID})
	if err != nil {
		return 0, false, fmt.Errorf("encode notification audience: %w", err)
	}
	row := NotificationBatch{TournamentID: batch.TournamentID, RequestKey: batch.RequestKey, Audience: audience,
		Body: batch.Body, SentBy: &batch.SentBy, CreatedAt: batch.At}
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
		if len(batch.Jobs) == 0 {
			return nil
		}
		jobs := make([]NotificationJob, len(batch.Jobs))
		for i, job := range batch.Jobs {
			jobs[i] = NotificationJob{BatchID: row.ID, LineUserID: job.LineUserID, Name: job.Name, Teams: job.Teams,
				Status: string(job.Status)}
		}
		if err := tx.CreateInBatches(jobs, notificationJobsBatchSize).Error; err != nil {
			return writeError("notification recipients", err)
		}
		return nil
	})
	if err != nil {
		return 0, false, err
	}
	return row.ID, created, nil
}

// FinishJobs は notify.Store.FinishJobs。
func (s *NotifyStore) FinishJobs(
	ctx context.Context,
	batchID int64,
	lineUserIDs []int64,
	result notify.JobResult,
) error {
	updates := map[string]any{"status": string(result.Status), "error": result.Error, "sent_at": result.At}
	err := s.db.WithContext(ctx).Model(&NotificationJob{}).Where(sqlJobsOfBatch, batchID, lineUserIDs).
		Updates(updates).Error
	if err != nil {
		return writeError("notification result", err)
	}
	return nil
}

// Batches は notify.Store.Batches。問い合わせは送信の数によらず 3 回(送信、結果の人数、送った人)。
func (s *NotifyStore) Batches(ctx context.Context, tournamentID int64) ([]notify.Batch, error) {
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
	}
	err = db.Model(&NotificationJob{}).Select(sqlSelectJobCounts).Where(sqlByBatches, ids).
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
		addCount(&out[index[count.BatchID]].Counts, notify.JobStatus(count.Status), count.Count)
	}
	return out, nil
}

// Batch は notify.Store.Batch。
func (s *NotifyStore) Batch(ctx context.Context, id int64) (notify.Batch, error) {
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
		addCount(&out.Counts, status, 1)
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
		CreatedAt: row.CreatedAt, Audience: notify.Audience{Kind: notify.AudienceKind(audience.Kind),
			Status: entry.Status(audience.Status), DivisionID: audience.DivisionID}}
	if row.SentBy != nil {
		out.SentByName = names[*row.SentBy]
	}
	return out, nil
}

func addCount(counts *notify.Counts, status notify.JobStatus, count int) {
	switch status {
	case notify.JobPending:
		counts.Pending += count
	case notify.JobSent:
		counts.Sent += count
	case notify.JobSkipped:
		counts.Skipped += count
	case notify.JobFailed:
		counts.Failed += count
	}
}
