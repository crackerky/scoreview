package parse

import (
	"strings"
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

// ナビが「エントリー一覧」の大会は、チーム名がリンクのチーム戦でも見出しが Players(237)。
func TestParseTeamListPlayersWithLinksIsTeam(t *testing.T) {
	html := `<html><body><div id="score_view">
<div class='TeamListMIDA'>架空ミックス&nbsp;&nbsp;&nbsp;--&nbsp;1 Players&nbsp;--&nbsp;&nbsp;&nbsp;</div>
<div class='TeamListBLOCK'><div class='px12 FTitle_Mix'><span class='teamimg_rnk'><a href="#"
 onclick="document.location = 'https://score.clubjr.com/pscore/scv_team/open/9?lang=jp'; return false;">架空</a></span></div></div>
<div class='TeamListBLOCK'><div class='px12 AreaSepa RM'>x7</div></div>
</div></body></html>`
	tl, err := ParseTeamList(strings.NewReader(html), site.LangJP)
	require.NoError(t, err)
	require.Len(t, tl.Divisions, 1)
	assert.Equal(t, "team", tl.Divisions[0].EntryKind)
	assert.Equal(t, ptr(int64(9)), tl.Divisions[0].Entries[0].ID)
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
