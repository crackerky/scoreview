package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
)

// 記録係の割り当て(docs/operations/operator-experience.md「記録係を試合やコートごとに割り当てる」)。
// 記録係(staff)は担当大会(tournament_staff)の試合を読めるが、スコアを入力できるのは自分の割り当てが
// 覆う試合だけ。

const (
	sqlUserInTournaments = "user_id = ? AND tournament_id IN ?"
	sqlOrderAssignments  = "user_id, match_date NULLS FIRST, start_time NULLS FIRST, field, id"
)

// AssignmentFields は割り当ての登録値。
type AssignmentFields struct {
	UserID int64
	domain.Assignment
}

// AssignmentRecord は管理画面の割り当て。UserName は表示名(無ければメールアドレス)。
// MatchCode は試合を指す割り当ての試合 CD。
type AssignmentRecord struct {
	ID           int64
	TournamentID int64
	UserID       int64
	UserName     string
	MatchCode    *string
	domain.Assignment
}

// AssignedMatch は記録係の担当試合の 1 件(自分の担当試合の一覧に出す)。
type AssignedMatch struct {
	Match          MatchRecord
	TournamentID   int64
	TournamentName string
	DivisionName   string
}

func toAssignment(row StaffAssignment) domain.Assignment {
	clock := func(value *string) *string {
		if value == nil {
			return nil
		}
		text := *value
		if len(text) > len("15:04") {
			text = text[:len("15:04")]
		}
		return &text
	}
	var date *string
	if row.MatchDate != nil {
		text := row.MatchDate.Format(time.DateOnly)
		date = &text
	}
	return domain.Assignment{
		MatchID: row.MatchID, Field: row.Field, Date: date,
		From: clock(row.StartTime), To: clock(row.EndTime),
	}
}

func matchSlot(match Match) domain.MatchSlot {
	return domain.MatchSlot{
		ID: match.ID, Field: match.Field, Date: formatDate(match), Time: formatClock(match),
	}
}

// ListAssignments は大会の割り当てを記録係、日付、時刻、コートの順で返す。
func ListAssignments(
	ctx context.Context,
	db *gorm.DB,
	tournamentID int64,
) ([]AssignmentRecord, error) {
	if _, err := findByID[Tournament](ctx, db, tournamentID, domain.ResourceTournament); err != nil {
		return nil, err
	}
	var rows []StaffAssignment
	if err := db.WithContext(ctx).
		Where(sqlByTournament, tournamentID).
		Order(sqlOrderAssignments).
		Find(&rows).
		Error; err != nil {
		return nil, fmt.Errorf("list assignments of tournament %d: %w", tournamentID, err)
	}
	var userIDs, matchIDs idSet
	for _, row := range rows {
		userIDs.add(row.UserID)
		matchIDs.addPtr(row.MatchID)
	}
	users := map[int64]User{}
	if len(userIDs.list) > 0 {
		var found []User
		if err := db.WithContext(ctx).Where(sqlByIDs, userIDs.list).Find(&found).Error; err != nil {
			return nil, fmt.Errorf("load users of assignments: %w", err)
		}
		for _, user := range found {
			users[user.ID] = user
		}
	}
	codes := map[int64]string{}
	if len(matchIDs.list) > 0 {
		var found []Match
		if err := db.WithContext(ctx).Where(sqlByIDs, matchIDs.list).Find(&found).Error; err != nil {
			return nil, fmt.Errorf("load matches of assignments: %w", err)
		}
		for _, match := range found {
			codes[match.ID] = match.Code
		}
	}
	out := make([]AssignmentRecord, 0, len(rows))
	for _, row := range rows {
		record := AssignmentRecord{
			ID: row.ID, TournamentID: row.TournamentID, UserID: row.UserID,
			Assignment: toAssignment(row),
		}
		if user, ok := users[row.UserID]; ok {
			record.UserName = user.DisplayName
			if record.UserName == "" {
				record.UserName = user.Email
			}
		}
		if row.MatchID != nil {
			code := codes[*row.MatchID]
			record.MatchCode = &code
		}
		out = append(out, record)
	}
	return out, nil
}

// CreateAssignment は記録係を割り当てる。記録係はまだ大会の担当でなければ担当にする(割り当てた
// 試合のデータを読めるように)。試合を指すなら同じ大会の試合に限る。
func CreateAssignment(
	ctx context.Context,
	db *gorm.DB,
	tournamentID int64,
	fields AssignmentFields,
) (AssignmentRecord, error) {
	if err := fields.Validate(); err != nil {
		return AssignmentRecord{}, err
	}
	var id int64
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findByID[Tournament](ctx, tx, tournamentID, domain.ResourceTournament); err != nil {
			return err
		}
		var users []User
		if err := tx.Where(sqlByID, fields.UserID).Limit(1).Find(&users).Error; err != nil {
			return fmt.Errorf("load user %d: %w", fields.UserID, err)
		}
		if len(users) == 0 || users[0].Role != domain.RoleStaff {
			return domain.Invalid("userId", "must be a staff user")
		}
		if fields.MatchID != nil {
			owner, err := TournamentOfMatch(ctx, tx, *fields.MatchID)
			if err != nil || owner != tournamentID {
				return domain.Invalid("matchId", "must be a match of the tournament")
			}
		}
		row := StaffAssignment{
			TournamentID: tournamentID, UserID: fields.UserID, MatchID: fields.MatchID,
			Field: fields.Field, StartTime: fields.From, EndTime: fields.To,
		}
		if fields.Date != nil {
			day, err := domain.ParseDay("date", *fields.Date)
			if err != nil {
				return err
			}
			row.MatchDate = &day
		}
		if err := tx.Create(&row).Error; err != nil {
			return writeError("create assignment", err)
		}
		id = row.ID
		staff := TournamentStaff{UserID: fields.UserID, TournamentID: tournamentID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&staff).Error; err != nil {
			return writeError("add staff to tournament", err)
		}
		return nil
	})
	if err != nil {
		return AssignmentRecord{}, err
	}
	records, err := ListAssignments(ctx, db, tournamentID)
	if err != nil {
		return AssignmentRecord{}, err
	}
	for _, record := range records {
		if record.ID == id {
			return record, nil
		}
	}
	return AssignmentRecord{}, fmt.Errorf("assignment %d: %w", id, domain.ErrNotFound)
}

// DeleteAssignment は割り当てを消す。無ければ NotFound。
func DeleteAssignment(ctx context.Context, db *gorm.DB, id int64) error {
	return deleteByID[StaffAssignment](ctx, db, id, domain.ResourceAssignment)
}

// TournamentOfAssignment は割り当ての大会 ID。無ければ NotFound。
func TournamentOfAssignment(ctx context.Context, db *gorm.DB, id int64) (int64, error) {
	row, err := findByID[StaffAssignment](ctx, db, id, domain.ResourceAssignment)
	if err != nil {
		return 0, err
	}
	return row.TournamentID, nil
}

// CanScoreMatch は記録係が試合のスコアを入力してよいか(大会の担当で、割り当てのどれかが試合を覆う)。
func CanScoreMatch(ctx context.Context, db *gorm.DB, userID, matchID int64) (bool, error) {
	match, err := findByID[Match](ctx, db, matchID, domain.ResourceMatch)
	if err != nil {
		return false, err
	}
	division, err := findInternal[Division](ctx, db, match.DivisionID, "division")
	if err != nil {
		return false, err
	}
	staff, err := IsTournamentStaff(ctx, db, userID, division.TournamentID)
	if err != nil || !staff {
		return false, err
	}
	assignments, err := userAssignments(ctx, db, userID, []int64{division.TournamentID})
	if err != nil {
		return false, err
	}
	return domain.AnyCovers(assignments[division.TournamentID], matchSlot(match)), nil
}

// userAssignments は利用者の割り当てを大会ごとに返す。
func userAssignments(
	ctx context.Context,
	db *gorm.DB,
	userID int64,
	tournamentIDs []int64,
) (map[int64][]domain.Assignment, error) {
	var rows []StaffAssignment
	if err := db.WithContext(ctx).
		Where(sqlUserInTournaments, userID, tournamentIDs).
		Find(&rows).
		Error; err != nil {
		return nil, fmt.Errorf("load assignments of user %d: %w", userID, err)
	}
	out := map[int64][]domain.Assignment{}
	for _, row := range rows {
		out[row.TournamentID] = append(out[row.TournamentID], toAssignment(row))
	}
	return out, nil
}

// ListAssignedMatches は利用者の割り当てが覆う試合を、日付、時刻、コートの順で返す(記録係の
// トップ画面の「自分の担当試合」)。担当大会でなくなった大会の割り当ては使わない。
func ListAssignedMatches(ctx context.Context, db *gorm.DB, userID int64) ([]AssignedMatch, error) {
	tournamentIDs, err := StaffTournamentIDs(ctx, db, userID)
	if err != nil || len(tournamentIDs) == 0 {
		return []AssignedMatch{}, err
	}
	assignments, err := userAssignments(ctx, db, userID, tournamentIDs)
	if err != nil {
		return nil, err
	}
	var divisions []Division
	if err := db.WithContext(ctx).
		Where(sqlByTournaments, tournamentIDs).
		Find(&divisions).
		Error; err != nil {
		return nil, fmt.Errorf("load divisions of user %d: %w", userID, err)
	}
	byDivision := map[int64]Division{}
	divisionIDs := make([]int64, 0, len(divisions))
	for _, division := range divisions {
		byDivision[division.ID] = division
		divisionIDs = append(divisionIDs, division.ID)
	}
	var matches []Match
	if len(divisionIDs) > 0 {
		if err := db.WithContext(ctx).
			Where(sqlByDivisions, divisionIDs).
			Find(&matches).
			Error; err != nil {
			return nil, fmt.Errorf("load matches of user %d: %w", userID, err)
		}
	}
	var covered []Match
	for _, match := range matches {
		division := byDivision[match.DivisionID]
		if domain.AnyCovers(assignments[division.TournamentID], matchSlot(match)) {
			covered = append(covered, match)
		}
	}
	sortMatches(covered)
	records, err := toMatchRecords(ctx, db, covered)
	if err != nil {
		return nil, err
	}
	var tournaments []Tournament
	if err := db.WithContext(ctx).Where(sqlByIDs, tournamentIDs).Find(&tournaments).Error; err != nil {
		return nil, fmt.Errorf("load tournaments of user %d: %w", userID, err)
	}
	names := map[int64]string{}
	for _, tournament := range tournaments {
		names[tournament.ID] = tournament.NameJa
	}
	out := make([]AssignedMatch, len(records))
	for i, record := range records {
		division := byDivision[record.DivisionID]
		out[i] = AssignedMatch{
			Match: record, TournamentID: division.TournamentID,
			TournamentName: names[division.TournamentID], DivisionName: division.NameJa,
		}
	}
	return out, nil
}
