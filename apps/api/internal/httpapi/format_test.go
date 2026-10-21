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
	assert.Equal(t, "4 プール(各 3 チーム) → 4 チームのトーナメント × 3(1〜4 位、5〜8 位、9〜12 位)", progress["title"])
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

// TestCorrectionScoreMismatch は本部の結果入力の楽観的ロック。確かめたスコアと今のスコアが違えば
// 409(score_mismatch)で、記録係の試合終了の照合と同じエラーコードにする。
func TestCorrectionScoreMismatch(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	division := b.Division(b.Tournament().ID)
	left, right := b.Team(division), b.Team(division)
	match := b.Match(division, &left, &right)
	path := fmt.Sprintf("/api/v1/admin/matches/%d/correction", match.ID)
	opened := map[string]any{"left": 0, "right": 0}
	f.call(t, http.StatusOK, http.MethodPost, path, map[string]any{"score": map[string]any{"left": 11, "right": 9},
		"winnerSide": nil, "downstream": nil, "expectedScore": opened})
	body := f.call(t, http.StatusConflict, http.MethodPost, path, map[string]any{
		"score": map[string]any{"left": 9, "right": 11}, "winnerSide": nil, "downstream": nil, "expectedScore": opened})
	errBody, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "score_mismatch", errBody["code"])
	assert.Contains(t, errBody["message"], "11-9")
}

// TestDayOfOperations は本部の一括操作(棄権、締め、中断と再開、遅れの調整の確認)を API で通す。
func TestDayOfOperations(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	var seeds []int64
	for range 12 {
		seeds = append(seeds, b.Team(division).ID)
	}
	base := fmt.Sprintf("/api/v1/admin/divisions/%d", division.ID)
	f.call(t, http.StatusOK, http.MethodPut, base+"/format", map[string]any{
		"template": "pool_bands", "seeds": seeds, "tiebreak": "current",
		"params": map[string]any{"teams": 12, "pools": 4, "bands": []map[string]any{
			{"ranks": []int{1}, "shape": "T4"}, {"ranks": []int{2}, "shape": "T4"}, {"ranks": []int{3}, "shape": "T4"}}},
		"forfeit": map[string]any{"winner": 15, "loser": 0},
		"schedule": map[string]any{"date": "2026-10-18", "start": "09:00", "slotMinutes": 20,
			"courts": []string{"1", "2", "3", "4"}, "teamRestSlots": 0, "stageGapSlots": 0, "end": nil},
	})

	team := fmt.Sprintf("/api/v1/admin/teams/%d/withdrawal", seeds[7])
	plan := f.call(t, http.StatusOK, http.MethodGet, team, nil)
	targets, ok := plan["targets"].([]any)
	require.True(t, ok)
	require.Len(t, targets, 2)
	var matchIDs []int64
	for _, target := range targets {
		row, ok := target.(map[string]any)
		require.True(t, ok)
		matchIDs = append(matchIDs, id(row["id"]))
	}
	f.call(t, http.StatusConflict, http.MethodPost, team, map[string]any{"matchIds": matchIDs[:1]})
	done := f.call(t, http.StatusOK, http.MethodPost, team, map[string]any{"matchIds": matchIDs})
	assert.NotNil(t, done["withdrawnAt"])

	closing := f.call(t, http.StatusOK, http.MethodGet, base+"/closing", nil)
	assert.NotEmpty(t, closing["remaining"])
	assert.Nil(t, closing["closedAt"])
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodGet, base+"/closing", nil)

	dry := f.call(t, http.StatusOK, http.MethodPost, base+"/schedule-shift",
		map[string]any{"court": "1", "minutes": 15, "dryRun": true})
	assert.Positive(t, dry["moved"])

	tournament := fmt.Sprintf("/api/v1/admin/tournaments/%d", tour.ID)
	f.call(t, http.StatusBadRequest, http.MethodPost, tournament+"/suspension",
		map[string]any{"reason": "snow", "note": "", "resumeAt": nil})
	f.call(t, http.StatusOK, http.MethodPost, tournament+"/suspension",
		map[string]any{"reason": "thunder", "note": "", "resumeAt": "13:30"})
	top := f.call(t, http.StatusOK, http.MethodGet, fmt.Sprintf("/api/v1/tournaments/%d", tour.ID), nil)
	frame, ok := top["frame"].(map[string]any)
	require.True(t, ok)
	suspension, ok := frame["suspension"].(map[string]any)
	require.True(t, ok, "閲覧の応答に中断が出る")
	assert.Equal(t, "thunder", suspension["reason"])
	progress := f.call(t, http.StatusOK, http.MethodGet, base+"/progress", nil)
	assert.NotNil(t, progress["suspension"])
	resumed := f.call(t, http.StatusOK, http.MethodPost, tournament+"/resume", map[string]any{"minutes": 30})
	assert.InDelta(t, 30, resumed["minutes"], 0)
	f.call(t, http.StatusConflict, http.MethodPost, tournament+"/resume", map[string]any{"minutes": nil})
}
