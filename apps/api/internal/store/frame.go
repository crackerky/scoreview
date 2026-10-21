package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const sqlPublishedGroups = "division_id IN ? AND published = ?"

// tournamentScope は大会に属するページが共通で読むもの。divisions は sort_order, id 順。
type tournamentScope struct {
	tournament Tournament
	divisions  []Division
	frame      Frame
}

// divisionOrder はディビジョンの並び順(divisions の位置)を ID から引く。
func (s tournamentScope) divisionOrder() map[int64]int {
	order := make(map[int64]int, len(s.divisions))
	for i, division := range s.divisions {
		order[division.ID] = i
	}
	return order
}

// loadTournamentScope は大会とディビジョン、ナビの骨格(Frame)を読む。問い合わせは大会の数に
// よらず 4 回(ディビジョン、グループ、ギャラリー、バナー)。全ページで共有する。
func loadTournamentScope(
	ctx context.Context,
	db *gorm.DB,
	tournament Tournament,
) (tournamentScope, error) {
	tx := db.WithContext(ctx)
	var divisions []Division
	if err := tx.Where(sqlByTournament, tournament.ID).Order(sqlOrderSortID).Find(&divisions).Error; err != nil {
		return tournamentScope{}, fmt.Errorf("load divisions of tournament %d: %w", tournament.ID, err)
	}
	divisionIDs := make([]int64, len(divisions))
	for i, division := range divisions {
		divisionIDs[i] = division.ID
	}
	var groups []Group
	if len(divisionIDs) > 0 {
		// bracket_layout は大きいのでナビでは読まない
		err := tx.Select("id", "division_id", "name", "name_en", "linked").
			Where(sqlPublishedGroups, divisionIDs, true).Order(sqlOrderSortID).Find(&groups).Error
		if err != nil {
			return tournamentScope{}, fmt.Errorf("load groups of tournament %d: %w", tournament.ID, err)
		}
	}
	var galleries []Gallery
	if err := tx.Where(sqlByTournament, tournament.ID).Order(sqlOrderSortID).Find(&galleries).Error; err != nil {
		return tournamentScope{}, fmt.Errorf("load galleries of tournament %d: %w", tournament.ID, err)
	}
	var banners []TournamentBanner
	if err := tx.Where(sqlByTournament, tournament.ID).Order(sqlOrderSortID).Find(&banners).Error; err != nil {
		return tournamentScope{}, fmt.Errorf("load banners of tournament %d: %w", tournament.ID, err)
	}
	return tournamentScope{
		tournament: tournament,
		divisions:  divisions,
		frame:      buildFrame(tournament, divisions, groups, galleries, banners),
	}, nil
}

func buildFrame(
	tournament Tournament,
	divisions []Division,
	groups []Group,
	galleries []Gallery,
	banners []TournamentBanner,
) Frame {
	groupsByDivision := map[int64][]FrameGroup{}
	for _, group := range groups {
		groupsByDivision[group.DivisionID] = append(groupsByDivision[group.DivisionID],
			FrameGroup{ID: group.ID, Name: i18n(group.Name, group.NameEn), Linked: group.Linked})
	}
	frame := Frame{
		ID:            tournament.ID,
		Name:          i18n(tournament.NameJa, tournament.NameEn),
		ShortName:     tournament.ShortName,
		TeamListLabel: tournament.TeamListLabel,
		GroupsLabel:   tournament.GroupLabel,
		ShowSchedule:  tournament.ShowSchedule,
		Divisions:     make([]FrameDivision, 0, len(divisions)),
		Galleries:     make([]FrameGallery, 0, len(galleries)),
		Banners:       make([]FrameBanner, 0, len(banners)),
	}
	for _, division := range divisions {
		fg := groupsByDivision[division.ID]
		if fg == nil {
			fg = []FrameGroup{}
		}
		frame.Divisions = append(
			frame.Divisions,
			FrameDivision{
				ID: division.ID, Name: i18n(division.NameJa, division.NameEn),
				ScheduleEnabled: division.ScheduleEnabled, Groups: fg,
			},
		)
	}
	for _, gallery := range galleries {
		frame.Galleries = append(
			frame.Galleries,
			FrameGallery{ID: gallery.ID, Name: i18n(gallery.Name, gallery.NameEn)},
		)
	}
	for _, banner := range banners {
		frame.Banners = append(
			frame.Banners,
			FrameBanner{ImageURL: banner.ImageURL, LinkURL: banner.LinkURL},
		)
	}
	return frame
}

// loadScopeByTournamentID は利用者が指定した大会 ID から読む。無ければ TOURNAMENT_ID の NotFound。
func loadScopeByTournamentID(ctx context.Context, db *gorm.DB, id int64) (tournamentScope, error) {
	tournament, err := findByID[Tournament](ctx, db, id, domain.ResourceTournament)
	if err != nil {
		return tournamentScope{}, err
	}
	return loadTournamentScope(ctx, db, tournament)
}

// loadScopeOf は別の行(ディビジョン、チームなど)が参照する大会から読む。
func loadScopeOf(ctx context.Context, db *gorm.DB, tournamentID int64) (tournamentScope, error) {
	tournament, err := findInternal[Tournament](ctx, db, tournamentID, "tournament")
	if err != nil {
		return tournamentScope{}, err
	}
	return loadTournamentScope(ctx, db, tournament)
}
