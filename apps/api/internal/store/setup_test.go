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

// TestCopyTournament は大会のコピーが、部門とチームを引き継ぎ、試合と結果を引き継がず、新しい部門で
// 形を決めるときにコピー元の形(シードはコピーしたチーム)を初期値にできることを確かめる(H9)。
func TestCopyTournament(t *testing.T) {
	ctx := context.Background()
	f := newFormatFixture(t)
	f.finish(t, "A2", 15, 3) // 結果は引き継がない
	var division store.Division
	require.NoError(t, f.db.Where("id = ?", f.division.ID).First(&division).Error)
	source, err := store.GetAdminTournament(ctx, f.db, division.TournamentID)
	require.NoError(t, err)

	fields := source.TournamentFields
	fields.Name = domain.I18nText{Ja: "次回の大会"}
	date := "2027-10-18"
	fields.StartDate, fields.Listed = &date, false
	copied, err := store.CopyTournament(ctx, f.db, source.ID, fields, true)
	require.NoError(t, err)
	assert.Equal(t, "次回の大会", copied.Name.Ja)
	assert.False(t, copied.Listed)

	var divisions []store.Division
	require.NoError(t, f.db.Where("tournament_id = ?", copied.ID).Find(&divisions).Error)
	require.Len(t, divisions, 1)
	newDivision := divisions[0]
	var teams []store.Team
	require.NoError(t, f.db.Where("division_id = ?", newDivision.ID).Order("sort_order, id").Find(&teams).Error)
	require.Len(t, teams, 12)
	var matches int64
	require.NoError(t, f.db.Model(&store.Match{}).Where("division_id = ?", newDivision.ID).Count(&matches).Error)
	assert.Zero(t, matches, "試合は引き継がない")

	inherited, ok, err := store.GetInheritedFormat(ctx, f.db, newDivision.ID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "pool_bands", inherited.Template)
	require.Len(t, inherited.Seeds, 12, "シードはコピーしたチームに置き換わる")
	byID := map[int64]string{}
	for _, team := range teams {
		byID[team.ID] = team.NameFullJa
	}
	for i, seed := range inherited.Seeds {
		assert.Equal(t, f.teams[i].NameFullJa, byID[seed])
	}

	_, ok, err = store.GetInheritedFormat(ctx, f.db, f.division.ID)
	require.NoError(t, err)
	assert.False(t, ok, "コピーで作っていない部門")

	// チームを引き継がないコピーでも、形は初期値になる(シードは空)
	plain, err := store.CopyTournament(ctx, f.db, source.ID, fields, false)
	require.NoError(t, err)
	var plainDivision store.Division
	require.NoError(t, f.db.Where("tournament_id = ?", plain.ID).First(&plainDivision).Error)
	inherited, ok, err = store.GetInheritedFormat(ctx, f.db, plainDivision.ID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Empty(t, inherited.Seeds)
}

func TestDeleteDivisionTeams(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	b := storetest.New(t, db)
	division := b.Division(b.Tournament().ID)
	b.Team(division)
	b.Team(division)
	deleted, err := store.DeleteDivisionTeams(ctx, db, division.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)

	f := newFormatFixture(t)
	_, err = store.DeleteDivisionTeams(ctx, f.db, f.division.ID)
	require.ErrorIs(t, err, domain.ErrConflict, "試合がある部門は消さない")
}
