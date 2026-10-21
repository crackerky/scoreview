package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
)

// 記録係から届く数値が DB の integer を超えても 500 にせず 400 にする(セキュリティ点検 L-2)。
func TestScoringRejectsOutOfRangeNumbers(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	path := fmt.Sprintf("/api/v1/admin/matches/%d", s.match)
	stat := func(blocks, greatests int) map[string]any {
		return map[string]any{"stats": []map[string]any{
			{"playerId": s.scorer, "side": "left", "blocks": blocks, "greatests": greatests},
		}}
	}
	for name, req := range map[string]struct {
		method, path string
		body         any
	}{
		"blocks 3e9":    {http.MethodPut, path + "/player-stats", stat(3_000_000_000, 0)},
		"greatests":     {http.MethodPut, path + "/player-stats", stat(0, domain.MaxPlayerStat+1)},
		"negative":      {http.MethodPut, path + "/player-stats", stat(-1, 0)},
		"elapsedSec":    {http.MethodPost, path + "/events", map[string]any{"kind": "timeout", "side": "left", "elapsedSec": 3_000_000_000}},
		"score":         {http.MethodPost, path + "/status", map[string]any{"status": "finished", "score": map[string]int{"left": 1000, "right": 0}}},
		"forfeit score": {http.MethodPost, path + "/result", map[string]any{"resultType": "forfeit", "winnerSide": "left", "forfeitScore": map[string]int{"winner": 3_000_000_000, "loser": 0}}},
	} {
		rec := f.do(t, f.staffToken, req.method, req.path, req.body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "%s: %s", name, rec.Body.String())
	}
	// 上限ちょうどは受け付ける
	f.callAs(t, f.staffToken, http.StatusOK, http.MethodPut, path+"/player-stats",
		stat(domain.MaxPlayerStat, domain.MaxPlayerStat))
	f.callAs(t, f.staffToken, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"kind": "timeout", "side": "left", "elapsedSec": domain.MaxElapsedSec})
}
