package store

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

// 本人用のリンクとメンバー登録(migrations/00041)の読み書き。決まりは entry にある。

const (
	// 受付中で一覧に載せた大会と、この人が申込を出した終わっていない大会(サブクエリは値で渡す)
	sqlLinkTournaments = "(status = ? AND listed) OR (status <> ? AND id IN (?))"
	sqlRepActive       = "rep_line_user_id = ? AND status <> ?"
	sqlMembersOfEntry  = "id IN ? AND team_entry_id = ?"
	// 前に申し込んだチーム: 同じチーム名(前後の空白と大文字小文字を除く)は最後の申込だけ
	sqlSelectPastTeams   = "DISTINCT ON (lower(btrim(team_name_full))) id, team_name_full, team_name_short, submitted_at"
	sqlPastTeamsOf       = "rep_line_user_id = ? AND tournament_id <> ?"
	sqlOrderPastTeams    = "lower(btrim(team_name_full)), submitted_at DESC, id DESC"
	sqlOrderStartDate    = "start_date NULLS LAST, id"
	sqlByInviteCode      = "invite_code = ?"
	sqlByRep             = "tournament_id = ? AND rep_line_user_id = ?"
	sqlByMemberEntry     = "member_entry_id = ?"
	sqlByMemberOfEntry   = "id = ? AND team_entry_id = ?"
	sqlByLineUserOfEntry = "team_entry_id = ? AND line_user_id = ?"
	// メンバーとして登録したチーム: 取り消していない申込の、終わっていない大会
	sqlSelectJoined = "team_entries.id AS team_entry_id, team_entries.team_name_full, " +
		"tournaments.id AS tournament_id, tournaments.name_ja"
	sqlJoinEntryOfMember   = "JOIN team_entries ON team_entries.id = member_entries.team_entry_id"
	sqlJoinTournamentEntry = "JOIN tournaments ON tournaments.id = team_entries.tournament_id"
	sqlJoinedBy            = "member_entries.line_user_id = ? AND team_entries.status <> ? AND tournaments.status <> ?"
	sqlOrderJoined         = "tournaments.start_date NULLS LAST, tournaments.id, team_entries.id"
	// 記録係に選ばれたチーム: 参加が決まった申込の、終わっていない大会。メンバーの行を通して LINE の人を見る
	sqlSelectScorerTeams = sqlSelectJoined + ", member_entries.name AS member_name"
	sqlJoinScorerEntry   = "JOIN team_entries ON team_entries.scorer_member_id = member_entries.id"
	sqlScorerBy          = "member_entries.line_user_id = ? AND team_entries.status = ? AND tournaments.status <> ?"
	// 記録係の利用者のメールアドレス(ログインには使わない。users.email の一意を満たすための値)
	scorerEmailFormat = "line-scorer-%d@scorer.invalid"
)

// EntryLink は entry_links の行。
type EntryLink struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	TokenHash    []byte    `gorm:"column:token_hash"`
	LineUserID   int64     `gorm:"column:line_user_id"`
	Purpose      string    `gorm:"column:purpose"`
	TournamentID *int64    `gorm:"column:tournament_id"`
	TeamEntryID  *int64    `gorm:"column:team_entry_id"`
	ExpiresAt    time.Time `gorm:"column:expires_at"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

// TableName は GORM に表名を伝える。
func (EntryLink) TableName() string { return "entry_links" }

// LinkTournaments は entry.Store.LinkTournaments。受付中でも一覧に載せない大会(準備中など)は出さない。
func (s *EntryStore) LinkTournaments(ctx context.Context, lineUserID int64) ([]entry.TournamentRef, error) {
	db := s.db.WithContext(ctx)
	applied := db.Model(&TeamEntry{}).Select("tournament_id").
		Where(sqlRepActive, lineUserID, string(entry.StatusCancelled))
	var rows []Tournament
	err := db.Select("id", "name_ja").
		Where(sqlLinkTournaments, domain.TournamentEntryOpen, domain.TournamentFinished, applied).
		Order(sqlOrderStartDate).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list tournaments for entry links: %w", err)
	}
	out := make([]entry.TournamentRef, len(rows))
	for i, row := range rows {
		out[i] = entry.TournamentRef{ID: row.ID, Name: row.NameJa}
	}
	return out, nil
}

// JoinedTeams は entry.Store.JoinedTeams。
func (s *EntryStore) JoinedTeams(ctx context.Context, lineUserID int64) ([]entry.JoinedTeam, error) {
	var rows []struct {
		TeamEntryID  int64
		TeamNameFull string
		TournamentID int64
		NameJa       string
	}
	err := s.db.WithContext(ctx).Model(&MemberEntry{}).Select(sqlSelectJoined).
		Joins(sqlJoinEntryOfMember).Joins(sqlJoinTournamentEntry).
		Where(sqlJoinedBy, lineUserID, string(entry.StatusCancelled), domain.TournamentFinished).
		Order(sqlOrderJoined).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list joined teams: %w", err)
	}
	out := make([]entry.JoinedTeam, len(rows))
	for i, row := range rows {
		out[i] = entry.JoinedTeam{TeamEntryID: row.TeamEntryID, TeamName: row.TeamNameFull,
			Tournament: entry.TournamentRef{ID: row.TournamentID, Name: row.NameJa}}
	}
	return out, nil
}

// ScorerTeams は entry.Store.ScorerTeams。
func (s *EntryStore) ScorerTeams(ctx context.Context, lineUserID int64) ([]entry.ScorerTeam, error) {
	var rows []struct {
		TeamEntryID  int64
		TeamNameFull string
		TournamentID int64
		NameJa       string
		MemberName   string
	}
	err := s.db.WithContext(ctx).Model(&MemberEntry{}).Select(sqlSelectScorerTeams).
		Joins(sqlJoinScorerEntry).Joins(sqlJoinTournamentEntry).
		Where(sqlScorerBy, lineUserID, string(entry.StatusApproved), domain.TournamentFinished).
		Order(sqlOrderJoined).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list scorer teams: %w", err)
	}
	out := make([]entry.ScorerTeam, len(rows))
	for i, row := range rows {
		out[i] = entry.ScorerTeam{MemberName: row.MemberName, JoinedTeam: entry.JoinedTeam{
			TeamEntryID: row.TeamEntryID, TeamName: row.TeamNameFull,
			Tournament: entry.TournamentRef{ID: row.TournamentID, Name: row.NameJa}}}
	}
	return out, nil
}

// EnsureScorer は entry.Store.EnsureScorer。LINE の人の行をロックして、記録係の利用者を 2 つ作らない。
func (s *EntryStore) EnsureScorer(ctx context.Context, lineUserID, tournamentID int64, name string) (int64, error) {
	var userID int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lineUser LineUser
		result := tx.Clauses(clause.Locking{Strength: lockStrengthNoKeyUpdate}).
			Where(sqlByID, lineUserID).Limit(1).Find(&lineUser)
		if result.Error != nil {
			return fmt.Errorf("lock line user %d: %w", lineUserID, result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.NotFound(domain.ResourceEntryLink, 0)
		}
		if lineUser.UserID != nil {
			userID = *lineUser.UserID
		} else {
			user := User{Email: fmt.Sprintf(scorerEmailFormat, lineUserID), DisplayName: name,
				Role: domain.RoleStaff, TeamScorer: true}
			if err := tx.Create(&user).Error; err != nil {
				return writeError("create scorer user", err)
			}
			userID = user.ID
			if err := tx.Model(&LineUser{}).Where(sqlByID, lineUserID).Update("user_id", userID).Error; err != nil {
				return writeError("link scorer user", err)
			}
		}
		staff := TournamentStaff{UserID: userID, TournamentID: tournamentID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&staff).Error; err != nil {
			return writeError("make scorer staff of tournament", err)
		}
		return nil
	})
	return userID, err
}

// SaveLink は entry.Store.SaveLink。
func (s *EntryStore) SaveLink(ctx context.Context, tokenHash []byte, link entry.Link) error {
	row := EntryLink{TokenHash: tokenHash, LineUserID: link.LineUserID, Purpose: string(link.Purpose),
		ExpiresAt: link.ExpiresAt}
	if link.Purpose == entry.LinkTeam {
		row.TournamentID = &link.TournamentID
	} else {
		row.TeamEntryID = &link.TeamEntryID
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return writeError("save entry link", err)
	}
	return nil
}

// FindLink は entry.Store.FindLink。
func (s *EntryStore) FindLink(ctx context.Context, tokenHash []byte) (entry.Link, error) {
	var rows []EntryLink
	if err := s.db.WithContext(ctx).Where(sqlByTokenHash, tokenHash).Limit(1).Find(&rows).Error; err != nil {
		return entry.Link{}, fmt.Errorf("find entry link: %w", err)
	}
	if len(rows) == 0 {
		return entry.Link{}, domain.NotFound(domain.ResourceEntryLink, 0)
	}
	row := rows[0]
	link := entry.Link{LineUserID: row.LineUserID, Purpose: entry.LinkPurpose(row.Purpose),
		ExpiresAt: row.ExpiresAt}
	if row.TournamentID != nil {
		link.TournamentID = *row.TournamentID
	}
	if row.TeamEntryID != nil {
		link.TeamEntryID = *row.TeamEntryID
	}
	return link, nil
}

// FindByInvite は entry.Store.FindByInvite。
func (s *EntryStore) FindByInvite(ctx context.Context, inviteCode string) (entry.Summary, error) {
	var rows []TeamEntry
	if err := s.db.WithContext(ctx).Where(sqlByInviteCode, inviteCode).Limit(1).Find(&rows).Error; err != nil {
		return entry.Summary{}, fmt.Errorf("find entry by invite code: %w", err)
	}
	if len(rows) == 0 {
		return entry.Summary{}, domain.NotFound(domain.ResourceEntry, 0)
	}
	return toEntrySummary(rows[0], entry.Division{ID: rows[0].DivisionID}, 0, 0), nil
}

// RepEntries は entry.Store.RepEntries。
func (s *EntryStore) RepEntries(ctx context.Context, tournamentID, lineUserID int64) ([]entry.Detail, error) {
	var rows []TeamEntry
	err := s.db.WithContext(ctx).Where(sqlByRep, tournamentID, lineUserID).Order(sqlOrderSubmitted).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list entries of representative: %w", err)
	}
	return loadDetails(ctx, s.db, rows)
}

// MemberAnswers は entry.Store.MemberAnswers。
func (s *EntryStore) MemberAnswers(ctx context.Context, memberID int64) ([]entry.Answer, error) {
	var rows []EntryAnswer
	if err := s.db.WithContext(ctx).Where(sqlByMemberEntry, memberID).Order(sqlOrderID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load answers of member %d: %w", memberID, err)
	}
	out := make([]entry.Answer, len(rows))
	for i, row := range rows {
		out[i] = entry.Answer{QuestionID: row.QuestionID, Value: row.Value}
	}
	return out, nil
}

// Profile は entry.Store.Profile。
func (s *EntryStore) Profile(ctx context.Context, lineUserID int64) (*entry.Profile, error) {
	var rows []LineUser
	if err := s.db.WithContext(ctx).Where(sqlByID, lineUserID).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load profile of line user %d: %w", lineUserID, err)
	}
	if len(rows) == 0 || rows[0].ProfileUpdatedAt == nil {
		return nil, nil
	}
	return &entry.Profile{Name: rows[0].Name, Phone: rows[0].Phone, Email: rows[0].Email}, nil
}

// PastTeams は entry.Store.PastTeams。問い合わせはチームの数によらず 2 回(申込、メンバー)。
func (s *EntryStore) PastTeams(
	ctx context.Context,
	lineUserID, exceptTournamentID int64,
	limit int,
) ([]entry.PastTeam, error) {
	db := s.db.WithContext(ctx)
	var rows []TeamEntry
	err := db.Model(&TeamEntry{}).Select(sqlSelectPastTeams).Where(sqlPastTeamsOf, lineUserID, exceptTournamentID).
		Order(sqlOrderPastTeams).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list past teams of line user %d: %w", lineUserID, err)
	}
	slices.SortFunc(rows, func(a, b TeamEntry) int {
		if c := b.SubmittedAt.Compare(a.SubmittedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]entry.PastTeam, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]int64, len(rows))
	index := make(map[int64]int, len(rows))
	for i, row := range rows {
		ids[i], index[row.ID] = row.ID, i
		out[i] = entry.PastTeam{FullName: row.TeamNameFull, ShortName: row.TeamNameShort, Members: []entry.MemberDraft{}}
	}
	var members []MemberEntry
	if err := db.Where(sqlByTeamEntries, ids).Order(sqlOrderSortID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("load members of past teams: %w", err)
	}
	for _, member := range members {
		team := &out[index[member.TeamEntryID]]
		team.Members = append(team.Members, entry.MemberDraft{Name: member.Name, Number: member.Number})
	}
	return out, nil
}

// saveProfile は参加者の前回の入力を上書きする(entry.ProfileUpdate)。
func saveProfile(ctx context.Context, tx *gorm.DB, lineUserID int64, update entry.ProfileUpdate, at time.Time) error {
	updates := map[string]any{"name": update.Name, "profile_updated_at": at, "updated_at": at}
	if update.Contact != nil {
		updates["phone"], updates["email"] = update.Contact.Phone, update.Contact.Email
	}
	if err := tx.WithContext(ctx).Model(&LineUser{}).Where(sqlByID, lineUserID).Updates(updates).Error; err != nil {
		return writeError("save profile of line user", err)
	}
	return nil
}

// Join は entry.Store.Join。申込の行をロックしてから(同じチームへの同時の登録と、運営の承認と送り
// 直しを並べる)、メンバーの行を決めて書き、答えを置き換える。背番号の重なりは、検査のあとに別の人が
// 登録したかもしれないので、ロックの中で読み直したメンバーで確かめ直す。承認した申込なら、その行の
// 選手も直す(無ければ作る)。登録した人の前回の入力(名前)も同じトランザクションで上書きする。
func (s *EntryStore) Join(ctx context.Context, join entry.Join) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		team, err := lockEntry(ctx, tx, join.TeamEntryID)
		if err != nil {
			return err
		}
		if err := entry.CanJoin(entry.Status(team.Status)); err != nil {
			return err
		}
		member, err := memberToJoin(ctx, tx, join)
		if err != nil {
			return err
		}
		members, err := loadMembers(ctx, tx, join.TeamEntryID)
		if err != nil {
			return err
		}
		if err := entry.CheckNumberFree(members, member.ID, join.Number); err != nil {
			return err
		}
		member.Name, member.Number, member.LineUserID = join.Name, join.Number, &join.LineUserID
		member.UpdatedAt = join.At
		if team.Status == string(entry.StatusApproved) && team.TeamID != nil {
			if err := syncPlayer(ctx, tx, *team.TeamID, &member); err != nil {
				return err
			}
		}
		if member.ID == 0 {
			err = tx.Create(&member).Error
		} else {
			err = tx.Save(&member).Error
		}
		if err != nil {
			return writeError("register member: the member is already registered", err)
		}
		if err := replaceMemberAnswers(ctx, tx, member.ID, join.Answers); err != nil {
			return err
		}
		return saveProfile(ctx, tx, join.LineUserID, join.Profile, join.At)
	})
}

// Resubmit は entry.Store.Resubmit。申込の行をロックしてから、メンバーの行の変え方をロックの中で読んだ
// メンバーから決める(検査のあとに登録したメンバーの行を消さないため)。チーム名を同じ部門の別の申込と
// 同じ名前に直したら、一意索引(team_entries_name_uniq)の違反が domain.ErrConflict になる。
func (s *EntryStore) Resubmit(ctx context.Context, id int64, resubmission entry.Resubmission) (entry.Detail, error) {
	draft := resubmission.Draft
	var out entry.Detail
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := lockEntry(ctx, tx, id)
		if err != nil {
			return err
		}
		if row.Status != string(entry.StatusReturned) {
			return fmt.Errorf("%w: the entry was changed by someone else; reload it", domain.ErrConflict)
		}
		members, err := loadMembers(ctx, tx, id)
		if err != nil {
			return err
		}
		if !entry.SameRegistered(members, resubmission.Registered) {
			return fmt.Errorf("%w: the members of the entry changed; reload it", domain.ErrConflict)
		}
		plan, err := entry.PlanMembers(members, draft.Members)
		if err != nil {
			return err
		}
		err = tx.Model(&TeamEntry{}).Where(sqlByID, id).Updates(map[string]any{
			"division_id": draft.DivisionID, "team_name_full": draft.FullName,
			"team_name_short": draft.ShortName, "rep_name": draft.RepName, "rep_phone": draft.RepPhone,
			"rep_email": draft.RepEmail, "status": string(entry.StatusPending), "updated_at": resubmission.At,
			// 申込中は運営がまだ決めていない状態なので、差し戻しの理由と決めた記録を空に戻す
			"return_note": "", "decided_at": nil, "decided_by": nil,
		}).Error
		if err != nil {
			return writeError("resubmit entry: a team with the same name has already applied", err)
		}
		if err := applyMemberPlan(ctx, tx, id, plan, resubmission.At); err != nil {
			return err
		}
		if err := replaceTeamAnswers(ctx, tx, id, resubmission.Answers); err != nil {
			return err
		}
		if err := saveProfile(ctx, tx, resubmission.RepLineUserID, resubmission.Profile, resubmission.At); err != nil {
			return err
		}
		out, err = loadEntry(ctx, tx, id)
		return err
	})
	return out, err
}

// SetScorer は entry.Store.SetScorer。
func (s *EntryStore) SetScorer(ctx context.Context, entryID int64, memberID *int64, at time.Time) (entry.Detail, error) {
	var out entry.Detail
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockEntry(ctx, tx, entryID); err != nil {
			return err
		}
		err := tx.Model(&TeamEntry{}).Where(sqlByID, entryID).
			Updates(map[string]any{"scorer_member_id": memberID, "updated_at": at}).Error
		if err != nil {
			return writeError("set scorer of entry", err)
		}
		out, err = loadEntry(ctx, tx, entryID)
		return err
	})
	return out, err
}

// applyMemberPlan はメンバーの行を plan のとおりにする。消す行の答えは外部キーの ON DELETE CASCADE で
// 消える。
func applyMemberPlan(ctx context.Context, tx *gorm.DB, teamEntryID int64, plan entry.MemberPlan, at time.Time) error {
	db := tx.WithContext(ctx)
	if len(plan.Remove) > 0 {
		if err := db.Where(sqlMembersOfEntry, plan.Remove, teamEntryID).Delete(&MemberEntry{}).Error; err != nil {
			return writeError("remove members of entry", err)
		}
	}
	for _, planned := range plan.Rows {
		if planned.ID == 0 {
			row := MemberEntry{TeamEntryID: teamEntryID, Name: planned.Name, Number: planned.Number,
				SortOrder: planned.SortOrder, CreatedAt: at, UpdatedAt: at}
			if err := db.Create(&row).Error; err != nil {
				return writeError("add member to entry", err)
			}
			continue
		}
		err := db.Model(&MemberEntry{}).Where(sqlByMemberOfEntry, planned.ID, teamEntryID).Updates(map[string]any{
			"name": planned.Name, "number": planned.Number, "sort_order": planned.SortOrder, "updated_at": at,
		}).Error
		if err != nil {
			return writeError("update member of entry", err)
		}
	}
	return nil
}

// memberToJoin は登録する行を決める。代表の行を選んだならその行(別の人が結んでいたら衝突)、もう
// 登録した人ならその人の行、どちらでもなければ最後に足す新しい行。
func memberToJoin(ctx context.Context, tx *gorm.DB, join entry.Join) (MemberEntry, error) {
	var mine []MemberEntry
	err := tx.WithContext(ctx).Where(sqlByLineUserOfEntry, join.TeamEntryID, join.LineUserID).Limit(1).
		Find(&mine).Error
	if err != nil {
		return MemberEntry{}, fmt.Errorf("find registered member: %w", err)
	}
	if join.MemberID != nil {
		var rows []MemberEntry
		err := tx.WithContext(ctx).Where(sqlByMemberOfEntry, *join.MemberID, join.TeamEntryID).Limit(1).
			Find(&rows).Error
		if err != nil {
			return MemberEntry{}, fmt.Errorf("find member %d: %w", *join.MemberID, err)
		}
		if len(rows) == 0 {
			return MemberEntry{}, domain.Invalid("memberId", "is not a member of this team")
		}
		picked := rows[0]
		taken := picked.LineUserID != nil && *picked.LineUserID != join.LineUserID
		if taken || (len(mine) > 0 && mine[0].ID != picked.ID) {
			return MemberEntry{}, fmt.Errorf("%w: someone else has already registered as this member",
				domain.ErrConflict)
		}
		return picked, nil
	}
	if len(mine) > 0 {
		return mine[0], nil
	}
	var last int
	err = tx.WithContext(ctx).Model(&MemberEntry{}).Select(sqlSelectMaxSort).
		Where(sqlByTeamEntry, join.TeamEntryID).Scan(&last).Error
	if err != nil {
		return MemberEntry{}, fmt.Errorf("find the last member of entry %d: %w", join.TeamEntryID, err)
	}
	return MemberEntry{TeamEntryID: join.TeamEntryID, SortOrder: last + 1}, nil
}

// syncPlayer は承認した申込のメンバー登録を選手に写す。選手があれば名前と背番号を直し(英語名が
// 日本語名と同じなら英語名も直す。承認のときに日本語名を写したもの)、無ければ作る。
func syncPlayer(ctx context.Context, tx *gorm.DB, teamID int64, member *MemberEntry) error {
	if member.PlayerID == nil {
		created, err := CreatePlayer(ctx, tx, teamID, PlayerFields{Number: member.Number,
			Name: domain.I18nText{Ja: member.Name}})
		if err != nil {
			return err
		}
		member.PlayerID = &created.ID
		return nil
	}
	player, err := findByID[Player](ctx, tx, *member.PlayerID, domain.ResourcePlayer)
	if err != nil {
		return err
	}
	if player.NameEn == player.NameJa {
		player.NameEn = member.Name
	}
	player.NameJa, player.Number = member.Name, member.Number
	if err := tx.WithContext(ctx).Save(&player).Error; err != nil {
		return writeError("update player of member", err)
	}
	return nil
}

// replaceTeamAnswers は申込(代表が答えるチーム向けの質問)の答えを置き換える。
func replaceTeamAnswers(ctx context.Context, tx *gorm.DB, teamEntryID int64, answers []entry.Answer) error {
	if err := tx.WithContext(ctx).Where(sqlByTeamEntry, teamEntryID).Delete(&EntryAnswer{}).Error; err != nil {
		return writeError("delete entry answers", err)
	}
	rows := make([]EntryAnswer, len(answers))
	for i, answer := range answers {
		rows[i] = EntryAnswer{QuestionID: answer.QuestionID, TeamEntryID: &teamEntryID, Value: answer.Value}
	}
	return createAnswers(ctx, tx, rows)
}

// replaceMemberAnswers はメンバーエントリーの答えを置き換える。
func replaceMemberAnswers(ctx context.Context, tx *gorm.DB, memberID int64, answers []entry.Answer) error {
	if err := tx.WithContext(ctx).Where(sqlByMemberEntry, memberID).Delete(&EntryAnswer{}).Error; err != nil {
		return writeError("delete member answers", err)
	}
	rows := make([]EntryAnswer, len(answers))
	for i, answer := range answers {
		rows[i] = EntryAnswer{QuestionID: answer.QuestionID, MemberEntryID: &memberID, Value: answer.Value}
	}
	return createAnswers(ctx, tx, rows)
}

func createAnswers(ctx context.Context, tx *gorm.DB, rows []EntryAnswer) error {
	if len(rows) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
		return writeError("create answers", err)
	}
	return nil
}
