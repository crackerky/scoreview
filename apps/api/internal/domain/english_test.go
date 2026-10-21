package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
)

func TestWithEnglish(t *testing.T) {
	assert.Equal(t, domain.I18nText{Ja: "A 組", En: "A 組"}, domain.I18nText{Ja: "A 組"}.WithEnglish())
	assert.Equal(t, domain.I18nText{Ja: "A 組", En: "Pool A"},
		domain.I18nText{Ja: "A 組", En: "Pool A"}.WithEnglish())
	assert.Equal(t, domain.I18nText{}, domain.I18nText{}.WithEnglish())
}

func TestTeamNamesWithEnglish(t *testing.T) {
	cases := []struct {
		name                string
		full, short         domain.I18nText
		wantFull, wantShort domain.I18nText
	}{
		{
			name:      "japanese only",
			full:      domain.I18nText{Ja: "架空フライヤーズ東京"},
			short:     domain.I18nText{Ja: "フライヤーズ"},
			wantFull:  domain.I18nText{Ja: "架空フライヤーズ東京", En: "架空フライヤーズ東京"},
			wantShort: domain.I18nText{Ja: "フライヤーズ", En: "フライヤーズ"},
		},
		{
			name:      "english full name only",
			full:      domain.I18nText{Ja: "架空フライヤーズ東京", En: "Fictional Flyers Tokyo"},
			short:     domain.I18nText{Ja: "フライヤーズ"},
			wantFull:  domain.I18nText{Ja: "架空フライヤーズ東京", En: "Fictional Flyers Tokyo"},
			wantShort: domain.I18nText{Ja: "フライヤーズ", En: "Fictional Flyers Tokyo"},
		},
		{
			name:      "both english",
			full:      domain.I18nText{Ja: "架空フライヤーズ東京", En: "Fictional Flyers Tokyo"},
			short:     domain.I18nText{Ja: "フライヤーズ", En: "Flyers"},
			wantFull:  domain.I18nText{Ja: "架空フライヤーズ東京", En: "Fictional Flyers Tokyo"},
			wantShort: domain.I18nText{Ja: "フライヤーズ", En: "Flyers"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			full, short := domain.TeamNamesWithEnglish(tc.full, tc.short)
			assert.Equal(t, tc.wantFull, full)
			assert.Equal(t, tc.wantShort, short)
		})
	}
}
