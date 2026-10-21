package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// DivisionFields はディビジョンの編集できる値。
type DivisionFields struct {
	Name               domain.I18nText
	Kind               domain.DivisionKind
	SortOrder          int
	ScheduleEnabled    bool
	StandingsPublished bool
	EntryKind          domain.EntryKind
	StatsKind          domain.StatsKind
}

// DivisionRecord は管理画面のディビジョン。
type DivisionRecord struct {
	ID           int64
	TournamentID int64
	DivisionFields
}

func (f DivisionFields) validate() error {
	if err := domain.CheckRequired("name.ja", f.Name.Ja); err != nil {
		return err
	}
	switch f.Kind {
	case domain.DivisionMen, domain.DivisionWomen, domain.DivisionMixed, domain.DivisionNone:
	default:
		return domain.Invalid("kind", "must be men, women, mixed or none")
	}
	if f.EntryKind != domain.EntryTeam && f.EntryKind != domain.EntryIndividual {
		return domain.Invalid("entryKind", "must be team or individual")
	}
	if f.StatsKind != domain.StatsUltimate && f.StatsKind != domain.StatsDiscgolf {
		return domain.Invalid("statsKind", "must be ultimate or discgolf")
	}
	return nil
}

func (f DivisionFields) apply(row *Division) {
	row.NameJa, row.NameEn, row.Kind, row.SortOrder = f.Name.Ja, f.Name.En, f.Kind, f.SortOrder
	row.ScheduleEnabled, row.StandingsPublished = f.ScheduleEnabled, f.StandingsPublished
	row.EntryKind, row.StatsKind = f.EntryKind, f.StatsKind
}

func toDivisionRecord(row Division) DivisionRecord {
	return DivisionRecord{
		ID:           row.ID,
		TournamentID: row.TournamentID,
		DivisionFields: DivisionFields{
			Name: i18n(row.NameJa, row.NameEn), Kind: row.Kind, SortOrder: row.SortOrder,
			ScheduleEnabled: row.ScheduleEnabled, StandingsPublished: row.StandingsPublished,
			EntryKind: row.EntryKind, StatsKind: row.StatsKind,
		},
	}
}

// ListDivisions は大会のディビジョンを sort_order, id 順で返す。大会が無ければ NotFound。
func ListDivisions(ctx context.Context, db *gorm.DB, tournamentID int64) ([]DivisionRecord, error) {
	if _, err := findByID[Tournament](
		ctx,
		db,
		tournamentID,
		domain.ResourceTournament,
	); err != nil {
		return nil, err
	}
	var rows []Division
	err := db.WithContext(ctx).
		Where(sqlByTournament, tournamentID).
		Order(sqlOrderSortID).
		Find(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("list divisions of tournament %d: %w", tournamentID, err)
	}
	out := make([]DivisionRecord, len(rows))
	for i, row := range rows {
		out[i] = toDivisionRecord(row)
	}
	return out, nil
}

// GetDivision はディビジョンを返す。無ければ EVENT_DIVISION_ID の NotFound。
func GetDivision(ctx context.Context, db *gorm.DB, id int64) (DivisionRecord, error) {
	row, err := findByID[Division](ctx, db, id, domain.ResourceDivision)
	if err != nil {
		return DivisionRecord{}, err
	}
	return toDivisionRecord(row), nil
}

// CreateDivision は大会にディビジョンを作る。
func CreateDivision(
	ctx context.Context,
	db *gorm.DB,
	tournamentID int64,
	fields DivisionFields,
) (DivisionRecord, error) {
	if err := fields.validate(); err != nil {
		return DivisionRecord{}, err
	}
	if _, err := findByID[Tournament](
		ctx,
		db,
		tournamentID,
		domain.ResourceTournament,
	); err != nil {
		return DivisionRecord{}, err
	}
	row := Division{TournamentID: tournamentID}
	fields.apply(&row)
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return DivisionRecord{}, writeError("create division", err)
	}
	return toDivisionRecord(row), nil
}

// UpdateDivision はディビジョンの値を fields で置き換える。
func UpdateDivision(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	fields DivisionFields,
) (DivisionRecord, error) {
	if err := fields.validate(); err != nil {
		return DivisionRecord{}, err
	}
	row, err := findByID[Division](ctx, db, id, domain.ResourceDivision)
	if err != nil {
		return DivisionRecord{}, err
	}
	fields.apply(&row)
	if err := db.WithContext(ctx).Save(&row).Error; err != nil {
		return DivisionRecord{}, writeError("update division", err)
	}
	return toDivisionRecord(row), nil
}

// DeleteDivision はディビジョンと配下のグループ、チーム、試合を消す。
func DeleteDivision(ctx context.Context, db *gorm.DB, id int64) error {
	return deleteByID[Division](ctx, db, id, domain.ResourceDivision)
}

// TournamentOfDivision はディビジョンの大会 ID を返す(記録係の権限の判定に使う)。
func TournamentOfDivision(ctx context.Context, db *gorm.DB, id int64) (int64, error) {
	row, err := findByID[Division](ctx, db, id, domain.ResourceDivision)
	if err != nil {
		return 0, err
	}
	return row.TournamentID, nil
}

// TournamentOfMatch は試合の大会 ID を返す(記録係の権限の判定に使う)。
func TournamentOfMatch(ctx context.Context, db *gorm.DB, id int64) (int64, error) {
	match, err := findByID[Match](ctx, db, id, domain.ResourceMatch)
	if err != nil {
		return 0, err
	}
	division, err := findInternal[Division](ctx, db, match.DivisionID, "division")
	if err != nil {
		return 0, err
	}
	return division.TournamentID, nil
}

// GalleryFields はギャラリーの編集できる値。
type GalleryFields struct {
	Name      string
	SortOrder int
}

// PhotoFields はギャラリーの写真。
type PhotoFields struct {
	ThumbURL string
	FullURL  string
}

// GalleryRecord は管理画面のギャラリー。Photos は表示順。
type GalleryRecord struct {
	ID           int64
	TournamentID int64
	GalleryFields
	Photos []PhotoFields
}

func (f GalleryFields) validate() error { return domain.CheckRequired("name", f.Name) }

// ListGalleries は大会のギャラリーを写真つきで返す。
func ListGalleries(ctx context.Context, db *gorm.DB, tournamentID int64) ([]GalleryRecord, error) {
	if _, err := findByID[Tournament](
		ctx,
		db,
		tournamentID,
		domain.ResourceTournament,
	); err != nil {
		return nil, err
	}
	var rows []Gallery
	err := db.WithContext(ctx).
		Where(sqlByTournament, tournamentID).
		Order(sqlOrderSortID).
		Find(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("list galleries of tournament %d: %w", tournamentID, err)
	}
	out := make([]GalleryRecord, 0, len(rows))
	for _, row := range rows {
		record, err := toGalleryRecord(ctx, db, row)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

func toGalleryRecord(ctx context.Context, db *gorm.DB, row Gallery) (GalleryRecord, error) {
	var photos []GalleryPhoto
	err := db.WithContext(ctx).Where(sqlByGallery, row.ID).Order(sqlOrderSortID).Find(&photos).Error
	if err != nil {
		return GalleryRecord{}, fmt.Errorf("load photos of gallery %d: %w", row.ID, err)
	}
	record := GalleryRecord{
		ID: row.ID, TournamentID: row.TournamentID,
		GalleryFields: GalleryFields{Name: row.Name, SortOrder: row.SortOrder},
		Photos:        make([]PhotoFields, len(photos)),
	}
	for i, photo := range photos {
		record.Photos[i] = PhotoFields{ThumbURL: photo.ThumbURL, FullURL: photo.FullURL}
	}
	return record, nil
}

// GetAdminGallery はギャラリーを写真つきで返す。無ければ TGALLERY_ID の NotFound。
func GetAdminGallery(ctx context.Context, db *gorm.DB, id int64) (GalleryRecord, error) {
	row, err := findByID[Gallery](ctx, db, id, domain.ResourceGallery)
	if err != nil {
		return GalleryRecord{}, err
	}
	return toGalleryRecord(ctx, db, row)
}

// CreateGallery は大会にギャラリーを作る(写真は ReplacePhotos で入れる)。
func CreateGallery(
	ctx context.Context,
	db *gorm.DB,
	tournamentID int64,
	fields GalleryFields,
) (GalleryRecord, error) {
	if err := fields.validate(); err != nil {
		return GalleryRecord{}, err
	}
	if _, err := findByID[Tournament](
		ctx,
		db,
		tournamentID,
		domain.ResourceTournament,
	); err != nil {
		return GalleryRecord{}, err
	}
	row := Gallery{TournamentID: tournamentID, Name: fields.Name, SortOrder: fields.SortOrder}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return GalleryRecord{}, writeError("create gallery", err)
	}
	return toGalleryRecord(ctx, db, row)
}

// UpdateGallery はギャラリーの名前と並び順を変える。
func UpdateGallery(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	fields GalleryFields,
) (GalleryRecord, error) {
	if err := fields.validate(); err != nil {
		return GalleryRecord{}, err
	}
	row, err := findByID[Gallery](ctx, db, id, domain.ResourceGallery)
	if err != nil {
		return GalleryRecord{}, err
	}
	row.Name, row.SortOrder = fields.Name, fields.SortOrder
	if err := db.WithContext(ctx).Save(&row).Error; err != nil {
		return GalleryRecord{}, writeError("update gallery", err)
	}
	return toGalleryRecord(ctx, db, row)
}

// DeleteGallery はギャラリーと写真を消す。
func DeleteGallery(ctx context.Context, db *gorm.DB, id int64) error {
	return deleteByID[Gallery](ctx, db, id, domain.ResourceGallery)
}

// ReplacePhotos はギャラリーの写真を photos(この順で表示)で置き換える。
func ReplacePhotos(
	ctx context.Context,
	db *gorm.DB,
	galleryID int64,
	photos []PhotoFields,
) (GalleryRecord, error) {
	for i, photo := range photos {
		if err := domain.CheckURL(
			fmt.Sprintf("photos[%d].thumbUrl", i),
			photo.ThumbURL,
		); err != nil {
			return GalleryRecord{}, err
		}
		if err := domain.CheckURL(fmt.Sprintf("photos[%d].fullUrl", i), photo.FullURL); err != nil {
			return GalleryRecord{}, err
		}
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := findByID[Gallery](ctx, tx, galleryID, domain.ResourceGallery); err != nil {
			return err
		}
		if err := tx.Where(sqlByGallery, galleryID).Delete(&GalleryPhoto{}).Error; err != nil {
			return fmt.Errorf("clear photos: %w", err)
		}
		if len(photos) == 0 {
			return nil
		}
		rows := make([]GalleryPhoto, len(photos))
		for i, photo := range photos {
			rows[i] = GalleryPhoto{
				GalleryID: galleryID,
				ThumbURL:  photo.ThumbURL,
				FullURL:   photo.FullURL,
				SortOrder: i,
			}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return writeError("create photos", err)
		}
		return nil
	})
	if err != nil {
		return GalleryRecord{}, err
	}
	return GetAdminGallery(ctx, db, galleryID)
}
