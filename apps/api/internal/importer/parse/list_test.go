package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParseTournamentList(t *testing.T) {
	list := parseBoth(t, "list_jp.html", "list_en.html", ParseTournamentList, MergeTournamentListEN)

	require.Len(t, list.Entries, 5)
	assert.Equal(t, ListEntry{
		ID: 539, Name: Text{JA: "2027 CLUB Jr ULTIMATE DREAM CUP", EN: "2027 CLUB Jr ULTIMATE DREAM CUP"},
		Status: StatusEntryOpen, Pos: 1,
	}, list.Entries[0])
	// 英語名が未登録の大会は en が空(clone-spec A 3)
	assert.Equal(t, int64(537), list.Entries[1].ID)
	assert.Equal(t, "", list.Entries[1].Name.EN)
	assert.NotEmpty(t, list.Entries[1].Name.JA)

	statuses := []TournamentStatus{}
	for _, e := range list.Entries {
		statuses = append(statuses, e.Status)
	}
	assert.Equal(t, []TournamentStatus{
		StatusEntryOpen, StatusEntryOpen, StatusEntryClosed, StatusFinished, StatusFinished,
	}, statuses)
	assert.Equal(t, 5, list.Entries[4].Pos)
}

func TestMergeTournamentListMismatch(t *testing.T) {
	jp, err := ParseTournamentList(fixture(t, "list_jp.html"), site.LangJP)
	require.NoError(t, err)
	en, err := ParseTournamentList(fixture(t, "list_en.html"), site.LangEN)
	require.NoError(t, err)
	en.Entries = en.Entries[1:]
	_, err = MergeTournamentListEN(jp, en)
	assert.ErrorIs(t, err, ErrMismatch)
}
