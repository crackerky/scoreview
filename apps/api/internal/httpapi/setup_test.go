package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
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

// TestCreateTeamAndFillEnglishNames は、API で作るチームと選手の英語名が空なら日本語名が入ることと、
// 既存の部門の英語名が空のチーム、選手、組に日本語名を入れる一括の操作を通す。
func TestCreateTeamAndFillEnglishNames(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	division := b.Division(b.Tournament().ID)
	old := b.Team(division, func(row *store.Team) { row.NameFullEn, row.NameShortEn = "", "" })
	b.Group(division.ID, domain.GroupPool, func(row *store.Group) { row.NameEn = "" })

	team := f.call(t, http.StatusCreated, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/teams", division.ID), map[string]any{
			"fullName": map[string]any{"ja": "架空サンズ", "en": ""}, "shortName": map[string]any{"ja": "サンズ"},
		})
	assert.Equal(t, map[string]any{"ja": "架空サンズ", "en": "架空サンズ"}, team["fullName"])
	assert.Equal(t, map[string]any{"ja": "サンズ", "en": "サンズ"}, team["shortName"])
	player := f.call(t, http.StatusCreated, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/teams/%v/players", team["id"]), map[string]any{
			"number": "7", "name": map[string]any{"ja": "架空 一郎", "en": ""},
		})
	assert.Equal(t, map[string]any{"ja": "架空 一郎", "en": "架空 一郎"}, player["name"])

	path := fmt.Sprintf("/api/v1/admin/divisions/%d/english-names", division.ID)
	filled := f.call(t, http.StatusOK, http.MethodPost, path, nil)
	assert.Equal(t, map[string]any{"teams": 1.0, "players": 0.0, "groups": 1.0}, filled)
	got := f.call(t, http.StatusOK, http.MethodGet, fmt.Sprintf("/api/v1/admin/teams/%d", old.ID), nil)
	assert.Equal(t, map[string]any{"ja": old.NameFullJa, "en": old.NameFullJa}, got["fullName"])
	f.call(t, http.StatusNotFound, http.MethodPost, "/api/v1/admin/divisions/999999/english-names", nil)
}
