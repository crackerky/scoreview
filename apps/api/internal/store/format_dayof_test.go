package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 本部の「いまやること」は緊急度の順(時刻超過 → 同じチームの時刻の重なり・食い違い → 確定待ち →
// 未決 → コートの遅れ)に並び、グループと枠は運営の言葉で出る(レビュー H1、M1)。
func TestProgressTodoOrderAndOverlap(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	a1, a2 := f.match(t, "A1"), f.match(t, "A2")
	// A1 と A2 はどちらもシード 1 の試合。同じ時刻に動かすと重なる
	require.NoError(t, f.db.Model(&store.Match{}).Where("id = ?", a2.ID).Updates(map[string]any{
		"start_time": *a1.StartTime, "match_date": a1.MatchDate,
	}).Error)

	progress, err := store.GetFormatProgress(ctx, f.db, f.division.ID, fixtureNow())
	require.NoError(t, err)
	rank := map[store.TodoKind]int{store.TodoOverdue: 0, store.TodoOverlap: 1, store.TodoMismatch: 1,
		store.TodoConfirmGroup: 2, store.TodoConfirmStage: 2, store.TodoTie: 3, store.TodoCourtDelay: 4}
	var overlaps []string
	last := 0
	for _, item := range progress.Todo {
		assert.GreaterOrEqual(t, rank[item.Kind], last, "緊急度の順: %s", item.Text)
		last = rank[item.Kind]
		if item.Kind == store.TodoOverlap {
			overlaps = append(overlaps, item.Text)
		}
	}
	require.Len(t, overlaps, 1)
	assert.Contains(t, overlaps[0], "T01")
	assert.Contains(t, overlaps[0], "A1")
	assert.Contains(t, overlaps[0], "A2")

	assert.Equal(t, "予選 A 組", progress.Groups[0].Label)
	assert.Equal(t, "Qualifying pool A", progress.Groups[0].Name, "グループ名のデータは変えない")
	assert.Equal(t, "1〜4 位を決めるトーナメント", progress.Groups[4].Label)
	assert.Equal(t, "A1", progress.Groups[4].Slots[0].Label)

	matches, err := store.ListFormatMatches(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	for _, m := range matches {
		if m.Code == "E1" {
			assert.Equal(t, "[E2-勝ち]", m.Left.Label)
			assert.True(t, strings.HasSuffix(m.GroupLabel, "トーナメント"))
		}
	}
}

// プールの試合の前は「未決」を出さない(全試合が終わる前に並んでいるのは未決ではない。H1)。
func TestProgressNoTieBeforePoolGames(t *testing.T) {
	f := newFormatFixture(t)
	assert.Empty(t, f.todo(t, store.TodoTie))
	progress, err := store.GetFormatProgress(context.Background(), f.db, f.division.ID, fixtureNow())
	require.NoError(t, err)
	for _, g := range progress.Groups {
		assert.NotEqual(t, "tied", string(g.State), g.Label)
		for _, row := range g.Rows {
			assert.False(t, row.Unresolved, g.Label)
		}
	}
}

// 本部の結果入力の楽観的ロック: 画面を開いたときのスコアと保存時のスコアが違えば保存しない(H7)。
func TestCorrectionRefusesAStaleScore(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	a1 := f.match(t, "A1").ID
	opened := domain.Score{} // 試合前に画面を開いた(スコアなしは 0-0)
	_, err := store.ApplyCorrection(ctx, f.db, f.actor, a1,
		store.CorrectionInput{Score: domain.Score{Left: 11, Right: 9}, ExpectedScore: &opened}, store.DownstreamNone)
	require.NoError(t, err, "1 人目は保存できる")
	_, err = store.ApplyCorrection(ctx, f.db, f.actor, a1,
		store.CorrectionInput{Score: domain.Score{Left: 9, Right: 11}, ExpectedScore: &opened}, store.DownstreamNone)
	var mismatch *domain.ScoreMismatchError
	require.ErrorAs(t, err, &mismatch, "古い画面からの保存は止める")
	assert.Equal(t, domain.Score{Left: 11, Right: 9}, mismatch.Actual, "今の値を返す")
	assert.True(t, errors.Is(err, domain.ErrConflict))
	m := f.match(t, "A1")
	assert.Equal(t, 11, *m.LeftScore, "上書きされていない")

	current := domain.Score{Left: 11, Right: 9}
	_, err = store.ApplyCorrection(ctx, f.db, f.actor, a1,
		store.CorrectionInput{Score: domain.Score{Left: 9, Right: 11}, ExpectedScore: &current}, store.DownstreamNone)
	require.NoError(t, err, "今の値を確かめてからなら上書きできる")
	audit, err := store.ListMatchAudit(ctx, f.db, a1)
	require.NoError(t, err)
	assert.Len(t, audit, 2, "1 回の保存は監査ログの 1 行(L2)")
}

// 記録途中(得点経過あり)の試合を、得点経過を残したまま本部の最終スコアで確定できる(H5)。
func TestCorrectionKeepsEventsAndNotesTheDifference(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	a1 := f.match(t, "A1").ID
	left := domain.SideLeft
	_, err := store.AddMatchEvent(ctx, f.db, f.actor, a1,
		store.EventInput{NewEvent: domain.NewEvent{Kind: domain.EventGoal, Side: &left}}, time.Now())
	require.NoError(t, err)

	final := store.CorrectionInput{Score: domain.Score{Left: 11, Right: 9}}
	_, err = store.ApplyCorrection(ctx, f.db, f.actor, a1, final, store.DownstreamNone)
	require.ErrorIs(t, err, domain.ErrBadRequest, "得点経過を残すと選ばなければ断る")
	final.KeepEvents = true
	_, err = store.ApplyCorrection(ctx, f.db, f.actor, a1, final, store.DownstreamNone)
	require.NoError(t, err)

	m := f.match(t, "A1")
	assert.Equal(t, domain.MatchFinished, m.Status)
	assert.Equal(t, []int{11, 9}, []int{*m.LeftScore, *m.RightScore}, "閲覧ページは最終スコアを出す")
	detail, err := store.GetMatchDetail(ctx, f.db, a1)
	require.NoError(t, err)
	assert.Len(t, detail.Events, 1, "得点経過は残る")
	audit, err := store.ListMatchAudit(ctx, f.db, a1)
	require.NoError(t, err)
	assert.Equal(t, "得点経過(1-0)と違う最終スコア(11-9)で本部が確定", audit[0].After.Note)
}

// 影響を確かめる段階で、保存のときと同じ検査をして先に知らせる(H6)。
func TestCorrectionImpactValidatesFirst(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finishPools(t)
	f.confirmPools(t)
	_, err := store.PreviewCorrection(ctx, f.db, f.match(t, "E2").ID,
		store.CorrectionInput{Score: domain.Score{Left: 5, Right: 5}})
	var invalid *domain.ValidationError
	require.ErrorAs(t, err, &invalid, "トーナメントの同点は勝者が要る")
	assert.Equal(t, "winnerSide", invalid.Field)
	_, err = store.PreviewCorrection(ctx, f.db, f.match(t, "E1").ID,
		store.CorrectionInput{Score: domain.Score{Left: 5, Right: 3}})
	require.ErrorIs(t, err, domain.ErrConflict, "対戦チームが決まる前の試合")
}
