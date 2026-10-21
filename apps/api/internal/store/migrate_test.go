package store_test

import (
	"context"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/store"
	"scoreview/api/internal/testdb"
)

// TestMigrationsRoundTrip は最新のマイグレーションを 1 つ戻してから当て直せることを確かめる
// (Down が壊れていると、適用済みの環境で戻せなくなるため)。
func TestMigrationsRoundTrip(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	version, err := goose.GetDBVersionContext(ctx, sqlDB)
	require.NoError(t, err)
	require.NoError(t, goose.DownToContext(ctx, sqlDB, ".", version-1))
	require.NoError(t, store.Migrate(ctx, db))
	after, err := goose.GetDBVersionContext(ctx, sqlDB)
	require.NoError(t, err)
	require.Equal(t, version, after)
}
