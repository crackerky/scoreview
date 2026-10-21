package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/store/storetest"
)

// TestCopyTournamentAndDeleteTeams は大会のコピー(非公開、新しい開催日)と、試合の無い部門のチームの
// 一括削除を API で通す。
func TestCopyTournamentAndDeleteTeams(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	b.Team(division)
	b.Team(division)

	path := fmt.Sprintf("/api/v1/admin/tournaments/%d/copy", tour.ID)
	f.call(t, http.StatusBadRequest, http.MethodPost, path, map[string]any{"withTeams": true})
	copied := f.call(t, http.StatusCreated, http.MethodPost, path, map[string]any{
		"name": map[string]any{"ja": "次回", "en": ""}, "startDate": "2027-05-03", "withTeams": true,
	})
	assert.Equal(t, false, copied["listed"])
	assert.Equal(t, "2027-05-03", copied["startDate"])
	divisions := f.call(t, http.StatusOK, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/tournaments/%v/divisions", copied["id"]), nil)
	list, ok := divisions["divisions"].([]any)
	assert.True(t, ok)
	assert.Len(t, list, 1)

	deleted := f.call(t, http.StatusOK, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/divisions/%d/teams", division.ID), nil)
	assert.InDelta(t, 2, deleted["deleted"], 0)
	saved := f.call(t, http.StatusOK, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/divisions/%d/format", division.ID), nil)
	assert.Nil(t, saved["format"])
	assert.Nil(t, saved["inherited"])
}
