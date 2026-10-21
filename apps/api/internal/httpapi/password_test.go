package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 記録係も自分のパスワードを変えられる。変更に使ったセッションは残し、他の端末のセッションを消す
// (セキュリティ点検 L-8)。
func TestChangeOwnPassword(t *testing.T) {
	f := newAdminFixture(t)
	other := f.login(t, "staff@example.com") // 同じ記録係の別の端末
	change := func(current, next string) *httptest.ResponseRecorder {
		return f.do(t, f.staffToken, http.MethodPost, "/api/v1/auth/password",
			map[string]string{"currentPassword": current, "newPassword": next})
	}

	for _, tc := range []struct{ current, next, field string }{
		{"wrong password!", "a brand new staff password", "currentPassword"},
		{adminPassword, "aaaaaaaaaaaa", "newPassword"},
		{adminPassword, "1qaz2wsx3edc", "newPassword"},
		{adminPassword, "short", "newPassword"},
		{adminPassword, adminPassword, "newPassword"},
	} {
		rec := change(tc.current, tc.next)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), tc.field)
	}

	rec := change(adminPassword, "a brand new staff password")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	// このセッションは残り、他の端末は締め出される
	assert.Equal(t, http.StatusOK, f.do(t, f.staffToken, http.MethodGet, "/api/v1/auth/me", nil).Code)
	assert.Equal(t, http.StatusUnauthorized, f.do(t, other, http.MethodGet, "/api/v1/auth/me", nil).Code)
	// 古いパスワードでは入れず、新しいパスワードで入れる
	login := func(password string) int {
		return f.do(t, "", http.MethodPost, "/api/v1/auth/login",
			map[string]string{"email": "staff@example.com", "password": password}).Code
	}
	assert.Equal(t, http.StatusUnauthorized, login(adminPassword))
	assert.Equal(t, http.StatusOK, login("a brand new staff password"))
	// ログインしていなければ変えられない
	assert.Equal(t, http.StatusUnauthorized, f.do(t, "", http.MethodPost, "/api/v1/auth/password",
		map[string]string{"currentPassword": "x", "newPassword": "y"}).Code)
}
