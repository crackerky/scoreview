package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParseTeamList(t *testing.T) {
	tl := parseBoth(t, "teamlist_525_jp.html", "teamlist_525_en.html", ParseTeamList, MergeTeamListEN)

	require.Len(t, tl.Divisions, 1)
	d := tl.Divisions[0]
	assert.Equal(t, Text{JA: "ビーチミックス", EN: "Beach Mixed"}, d.Name)
	assert.Equal(t, 3, d.Count)
	assert.Equal(t, "team", d.EntryKind)
	require.Len(t, d.Entries, 3)
	assert.Equal(t, TeamListEntry{
		ID: ptr(int64(17491)), Name: Text{JA: "雨ニモマケズ", EN: "AmenimoMakezu"},
		BarStyle: "mixed", Members: ptr(11),
	}, d.Entries[0])
	assert.Equal(t, Text{JA: "いーあーる", EN: "ER"}, d.Entries[1].Name)
}

// ディスクゴルフ個人戦: "-- N Players --"、名前にリンクなし、人数セルは &nbsp;(clone-spec A 5)
func TestParseTeamListIndividual(t *testing.T) {
	tl, err := ParseTeamList(fixture(t, "teamlist_537_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.Len(t, tl.Divisions, 2)
	for _, d := range tl.Divisions {
		assert.Equal(t, "individual", d.EntryKind)
		require.Len(t, d.Entries, 2)
		for _, e := range d.Entries {
			assert.Nil(t, e.ID)
			assert.Nil(t, e.Members)
			assert.Equal(t, "default", e.BarStyle)
			assert.NotEmpty(t, e.Name.JA)
		}
	}
	assert.Equal(t, "ＤＧシングルス（土曜日）", tl.Divisions[0].Name.JA)
}

func TestParseTeamListBarStyles(t *testing.T) {
	tl, err := ParseTeamList(fixture(t, "teamlist_151_jp.html"), site.LangJP)
	require.NoError(t, err)
	var styles []string
	for _, d := range tl.Divisions {
		for _, e := range d.Entries {
			styles = append(styles, e.BarStyle)
		}
	}
	// メンの 1 チームだけ FTitle_Univ(clone-spec A 5)
	assert.Equal(t, []string{"univ", "default", "mixed", "mixed", "women", "women"}, styles)
}
