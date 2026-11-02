package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/store/storetest"
)

// 観戦アプリの表: 見せ方は管理者だけが読み書きし、観戦アプリは見せ方とトーナメントの形を読む。
func TestLiveTables(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	var seeds []int64
	for range 12 {
		seeds = append(seeds, b.Team(division).ID)
	}
	// 提案の先頭(4 プール → T4 × 3)を適用する
	base := fmt.Sprintf("/api/v1/admin/divisions/%d", division.ID)
	proposals := f.call(t, http.StatusOK, http.MethodPost, base+"/format/proposals", map[string]any{
		"teams": 12, "courts": 3, "slotMinutes": 20, "date": "2026-10-18", "start": "09:00", "sport": "four",
	})
	list, ok := proposals["proposals"].([]any)
	require.True(t, ok)
	first, ok := list[0].(map[string]any)
	require.True(t, ok)
	f.call(t, http.StatusOK, http.MethodPut, base+"/format", map[string]any{
		"template": first["template"], "params": first["params"], "seeds": seeds, "tiebreak": "current",
		"forfeit": map[string]any{"winner": 15, "loser": 0},
		"schedule": map[string]any{"date": "2026-10-18", "start": "09:00", "slotMinutes": 20,
			"courts": []string{"1", "2", "3"}, "teamRestSlots": 0, "stageGapSlots": 0, "end": nil},
	})

	admin := fmt.Sprintf("/api/v1/admin/tournaments/%d/live-tables", tour.ID)
	config := f.call(t, http.StatusOK, http.MethodGet, admin, nil)
	assert.Equal(t, true, config["shown"])
	assert.Equal(t, "maru", config["marks"])
	// ボディに無いキーは今のまま
	saved := f.call(t, http.StatusOK, http.MethodPut, admin, map[string]any{"marks": "kanji", "bracketShown": false})
	assert.Equal(t, "kanji", saved["marks"])
	assert.Equal(t, false, saved["bracketShown"])
	assert.Equal(t, true, saved["poolShown"])
	f.call(t, http.StatusBadRequest, http.MethodPut, admin, map[string]any{"order": "random"})
	// 記録係は読めない
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodGet, admin, nil)

	tables := f.call(t, http.StatusOK, http.MethodGet, fmt.Sprintf("/api/v1/tournaments/%d/live-tables", tour.ID), nil)
	cfg, ok := tables["config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "kanji", cfg["marks"])
	divisions, ok := tables["divisions"].([]any)
	require.True(t, ok)
	require.Len(t, divisions, 1)
	d, ok := divisions[0].(map[string]any)
	require.True(t, ok)
	pools, ok := d["pools"].([]any)
	require.True(t, ok)
	require.Len(t, pools, 4)
	poolA, ok := pools[0].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 1, poolA["advance"], 0)
	brackets, ok := d["brackets"].([]any)
	require.True(t, ok)
	require.Len(t, brackets, 3)
	top, ok := brackets[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "1〜4 位を決めるトーナメント", top["name"])
	matches, ok := top["matches"].([]any)
	require.True(t, ok)
	require.Len(t, matches, 4)
	f.call(t, http.StatusNotFound, http.MethodGet, "/api/v1/tournaments/999999/live-tables", nil)
}
