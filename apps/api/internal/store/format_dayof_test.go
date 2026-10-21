package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	assert.Equal(t, "A 組 1 位", progress.Groups[4].Slots[0].Label)

	matches, err := store.ListFormatMatches(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	for _, m := range matches {
		if m.Code == "E1" {
			assert.Equal(t, "E2 の勝ちチーム", m.Left.Label)
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
