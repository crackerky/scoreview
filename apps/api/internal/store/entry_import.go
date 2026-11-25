package store

import (
	"context"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

// 前の大会の申込を移す(entry.PastTournament)。決まりと検査は entry にある。

const (
	sqlByLineUsers = "line_user_id IN ?"
	// 台帳の名前と連絡先は、移す申込のほうが新しいときだけ書き換える(こちらで申し込み直した人の
	// 入力を、前の大会の古い入力で戻さない)
	sqlProfileIsOlder = "line_users.profile_updated_at IS NULL OR line_users.profile_updated_at < excluded.profile_updated_at"
	// メールアドレスはメンバーの行には無いので、空なら前の値を残す
	sqlKeepEmail    = "COALESCE(NULLIF(excluded.email, ''), line_users.email)"
	importBatchSize = 500
)

// pastProfile は参加者の台帳に書く、1 人の最も新しい入力。
type pastProfile struct {
	name  string
	email string
	at    time.Time
}

// ImportPast は前の大会を、終わった、一覧に載せない大会として作り、部門、承認済みの申込(LINE から
// 受けた申込として、代表とメンバーの LINE を結ぶ)、参加者の台帳を 1 つのトランザクションで書く。
// 作った大会の ID を返す。検査(entry.CheckPast)は済ませてある。
func (s *EntryStore) ImportPast(ctx context.Context, past entry.PastTournament) (int64, error) {
	var tournamentID int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fields := TournamentFields{Name: domain.I18nText{Ja: past.Name}, ShortName: past.Name,
			Status: domain.TournamentFinished, TeamListLabel: "teams", GroupLabel: "groups"}
		if past.StartDate != nil {
			day := past.StartDate.Format(time.DateOnly)
			fields.StartDate = &day
		}
		tournament, err := CreateTournament(ctx, tx, fields)
		if err != nil {
			return err
		}
		tournamentID = tournament.ID
		divisions := make([]int64, len(past.Divisions))
		for i, division := range past.Divisions {
			created, err := CreateDivision(ctx, tx, tournamentID, DivisionFields{
				Name: domain.I18nText{Ja: division.Name}, Kind: division.Kind, SortOrder: i + 1,
				EntryKind: domain.EntryTeam, StatsKind: domain.StatsUltimate})
			if err != nil {
				return err
			}
			divisions[i] = created.ID
		}
		lineUsers, err := importLineUsers(ctx, tx, past)
		if err != nil {
			return err
		}
		return importEntries(ctx, tx, tournamentID, divisions, lineUsers, past.Entries)
	})
	if err != nil {
		return 0, err
	}
	return tournamentID, nil
}

// pastProfiles は移す申込に出てくる人ごとの、最も新しい名前とメールアドレス。
func pastProfiles(past entry.PastTournament) map[string]pastProfile {
	profiles := map[string]pastProfile{}
	keep := func(lineID, name, email string, at time.Time) {
		current, ok := profiles[lineID]
		if !ok || current.at.Before(at) {
			if email == "" {
				email = current.email
			}
			profiles[lineID] = pastProfile{name: name, email: email, at: at}
		} else if current.email == "" && email != "" {
			current.email = email
			profiles[lineID] = current
		}
	}
	for _, pastEntry := range past.Entries {
		keep(pastEntry.RepLineID, pastEntry.RepName, pastEntry.RepEmail, pastEntry.SubmittedAt)
		for _, member := range pastEntry.Members {
			if member.LineID != "" {
				keep(member.LineID, member.Name, "", member.SubmittedAt)
			}
		}
	}
	return profiles
}

// importLineUsers は参加者の台帳に人を足し(友だちかどうかは分からないので、友だちとして作る。切り
// 替えた後の Webhook で直る)、名前と連絡先を新しければ書き換えて、ユーザー ID ごとの行の ID を返す。
func importLineUsers(ctx context.Context, tx *gorm.DB, past entry.PastTournament) (map[string]int64, error) {
	profiles := pastProfiles(past)
	lineIDs := make([]string, 0, len(profiles))
	for lineID := range profiles {
		lineIDs = append(lineIDs, lineID)
	}
	slices.Sort(lineIDs)
	rows := make([]LineUser, len(lineIDs))
	for i, lineID := range lineIDs {
		profile := profiles[lineID]
		at := profile.at
		rows[i] = LineUser{LineUserID: lineID, Friend: true, Name: profile.name, Email: profile.email,
			ProfileUpdatedAt: &at, CreatedAt: at, UpdatedAt: at}
	}
	update := clause.OnConflict{
		Columns: []clause.Column{{Name: "line_user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"name":               gorm.Expr("excluded.name"),
			"email":              gorm.Expr(sqlKeepEmail),
			"profile_updated_at": gorm.Expr("excluded.profile_updated_at"),
		}),
		Where: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: sqlProfileIsOlder}}},
	}
	if len(rows) > 0 {
		if err := tx.WithContext(ctx).Clauses(update).CreateInBatches(rows, importBatchSize).Error; err != nil {
			return nil, writeError("import line users", err)
		}
	}
	var found []LineUser
	err := tx.WithContext(ctx).Select("id", "line_user_id").Where(sqlByLineUsers, lineIDs).Find(&found).Error
	if err != nil {
		return nil, fmt.Errorf("read imported line users: %w", err)
	}
	out := make(map[string]int64, len(found))
	for _, row := range found {
		out[row.LineUserID] = row.ID
	}
	return out, nil
}

// importEntries は承認済みの申込とメンバーを書く。承認の時刻は申込の時刻にする(移した申込に、
// 運営が決めた時刻の記録は無い)。チームと選手は作らない(移した大会は試合を持たない)。
func importEntries(
	ctx context.Context,
	tx *gorm.DB,
	tournamentID int64,
	divisions []int64,
	lineUsers map[string]int64,
	entries []entry.PastEntry,
) error {
	for _, pastEntry := range entries {
		rep := lineUsers[pastEntry.RepLineID]
		decided := pastEntry.SubmittedAt
		row := TeamEntry{TournamentID: tournamentID, DivisionID: divisions[pastEntry.Division],
			TeamNameFull: pastEntry.FullName, TeamNameShort: pastEntry.FullName, RepName: pastEntry.RepName,
			RepEmail: pastEntry.RepEmail, RepLineUserID: &rep, Channel: string(entry.ChannelLine),
			Status: string(entry.StatusApproved), InviteCode: pastEntry.InviteCode,
			SubmittedAt: pastEntry.SubmittedAt, DecidedAt: &decided, CreatedAt: decided, UpdatedAt: decided}
		if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
			return writeError("import entry "+pastEntry.FullName, err)
		}
		if len(pastEntry.Members) == 0 {
			continue
		}
		members := make([]MemberEntry, len(pastEntry.Members))
		for i, member := range pastEntry.Members {
			members[i] = MemberEntry{TeamEntryID: row.ID, Name: member.Name, Number: member.Number,
				SortOrder: i + 1, CreatedAt: member.SubmittedAt, UpdatedAt: member.SubmittedAt}
			if member.LineID != "" {
				id := lineUsers[member.LineID]
				members[i].LineUserID = &id
			}
		}
		if err := tx.WithContext(ctx).CreateInBatches(members, importBatchSize).Error; err != nil {
			return writeError("import members of "+pastEntry.FullName, err)
		}
	}
	return nil
}
