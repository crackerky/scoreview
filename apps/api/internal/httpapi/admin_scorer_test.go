package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	deviceA = "6f1c2b9e-7a43-4d1e-9b0a-1f2e3d4c5b01"
	deviceB = "6f1c2b9e-7a43-4d1e-9b0a-1f2e3d4c5b02"
	opMove  = "6f1c2b9e-7a43-4d1e-9b0a-1f2e3d4c5b03"
)

func score(detail map[string]any) string {
	match := detail["match"].(map[string]any)
	return fmt.Sprintf("%v-%v", match["left"].(map[string]any)["score"],
		match["right"].(map[string]any)["score"])
}

// 試合終了の確認シートを開いた後に別の端末で得点が入ったら、確認したスコアと食い違うので 409 にし、
// 記録係に最新のスコアで確認し直させる(古いスコアのまま確定しない)。
func TestFinishRejectsUnconfirmedScore(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	path := fmt.Sprintf("/api/v1/admin/matches/%d", s.match)
	for range 2 {
		f.callAs(t, f.staffToken, http.StatusCreated, http.MethodPost, path+"/events",
			map[string]any{"kind": "goal", "side": "left"})
	}
	// 記録係が 2-0 で確認シートを開いている間に、別の端末(本部)が右の得点を入れた
	f.call(t, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"kind": "goal", "side": "right"})

	end := map[string]any{"kind": "end", "expectedScore": map[string]int{"left": 2, "right": 0}}
	rec := f.do(t, f.staffToken, http.MethodPost, path+"/events", end)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	var failure struct {
		Error struct{ Code, Message string }
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &failure))
	assert.Equal(t, "score_mismatch", failure.Error.Code)
	assert.Contains(t, failure.Error.Message, "2-1", "理由に現在のスコアを書く")

	rec = f.do(t, f.staffToken, http.MethodPost, path+"/status",
		map[string]any{"status": "finished", "expectedScore": map[string]int{"left": 2, "right": 0}})
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	rec = f.do(t, f.staffToken, http.MethodPost, path+"/result",
		map[string]any{"resultType": "abandoned", "expectedScore": map[string]int{"left": 1, "right": 1}})
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	detail := f.callAs(t, f.staffToken, http.StatusOK, http.MethodGet, path, nil)
	assert.Equal(t, "live", detail["match"].(map[string]any)["status"], "食い違えば終了しない")
	assert.Len(t, detail["events"], 3)

	// 最新のスコアで確認し直せば終われる
	end["expectedScore"] = map[string]int{"left": 2, "right": 1}
	detail = f.callAs(t, f.staffToken, http.StatusCreated, http.MethodPost, path+"/events", end)
	assert.Equal(t, "finished", detail["match"].(map[string]any)["status"])
	assert.Equal(t, "2-1", score(detail))

	// 確認したスコアは試合終了の操作だけに付ける
	f.callAs(t, f.staffToken, http.StatusBadRequest, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/events", s.match),
		map[string]any{"kind": "goal", "side": "left", "expectedScore": map[string]int{}})
}

// 同じ試合を 2 台で開いたとき、先に開いた端末が記録担当になり、後の端末は「引き継ぐ」まで担当に
// ならない。延長が止まれば期限で空く。別の記録係がほぼ同時に入れた同じ種類の記録は知らせる。
func TestScorerClaimAndTakeover(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	// 試合の詳細は実時間で記録担当の期限を見るので、テストの時計を実時間に合わせる
	// (時計を進めたのでログインし直す)
	f.now = time.Now().UTC()
	f.adminToken, f.staffToken = f.login(t, "admin@example.com"), f.login(t, "staff@example.com")
	path := fmt.Sprintf("/api/v1/admin/matches/%d", s.match)
	claim := func(token, device string, takeover bool) map[string]any {
		return f.callAs(t, token, http.StatusOK, http.MethodPost, path+"/scorer",
			map[string]any{"deviceId": device, "takeover": takeover})
	}

	first := claim(f.staffToken, deviceA, false)
	assert.Equal(t, true, first["mine"])
	assert.Equal(t, "staff@example.com", first["scorer"].(map[string]any)["name"])

	second := claim(f.adminToken, deviceB, false)
	assert.Equal(t, false, second["mine"], "他の端末が記録中なら引き継ぎを確かめる")
	assert.Equal(t, deviceA, second["scorer"].(map[string]any)["deviceId"])
	detail := f.call(t, http.StatusOK, http.MethodGet, path, nil)
	assert.Equal(t, deviceA, detail["scorer"].(map[string]any)["deviceId"])

	taken := claim(f.adminToken, deviceB, true)
	assert.Equal(t, true, taken["mine"])
	renewed := claim(f.staffToken, deviceA, false)
	assert.Equal(t, false, renewed["mine"], "引き継がれた端末の延長は担当を取り返さない")

	// 端末を閉じると外れる。他の端末の担当は外さない
	f.callAs(t, f.staffToken, http.StatusNoContent, http.MethodDelete,
		path+"/scorer?deviceId="+deviceA, nil)
	detail = f.call(t, http.StatusOK, http.MethodGet, path, nil)
	assert.Equal(t, deviceB, detail["scorer"].(map[string]any)["deviceId"])
	f.call(t, http.StatusNoContent, http.MethodDelete, path+"/scorer?deviceId="+deviceB, nil)
	detail = f.call(t, http.StatusOK, http.MethodGet, path, nil)
	assert.Nil(t, detail["scorer"])

	// 延長が止まった担当は期限で空く
	f.now = time.Now().UTC().Add(-time.Minute)
	claim(f.adminToken, deviceB, false)
	f.now = time.Now().UTC()
	assert.Equal(t, true, claim(f.staffToken, deviceA, false)["mine"])

	// 割り当ての無い試合の担当にはなれない
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/scorer", s.unassigned),
		map[string]any{"deviceId": deviceA})
	f.callAs(t, f.staffToken, http.StatusBadRequest, http.MethodPost, path+"/scorer",
		map[string]any{"deviceId": "not-a-uuid"})

	// 2 人が同じ得点をほぼ同時に記録した
	f.callAs(t, f.staffToken, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"clientId": opGoal, "kind": "goal", "side": "left"})
	f.now = f.now.Add(2 * time.Second)
	detail = f.call(t, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"clientId": opGoal2, "kind": "goal", "side": "left"})
	records := detail["concurrentRecords"].([]any)
	require.Len(t, records, 1)
	record := records[0].(map[string]any)
	assert.Equal(t, "goal", record["kind"])
	assert.Equal(t, opGoal, record["firstClientId"])
	assert.Equal(t, opGoal2, record["secondClientId"])
	assert.Equal(t, "staff@example.com", record["firstUser"])
	assert.Equal(t, float64(2), record["secondsApart"])
	assert.Nil(t, detail["matchMinutes"], "時間割の無いディビジョンは試合時間なし")
}

// 数点前の得点のチームの付け間違いを、その得点だけ相手チームの得点に直す。監査ログには 1 つの訂正
// として残る。
func TestMoveGoalToOtherTeam(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	path := fmt.Sprintf("/api/v1/admin/matches/%d", s.match)
	f.callAs(t, f.staffToken, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"clientId": opGoal, "kind": "goal", "side": "left", "scorerId": s.scorer})
	f.callAs(t, f.staffToken, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"kind": "goal", "side": "left"})

	move := map[string]any{"clientId": opMove, "side": "right"}
	for range 2 {
		detail := f.callAs(t, f.staffToken, http.StatusOK, http.MethodPut,
			fmt.Sprintf("%s/events/%s/side", path, opGoal), move)
		assert.Equal(t, "1-1", score(detail), "送り直しても 1 回だけ")
		moved := detail["events"].([]any)[0].(map[string]any)
		assert.Equal(t, "right", moved["side"])
		assert.Nil(t, moved["scorer"], "元のチームの得点者は外す")
	}
	f.callAs(t, f.staffToken, http.StatusConflict, http.MethodPut,
		fmt.Sprintf("%s/events/%s/side", path, opGoal), map[string]any{"side": "right"})

	audit := f.callAs(t, f.staffToken, http.StatusOK, http.MethodGet, path+"/audit", nil)
	entry := audit["entries"].([]any)[0].(map[string]any)
	assert.Equal(t, "event.side", entry["action"])
	before := entry["before"].(map[string]any)
	after := entry["after"].(map[string]any)
	assert.Equal(t, "left", before["event"].(map[string]any)["side"])
	assert.Equal(t, "right", after["event"].(map[string]any)["side"])
	assert.Equal(t, float64(2), before["match"].(map[string]any)["leftScore"])
	assert.Equal(t, float64(1), after["match"].(map[string]any)["rightScore"])

	// 終了した試合は先に試合中へ戻す
	f.callAs(t, f.staffToken, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"kind": "end"})
	f.callAs(t, f.staffToken, http.StatusConflict, http.MethodPut,
		fmt.Sprintf("%s/events/%s/side", path, opGoal), map[string]any{"side": "left"})
}
