package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// TeamFields はチームの編集できる値。正式名は加工せずに保存する(後で大会をまたいで束ねるため。
// docs/line-integration.md)。FinalRank はディビジョンの最終順位(順位決定戦の結果。計算しない)。
type TeamFields struct {
	FullName  domain.I18nText
	ShortName domain.I18nText
	Comment   domain.I18nText
	BarStyle  domain.TeamBarStyle
	SortOrder int
	FinalRank *int
}

// TeamRecord は管理画面のチーム。
type TeamRecord struct {
	ID           int64
	TournamentID int64
	DivisionID   int64
	TeamFields
}

func (f TeamFields) validate() error {
	if err := domain.CheckRequired("fullName.ja", f.FullName.Ja); err != nil {
		return err
	}
	if err := domain.CheckRequired("shortName.ja", f.ShortName.Ja); err != nil {
		return err
	}
	switch f.BarStyle {
	case "default", "mixed", "women", "univ", "waku1", "waku2":
	default:
		return domain.Invalid("barStyle", "must be default, mixed, women, univ, waku1 or waku2")
	}
	if f.FinalRank != nil && *f.FinalRank <= 0 {
		return domain.Invalid("finalRank", "must be positive")
	}
	return nil
}

func (f TeamFields) apply(row *Team) {
	row.NameFullJa, row.NameFullEn = f.FullName.Ja, f.FullName.En
	row.NameShortJa, row.NameShortEn = f.ShortName.Ja, f.ShortName.En
	row.CommentJa, row.CommentEn = f.Comment.Ja, f.Comment.En
	row.BarStyle, row.SortOrder, row.FinalRank = f.BarStyle, f.SortOrder, f.FinalRank
}

func toTeamRecord(row Team) TeamRecord {
	return TeamRecord{
		ID: row.ID, TournamentID: row.TournamentID, DivisionID: row.DivisionID,
		TeamFields: TeamFields{
			FullName:  i18n(row.NameFullJa, row.NameFullEn),
			ShortName: i18n(row.NameShortJa, row.NameShortEn),
			Comment:   i18n(row.CommentJa, row.CommentEn),
			BarStyle:  row.BarStyle, SortOrder: row.SortOrder, FinalRank: row.FinalRank,
		},
	}
}

// ListTeams はディビジョンのチームを sort_order, id 順で返す。
func ListTeams(ctx context.Context, db *gorm.DB, divisionID int64) ([]TeamRecord, error) {
	if _, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision); err != nil {
		return nil, err
	}
	var rows []Team
	err := db.WithContext(ctx).
		Where(sqlByDivision, divisionID).
		Order(sqlOrderSortID).
		Find(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("list teams of division %d: %w", divisionID, err)
	}
	out := make([]TeamRecord, len(rows))
	for i, row := range rows {
		out[i] = toTeamRecord(row)
	}
	return out, nil
}

// GetAdminTeam はチームを返す。無ければ ENTRY_TEAM_ID の NotFound。
func GetAdminTeam(ctx context.Context, db *gorm.DB, id int64) (TeamRecord, error) {
	row, err := findByID[Team](ctx, db, id, domain.ResourceTeam)
	if err != nil {
		return TeamRecord{}, err
	}
	return toTeamRecord(row), nil
}

// CreateTeam はディビジョンにチームを作る。英語の正式名と短縮名が空なら日本語の値を入れる
// (domain.TeamNamesWithEnglish。英語の閲覧ページでチーム名が空にならないように)。
func CreateTeam(
	ctx context.Context,
	db *gorm.DB,
	divisionID int64,
	fields TeamFields,
) (TeamRecord, error) {
	if err := fields.validate(); err != nil {
		return TeamRecord{}, err
	}
	division, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision)
	if err != nil {
		return TeamRecord{}, err
	}
	row := Team{TournamentID: division.TournamentID, DivisionID: division.ID}
	fields.FullName, fields.ShortName = domain.TeamNamesWithEnglish(fields.FullName, fields.ShortName)
	fields.apply(&row)
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return TeamRecord{}, writeError("create team", err)
	}
	return toTeamRecord(row), nil
}

// UpdateTeam はチームの値を fields で置き換える。
func UpdateTeam(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	id int64,
	fields TeamFields,
) (TeamRecord, error) {
	if err := fields.validate(); err != nil {
		return TeamRecord{}, err
	}
	var row Team
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		row, err = findByID[Team](ctx, tx, id, domain.ResourceTeam)
		if err != nil {
			return err
		}
		before := row.FinalRank
		fields.apply(&row)
		if err := tx.Save(&row).Error; err != nil {
			return writeError("update team", err)
		}
		return auditFinalRank(ctx, tx, actor, row, before)
	})
	if err != nil {
		return TeamRecord{}, err
	}
	return toTeamRecord(row), nil
}

// DeleteTeam はチームと選手を消す。試合の対戦チームは NULL になる(ON DELETE SET NULL)。
func DeleteTeam(ctx context.Context, db *gorm.DB, id int64) error {
	return deleteByID[Team](ctx, db, id, domain.ResourceTeam)
}

// FinalRank はディビジョンの最終順位の 1 件。Rank が nil なら順位を消す。
type FinalRank struct {
	TeamID int64
	Rank   *int
}

// SetFinalRanks はディビジョンのチームの最終順位(teams.final_rank)をまとめて書く。ranks に無い
// チームは変えない。同順位(5、5、7、7)を許す(clone-spec C 1)。[最終順位] の公開は
// divisions.standings_published で別に切り替える。
func SetFinalRanks(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	divisionID int64,
	ranks []FinalRank,
) ([]TeamRecord, error) {
	for i, rank := range ranks {
		if rank.Rank != nil && *rank.Rank <= 0 {
			return nil, domain.Invalid(fmt.Sprintf("[%d].rank", i), "must be positive")
		}
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findByID[Division](ctx, tx, divisionID, domain.ResourceDivision); err != nil {
			return err
		}
		for i, rank := range ranks {
			var teams []Team
			if err := tx.Where(sqlTeamInDivision, rank.TeamID, divisionID).Limit(1).
				Find(&teams).Error; err != nil {
				return fmt.Errorf("load team %d: %w", rank.TeamID, err)
			}
			if len(teams) == 0 {
				return domain.Invalid(
					fmt.Sprintf("[%d].teamId", i),
					"is not a team of the division",
				)
			}
			team := teams[0]
			before := team.FinalRank
			err := tx.Model(&Team{}).Where(sqlByID, team.ID).Update("final_rank", rank.Rank).Error
			if err != nil {
				return writeError("set final rank", err)
			}
			team.FinalRank = rank.Rank
			if err := auditFinalRank(ctx, tx, actor, team, before); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ListTeams(ctx, db, divisionID)
}

const sqlTeamInDivision = "id = ? AND division_id = ?"

// auditFinalRank は最終順位が変わったときだけ監査ログに残す(docs/format-engine.md「監査ログ」)。
func auditFinalRank(
	ctx context.Context,
	tx *gorm.DB,
	actor domain.Actor,
	team Team,
	before *int,
) error {
	if sameRank(before, team.FinalRank) {
		return nil
	}
	id := team.ID
	return writeAudit(ctx, tx, auditEntry{
		tournamentID: team.TournamentID,
		teamID:       &id,
		actor:        actor,
		action:       AuditFinalRank,
		before:       AuditSnapshot{FinalRank: before},
		after:        AuditSnapshot{FinalRank: team.FinalRank},
	})
}

func sameRank(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// PlayerFields は選手(またはスタッフ)の編集できる値。Role があればスタッフ。
type PlayerFields struct {
	Number string
	Name   domain.I18nText
	Gender *domain.Gender
	Role   *domain.I18nText
}

// PlayerRecord は管理画面の選手。
type PlayerRecord struct {
	ID     int64
	TeamID int64
	PlayerFields
}

func (f PlayerFields) validate() error {
	if err := domain.CheckRequired("name.ja", f.Name.Ja); err != nil {
		return err
	}
	if f.Gender != nil && *f.Gender != "M" && *f.Gender != "F" {
		return domain.Invalid("gender", "must be M, F or null")
	}
	if f.Role != nil {
		return domain.CheckRequired("role.ja", f.Role.Ja)
	}
	return nil
}

func (f PlayerFields) apply(row *Player) {
	row.Number, row.NameJa, row.NameEn = f.Number, f.Name.Ja, f.Name.En
	row.Gender, row.RoleJa, row.RoleEn = nil, nil, nil
	if f.Gender != nil {
		gender := string(*f.Gender)
		row.Gender = &gender
	}
	if f.Role != nil {
		ja, en := f.Role.Ja, f.Role.En
		row.RoleJa, row.RoleEn = &ja, &en
	}
}

func toPlayerRecord(row Player) PlayerRecord {
	record := PlayerRecord{ID: row.ID, TeamID: row.TeamID, PlayerFields: PlayerFields{
		Number: row.Number, Name: i18n(row.NameJa, row.NameEn), Role: playerRole(row),
	}}
	if row.Gender != nil {
		gender := domain.Gender(*row.Gender)
		record.Gender = &gender
	}
	return record
}

// ListPlayers はチームの選手とスタッフを ID 順で返す。
func ListPlayers(ctx context.Context, db *gorm.DB, teamID int64) ([]PlayerRecord, error) {
	if _, err := findByID[Team](ctx, db, teamID, domain.ResourceTeam); err != nil {
		return nil, err
	}
	return listPlayersOf(ctx, db, teamID)
}

func listPlayersOf(ctx context.Context, db *gorm.DB, teamID int64) ([]PlayerRecord, error) {
	var rows []Player
	if err := db.WithContext(ctx).
		Where(sqlByTeam, teamID).
		Order(sqlOrderID).
		Find(&rows).
		Error; err != nil {
		return nil, fmt.Errorf("list players of team %d: %w", teamID, err)
	}
	out := make([]PlayerRecord, len(rows))
	for i, row := range rows {
		out[i] = toPlayerRecord(row)
	}
	return out, nil
}

// GetAdminPlayer は選手を返す。無ければ ENTRY_PLAYER_ID の NotFound。
func GetAdminPlayer(ctx context.Context, db *gorm.DB, id int64) (PlayerRecord, error) {
	row, err := findByID[Player](ctx, db, id, domain.ResourcePlayer)
	if err != nil {
		return PlayerRecord{}, err
	}
	return toPlayerRecord(row), nil
}

// CreatePlayer はチームに選手を作る。英語の名前が空なら日本語の名前を入れる(CreateTeam と同じ)。
func CreatePlayer(
	ctx context.Context,
	db *gorm.DB,
	teamID int64,
	fields PlayerFields,
) (PlayerRecord, error) {
	if err := fields.validate(); err != nil {
		return PlayerRecord{}, err
	}
	if _, err := findByID[Team](ctx, db, teamID, domain.ResourceTeam); err != nil {
		return PlayerRecord{}, err
	}
	row := Player{TeamID: teamID}
	fields.Name = fields.Name.WithEnglish()
	fields.apply(&row)
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return PlayerRecord{}, writeError("create player", err)
	}
	return toPlayerRecord(row), nil
}

// UpdatePlayer は選手の値を fields で置き換える。
func UpdatePlayer(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	fields PlayerFields,
) (PlayerRecord, error) {
	if err := fields.validate(); err != nil {
		return PlayerRecord{}, err
	}
	row, err := findByID[Player](ctx, db, id, domain.ResourcePlayer)
	if err != nil {
		return PlayerRecord{}, err
	}
	fields.apply(&row)
	if err := db.WithContext(ctx).Save(&row).Error; err != nil {
		return PlayerRecord{}, writeError("update player", err)
	}
	return toPlayerRecord(row), nil
}

// DeletePlayer は選手を消す。得点経過の得点者とアシストは NULL になる(ON DELETE SET NULL)。
func DeletePlayer(ctx context.Context, db *gorm.DB, id int64) error {
	return deleteByID[Player](ctx, db, id, domain.ResourcePlayer)
}
