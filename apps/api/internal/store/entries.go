package store

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

// 大会への申込(migrations/00040)の読み書き。決まり(検査、状態の移り方)は entry にあり、ここは
// 検査済みの値を保存し、読んだ行を entry の型にするだけ。

const (
	sqlByQuestions       = "question_id IN ?"
	sqlByTeamEntry       = "team_entry_id = ?"
	sqlByTeamEntries     = "team_entry_id IN ?"
	sqlQuestionsNotIn    = "tournament_id = ? AND id NOT IN ?"
	sqlOrderSubmitted    = "submitted_at, id"
	sqlSelectAnswerCount = "question_id, count(*) AS answered"
	sqlSelectMemberCount = "team_entry_id, count(*) AS members, count(user_id) AS registered"
	sqlSelectMaxSort     = "COALESCE(MAX(sort_order), 0)"
	sqlGroupQuestion     = "question_id"
	sqlGroupTeamEntry    = "team_entry_id"
)

// EntryQuestion は entry_questions の行。Options は文字列の配列の JSON。
type EntryQuestion struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	TournamentID int64     `gorm:"column:tournament_id"`
	Scope        string    `gorm:"column:scope"`
	Label        string    `gorm:"column:label"`
	Kind         string    `gorm:"column:kind"`
	Required     bool      `gorm:"column:required"`
	Options      []byte    `gorm:"column:options;type:jsonb"`
	SortOrder    int       `gorm:"column:sort_order"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (EntryQuestion) TableName() string { return "entry_questions" }

// TeamEntry は team_entries の行。
type TeamEntry struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	TournamentID  int64      `gorm:"column:tournament_id"`
	DivisionID    int64      `gorm:"column:division_id"`
	TeamNameFull  string     `gorm:"column:team_name_full"`
	TeamNameShort string     `gorm:"column:team_name_short"`
	RepName       string     `gorm:"column:rep_name"`
	RepPhone      string     `gorm:"column:rep_phone"`
	RepEmail      string     `gorm:"column:rep_email"`
	RepUserID     *int64     `gorm:"column:rep_user_id"`
	Channel       string     `gorm:"column:channel"`
	Status        string     `gorm:"column:status"`
	ReturnNote    string     `gorm:"column:return_note"`
	InviteCode    string     `gorm:"column:invite_code"`
	TeamID        *int64     `gorm:"column:team_id"`
	SubmittedAt   time.Time  `gorm:"column:submitted_at"`
	DecidedAt     *time.Time `gorm:"column:decided_at"`
	DecidedBy     *int64     `gorm:"column:decided_by"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (TeamEntry) TableName() string { return "team_entries" }

// MemberEntry は member_entries の行。
type MemberEntry struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	TeamEntryID int64     `gorm:"column:team_entry_id"`
	UserID      *int64    `gorm:"column:user_id"`
	Name        string    `gorm:"column:name"`
	Number      string    `gorm:"column:number"`
	PlayerID    *int64    `gorm:"column:player_id"`
	SortOrder   int       `gorm:"column:sort_order"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (MemberEntry) TableName() string { return "member_entries" }

// EntryAnswer は entry_answers の行。TeamEntryID と MemberEntryID のどちらか一方だけが入る。
type EntryAnswer struct {
	ID            int64  `gorm:"column:id;primaryKey"`
	QuestionID    int64  `gorm:"column:question_id"`
	TeamEntryID   *int64 `gorm:"column:team_entry_id"`
	MemberEntryID *int64 `gorm:"column:member_entry_id"`
	Value         string `gorm:"column:value"`
}

// TableName は GORM に表名を伝える。
func (EntryAnswer) TableName() string { return "entry_answers" }

// EntryStore は entry.Store を GORM で実装する。
type EntryStore struct {
	db *gorm.DB
}

var _ entry.Store = (*EntryStore)(nil)

// NewEntryStore は db で読み書きする EntryStore を返す。
func NewEntryStore(db *gorm.DB) *EntryStore {
	return &EntryStore{db: db}
}

// Tournament は entry.Store.Tournament。
func (s *EntryStore) Tournament(ctx context.Context, id int64) (entry.Tournament, error) {
	row, err := findByID[Tournament](ctx, s.db, id, domain.ResourceTournament)
	if err != nil {
		return entry.Tournament{}, err
	}
	var divisions []Division
	err = s.db.WithContext(ctx).Select("id", "name_ja", "name_en").
		Where(sqlByTournament, id).Order(sqlOrderSortID).Find(&divisions).Error
	if err != nil {
		return entry.Tournament{}, fmt.Errorf("load divisions of tournament %d: %w", id, err)
	}
	out := entry.Tournament{ID: row.ID, Status: row.Status, Divisions: make([]entry.Division, len(divisions))}
	for i, division := range divisions {
		out.Divisions[i] = entry.Division{ID: division.ID, Name: i18n(division.NameJa, division.NameEn)}
	}
	return out, nil
}

// Questions は entry.Store.Questions。問い合わせは 2 回(質問、答えの数)。
func (s *EntryStore) Questions(ctx context.Context, tournamentID int64) ([]entry.Question, error) {
	return loadQuestions(ctx, s.db, tournamentID)
}

func loadQuestions(ctx context.Context, db *gorm.DB, tournamentID int64) ([]entry.Question, error) {
	var rows []EntryQuestion
	err := db.WithContext(ctx).Where(sqlByTournament, tournamentID).Order(sqlOrderSortID).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load entry questions of tournament %d: %w", tournamentID, err)
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	answered := map[int64]int{}
	if len(ids) > 0 {
		var counts []struct {
			QuestionID int64
			Answered   int
		}
		err := db.WithContext(ctx).Model(&EntryAnswer{}).Select(sqlSelectAnswerCount).
			Where(sqlByQuestions, ids).Group(sqlGroupQuestion).Scan(&counts).Error
		if err != nil {
			return nil, fmt.Errorf("count answers of tournament %d: %w", tournamentID, err)
		}
		for _, count := range counts {
			answered[count.QuestionID] = count.Answered
		}
	}
	out := make([]entry.Question, len(rows))
	for i, row := range rows {
		var options []string
		if err := json.Unmarshal(row.Options, &options); err != nil {
			return nil, fmt.Errorf("decode options of entry question %d: %w", row.ID, err)
		}
		out[i] = entry.Question{ID: row.ID, Scope: entry.Scope(row.Scope), Label: row.Label,
			Kind: entry.Kind(row.Kind), Required: row.Required, Options: options, Answered: answered[row.ID]}
	}
	return out, nil
}

// ReplaceQuestions は entry.Store.ReplaceQuestions。並びに無い質問は答えごと消す(ON DELETE CASCADE)。
func (s *EntryStore) ReplaceQuestions(
	ctx context.Context,
	tournamentID int64,
	questions []entry.Question,
) ([]entry.Question, error) {
	var out []entry.Question
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		kept := []int64{0} // NOT IN () は SQL の誤りなので、使われない ID 0 を入れておく
		for _, question := range questions {
			if question.ID != 0 {
				kept = append(kept, question.ID)
			}
		}
		if err := tx.Where(sqlQuestionsNotIn, tournamentID, kept).Delete(&EntryQuestion{}).Error; err != nil {
			return writeError("delete entry questions", err)
		}
		for i, question := range questions {
			options, err := json.Marshal(question.Options)
			if err != nil {
				return fmt.Errorf("encode options of entry question: %w", err)
			}
			row := EntryQuestion{ID: question.ID, TournamentID: tournamentID, Scope: string(question.Scope),
				Label: question.Label, Kind: string(question.Kind), Required: question.Required,
				Options: options, SortOrder: i + 1}
			if question.ID == 0 {
				err = tx.Create(&row).Error
			} else {
				err = tx.Model(&EntryQuestion{}).Where(sqlByID, question.ID).Updates(map[string]any{
					"scope": row.Scope, "label": row.Label, "kind": row.Kind, "required": row.Required,
					"options": row.Options, "sort_order": row.SortOrder, "updated_at": time.Now(),
				}).Error
			}
			if err != nil {
				return writeError("save entry question", err)
			}
		}
		var err error
		out, err = loadQuestions(ctx, tx, tournamentID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Create は entry.Store.Create。申込、メンバー、答えを 1 つのトランザクションで保存する。同じ部門に同じ
// チーム名の申込があれば、一意索引(team_entries_name_uniq)の違反が domain.ErrConflict になる。
func (s *EntryStore) Create(
	ctx context.Context,
	tournamentID int64,
	newEntry entry.NewEntry,
) (entry.Detail, error) {
	var out entry.Detail
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := TeamEntry{TournamentID: tournamentID, DivisionID: newEntry.DivisionID,
			TeamNameFull: newEntry.FullName, TeamNameShort: newEntry.ShortName, RepName: newEntry.RepName,
			RepPhone: newEntry.RepPhone, RepEmail: newEntry.RepEmail, Channel: string(newEntry.Channel),
			Status: string(entry.StatusPending), InviteCode: newEntry.InviteCode, SubmittedAt: newEntry.At}
		if err := tx.Create(&row).Error; err != nil {
			return writeError("create entry: a team with the same name has already applied", err)
		}
		if len(newEntry.Members) > 0 {
			members := make([]MemberEntry, len(newEntry.Members))
			for i, member := range newEntry.Members {
				members[i] = MemberEntry{TeamEntryID: row.ID, Name: member.Name, Number: member.Number,
					SortOrder: i + 1}
			}
			if err := tx.Create(&members).Error; err != nil {
				return writeError("create entry members", err)
			}
		}
		if len(newEntry.Answers) > 0 {
			answers := make([]EntryAnswer, len(newEntry.Answers))
			for i, answer := range newEntry.Answers {
				answers[i] = EntryAnswer{QuestionID: answer.QuestionID, TeamEntryID: &row.ID, Value: answer.Value}
			}
			if err := tx.Create(&answers).Error; err != nil {
				return writeError("create entry answers", err)
			}
		}
		var err error
		out, err = loadEntry(ctx, tx, row.ID)
		return err
	})
	return out, err
}

// List は entry.Store.List。問い合わせは申込の数によらず 3 回(申込、部門、メンバーの数)。
func (s *EntryStore) List(ctx context.Context, tournamentID int64) ([]entry.Summary, error) {
	var rows []TeamEntry
	err := s.db.WithContext(ctx).Where(sqlByTournament, tournamentID).Order(sqlOrderSubmitted).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list entries of tournament %d: %w", tournamentID, err)
	}
	if len(rows) == 0 {
		return []entry.Summary{}, nil
	}
	divisions, err := entryDivisions(ctx, s.db, rows)
	if err != nil {
		return nil, err
	}
	counts, err := memberCounts(ctx, s.db, rows)
	if err != nil {
		return nil, err
	}
	out := make([]entry.Summary, len(rows))
	for i, row := range rows {
		count := counts[row.ID]
		out[i] = toEntrySummary(row, divisions[row.DivisionID], count.Members, count.Registered)
	}
	return out, nil
}

type memberCount struct {
	TeamEntryID int64
	Members     int
	Registered  int
}

func memberCounts(ctx context.Context, db *gorm.DB, rows []TeamEntry) (map[int64]memberCount, error) {
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	var counts []memberCount
	err := db.WithContext(ctx).Model(&MemberEntry{}).Select(sqlSelectMemberCount).
		Where(sqlByTeamEntries, ids).Group(sqlGroupTeamEntry).Scan(&counts).Error
	if err != nil {
		return nil, fmt.Errorf("count entry members: %w", err)
	}
	out := make(map[int64]memberCount, len(counts))
	for _, count := range counts {
		out[count.TeamEntryID] = count
	}
	return out, nil
}

func entryDivisions(ctx context.Context, db *gorm.DB, rows []TeamEntry) (map[int64]entry.Division, error) {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if !slices.Contains(ids, row.DivisionID) {
			ids = append(ids, row.DivisionID)
		}
	}
	var divisions []Division
	err := db.WithContext(ctx).Select("id", "name_ja", "name_en").Where(sqlByIDs, ids).Find(&divisions).Error
	if err != nil {
		return nil, fmt.Errorf("load divisions of entries: %w", err)
	}
	out := make(map[int64]entry.Division, len(divisions))
	for _, division := range divisions {
		out[division.ID] = entry.Division{ID: division.ID, Name: i18n(division.NameJa, division.NameEn)}
	}
	return out, nil
}

func toEntrySummary(row TeamEntry, division entry.Division, members, registered int) entry.Summary {
	return entry.Summary{ID: row.ID, Division: division, FullName: row.TeamNameFull,
		ShortName: row.TeamNameShort, RepName: row.RepName, Channel: entry.Channel(row.Channel),
		Status: entry.Status(row.Status), Members: members, Registered: registered,
		SubmittedAt: row.SubmittedAt, TeamID: row.TeamID}
}

// Get は entry.Store.Get。
func (s *EntryStore) Get(ctx context.Context, id int64) (entry.Detail, error) {
	return loadEntry(ctx, s.db, id)
}

// Details は entry.Store.Details。問い合わせは申込の数によらず 4 回(申込と loadDetails の 3 回)。
func (s *EntryStore) Details(ctx context.Context, tournamentID int64) ([]entry.Detail, error) {
	var rows []TeamEntry
	err := s.db.WithContext(ctx).Where(sqlByTournament, tournamentID).Order(sqlOrderSubmitted).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list entries of tournament %d: %w", tournamentID, err)
	}
	return loadDetails(ctx, s.db, rows)
}

// loadEntry は申込 1 件を読む。問い合わせは 4 回(申込、部門、メンバー、答え)。
func loadEntry(ctx context.Context, db *gorm.DB, id int64) (entry.Detail, error) {
	row, err := findByID[TeamEntry](ctx, db, id, domain.ResourceEntry)
	if err != nil {
		return entry.Detail{}, err
	}
	details, err := loadDetails(ctx, db, []TeamEntry{row})
	if err != nil {
		return entry.Detail{}, err
	}
	return details[0], nil
}

// loadDetails は申込の行に部門、メンバー、答えを付ける。問い合わせは申込の数によらず 3 回(部門、
// メンバー、答え)。返す並びは rows と同じ。
func loadDetails(ctx context.Context, db *gorm.DB, rows []TeamEntry) ([]entry.Detail, error) {
	if len(rows) == 0 {
		return []entry.Detail{}, nil
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	divisions, err := entryDivisions(ctx, db, rows)
	if err != nil {
		return nil, err
	}
	var members []MemberEntry
	err = db.WithContext(ctx).Where(sqlByTeamEntries, ids).Order(sqlOrderSortID).Find(&members).Error
	if err != nil {
		return nil, fmt.Errorf("load members of entries: %w", err)
	}
	var answers []EntryAnswer
	if err := db.WithContext(ctx).Where(sqlByTeamEntries, ids).Order(sqlOrderID).Find(&answers).Error; err != nil {
		return nil, fmt.Errorf("load answers of entries: %w", err)
	}
	membersOf := make(map[int64][]entry.Member, len(rows))
	for _, member := range members {
		membersOf[member.TeamEntryID] = append(membersOf[member.TeamEntryID], toEntryMember(member))
	}
	answersOf := make(map[int64][]entry.Answer, len(rows))
	for _, answer := range answers {
		// 代表エントリーの答えだけを読む(Where が team_entry_id で絞っているので nil にならない)
		owner := *answer.TeamEntryID
		answersOf[owner] = append(answersOf[owner], entry.Answer{QuestionID: answer.QuestionID, Value: answer.Value})
	}
	out := make([]entry.Detail, len(rows))
	for i, row := range rows {
		list := membersOf[row.ID]
		if list == nil {
			list = []entry.Member{}
		}
		registered := 0
		for _, member := range list {
			if member.Registered {
				registered++
			}
		}
		given := answersOf[row.ID]
		if given == nil {
			given = []entry.Answer{}
		}
		out[i] = entry.Detail{Summary: toEntrySummary(row, divisions[row.DivisionID], len(list), registered),
			TournamentID: row.TournamentID, RepPhone: row.RepPhone, RepEmail: row.RepEmail,
			ReturnNote: row.ReturnNote, InviteCode: row.InviteCode, DecidedAt: row.DecidedAt,
			MemberList: list, Answers: given}
	}
	return out, nil
}

func toEntryMember(row MemberEntry) entry.Member {
	return entry.Member{ID: row.ID, Name: row.Name, Number: row.Number, Registered: row.UserID != nil,
		PlayerID: row.PlayerID}
}

func loadMembers(ctx context.Context, db *gorm.DB, teamEntryID int64) ([]entry.Member, error) {
	var rows []MemberEntry
	err := db.WithContext(ctx).Where(sqlByTeamEntry, teamEntryID).Order(sqlOrderSortID).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load members of entry %d: %w", teamEntryID, err)
	}
	out := make([]entry.Member, len(rows))
	for i, row := range rows {
		out[i] = toEntryMember(row)
	}
	return out, nil
}

// Decide は entry.Store.Decide。
//
// 承認では、部門の行を先にロックしてから(大会の形の適用や組分けとチームの数を食い違わせないため。
// ロックの順序は locks.go)、申込の行をロックする。申込が decision.From の状態でなければ、読んでから
// 別の運営が変えたので domain.ErrConflict にする。承認するメンバーが読んだときと違っても(その間に
// メンバーが登録した)同じく domain.ErrConflict にし、読み直してから承認させる。
func (s *EntryStore) Decide(ctx context.Context, id int64, decision entry.Decision) (entry.Detail, error) {
	var out entry.Detail
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if decision.Approval != nil {
			if err := lockDivision(ctx, tx, decision.Approval.DivisionID); err != nil {
				return err
			}
		}
		var row TeamEntry
		result := tx.Clauses(clause.Locking{Strength: lockStrengthNoKeyUpdate}).
			Where(sqlByID, id).Limit(1).Find(&row)
		if result.Error != nil {
			return fmt.Errorf("lock entry %d: %w", id, result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.NotFound(domain.ResourceEntry, id)
		}
		if row.Status != string(decision.From) {
			return fmt.Errorf("%w: the entry was changed by someone else; reload it", domain.ErrConflict)
		}
		updates := map[string]any{"status": string(decision.To), "return_note": decision.Note,
			"decided_at": decision.At, "updated_at": decision.At, "decided_by": nil}
		if decision.DecidedBy != 0 {
			updates["decided_by"] = decision.DecidedBy
		}
		if approval := decision.Approval; approval != nil {
			teamID, err := approveEntry(ctx, tx, row, *approval)
			if err != nil {
				return err
			}
			updates["team_id"] = teamID
		}
		if err := tx.Model(&TeamEntry{}).Where(sqlByID, id).Updates(updates).Error; err != nil {
			return writeError("decide entry", err)
		}
		var err error
		out, err = loadEntry(ctx, tx, id)
		return err
	})
	return out, err
}

// approveEntry は申込から部門のチームと選手を作り、メンバーエントリーを選手に結びつける。チームは部門の
// 最後に並べる。チームと選手の作り方は管理画面の手入力と同じ(CreateTeam と CreatePlayer)。
func approveEntry(ctx context.Context, tx *gorm.DB, row TeamEntry, approval entry.Approval) (int64, error) {
	if row.DivisionID != approval.DivisionID {
		return 0, fmt.Errorf("%w: the entry was changed by someone else; reload it", domain.ErrConflict)
	}
	members, err := loadMembers(ctx, tx, row.ID)
	if err != nil {
		return 0, err
	}
	if !sameEntryMembers(members, approval.Players) {
		return 0, fmt.Errorf("%w: the members of the entry changed; reload it", domain.ErrConflict)
	}
	var last int
	err = tx.Model(&Team{}).Select(sqlSelectMaxSort).Where(sqlByDivision, row.DivisionID).Scan(&last).Error
	if err != nil {
		return 0, fmt.Errorf("find the last team of division %d: %w", row.DivisionID, err)
	}
	team, err := CreateTeam(ctx, tx, row.DivisionID, TeamFields{
		FullName: domain.I18nText{Ja: approval.FullName}, ShortName: domain.I18nText{Ja: approval.ShortName},
		BarStyle: "default", SortOrder: last + 1,
	})
	if err != nil {
		return 0, err
	}
	for _, player := range approval.Players {
		created, err := CreatePlayer(ctx, tx, team.ID, PlayerFields{Number: player.Number,
			Name: domain.I18nText{Ja: player.Name}})
		if err != nil {
			return 0, err
		}
		err = tx.Model(&MemberEntry{}).Where(sqlByID, player.MemberID).Update("player_id", created.ID).Error
		if err != nil {
			return 0, writeError("link member to player", err)
		}
	}
	return team.ID, nil
}

func sameEntryMembers(members []entry.Member, players []entry.ApprovedPlayer) bool {
	if len(members) != len(players) {
		return false
	}
	for i, member := range members {
		if member.ID != players[i].MemberID {
			return false
		}
	}
	return true
}
