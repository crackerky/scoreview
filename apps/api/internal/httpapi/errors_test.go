package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 400 と 409 の文面に Go の型名と DB の制約名を出さない(セキュリティ点検 I-2)。
func TestErrorMessagesHideInternals(t *testing.T) {
	f := newAdminFixture(t)
	tournament := id(f.call(t, http.StatusCreated, http.MethodPost, "/api/v1/admin/tournaments",
		map[string]any{"name": map[string]string{"ja": "大会"}})["id"])
	path := fmt.Sprintf("/api/v1/admin/tournaments/%d", tournament)
	user := map[string]any{
		"email": "dup@example.com", "role": "staff", "password": "a long staff password",
	}
	f.call(t, http.StatusCreated, http.MethodPost, "/api/v1/admin/users", user)

	for _, tc := range []struct {
		method, path string
		body         any
		status       int
		message      string
	}{
		{http.MethodPatch, path, `{"name": 1}`, http.StatusBadRequest, "name: must be an object"},
		{http.MethodPatch, path, `{"name": {"ja": 1}}`, http.StatusBadRequest, "name.ja: must be a string"},
		{http.MethodPatch, path, `{"bogus": 1}`, http.StatusBadRequest, `body: unknown key "bogus"`},
		{http.MethodPatch, path, `{"name":`, http.StatusBadRequest, "body: must be a single valid JSON value"},
		{http.MethodPost, "/api/v1/admin/users", user, http.StatusConflict,
			"conflict: create user: the same value is already registered"},
	} {
		rec := f.do(t, f.adminToken, tc.method, tc.path, tc.body)
		require.Equal(t, tc.status, rec.Code, rec.Body.String())
		var body struct{ Error struct{ Message string } }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, tc.message, body.Error.Message)
		for _, internal := range []string{"Go struct", "httpapi.", "DTO", "_key", "_fkey", "_check"} {
			assert.NotContains(t, body.Error.Message, internal)
		}
	}
}
