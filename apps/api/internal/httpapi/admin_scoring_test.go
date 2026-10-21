package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	opGoal  = "0b6c8a52-0d8e-4b55-9f3e-2f1d6f2a0001"
	opUndo  = "0b6c8a52-0d8e-4b55-9f3e-2f1d6f2a0002"
	opGoal2 = "0b6c8a52-0d8e-4b55-9f3e-2f1d6f2a0003"
	opNames = "0b6c8a52-0d8e-4b55-9f3e-2f1d6f2a0004"
	opEnd   = "0b6c8a52-0d8e-4b55-9f3e-2f1d6f2a0005"
)

// 通信が切れている間に溜めた操作を送り直しても、同じ clientId の操作は 1 回しか適用しない。
func TestScoringOpsAreIdempotent(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	staff := f.staffToken
	path := fmt.Sprintf("/api/v1/admin/matches/%d", s.match)
	goal := map[string]any{"clientId": opGoal, "kind": "goal", "side": "left", "elapsedSec": 30}

	for range 2 {
		detail := f.callAs(t, staff, http.StatusCreated, http.MethodPost, path+"/events", goal)
		events := detail["events"].([]any)
		require.Len(t, events, 1, "同じ clientId の得点は 1 件")
		assert.Equal(t, opGoal, events[0].(map[string]any)["clientId"])
		assert.Equal(t, float64(1), detail["match"].(map[string]any)["left"].(map[string]any)["score"])
	}

	// まだ api の id を知らない得点経過を clientId で取り消す。送り直しても 2 回目は何もしない
	for range 2 {
		detail := f.callAs(t, staff, http.StatusOK, http.MethodDelete,
			fmt.Sprintf("%s/events/%s?clientId=%s", path, opGoal, opUndo), nil)
		assert.Empty(t, detail["events"])
	}

	// 得点者を後から付ける(clientId で指す)
	f.callAs(t, staff, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"clientId": opGoal2, "kind": "goal", "side": "left"})
	detail := f.callAs(t, staff, http.StatusOK, http.MethodPatch,
		fmt.Sprintf("%s/events/%s", path, opGoal2),
		map[string]any{"clientId": opNames, "scorerId": s.scorer})
	assert.Equal(t, "7",
		detail["events"].([]any)[0].(map[string]any)["scorer"].(map[string]any)["number"])

	end := map[string]any{"clientId": opEnd, "status": "finished"}
	f.callAs(t, staff, http.StatusOK, http.MethodPost, path+"/status", end)
	detail = f.callAs(t, staff, http.StatusOK, http.MethodPost, path+"/status", end)
	assert.Equal(t, "finished", detail["match"].(map[string]any)["status"],
		"終了の送り直しは 409 にならず現在の状態を返す")

	// 監査ログ: 1 操作 1 行で、送り直しの分は増えない。新しい順
	audit := f.callAs(t, staff, http.StatusOK, http.MethodGet, path+"/audit", nil)
	entries := audit["entries"].([]any)
	actions := make([]string, len(entries))
	for i, entry := range entries {
		actions[i] = entry.(map[string]any)["action"].(string)
	}
	assert.Equal(t,
		[]string{"match.status", "event.update", "event.add", "event.remove", "event.add"}, actions)
	last := entries[0].(map[string]any)
	assert.Equal(t, "staff@example.com", last["user"].(map[string]any)["name"])
	assert.Equal(t, "live", last["before"].(map[string]any)["match"].(map[string]any)["status"])
	assert.Equal(t, "finished", last["after"].(map[string]any)["match"].(map[string]any)["status"])
	named := entries[1].(map[string]any)
	assert.Nil(t, named["before"].(map[string]any)["event"].(map[string]any)["scorer"])
	assert.Equal(t, "選手",
		named["after"].(map[string]any)["event"].(map[string]any)["scorer"].(map[string]any)["name"])

	// clientId は UUID の形に限る
	f.callAs(t, staff, http.StatusBadRequest, http.MethodPost, path+"/status",
		map[string]any{"clientId": "not-a-uuid", "status": "finished"})
}

func TestResultTypesOverHTTP(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	path := fmt.Sprintf("/api/v1/admin/matches/%d", s.match)

	// 大会に規定スコアが無ければ、不戦勝にはスコアの入力が要る
	f.callAs(t, f.staffToken, http.StatusBadRequest, http.MethodPost, path+"/result",
		map[string]any{"resultType": "forfeit", "winnerSide": "right"})
	f.call(t, http.StatusOK, http.MethodPatch, fmt.Sprintf("/api/v1/admin/tournaments/%d", s.own),
		map[string]any{"forfeitScore": map[string]int{"winner": 15, "loser": 0}})
	detail := f.callAs(t, f.staffToken, http.StatusOK, http.MethodPost, path+"/result",
		map[string]any{"resultType": "forfeit", "winnerSide": "right"})
	match := detail["match"].(map[string]any)
	assert.Equal(t, "forfeit", match["resultType"])
	assert.Equal(t, "finished", match["status"])
	assert.Equal(t, float64(0), match["left"].(map[string]any)["score"])
	assert.Equal(t, float64(15), match["right"].(map[string]any)["score"])
	assert.Equal(t, map[string]any{"winner": float64(15), "loser": float64(0)},
		detail["forfeitScore"])

	// 終了した試合の結果を変えるには、管理者が試合中へ戻す(結果の種類は通常に戻る)
	f.callAs(t, f.staffToken, http.StatusConflict, http.MethodPost, path+"/result",
		map[string]any{"resultType": "unplayed"})
	detail = f.call(t, http.StatusOK, http.MethodPost, path+"/status",
		map[string]any{"status": "live"})
	match = detail["match"].(map[string]any)
	assert.Equal(t, "normal", match["resultType"])
	assert.Equal(t, float64(0), match["right"].(map[string]any)["score"], "規定スコアを引き継がない")

	detail = f.callAs(t, f.staffToken, http.StatusOK, http.MethodPost, path+"/result",
		map[string]any{"resultType": "double_forfeit"})
	match = detail["match"].(map[string]any)
	assert.Nil(t, match["left"].(map[string]any)["score"], "両チーム不戦敗はスコアを記録しない")

	// 閲覧用のチームページの成績: 両チーム不戦敗は負け
	rec := f.do(t, "", http.MethodGet, fmt.Sprintf("/api/v1/teams/%d", s.left), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"losses":1`)
}

func TestAssignmentsAndMyMatches(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	base := fmt.Sprintf("/api/v1/admin/tournaments/%d/assignments", s.own)

	mine := f.callAs(t, f.staffToken, http.StatusOK, http.MethodGet, "/api/v1/admin/me/matches", nil)
	matches := mine["matches"].([]any)
	require.Len(t, matches, 1)
	assert.Equal(t, float64(s.match), matches[0].(map[string]any)["id"])
	assert.Equal(t, "担当大会", matches[0].(map[string]any)["tournamentName"])

	// 大会の全試合(条件なし)を割り当てると、割り当てのない試合も入力できる
	created := f.call(t, http.StatusCreated, http.MethodPost, base,
		map[string]any{"userId": f.staffID})
	f.callAs(t, f.staffToken, http.StatusCreated, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/events", s.unassigned),
		map[string]any{"kind": "pull", "side": "left"})
	list := f.call(t, http.StatusOK, http.MethodGet, base, nil)
	assert.Len(t, list["assignments"], 2)
	f.call(t, http.StatusNoContent, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/assignments/%d", id(created["id"])), nil)
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/events", s.unassigned),
		map[string]any{"kind": "pull", "side": "left"})

	// 管理者は記録係に割り当てられない。他の大会の試合も指せない
	f.call(t, http.StatusBadRequest, http.MethodPost, base, map[string]any{"userId": 1})
	f.call(t, http.StatusBadRequest, http.MethodPost, base,
		map[string]any{"userId": f.staffID, "matchId": s.otherMatch})
}
