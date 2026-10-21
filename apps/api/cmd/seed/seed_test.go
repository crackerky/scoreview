package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

func count(t *testing.T, db *gorm.DB, model any, query string, args ...any) int64 {
	t.Helper()
	var total int64
	tx := db.Model(model)
	if query != "" {
		tx = tx.Where(query, args...)
	}
	require.NoError(t, tx.Count(&total).Error)
	return total
}

func TestSeedCoversVariants(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	summary, err := seed(ctx, db, false)
	require.NoError(t, err)
	assert.Len(t, summary, 6)

	for _, status := range domain.TournamentStatusOrder() {
		assert.Positive(t, count(t, db, &store.Tournament{}, "status = ? AND listed", status), status)
	}
	assert.EqualValues(t, 1, count(t, db, &store.Tournament{}, "NOT listed"))
	assert.Positive(t, count(t, db, &store.Tournament{}, "name_en = ''"))
	assert.Positive(
		t,
		count(t, db, &store.Tournament{}, "team_list_label = 'entries' AND NOT show_schedule"),
	)
	for _, kind := range []string{"men", "women", "mixed", "none"} {
		assert.Positive(t, count(t, db, &store.Division{}, "kind = ?", kind), kind)
	}
	assert.Positive(
		t,
		count(t, db, &store.Division{}, "entry_kind = 'individual' AND stats_kind = 'discgolf'"),
	)
	for _, kind := range []string{"pool", "bracket", "image"} {
		assert.Positive(t, count(t, db, &store.Group{}, "kind = ?", kind), kind)
	}
	for _, status := range []string{"scheduled", "live", "finished"} {
		assert.Positive(t, count(t, db, &store.Match{}, "status = ?", status), status)
	}
	assert.Positive(t, count(t, db, &store.Match{}, "NOT has_detail"))
	assert.Positive(
		t,
		count(t, db, &store.Match{}, "status = 'finished' AND left_score = right_score"),
	)
	assert.Positive(t, count(t, db, &store.Match{}, "right_placeholder_code IS NOT NULL"))
	assert.Positive(t, count(t, db, &store.Match{}, "field !~ '^[0-9]+$'"), "a non-numeric field")
	assert.Positive(t, count(t, db, &store.MatchEvent{}, "callahan"))
	assert.Positive(t, count(t, db, &store.MatchPlayerStat{}, "blocks > 0"))
	assert.Positive(t, count(t, db, &store.Player{}, "role_ja IS NOT NULL"))
	assert.Positive(t, count(t, db, &store.GalleryPhoto{}, ""))
	assert.Positive(t, count(t, db, &store.TournamentBanner{}, ""))
}

// TestSeedIsReadable は投入したデータの全ページを store で読めること(壊れた参照が無いこと)と、
// 得点経過の累計が試合のスコアと一致することを確かめる。
func TestSeedIsReadable(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	_, err := seed(ctx, db, false)
	require.NoError(t, err)

	var tournaments []store.Tournament
	require.NoError(t, db.Find(&tournaments).Error)
	for _, tour := range tournaments {
		_, err := store.GetTournamentTop(ctx, db, tour.ID)
		require.NoError(t, err)
		_, err = store.GetTeamList(ctx, db, tour.ID)
		require.NoError(t, err)
		_, err = store.GetTournamentSchedule(ctx, db, tour.ID, store.ScheduleFilter{})
		require.NoError(t, err)
	}
	var divisions []store.Division
	require.NoError(t, db.Find(&divisions).Error)
	for _, division := range divisions {
		_, err := store.GetDivisionSchedule(ctx, db, division.ID, store.ScheduleFilter{})
		require.NoError(t, err)
		_, err = store.GetDivisionStandings(ctx, db, division.ID)
		require.NoError(t, err)
	}
	var groups []store.Group
	require.NoError(t, db.Find(&groups).Error)
	for _, group := range groups {
		_, err := store.GetGroup(ctx, db, group.ID)
		require.NoError(t, err, group.Name)
	}
	var players []store.Player
	require.NoError(t, db.Find(&players).Error)
	for _, player := range players {
		_, err := store.GetPlayer(ctx, db, player.ID)
		require.NoError(t, err)
	}
	var matches []store.Match
	require.NoError(t, db.Find(&matches).Error)
	for _, match := range matches {
		page, err := store.GetMatch(ctx, db, match.ID)
		require.NoError(t, err)
		if len(page.Events) == 0 {
			continue
		}
		last := page.Events[len(page.Events)-1]
		assert.Equal(t, *match.LeftScore, last.LeftScore, match.Code)
		assert.Equal(t, *match.RightScore, last.RightScore, match.Code)
	}
}

func TestSeedRefusesNonEmptyDatabaseUnlessReplace(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	_, err := seed(ctx, db, false)
	require.NoError(t, err)
	before := count(t, db, &store.Match{}, "")

	_, err = seed(ctx, db, false)
	require.ErrorIs(t, err, ErrNotEmpty)

	_, err = seed(ctx, db, true)
	require.NoError(t, err)
	assert.Equal(t, before, count(t, db, &store.Match{}, ""))
	assert.EqualValues(t, 6, count(t, db, &store.Tournament{}, ""))
}
