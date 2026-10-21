package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/store/storetest"
)

// TestFormatWizardAndProgress は形の提案 → プレビュー → 適用 → 進行状況 → 確定の流れを API で通す。
func TestFormatWizardAndProgress(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	var seeds []int64
	for range 12 {
		seeds = append(seeds, b.Team(division).ID)
	}
	base := fmt.Sprintf("/api/v1/admin/divisions/%d", division.ID)

	proposals := f.call(t, http.StatusOK, http.MethodPost, base+"/format/proposals", map[string]any{
		"teams": 12, "courts": 4, "slotMinutes": 20, "date": "2026-10-18", "start": "09:00", "sport": "four",
	})
	list, ok := proposals["proposals"].([]any)
	require.True(t, ok)
	first, ok := list[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "pb4", first["id"])
	assert.Equal(t, true, first["recommended"])
	assert.InDelta(t, 24, first["matches"], 0)

	input := map[string]any{
		"template": first["template"], "params": first["params"], "seeds": seeds, "tiebreak": "current",
		"forfeit": map[string]any{"winner": 15, "loser": 0},
		"schedule": map[string]any{"date": "2026-10-18", "start": "09:00", "slotMinutes": 20,
			"courts": []string{"1", "2", "3", "4"}, "teamRestSlots": 0, "stageGapSlots": 0, "end": nil},
	}
	preview := f.call(t, http.StatusOK, http.MethodPost, base+"/format/preview", input)
	assert.Equal(t, "A", preview["startLetter"])
	groups, ok := preview["groups"].([]any)
	require.True(t, ok)
	assert.Len(t, groups, 7)
	f.call(t, http.StatusBadRequest, http.MethodPost, base+"/format/preview",
		map[string]any{"template": "pool_bands", "params": map[string]any{"teams": 12, "x": 1}, "seeds": seeds})

	f.call(t, http.StatusOK, http.MethodPut, base+"/format", input)
	saved := f.call(t, http.StatusOK, http.MethodGet, base+"/format", nil)
	require.NotNil(t, saved["format"])
	copies := f.call(t, http.StatusOK, http.MethodGet, "/api/v1/admin/formats", nil)
	assert.Len(t, copies["formats"], 1)

	progress := f.call(t, http.StatusOK, http.MethodGet, base+"/progress", nil)
	assert.Equal(t, "4 プール(各 3 チーム) → T4 × 3(1st / 5th / 9th)", progress["title"])
	progressGroups, ok := progress["groups"].([]any)
	require.True(t, ok)
	poolA, ok := progressGroups[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "playing", poolA["state"])
	f.call(t, http.StatusConflict, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/groups/%d/confirm", int64(poolA["id"].(float64))), // JSON の数値は float64
		map[string]any{"unplayed": false})
	// 試合が 1 つも無いプールは全チームが並ぶので、未消化のまま確定しても順位の指定を求める
	f.call(t, http.StatusConflict, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/groups/%d/confirm", int64(poolA["id"].(float64))), // 同上
		map[string]any{"unplayed": true})

	// 担当外の記録係は進行状況を読めない
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodGet, base+"/progress", nil)
	check := f.call(t, http.StatusOK, http.MethodGet, base+"/schedule-check", nil)
	assert.NotNil(t, check["violations"])
	f.call(t, http.StatusOK, http.MethodPost, base+"/schedule-shift", map[string]any{"court": nil, "minutes": 10})
}

// TestDownstreamChoiceIsValidated は訂正と状態の変更の downstream(下流の扱い)を検査する。
// keep / reset / null 以外は 400、対戦チームの決まらない試合への結果は 409。
func TestDownstreamChoiceIsValidated(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	division := b.Division(b.Tournament().ID)
	left, right := b.Team(division), b.Team(division)
	match := b.Match(division, &left, &right)
	path := fmt.Sprintf("/api/v1/admin/matches/%d", match.ID)
	score := map[string]any{"left": 3, "right": 1}
	f.call(t, http.StatusBadRequest, http.MethodPost, path+"/correction",
		map[string]any{"score": score, "winnerSide": nil, "downstream": "maybe"})
	f.call(t, http.StatusBadRequest, http.MethodPost, path+"/status",
		map[string]any{"status": "finished", "score": score, "downstream": ""})
	f.call(t, http.StatusOK, http.MethodPost, path+"/correction",
		map[string]any{"score": score, "winnerSide": nil, "downstream": "keep"})

	undecided := b.Match(division, &left, nil)
	f.call(t, http.StatusConflict, http.MethodPost, fmt.Sprintf("/api/v1/admin/matches/%d/status", undecided.ID),
		map[string]any{"status": "live"})
}
