package store_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/testdb"
)

// 失敗した問い合わせのログに SQL の値(パスワードのハッシュ、メールアドレス、名前)を出さない
// (セキュリティ点検 M-3)。
func TestLoggerHidesQueryValues(t *testing.T) {
	var buf bytes.Buffer
	db := testdb.New(t).Session(&gorm.Session{Logger: store.NewLogger(&buf)})
	ctx := context.Background()
	const email = "secret-person@example.com"
	const hash = "$argon2id$secret-hash-value"
	fields := store.UserFields{Email: email, DisplayName: "Secret Name", Role: domain.RoleStaff}
	_, err := store.CreateUser(ctx, db, fields, hash)
	require.NoError(t, err)
	_, err = store.CreateUser(ctx, db, fields, hash)
	require.ErrorIs(t, err, domain.ErrConflict)

	logged := buf.String()
	assert.Contains(t, logged, `INSERT INTO "users"`)
	assert.NotContains(t, logged, email)
	assert.NotContains(t, logged, hash)
	assert.NotContains(t, logged, "Secret Name")
}
