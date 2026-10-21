package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// 管理画面の大会、バナー、ディビジョン、ギャラリーの読み書き。*Fields は編集できる値で、*Record は
// それに ID と親の ID を足したもの。どちらも GORM のモデルとは別の型にして、モデルを外に出さない。

// TournamentFields は大会の編集できる値。StartDate は "YYYY-MM-DD"。TopHTML は無害化済みの HTML
// (httpapi が sanitize してから渡す)。
type TournamentFields struct {
	Name          domain.I18nText
	ShortName     string
	Status        domain.TournamentStatus
	Listed        bool
	StartDate     *string
	LogoURL       *string
	TopHTML       domain.I18nText
	TeamListLabel string // teams | entries
	GroupLabel    string // groups | results
	ShowSchedule  bool
	// ForfeitScore は不戦勝の規定スコア。nil は未設定(運営会社に未確認なので既定値を持たない)
	ForfeitScore *domain.ForfeitScore
}

// TournamentRecord は管理画面の大会。
type TournamentRecord struct {
	ID int64
	TournamentFields
}

func (f TournamentFields) validate() error {
	if err := domain.CheckRequired("name.ja", f.Name.Ja); err != nil {
		return err
	}
	switch f.Status {
	case domain.TournamentEntryOpen, domain.TournamentEntryClosed, domain.TournamentFinished:
	default:
		return domain.Invalid("status", "must be entry_open, entry_closed or finished")
	}
	if f.TeamListLabel != "teams" && f.TeamListLabel != "entries" {
		return domain.Invalid("labels.teamList", "must be teams or entries")
	}
	if f.GroupLabel != "groups" && f.GroupLabel != "results" {
		return domain.Invalid("labels.groups", "must be groups or results")
	}
	if f.ForfeitScore != nil {
		if err := f.ForfeitScore.Validate("forfeitScore"); err != nil {
			return err
		}
	}
	return domain.CheckOptionalURL("logoUrl", f.LogoURL)
}

func (f TournamentFields) apply(row *Tournament) error {
	row.NameJa, row.NameEn, row.ShortName = f.Name.Ja, f.Name.En, f.ShortName
	row.Status, row.Listed, row.LogoURL = f.Status, f.Listed, f.LogoURL
	row.TopHTMLJa, row.TopHTMLEn = f.TopHTML.Ja, f.TopHTML.En
	row.TeamListLabel, row.GroupLabel, row.ShowSchedule = f.TeamListLabel, f.GroupLabel, f.ShowSchedule
	row.ForfeitWinnerScore, row.ForfeitLoserScore = nil, nil
	if f.ForfeitScore != nil {
		winner, loser := f.ForfeitScore.Winner, f.ForfeitScore.Loser
		row.ForfeitWinnerScore, row.ForfeitLoserScore = &winner, &loser
	}
	row.StartDate = nil
	if f.StartDate != nil {
		day, err := domain.ParseDay("startDate", *f.StartDate)
		if err != nil {
			return err
		}
		row.StartDate = &day
	}
	return nil
}

func toTournamentRecord(row Tournament) TournamentRecord {
	var startDate *string
	if row.StartDate != nil {
		value := row.StartDate.Format(time.DateOnly)
		startDate = &value
	}
	return TournamentRecord{ID: row.ID, TournamentFields: TournamentFields{
		Name:          i18n(row.NameJa, row.NameEn),
		ShortName:     row.ShortName,
		Status:        row.Status,
		Listed:        row.Listed,
		StartDate:     startDate,
		LogoURL:       row.LogoURL,
		TopHTML:       i18n(row.TopHTMLJa, row.TopHTMLEn),
		TeamListLabel: row.TeamListLabel,
		GroupLabel:    row.GroupLabel,
		ShowSchedule:  row.ShowSchedule,
		ForfeitScore:  tournamentForfeitScore(row),
	}}
}

// tournamentForfeitScore は大会の不戦勝の規定スコア。未設定なら nil。
func tournamentForfeitScore(row Tournament) *domain.ForfeitScore {
	if row.ForfeitWinnerScore == nil || row.ForfeitLoserScore == nil {
		return nil
	}
	return &domain.ForfeitScore{Winner: *row.ForfeitWinnerScore, Loser: *row.ForfeitLoserScore}
}

// ListAdminTournaments は大会を開催日の新しい順(大会一覧と同じ並び)で返す。onlyIDs が nil なら全大会、
// nil でなければその ID の大会だけ(記録係の担当大会)。
func ListAdminTournaments(
	ctx context.Context,
	db *gorm.DB,
	onlyIDs []int64,
) ([]TournamentRecord, error) {
	query := db.WithContext(ctx).Order(sqlOrderTournamentList)
	if onlyIDs != nil {
		if len(onlyIDs) == 0 {
			return []TournamentRecord{}, nil
		}
		query = query.Where(sqlByIDs, onlyIDs)
	}
	var rows []Tournament
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list tournaments for admin: %w", err)
	}
	out := make([]TournamentRecord, len(rows))
	for i, row := range rows {
		out[i] = toTournamentRecord(row)
	}
	return out, nil
}

// GetAdminTournament は大会を返す。無ければ TOURNAMENT_ID の NotFound。
func GetAdminTournament(ctx context.Context, db *gorm.DB, id int64) (TournamentRecord, error) {
	row, err := findByID[Tournament](ctx, db, id, domain.ResourceTournament)
	if err != nil {
		return TournamentRecord{}, err
	}
	return toTournamentRecord(row), nil
}

// CreateTournament は大会を作る。
func CreateTournament(
	ctx context.Context,
	db *gorm.DB,
	fields TournamentFields,
) (TournamentRecord, error) {
	if err := fields.validate(); err != nil {
		return TournamentRecord{}, err
	}
	var row Tournament
	if err := fields.apply(&row); err != nil {
		return TournamentRecord{}, err
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return TournamentRecord{}, writeError("create tournament", err)
	}
	return toTournamentRecord(row), nil
}

// UpdateTournament は大会の値を fields で置き換える。
func UpdateTournament(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	fields TournamentFields,
) (TournamentRecord, error) {
	if err := fields.validate(); err != nil {
		return TournamentRecord{}, err
	}
	row, err := findByID[Tournament](ctx, db, id, domain.ResourceTournament)
	if err != nil {
		return TournamentRecord{}, err
	}
	if err := fields.apply(&row); err != nil {
		return TournamentRecord{}, err
	}
	if err := db.WithContext(ctx).Save(&row).Error; err != nil {
		return TournamentRecord{}, writeError("update tournament", err)
	}
	return toTournamentRecord(row), nil
}

// DeleteTournament は大会と配下の全データ(ON DELETE CASCADE)を消す。
func DeleteTournament(ctx context.Context, db *gorm.DB, id int64) error {
	return deleteByID[Tournament](ctx, db, id, domain.ResourceTournament)
}

// BannerFields はサイドバーのバナー。
type BannerFields struct {
	ImageURL string
	LinkURL  *string
}

// ListBanners は大会のバナーを表示順で返す。
func ListBanners(ctx context.Context, db *gorm.DB, tournamentID int64) ([]BannerFields, error) {
	if _, err := findByID[Tournament](
		ctx,
		db,
		tournamentID,
		domain.ResourceTournament,
	); err != nil {
		return nil, err
	}
	var rows []TournamentBanner
	err := db.WithContext(ctx).
		Where(sqlByTournament, tournamentID).
		Order(sqlOrderSortID).
		Find(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("list banners of tournament %d: %w", tournamentID, err)
	}
	out := make([]BannerFields, len(rows))
	for i, row := range rows {
		out[i] = BannerFields{ImageURL: row.ImageURL, LinkURL: row.LinkURL}
	}
	return out, nil
}

// maxBanners はバナーの上限。参照サイトの大会で観測した最大(30 枚。clone-spec A 2)。
const maxBanners = 30

// ReplaceBanners は大会のバナーを banners(この順で表示)で置き換える。
func ReplaceBanners(
	ctx context.Context,
	db *gorm.DB,
	tournamentID int64,
	banners []BannerFields,
) ([]BannerFields, error) {
	if len(banners) > maxBanners {
		return nil, domain.Invalid("banners", fmt.Sprintf("at most %d", maxBanners))
	}
	for i, banner := range banners {
		if err := domain.CheckURL(
			fmt.Sprintf("banners[%d].imageUrl", i),
			banner.ImageURL,
		); err != nil {
			return nil, err
		}
		if err := domain.CheckOptionalURL(
			fmt.Sprintf("banners[%d].linkUrl", i),
			banner.LinkURL,
		); err != nil {
			return nil, err
		}
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findByID[Tournament](
			ctx,
			tx,
			tournamentID,
			domain.ResourceTournament,
		); err != nil {
			return err
		}
		if err := tx.Where(sqlByTournament, tournamentID).
			Delete(&TournamentBanner{}).
			Error; err != nil {
			return fmt.Errorf("clear banners: %w", err)
		}
		if len(banners) == 0 {
			return nil
		}
		rows := make([]TournamentBanner, len(banners))
		for i, banner := range banners {
			rows[i] = TournamentBanner{
				TournamentID: tournamentID,
				ImageURL:     banner.ImageURL,
				LinkURL:      banner.LinkURL,
				SortOrder:    i,
			}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return writeError("create banners", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ListBanners(ctx, db, tournamentID)
}
