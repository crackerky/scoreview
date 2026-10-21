package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTournamentIDs(t *testing.T) {
	ids, err := ParseTournamentIDs("525, 440,,451")
	require.NoError(t, err)
	assert.Equal(t, []int64{525, 440, 451}, ids)

	ids, err = ParseTournamentIDs("")
	require.NoError(t, err)
	assert.Empty(t, ids)

	for _, bad := range []string{"abc", "0", "-1"} {
		_, err := ParseTournamentIDs(bad)
		assert.Error(t, err, bad)
	}
}

func TestRepresentativeTournaments(t *testing.T) {
	ids := RepresentativeTournaments()
	assert.Len(t, ids, 16, "docs/representative-tournaments.md の表の行数")
	assert.Equal(t, int64(525), ids[0])
}
