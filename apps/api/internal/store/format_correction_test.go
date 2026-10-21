package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 確定後の訂正と、結果のある下流の試合(docs/format-engine.md「訂正の影響を先に見せる」)。

// finalRank はチームの最終順位(無ければ 0)。
func (f formatFixture) finalRank(t *testing.T, team int64) int {
	t.Helper()
	var row store.Team
	require.NoError(t, f.db.Where("id = ?", team).First(&row).Error)
	if row.FinalRank == nil {
		return 0
	}
	return *row.FinalRank
}

// playFirstPlacement は全プールを確定し、1st placement(E)を上の枠の勝ちで終える。
// E2 = teams[0] 対 teams[1]、E3 = teams[2] 対 teams[11]、決勝 E1 = teams[0] 対 teams[2]、
// 3 位決定 E4 = teams[1] 対 teams[11]。
func (f formatFixture) playFirstPlacement(t *testing.T) {
	t.Helper()
	f.finishPools(t)
	f.confirmPools(t)
	for _, code := range []string{"E2", "E3", "E1", "E4"} {
		f.finish(t, code, 5, 3)
	}
	require.Equal(t, 1, f.finalRank(t, f.teams[0].ID))
	require.Equal(t, 3, f.finalRank(t, f.teams[1].ID))
}

// reverseE2 は準決勝 E2 の勝者を逆にする訂正(teams[1] が勝つ)。
func (f formatFixture) reverseE2(t *testing.T, downstream store.Downstream) error {
	t.Helper()
	_, err := store.ApplyCorrection(context.Background(), f.db, f.actor, f.match(t, "E2").ID,
		store.CorrectionInput{Score: domain.Score{Left: 3, Right: 5}}, downstream)
	return err
}

func TestCorrectionNeedsADownstreamChoice(t *testing.T) {
	f := newFormatFixture(t)
	f.playFirstPlacement(t)
	err := f.reverseE2(t, store.DownstreamNone)
	require.ErrorIs(t, err, domain.ErrConflict)
	assert.Contains(t, err.Error(), "E1")
	assert.Equal(t, 5, *f.match(t, "E2").LeftScore, "承認なしでは保存しない")
}

// 下流の結果を残すと、決勝と 3 位決定は食い違いになり、その結果は最終順位に使わない(決勝に出て
// いないチームを優勝にしない)。本部が対戦チームを差し替えると、結果は新しいチームのものになる。
func TestCorrectionKeepingDownstreamWithholdsFinalRanks(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.playFirstPlacement(t)
	require.NoError(t, f.reverseE2(t, store.DownstreamKeep))

	e1 := f.match(t, "E1")
	assert.Equal(t, f.teams[0].ID, teamOf(e1.LeftTeamID), "結果のある決勝の対戦は変えない")
	assert.ElementsMatch(t, []string{"E1", "E4"}, f.mismatchCodes(t))
	for _, i := range []int{0, 1, 2, 11} {
		assert.Zero(t, f.finalRank(t, f.teams[i].ID), "食い違いが解けるまで teams[%d] の最終順位は書かない", i)
	}

	require.NoError(t, store.AlignMatchTeams(ctx, f.db, f.actor, e1.ID))
	require.NoError(t, store.AlignMatchTeams(ctx, f.db, f.actor, f.match(t, "E4").ID))
	assert.Empty(t, f.mismatchCodes(t))
	assert.Equal(t, 1, f.finalRank(t, f.teams[1].ID), "差し替えた決勝の左(teams[1])が 5-3 で勝った")
	assert.Equal(t, 2, f.finalRank(t, f.teams[2].ID))
	assert.Equal(t, 3, f.finalRank(t, f.teams[0].ID))
	assert.Equal(t, 4, f.finalRank(t, f.teams[11].ID))
}

// 下流の結果を未試合に戻すと、決勝と 3 位決定は訂正後の順位のチームで組み直され、最終順位は空になる。
func TestCorrectionResettingDownstream(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.playFirstPlacement(t)
	require.NoError(t, f.reverseE2(t, store.DownstreamReset))

	e1, e4 := f.match(t, "E1"), f.match(t, "E4")
	for _, m := range []store.Match{e1, e4} {
		assert.Equal(t, domain.MatchScheduled, m.Status, m.Code)
		assert.Nil(t, m.LeftScore, m.Code)
	}
	assert.Equal(t, []int64{f.teams[1].ID, f.teams[2].ID}, []int64{teamOf(e1.LeftTeamID), teamOf(e1.RightTeamID)})
	assert.Equal(t, []int64{f.teams[0].ID, f.teams[11].ID}, []int64{teamOf(e4.LeftTeamID), teamOf(e4.RightTeamID)})
	assert.Empty(t, f.mismatchCodes(t))
	assert.Zero(t, f.finalRank(t, f.teams[0].ID))
	entries, err := store.ListMatchAudit(ctx, f.db, e1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	assert.Equal(t, store.AuditMatchStatus, entries[0].Action)
	assert.Equal(t, domain.MatchScheduled, entries[0].After.Match.Status, "未試合に戻したことが監査ログに残る")
}

// 状態の変更(終了 → 試合中 → 終了)で同じ訂正をしても、承認を迂回できない。試合中へ戻すときに
// 承認し、終えるときは(下流の扱いが決まっているので)承認は要らない。
func TestReopeningAMatchNeedsTheSameApproval(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.playFirstPlacement(t)
	e2 := f.match(t, "E2").ID
	_, err := store.ChangeMatchStatus(ctx, f.db, f.actor, e2, store.StatusInput{Status: domain.MatchLive})
	require.ErrorIs(t, err, domain.ErrConflict)
	assert.Equal(t, domain.MatchFinished, f.match(t, "E2").Status)

	_, err = store.ChangeMatchStatus(ctx, f.db, f.actor, e2,
		store.StatusInput{Status: domain.MatchLive, Downstream: store.DownstreamKeep})
	require.NoError(t, err)
	score := domain.Score{Left: 3, Right: 5}
	_, err = store.ChangeMatchStatus(ctx, f.db, f.actor, e2,
		store.StatusInput{Status: domain.MatchFinished, Score: &score})
	require.NoError(t, err)
	assert.Zero(t, f.finalRank(t, f.teams[1].ID), "決勝に出ていない teams[1] を優勝にしない")
	assert.ElementsMatch(t, []string{"E1", "E4"}, f.mismatchCodes(t))
}

// 得点経過のある試合も、終了の取り消し(end の得点経過の取り消し)で承認を迂回できない。
func TestRemovingTheEndEventNeedsApproval(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finishPools(t)
	f.confirmPools(t)
	e2 := f.match(t, "E2").ID
	now := time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)
	left := domain.SideLeft
	for _, kind := range []domain.EventKind{domain.EventGoal, domain.EventEnd} {
		event := store.EventInput{NewEvent: domain.NewEvent{Kind: kind}}
		if kind == domain.EventGoal {
			event.Side = &left
		}
		_, err := store.AddMatchEvent(ctx, f.db, f.actor, e2, event, now)
		require.NoError(t, err)
	}
	for _, code := range []string{"E3", "E1"} {
		f.finish(t, code, 5, 3)
	}
	detail, err := store.GetMatchDetail(ctx, f.db, e2)
	require.NoError(t, err)
	end := detail.Events[len(detail.Events)-1]
	_, err = store.RemoveMatchEvent(ctx, f.db, f.actor, e2, end.ID)
	require.ErrorIs(t, err, domain.ErrConflict, "決勝に結果があるので承認が要る")
	assert.Equal(t, domain.MatchFinished, f.match(t, "E2").Status)
}

// 対戦チームが決まる前の試合には、どの経路でも結果を入れない。
func TestNoResultBeforeTeamsAreKnown(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	e2 := f.match(t, "E2").ID
	score := domain.Score{Left: 1, Right: 0}
	left := domain.SideLeft
	_, err := store.ChangeMatchStatus(ctx, f.db, f.actor, e2,
		store.StatusInput{Status: domain.MatchFinished, Score: &score})
	require.ErrorIs(t, err, domain.ErrConflict)
	_, err = store.ChangeMatchStatus(ctx, f.db, f.actor, e2, store.StatusInput{Status: domain.MatchLive})
	require.ErrorIs(t, err, domain.ErrConflict)
	_, err = store.SetMatchResult(ctx, f.db, f.actor, e2,
		store.ResultInput{Type: domain.ResultForfeit, WinnerSide: &left})
	require.ErrorIs(t, err, domain.ErrConflict)
	_, err = store.AddMatchEvent(ctx, f.db, f.actor, e2,
		store.EventInput{NewEvent: domain.NewEvent{Kind: domain.EventGoal, Side: &left}}, time.Now())
	require.ErrorIs(t, err, domain.ErrConflict)
	assert.Equal(t, domain.MatchScheduled, f.match(t, "E2").Status)
}

// チームなしで結果だけが入っていた試合(以前のデータ)は食い違いとして出し、勝者を決勝へ流さない。
// 差し替えると、その結果を新しいチームのものとして決勝へ流す。
func TestAligningALegacyResultAdvancesTheWinner(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	e2 := f.match(t, "E2").ID
	require.NoError(t, f.db.Model(&store.Match{}).Where("id = ?", e2).Updates(map[string]any{
		"status": domain.MatchFinished, "left_score": 1, "right_score": 0}).Error)
	f.finishPools(t)
	f.confirmPools(t)
	assert.Contains(t, f.mismatchCodes(t), "E2")
	assert.Nil(t, f.match(t, "E1").LeftTeamID, "食い違う E2 の勝者は決勝へ流さない")

	require.NoError(t, store.AlignMatchTeams(ctx, f.db, f.actor, e2))
	assert.Equal(t, f.teams[0].ID, teamOf(f.match(t, "E1").LeftTeamID))
	assert.NotContains(t, f.mismatchCodes(t), "E2")
}

// 対戦チームが食い違う試合のあるプールは確定しない(未消化のまま確定でも)。
func TestPoolWithAMismatchIsNotConfirmed(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finishPools(t)
	a1 := f.match(t, "A1").ID
	require.NoError(t, f.db.Model(&store.Match{}).Where("id = ?", a1).
		Update("left_team_id", f.teams[3].ID).Error)
	poolA := f.groupID(t, "Qualifying pool A")
	err := store.ConfirmGroup(ctx, f.db, f.actor, poolA, true, fixtureNow())
	require.ErrorIs(t, err, domain.ErrConflict)
	assert.Contains(t, err.Error(), "A1")
}
