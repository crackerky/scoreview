package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const sqlByGallery = "gallery_id = ?"

// GetGallery はギャラリーを返す。無ければ TGALLERY_ID の NotFound。写真は sort_order, id 順で、
// ページングしない(参照サイトも全件を 1 ページに出す。clone-spec A 6)。
func GetGallery(ctx context.Context, db *gorm.DB, id int64) (GalleryPage, error) {
	gallery, err := findByID[Gallery](ctx, db, id, domain.ResourceGallery)
	if err != nil {
		return GalleryPage{}, err
	}
	scope, err := loadScopeOf(ctx, db, gallery.TournamentID)
	if err != nil {
		return GalleryPage{}, err
	}
	var photos []GalleryPhoto
	if err := db.WithContext(ctx).Where(sqlByGallery, id).Order(sqlOrderSortID).Find(&photos).Error; err != nil {
		return GalleryPage{}, fmt.Errorf("load photos of gallery %d: %w", id, err)
	}
	page := GalleryPage{
		Frame:  scope.frame,
		ID:     gallery.ID,
		Name:   gallery.Name,
		Photos: make([]Photo, 0, len(photos)),
	}
	for _, photo := range photos {
		page.Photos = append(page.Photos, Photo{ThumbURL: photo.ThumbURL, FullURL: photo.FullURL})
	}
	return page, nil
}
