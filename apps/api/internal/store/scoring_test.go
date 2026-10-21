package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

func strPtr(value string) *string { return &value }

// 同じ clientId の得点は 2 回送っても 1 件だけ記録し、監査ログも 1 行。
func TestAddMatchEventIsIdempotentByClientID(t *testing.T) {
	f := newMatchFixture(t)
	ctx := context.Background()
	input := goal(domain.SideLeft, &f.l7, nil)
	input.ClientID = strPtr("6f1c2b3a-1111-4c2d-8e9f-000000000001")
	for range 2 {
		detail, err := store.AddMatchEvent(ctx, f.db, staff, f.match.ID, input, t0)
		require.NoError(t, err)
		require.Len(t, detail.Events, 1)
		assert.Equal(t, *input.ClientID, detail.Events[0].ClientID)
		assert.Equal(t, 1, *detail.Match.Left.Score)
	}
	audit, err := store.ListMatchAudit(ctx, f.db, f.match.ID)
	require.NoError(t, err)
	require.Len(t, audit, 1)
	assert.Equal(t, store.AuditEventAdd, audit[0].Action)
	assert.Equal(t, domain.MatchScheduled, audit[0].Before.Match.Status)
	assert.Equal(t, domain.MatchLive, audit[0].After.Match.Status)
	assert.Equal(t, "7", audit[0].After.Event.Scorer.Number)

	// clientId の無い得点経過も DB が UUID を付ける(端末から後で指せる)
	detail, err := store.AddMatchEvent(ctx, f.db, staff, f.match.ID,
		goal(domain.SideRight, nil, nil), t0)
	require.NoError(t, err)
	assert.True(t, domain.IsClientID(detail.Events[1].ClientID))
}

func TestUpdateMatchEventAfterFinish(t *testing.T) {
	f := newMatchFixture(t)
	ctx := context.Background()
	_, err := store.AddMatchEvent(ctx, f.db, staff, f.match.ID, goal(domain.SideLeft, nil, nil), t0)
	require.NoError(t, err)
	detail, err := store.AddMatchEvent(ctx, f.db, staff, f.match.ID, endEvent(nil), t0)
	require.NoError(t, err)
	ref := store.EventRef{ClientID: detail.Events[0].ClientID}

	// 終了した試合でも得点者とアシストは後から付けられる(スコアと状態は変わらない)
	detail, err = store.UpdateMatchEvent(ctx, f.db, staff, f.match.ID, ref,
		store.EventPlayersInput{ScorerID: &f.l7.ID, AssistID: &f.l10.ID})
	require.NoError(t, err)
	assert.Equal(t, "7", detail.Events[0].Scorer.Number)
	assert.Equal(t, domain.MatchFinished, detail.Match.Status)

	// 相手チームの選手、得点以外の行は付けられない
	_, err = store.UpdateMatchEvent(ctx, f.db, staff, f.match.ID, ref,
		store.EventPlayersInput{ScorerID: &f.r1.ID})
	require.ErrorIs(t, err, domain.ErrBadRequest)
	_, err = store.UpdateMatchEvent(ctx, f.db, staff, f.match.ID,
		store.EventRef{ID: detail.Events[1].ID}, store.EventPlayersInput{})
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

// 結果の種類ごとの集計(不戦勝は規定スコアで勝敗、両チーム不戦敗は両方負け、不成立は勝敗のみ、
// 未消化は集計に入れない)をチームページの成績で確かめる。
func TestResultTypesAreTallied(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	b := storetest.New(t, db)
	tour := b.Tournament(func(row *store.Tournament) {
		row.ForfeitWinnerScore, row.ForfeitLoserScore = storetest.Ptr(15), storetest.Ptr(0)
	})
	division := b.Division(tour.ID)
	us := b.Team(division)
	them := b.Team(division)
	newMatch := func() int64 { return b.Match(division, &us, &them).ID }

	_, err := store.SetMatchResult(ctx, db, staff, newMatch(), store.ResultInput{
		Type: domain.ResultForfeit, WinnerSide: sidePtr(domain.SideLeft),
	})
	require.NoError(t, err)
	_, err = store.SetMatchResult(ctx, db, staff, newMatch(), store.ResultInput{
		Type: domain.ResultDoubleForfeit,
	})
	require.NoError(t, err)
	_, err = store.SetMatchResult(ctx, db, staff, newMatch(), store.ResultInput{
		Type: domain.ResultNoContest, WinnerSide: sidePtr(domain.SideRight),
	})
	require.NoError(t, err)
	unplayed := newMatch()
	_, err = store.AddMatchEvent(ctx, db, staff, unplayed, goal(domain.SideLeft, nil, nil), t0)
	require.NoError(t, err)
	detail, err := store.SetMatchResult(ctx, db, staff, unplayed, store.ResultInput{
		Type: domain.ResultUnplayed,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.ResultUnplayed, detail.Match.ResultType)

	// 規定スコアが大会の設定と違う不戦勝は受け付けない
	_, err = store.SetMatchResult(ctx, db, staff, newMatch(), store.ResultInput{
		Type: domain.ResultForfeit, WinnerSide: sidePtr(domain.SideLeft),
		ForfeitScore: &domain.ForfeitScore{Winner: 13, Loser: 0},
	})
	require.ErrorIs(t, err, domain.ErrBadRequest)

	page, err := store.GetTeam(ctx, db, us.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.Tally{Games: 3, Wins: 1, Losses: 2, GoalsFor: 15}, page.Record)
	page, err = store.GetTeam(ctx, db, them.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.Tally{Games: 3, Wins: 1, Losses: 2, GoalsAgainst: 15}, page.Record)
}

func TestFinalRankChangesAreAudited(t *testing.T) {
	f := newMatchFixture(t)
	ctx := context.Background()
	_, err := store.SetFinalRanks(ctx, f.db, admin, f.division.ID, []store.FinalRank{
		{TeamID: f.left.ID, Rank: storetest.Ptr(1)},
		{TeamID: f.right.ID, Rank: nil}, // 変わらないので記録しない
	})
	require.NoError(t, err)
	var rows []store.AuditLog
	require.NoError(t, f.db.Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, store.AuditFinalRank, rows[0].Action)
	assert.Equal(t, f.left.ID, *rows[0].TeamID)
	assert.JSONEq(t, `{}`, string(rows[0].Before))
	assert.JSONEq(t, `{"finalRank":1}`, string(rows[0].After))
}

func TestStaffCanScoreOnlyAssignedMatches(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	b := storetest.New(t, db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	left, right := b.Team(division), b.Team(division)
	court3 := b.Match(division, &left, &right, storetest.At(t, "2026-10-18", "10:30", "3"))
	court4 := b.Match(division, &left, &right, storetest.At(t, "2026-10-18", "09:00", "4"))
	user, err := store.CreateUser(ctx, db,
		store.UserFields{Email: "rec@example.com", Role: domain.RoleStaff}, "hash")
	require.NoError(t, err)

	ok, err := store.CanScoreMatch(ctx, db, user.ID, court3.ID)
	require.NoError(t, err)
	assert.False(t, ok, "割り当てが無ければ入力できない")

	_, err = store.CreateAssignment(ctx, db, tour.ID, store.AssignmentFields{
		UserID: user.ID,
		Assignment: domain.Assignment{
			Field: strPtr("3"), From: strPtr("10:00"), To: strPtr("12:00"),
		},
	})
	require.NoError(t, err)
	ok, err = store.CanScoreMatch(ctx, db, user.ID, court3.ID)
	require.NoError(t, err)
	assert.True(t, ok, "割り当てると大会の担当にもなる")
	ok, err = store.CanScoreMatch(ctx, db, user.ID, court4.ID)
	require.NoError(t, err)
	assert.False(t, ok)

	mine, err := store.ListAssignedMatches(ctx, db, user.ID)
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, court3.ID, mine[0].Match.ID)
}
