package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParseTournamentTopStandard(t *testing.T) {
	top := parseBoth(t, "top_525_jp.html", "top_525_en.html", ParseTournamentTop, MergeTournamentTopEN)

	assert.Equal(t, int64(525), top.ID)
	assert.Equal(t, Text{
		JA: "2026 Atami Beach Ultimate Beach Line Tournament",
		EN: "2026 Atami Beach Ultimate Beach Line Tournament",
	}, top.Name)
	require.NotNil(t, top.LogoURL)
	assert.Equal(t, "https://score.clubjr.com/zfiles/ScoreViewLogo/atami_2025.png", *top.LogoURL)

	require.Len(t, top.Divisions, 1)
	d := top.Divisions[0]
	assert.Equal(t, ptr(int64(949)), d.ID)
	assert.Equal(t, Text{JA: "ビーチミックス", EN: "Beach Mixed"}, d.Name)
	assert.True(t, d.ScheduleLinked)
	assert.True(t, d.StandingsLinked)
	require.Len(t, d.Groups, 8)
	assert.Equal(t, NavGroup{
		ID: ptr(int64(6623)), Name: "Tounament A", NameEN: "Tounament A", Linked: true,
	}, d.Groups[0])
	assert.Equal(t, "21st-29th placement tournament", d.Groups[7].Name)

	assert.Equal(t, Nav{
		TeamListLabel: "teams", GroupLabel: "groups", ShowSchedule: true,
		Galleries: []NavGallery{
			{ID: 362, Name: "Day2-AM", NameEN: "Day2-AM"}, {ID: 363, Name: "Day2-PM", NameEN: "Day2-PM"},
		},
	}, top.Nav)

	assert.Contains(t, top.TopHTML.JA, "見本チーム")
	assert.Contains(t, top.TopHTML.EN, "Sample Team")

	require.Len(t, top.Banners, 3)
	require.NotNil(t, top.Banners[0].LinkURL)
	assert.Equal(t, "http://www.clubjr.com/event_2026atami_4.Map.pdf", *top.Banners[0].LinkURL)
	assert.Nil(t, top.Banners[1].LinkURL)
	assert.Contains(t, top.Banners[0].ImageURL, "https://score.clubjr.com/zfiles/ScoreViewTop/")
}

// ナビの名前: 前後の半角空白は名前の一部、英語名が空のリンクは参照サイトが URL を出すので空にする。
func TestNavName(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<ul>
<li><a href="https://score.clubjr.com/pscore/scv_group/open/1?lang=jp">架空 placement tournament </a></li>
<li><a href="https://score.clubjr.com/pscore/scv_group/open/1?lang=en">https://score.clubjr.com/pscore/scv_group/open/1?lang=en</a></li>
<li><p>
	架空</p></li></ul>`))
	require.NoError(t, err)
	var names []string
	doc.Find("li > a, li > p").Each(func(_ int, sel *goquery.Selection) {
		names = append(names, navName(sel))
	})
	assert.Equal(t, []string{"架空 placement tournament ", "", "架空"}, names)
}

// ディスクゴルフ個人戦: エントリー一覧/成績、スケジュールなし、リンクはダミー、英語名が空(clone-spec E 3)
func TestParseTournamentTopDiscGolf(t *testing.T) {
	top := parseBoth(t, "top_537_jp.html", "top_537_en.html", ParseTournamentTop, MergeTournamentTopEN)

	assert.Equal(t, int64(537), top.ID)
	assert.Equal(t, "", top.Name.EN)
	assert.Equal(t, Nav{TeamListLabel: "entries", GroupLabel: "results", Galleries: []NavGallery{}}, top.Nav)
	require.Len(t, top.Divisions, 2)
	for _, d := range top.Divisions {
		assert.Nil(t, d.ID, "ダミーリンクだけのディビジョンは ID が分からない")
		assert.False(t, d.ScheduleLinked)
		assert.False(t, d.StandingsLinked)
		assert.Empty(t, d.Groups)
	}
	assert.Equal(t, Text{JA: "ＤＧシングルス（日曜日）", EN: "DG Singles (Sunday)"}, top.Divisions[1].Name)
	assert.Empty(t, top.Banners)
	assert.Equal(t, Text{}, top.TopHTML)
}

func TestParseTournamentTopDefaultLogo(t *testing.T) {
	top, err := ParseTournamentTop(fixture(t, "top_151_jp.html"), site.LangJP)
	require.NoError(t, err)
	assert.Nil(t, top.LogoURL, "既定ロゴはロゴなしとして扱う")
	ids := []int64{}
	for _, d := range top.Divisions {
		require.NotNil(t, d.ID)
		ids = append(ids, *d.ID)
	}
	// 並びは DB 順で ID 順ではない(clone-spec A 4)
	assert.Equal(t, []int64{338, 340, 339}, ids)
}

// 237: スケジュールのないディビジョンはナビのスケジュールに出ないので、名前で ID を対応させる
func TestParseTournamentTopPartialSchedule(t *testing.T) {
	top, err := ParseTournamentTop(fixture(t, "top_237_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.Len(t, top.Divisions, 2)
	assert.Equal(t, ptr(int64(485)), top.Divisions[0].ID)
	assert.True(t, top.Divisions[0].ScheduleLinked)
	assert.False(t, top.Divisions[0].StandingsLinked)
	assert.Nil(t, top.Divisions[1].ID)
	assert.Equal(t, "entries", top.Nav.TeamListLabel)
	assert.Equal(t, "results", top.Nav.GroupLabel)
	assert.True(t, top.Nav.ShowSchedule)
}
