package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/store"
)

// 大会形式の訂正と並行性のテストの共通部分。

// fixtureNow は進行画面の「いま」(予定時刻の判定だけに使う)。
func fixtureNow() time.Time { return time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC) }

// confirmPools は finishPools の後に全プールを確定する。D は 3 チームの未決をシード 12 → 4 → 5 で決める。
func (f formatFixture) confirmPools(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	for _, name := range []string{"Qualifying pool A", "Qualifying pool B", "Qualifying pool C"} {
		require.NoError(t, store.ConfirmGroup(ctx, f.db, f.actor, f.groupID(t, name), false, fixtureNow()))
	}
	poolD := f.groupID(t, "Qualifying pool D")
	d := []int64{f.teams[11].ID, f.teams[3].ID, f.teams[4].ID}
	require.NoError(t, store.DecideRank(ctx, f.db, f.actor, poolD,
		store.RankDecisionInput{Teams: d, Reason: "ディスクフリップ"}))
	require.NoError(t, store.ConfirmGroup(ctx, f.db, f.actor, poolD, false, fixtureNow()))
}

// todo は進行画面の「いまやること」のうち kind のもの。
func (f formatFixture) todo(t *testing.T, kind store.TodoKind) []store.TodoItem {
	t.Helper()
	progress, err := store.GetFormatProgress(context.Background(), f.db, f.division.ID, fixtureNow())
	require.NoError(t, err)
	var out []store.TodoItem
	for _, item := range progress.Todo {
		if item.Kind == kind {
			out = append(out, item)
		}
	}
	return out
}

// mismatchCodes は食い違いとして出ている試合の CD。
func (f formatFixture) mismatchCodes(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, item := range f.todo(t, store.TodoMismatch) {
		require.NotNil(t, item.MatchID)
		var m store.Match
		require.NoError(t, f.db.Where("id = ?", *item.MatchID).First(&m).Error)
		out = append(out, m.Code)
	}
	return out
}

// lockRows は別のトランザクションで model の行(where に合うもの)を FOR UPDATE でロックしたまま返す。
// 呼び出し側が Commit するまで、その行を書く操作は待つ(割り込みの順序をテストで決めるため)。
func lockRows(t *testing.T, db *gorm.DB, model any, where string, args ...any) *gorm.DB {
	t.Helper()
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })
	require.NoError(t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(where, args...).Find(model).Error)
	return tx
}

// waitLockWaiters は、このテストの DB で行ロックを待っている接続が n 以上になるまで待つ。
func waitLockWaiters(t *testing.T, db *gorm.DB, n int64) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int64
		require.NoError(t, db.Table("pg_stat_activity").
			Where("datname = current_database() AND wait_event_type = ?", "Lock").
			Count(&waiting).Error)
		if waiting >= n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%d 件の待ちになりませんでした", n)
}
