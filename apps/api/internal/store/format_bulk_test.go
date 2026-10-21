package store_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 本部の一括操作(棄権、大会の締め、中断と再開)のテスト。レビュー H3、H10、H2、M5。

func ids(matches []store.WithdrawalMatch) []int64 {
	out := make([]int64, len(matches))
	for i, m := range matches {
		out[i] = m.ID
	}
	return out
}

func auditCount(t *testing.T, f formatFixture, action string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, f.db.Model(&store.AuditLog{}).Where("action = ?", action).Count(&n).Error)
	return n
}

// 棄権: 残りの試合を相手の不戦勝として一括で記録し、監査ログに「棄権の一括処理」と残す。後で順位決定戦に
// 入ったら「いまやること」に出す。
func TestWithdrawTeam(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	t08 := f.teams[7] // 予選 A 組(シード 1、8、9)

	plan, err := store.GetWithdrawal(ctx, f.db, t08.ID)
	require.NoError(t, err)
	require.Len(t, plan.Targets, 2, "A 組の残り 2 試合")
	assert.Nil(t, plan.WithdrawnAt)
	require.NotNil(t, plan.ForfeitScore)

	_, err = store.WithdrawTeam(ctx, f.db, f.actor, t08.ID, ids(plan.Targets)[:1], fixtureNow())
	require.ErrorIs(t, err, domain.ErrConflict, "確かめた試合と違えば何もしない")
	done, err := store.WithdrawTeam(ctx, f.db, f.actor, t08.ID, ids(plan.Targets), fixtureNow())
	require.NoError(t, err)
	assert.NotNil(t, done.WithdrawnAt)
	assert.Empty(t, done.Targets)

	for _, code := range []string{"A1", "A3"} {
		m := f.match(t, code)
		assert.Equal(t, domain.ResultForfeit, m.ResultType, code)
		require.NotNil(t, m.LeftScore, code)
		winner := *m.RightTeamID
		if *m.LeftScore > *m.RightScore {
			winner = *m.LeftTeamID
		}
		assert.NotEqual(t, t08.ID, winner, "%s は相手の不戦勝", code)
	}
	audit, err := store.ListMatchAudit(ctx, f.db, f.match(t, "A1").ID)
	require.NoError(t, err)
	require.NotEmpty(t, audit)
	assert.Equal(t, store.AuditMatchWithdraw, audit[0].Action)
	assert.Equal(t, t08.ID, *audit[0].TeamID)
	assert.Contains(t, audit[0].After.Note, "棄権の一括処理")

	// 残りのプールを終えて確定すると、T08 は A 組 3 位で 9〜12 位のトーナメントに入る
	f.finish(t, "A2", 4, 1)
	for _, letter := range []string{"B", "C"} {
		f.finish(t, letter+"1", 3, 1)
		f.finish(t, letter+"2", 4, 1)
		f.finish(t, letter+"3", 2, 1)
	}
	for _, code := range []string{"D1", "D2", "D3"} {
		f.finish(t, code, 2, 2)
	}
	f.confirmPools(t)
	todo := f.todo(t, store.TodoWithdrawn)
	require.Len(t, todo, 1, "棄権したチームが入った試合を知らせる")
	assert.Equal(t, t08.ID, *todo[0].TeamID)
	assert.Contains(t, todo[0].Text, "G2")
	again, err := store.GetWithdrawal(ctx, f.db, t08.ID)
	require.NoError(t, err)
	require.Len(t, again.Targets, 1)
	_, err = store.WithdrawTeam(ctx, f.db, f.actor, t08.ID, ids(again.Targets), fixtureNow())
	require.NoError(t, err)
	assert.Empty(t, f.todo(t, store.TodoWithdrawn))
	assert.Equal(t, domain.MatchFinished, f.match(t, "G2").Status)
	assert.NotNil(t, f.match(t, "G1").LeftTeamID, "不戦勝の勝者が決勝へ進む")
}

// 大会の締め: 残りの試合をまとめて未消化にし、決まらない順位を同順位で提案して、公開まで進める。
func TestCloseDivision(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finishPools(t)
	f.confirmPools(t)
	for _, code := range []string{"E2", "E3", "E1", "E4", "F2", "F3"} {
		f.finish(t, code, 5, 3)
	}

	preview, err := store.GetClosing(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	var codes []string
	var remaining []int64
	for _, m := range preview.Remaining {
		codes = append(codes, m.Code)
		remaining = append(remaining, m.ID)
	}
	assert.ElementsMatch(t, []string{"F1", "F4", "G1", "G2", "G3", "G4"}, codes,
		"相手が未定の G1、G4 も未消化にする")
	ties := strings.Join(preview.Ties, "\n")
	assert.Contains(t, ties, "5〜6 位は 5 位タイ")
	assert.Contains(t, ties, "7〜8 位は 7 位タイ")
	assert.Contains(t, ties, "9〜12 位は 9 位タイ")
	require.Len(t, preview.Ranks, 12)
	assert.Nil(t, preview.ClosedAt)
	assert.False(t, preview.Published)

	_, err = store.CloseDivision(ctx, f.db, f.actor, f.division.ID, remaining[:2], true, fixtureNow())
	require.ErrorIs(t, err, domain.ErrConflict, "確かめた試合と違えば何もしない")
	closed, err := store.CloseDivision(ctx, f.db, f.actor, f.division.ID, remaining, true, fixtureNow())
	require.NoError(t, err)
	assert.Empty(t, closed.Remaining)
	assert.NotNil(t, closed.ClosedAt)
	assert.True(t, closed.Published)

	want := map[int64]int{}
	for _, r := range preview.Ranks {
		want[r.Team.ID] = r.Rank
	}
	var teams []store.Team
	require.NoError(t, f.db.Where("division_id = ?", f.division.ID).Find(&teams).Error)
	for _, team := range teams {
		require.NotNil(t, team.FinalRank, team.NameShortJa)
		assert.Equal(t, want[team.ID], *team.FinalRank, "確認画面の提案のとおり(%s)", team.NameShortJa)
	}
	var division store.Division
	require.NoError(t, f.db.Where("id = ?", f.division.ID).First(&division).Error)
	assert.True(t, division.StandingsPublished, "そのまま公開まで進める")
	g1 := f.match(t, "G1")
	assert.Equal(t, domain.ResultUnplayed, g1.ResultType)
	assert.Empty(t, f.mismatchCodes(t), "未消化の試合を食い違いにしない")
	assert.Equal(t, int64(1), auditCount(t, f, store.AuditDivisionClose))

	progress, err := store.GetFormatProgress(ctx, f.db, f.division.ID, fixtureNow())
	require.NoError(t, err)
	assert.Len(t, progress.Final, 12, "進行画面の最終順位にも締めた順位を出す")
	assert.NotNil(t, progress.ClosedAt)
}

// トーナメントの「時間切れで未消化のまま確定」は、残りの試合(相手が未定の試合も)を未消化にする。
func TestBracketTimeUp(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finishPools(t)
	f.confirmPools(t)
	g := f.groupID(t, "9th placement tournament")
	require.ErrorIs(t, store.ConfirmGroup(ctx, f.db, f.actor, g, false, fixtureNow()), domain.ErrConflict)
	require.NoError(t, store.ConfirmGroup(ctx, f.db, f.actor, g, true, fixtureNow()))
	for _, code := range []string{"G1", "G2", "G3", "G4"} {
		assert.Equal(t, domain.ResultUnplayed, f.match(t, code).ResultType, code)
	}
	closing, err := store.GetClosing(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	assert.Contains(t, strings.Join(closing.Ties, "\n"), "9〜12 位は 9 位タイ")
}

// 中断と再開: 中断中は閲覧ページの帯の情報を返し、再開で中断した分だけ未開始の試合をずらす。
func TestSuspendAndResume(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	tournament := f.division.TournamentID
	f.finish(t, "A1", 3, 1)
	a1Before, a2Before := f.match(t, "A1"), f.match(t, "A2")
	// 終了時刻 10:00 の時間割にする(ずらすと超える試合を知らせる)
	require.NoError(t, f.db.Model(&store.DivisionFormat{}).Where("division_id = ?", f.division.ID).
		Update("schedule", `{"date":"2026-10-18","start":"09:00","slotMinutes":20,"courts":["1","2","3","4"],`+
			`"teamRestSlots":0,"stageGapSlots":0,"end":"10:00"}`).Error)

	_, err := store.ResumeTournament(ctx, f.db, f.actor, tournament, nil, fixtureNow())
	require.ErrorIs(t, err, domain.ErrConflict, "中断していなければ再開できない")
	resume := "13:30"
	since := fixtureNow()
	_, err = store.SuspendTournament(ctx, f.db, f.actor, tournament,
		store.SuspensionInput{Reason: "lightning"}, since)
	require.ErrorIs(t, err, domain.ErrBadRequest)
	_, err = store.SuspendTournament(ctx, f.db, f.actor, tournament,
		store.SuspensionInput{Reason: "thunder", ResumeAt: &resume}, since)
	require.NoError(t, err)
	_, err = store.SuspendTournament(ctx, f.db, f.actor, tournament,
		store.SuspensionInput{Reason: "rain", ResumeAt: &resume}, since.Add(5*time.Minute))
	require.NoError(t, err)
	top, err := store.GetTournamentTop(ctx, f.db, tournament)
	require.NoError(t, err)
	require.NotNil(t, top.Frame.Suspension, "閲覧ページの上部に帯を出す")
	assert.Equal(t, "rain", top.Frame.Suspension.Reason)
	assert.Equal(t, "13:30", *top.Frame.Suspension.ResumeAt)
	assert.True(t, top.Frame.Suspension.Since.Equal(since), "中断した時刻は最初のまま")

	result, err := store.ResumeTournament(ctx, f.db, f.actor, tournament, nil, since.Add(17*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 20, result.Minutes, "17 分の中断は 5 分単位に切り上げて 20 分")
	assert.Equal(t, 23, result.Shifted, "終了した A1 以外の 23 試合")
	assert.NotEmpty(t, result.OverEnd, "終了時刻 10:00 を過ぎる試合を知らせる")
	assert.Equal(t, *a1Before.StartTime, *f.match(t, "A1").StartTime, "始まった試合の時刻は記録として残す")
	moved, err := time.Parse("15:04", (*f.match(t, "A2").StartTime)[:5])
	require.NoError(t, err)
	before, err := time.Parse("15:04", (*a2Before.StartTime)[:5])
	require.NoError(t, err)
	assert.Equal(t, 20*time.Minute, moved.Sub(before))
	top, err = store.GetTournamentTop(ctx, f.db, tournament)
	require.NoError(t, err)
	assert.Nil(t, top.Frame.Suspension)
	assert.Equal(t, int64(2), auditCount(t, f, store.AuditTournamentSuspend))
	assert.Equal(t, int64(1), auditCount(t, f, store.AuditTournamentResume))
}

// 遅れの調整の確認: 動く試合の数を保存せずに返す(M5)。
func TestCountShift(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	court := "1"
	var want int64
	require.NoError(t, f.db.Model(&store.Match{}).Where("division_id = ? AND field = ? AND status = ?",
		f.division.ID, court, domain.MatchScheduled).Count(&want).Error)
	before := f.match(t, "A1")
	moved, err := store.CountShift(ctx, f.db, f.division.ID, &court, 15)
	require.NoError(t, err)
	assert.Equal(t, int(want), moved)
	assert.Equal(t, *before.StartTime, *f.match(t, "A1").StartTime, "保存しない")
	_, err = store.CountShift(ctx, f.db, f.division.ID, &court, 0)
	require.ErrorIs(t, err, domain.ErrBadRequest)
}
