package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/httpapi"
	"scoreview/api/internal/testdb"
)

const testToken = "test-internal-token-0123456789abcdef"

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

func TestInternalTokenIsRequired(t *testing.T) {
	srv := httpapi.NewServer(testdb.New(t), testToken)

	cases := []struct {
		name   string
		token  string
		status int
	}{
		{"missing token", "", http.StatusUnauthorized},
		{"wrong token", "wrong", http.StatusUnauthorized},
		{"valid token", testToken, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
			if tc.token != "" {
				req.Header.Set("X-Internal-Token", tc.token)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			assert.Equal(t, tc.status, rec.Code)
		})
	}
}

func TestUnauthorizedBodyShape(t *testing.T) {
	srv := httpapi.NewServer(testdb.New(t), testToken)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.JSONEq(t, `{"error":{"code":"unauthorized","message":"unauthorized"}}`, rec.Body.String())
}
