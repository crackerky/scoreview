package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

func TestReadPassword(t *testing.T) {
	got, err := readPassword("from env", strings.NewReader("ignored\n"))
	require.NoError(t, err)
	assert.Equal(t, "from env", got)

	got, err = readPassword("", strings.NewReader("from stdin  \r\nsecond line\n"))
	require.NoError(t, err)
	assert.Equal(t, "from stdin  ", got, "前後の空白はパスワードの一部として残す")

	got, err = readPassword("", strings.NewReader("no newline"))
	require.NoError(t, err)
	assert.Equal(t, "no newline", got)

	_, err = readPassword("", strings.NewReader(""))
	assert.Error(t, err)
}

func TestCreateAdmin(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	user, err := createAdmin(ctx, db, " Admin@Example.com", "管理者", "a long admin password")
	require.NoError(t, err)
	assert.Equal(t, domain.RoleAdmin, user.Role)
	assert.Equal(t, "admin@example.com", user.Email)

	gate := auth.NewPasswordGate(1, time.Second)
	attempt := auth.Attempt{Email: "admin@example.com", Password: "a long admin password"}
	session, err := auth.Login(ctx, db, gate, attempt, time.Now())
	require.NoError(t, err)
	assert.Equal(t, user.ID, session.User.ID)

	_, err = createAdmin(ctx, db, "other@example.com", "", "short")
	assert.ErrorIs(t, err, domain.ErrBadRequest)
	_, err = createAdmin(ctx, db, "admin@example.com", "", "a long admin password")
	assert.ErrorIs(t, err, domain.ErrConflict)
}
