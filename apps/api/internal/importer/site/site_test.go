package site

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURL(t *testing.T) {
	cases := []struct {
		page Page
		id   int64
		lang Lang
		want string
	}{
		{PageTournamentList, 0, LangEN, "https://score.clubjr.com/sctnmt?lang=en"},
		{PageTournamentTop, 525, LangJP, "https://score.clubjr.com/scview/525?lang=jp"},
		{PageScheduleDiv, 949, LangEN, "https://score.clubjr.com/pscore/scv_search_div/open/949?lang=en"},
		{PageMatch, 45277, LangJP, "https://score.clubjr.com/pscore/scv_match/open/45277?lang=jp"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, URL(tc.page, tc.id, tc.lang))
	}
}

func TestLinkID(t *testing.T) {
	onclick := "document.location = 'https://score.clubjr.com/pscore/scv_team/open/17322?lang=jp'; " +
		"return false;"
	id, ok := LinkID(onclick, PageTeam)
	require.True(t, ok)
	assert.Equal(t, int64(17322), id)

	_, ok = LinkID(onclick, PageMatch)
	assert.False(t, ok, "種類が違うリンクは拾わない")

	id, ok = LinkID("https://score.clubjr.com/scview/539?lang=jp", PageTournamentTop)
	require.True(t, ok)
	assert.Equal(t, int64(539), id)
}

func TestLangOf(t *testing.T) {
	lang, ok := LangOf(`<a href="https://score.clubjr.com/pscore/scv_top/open/1?lang=en">`)
	require.True(t, ok)
	assert.Equal(t, LangEN, lang)
	_, ok = LangOf("<p>no link</p>")
	assert.False(t, ok)
}

func TestAbsURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{
			"/zfiles/TGalleryImg/2026 Atami-B/Day2-AM/mini/x_00001.jpg",
			"https://score.clubjr.com/zfiles/TGalleryImg/2026%20Atami-B/Day2-AM/mini/x_00001.jpg",
		},
		{
			"/zfiles/ScoreViewTop/%E6%96%B0Banner(2024)/a%20b.jpeg",
			"https://score.clubjr.com/zfiles/ScoreViewTop/%E6%96%B0Banner%282024%29/a%20b.jpeg",
		},
		{
			"/zfiles/ScoreViewLogo/GREATEST CUP 2025(600×213).png",
			"https://score.clubjr.com/zfiles/ScoreViewLogo/GREATEST%20CUP%202025%28600%C3%97213%29.png",
		},
		{"/zfiles/x/100%.jpg", "https://score.clubjr.com/zfiles/x/100%25.jpg"},
		{"http://www.clubjr.com/", "http://www.clubjr.com/"},
	}
	for _, tc := range cases {
		got, err := AbsURL(tc.in)
		require.NoError(t, err, tc.in)
		assert.Equal(t, tc.want, got, tc.in)
	}
	_, err := AbsURL("  ")
	assert.Error(t, err)
}
