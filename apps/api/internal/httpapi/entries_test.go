package httpapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
)

// 申込: 管理者が質問を決め、電話の申込を代わりに入れ、承認すると部門のチームと選手ができる。
func TestEntries(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	tour := b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen })
	division := b.Division(tour.ID)
	questionsURL := fmt.Sprintf("/api/v1/admin/tournaments/%d/entry-questions", tour.ID)
	entriesURL := fmt.Sprintf("/api/v1/admin/tournaments/%d/entries", tour.ID)

	empty := f.call(t, http.StatusOK, http.MethodGet, questionsURL, nil)
	assert.Equal(t, []any{}, empty["questions"])
	saved := f.call(t, http.StatusOK, http.MethodPut, questionsURL, map[string]any{"questions": []any{
		map[string]any{"id": nil, "scope": "team", "label": "宿泊", "kind": "yes_no", "required": true,
			"options": []string{}},
		map[string]any{"id": nil, "scope": "team", "label": "弁当", "kind": "choice", "required": false,
			"options": []string{"要る", "要らない"}},
	}})
	questions, ok := saved["questions"].([]any)
	require.True(t, ok)
	require.Len(t, questions, 2)
	first, ok := questions[0].(map[string]any)
	require.True(t, ok)
	stay := first["id"]
	f.call(t, http.StatusBadRequest, http.MethodPut, questionsURL, map[string]any{"questions": []any{
		map[string]any{"id": nil, "scope": "team", "label": "x", "kind": "choice", "required": false,
			"options": []string{}},
	}})

	draft := map[string]any{"divisionId": division.ID, "fullName": "熱海ディスクス", "shortName": "熱海",
		"repName": "山田 太郎", "repPhone": "090-0000-0000", "repEmail": "",
		"members": []any{map[string]any{"name": "山田 太郎", "number": "7"}},
		"answers": []any{map[string]any{"questionId": stay, "value": "yes"}}}
	created := f.call(t, http.StatusCreated, http.MethodPost, entriesURL, draft)
	assert.Equal(t, "pending", created["status"])
	assert.Equal(t, "desk", created["channel"])
	assert.Len(t, created["inviteCode"], 26)
	f.call(t, http.StatusConflict, http.MethodPost, entriesURL, draft)

	list := f.call(t, http.StatusOK, http.MethodGet, entriesURL, nil)
	entries, ok := list["entries"].([]any)
	require.True(t, ok)
	require.Len(t, entries, 1)

	entryURL := fmt.Sprintf("/api/v1/admin/entries/%v", created["id"])
	detail := f.call(t, http.StatusOK, http.MethodGet, entryURL, nil)
	assert.Equal(t, "090-0000-0000", detail["repPhone"])
	f.call(t, http.StatusBadRequest, http.MethodPost, entryURL+"/decision", map[string]any{"action": "return"})
	approved := f.call(t, http.StatusOK, http.MethodPost, entryURL+"/decision", map[string]any{"action": "approve"})
	assert.Equal(t, "approved", approved["status"])
	require.NotNil(t, approved["teamId"])
	f.call(t, http.StatusConflict, http.MethodPost, entryURL+"/decision", map[string]any{"action": "cancel"})

	teams := f.call(t, http.StatusOK, http.MethodGet, fmt.Sprintf("/api/v1/admin/divisions/%d/teams", division.ID), nil)
	teamList, ok := teams["teams"].([]any)
	require.True(t, ok)
	require.Len(t, teamList, 1)

	// 申込には代表の連絡先が入るので、記録係には読ませない
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodGet, entriesURL, nil)
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodGet, entryURL, nil)
	f.call(t, http.StatusNotFound, http.MethodGet, "/api/v1/admin/tournaments/999999/entries", nil)
}
