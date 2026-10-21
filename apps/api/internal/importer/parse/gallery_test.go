package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGallery(t *testing.T) {
	g := parseBoth(t, "gallery_362_jp.html", "gallery_362_en.html", ParseGallery, MergeGalleryEN)

	assert.Equal(t, "Day2-AM", g.Name)
	require.Len(t, g.Photos, 3)
	// HTML の生の空白("2026 Atami-B")を %20 にした絶対 URL(clone-spec A 6)
	assert.Equal(t, Photo{
		ThumbURL: "https://score.clubjr.com/zfiles/TGalleryImg/2026%20Atami-B/Day2-AM/mini/" +
			"event_2026atami_photo_day2_am_00001.jpg",
		FullURL: "https://score.clubjr.com/zfiles/TGalleryImg/2026%20Atami-B/Day2-AM/" +
			"event_2026atami_photo_day2_am_00001.jpg",
	}, g.Photos[0])
}
