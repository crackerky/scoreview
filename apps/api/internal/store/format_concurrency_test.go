package store_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 同じディビジョンへの同時の操作。割り込みの順序は、別のトランザクションで行ロックを握って決める
// (lockRows)。どの操作もディビジョンの行を先にロックするので、後から来た操作は先の操作の結果を
// 見てから動く。

// プールの最後の試合の終了と「未消化のまま確定」が同時に走っても、確定は最後の試合を数えた順位で
// 下流の枠を埋める(古い順位のまま残らない)。
func TestConcurrentFinishAndConfirmUseTheLastResult(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finish(t, "A1", 3, 1)
	f.finish(t, "A2", 4, 1)
	a3 := f.match(t, "A3").ID
	_, err := store.ChangeMatchStatus(ctx, f.db, f.actor, a3, store.StatusInput{Status: domain.MatchLive})
	require.NoError(t, err)
	poolA := f.groupID(t, "Qualifying pool A")

	// A3 の終了は group_teams の書き込みで止まる(ディビジョンのロックを握ったまま)
	blocker := lockRows(t, f.db, &[]store.GroupTeam{}, "group_id = ?", poolA)
	var wg sync.WaitGroup
	var finishErr, confirmErr error
	wg.Go(func() {
		score := domain.Score{Left: 1, Right: 2} // 枠 3(teams[8])が枠 2(teams[7])に勝つ
		_, finishErr = store.ChangeMatchStatus(ctx, f.db, f.actor, a3,
			store.StatusInput{Status: domain.MatchFinished, Score: &score})
	})
	waitLockWaiters(t, f.db, 1)
	// 確定はディビジョンのロックを待つ
	wg.Go(func() {
		confirmErr = store.ConfirmGroup(ctx, f.db, f.actor, poolA, true, fixtureNow())
	})
	waitLockWaiters(t, f.db, 2)
	require.NoError(t, blocker.Commit().Error)
	wg.Wait()
	require.NoError(t, finishErr)
	require.NoError(t, confirmErr)

	f2 := f.match(t, "F2")
	assert.Equal(t, f.teams[8].ID, teamOf(f2.LeftTeamID), "A3 を数えた 2 位(teams[8])が 5th の枠に入る")
	var unplayed bool
	require.NoError(t, f.db.Table("groups").Where("id = ?", poolA).
		Pluck("confirmed_unplayed", &unplayed).Error)
	assert.False(t, unplayed, "全試合が終わった後の確定なので未消化の印は付かない")
}

// 同じプールの 2 試合が同時に終わっても、どちらも保存できる(group_teams の作り直しで主キーが
// 重複しない)。
func TestConcurrentResultsInOnePool(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	poolA := f.groupID(t, "Qualifying pool A")
	blocker := lockRows(t, f.db, &[]store.GroupTeam{}, "group_id = ?", poolA)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, code := range []string{"A1", "A2"} {
		id := f.match(t, code).ID
		wg.Go(func() {
			score := domain.Score{Left: 3, Right: 1}
			_, errs[i] = store.ChangeMatchStatus(ctx, f.db, f.actor, id,
				store.StatusInput{Status: domain.MatchFinished, Score: &score})
		})
	}
	waitLockWaiters(t, f.db, 2)
	require.NoError(t, blocker.Commit().Error)
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])

	var members []store.GroupTeam
	require.NoError(t, f.db.Where("group_id = ?", poolA).Order("sort_order").Find(&members).Error)
	require.Len(t, members, 3)
	require.NotNil(t, members[0].RankOverride)
	assert.Equal(t, 1, *members[0].RankOverride, "2 勝のシード 1 が 1 位")
}

// 下流の試合が始まった直後に上流の訂正が走っても、試合中の試合の対戦チームは差し替えず、
// 食い違いとして出す。blocker は試合の行だけをロックして試合中にする経路を模す(UPDATE の
// status の条件だけで守れることを確かめる)。
func TestCorrectionDoesNotOverwriteMatchStartedMeanwhile(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finishPools(t)
	f.confirmPools(t)
	e2 := f.match(t, "E2")
	require.Equal(t, f.teams[0].ID, teamOf(e2.LeftTeamID))

	blocker := lockRows(t, f.db, &[]store.Match{}, "id = ?", e2.ID)
	require.NoError(t, blocker.Model(&store.Match{}).Where("id = ?", e2.ID).
		Update("status", domain.MatchLive).Error)
	var corrErr error
	var wg sync.WaitGroup
	wg.Go(func() {
		// A1 を逆にすると A の 1 位が teams[7] になる
		_, corrErr = store.ApplyCorrection(ctx, f.db, f.actor, f.match(t, "A1").ID,
			store.CorrectionInput{Score: domain.Score{Left: 1, Right: 3}}, true)
	})
	waitLockWaiters(t, f.db, 1)
	require.NoError(t, blocker.Commit().Error)
	wg.Wait()
	require.NoError(t, corrErr)

	e2 = f.match(t, "E2")
	assert.Equal(t, domain.MatchLive, e2.Status)
	assert.Equal(t, f.teams[0].ID, teamOf(e2.LeftTeamID), "試合中の E2 の左は変えない")
	assert.Contains(t, f.mismatchCodes(t), "E2")
}
