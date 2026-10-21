package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
)

func TestCheckURL(t *testing.T) {
	for _, ok := range []string{"/brand/logo.svg", "https://example.com/a.png", "http://example.com"} {
		assert.NoError(t, domain.CheckURL("url", ok), ok)
	}
	for _, bad := range []string{
		"javascript:alert(1)", "JaVaScRiPt:alert(1)", "data:text/html,<script>", "//evil.example",
		"logo.png", "ftp://example.com/a", "https://", "",
		// ブラウザが "//evil.example" と読むもの(セキュリティ点検 L-1)
		"/\\evil.example/x", "/\t/evil.example", "/\n/evil.example", "\\\\evil.example",
		" https://ok.example", "https://evil.example\\@good.example", "/a b",
	} {
		assert.ErrorIs(t, domain.CheckURL("url", bad), domain.ErrBadRequest, bad)
	}
	assert.NoError(t, domain.CheckOptionalURL("url", nil))
}

func TestCheckRequiredAndClock(t *testing.T) {
	assert.ErrorIs(t, domain.CheckRequired("name", "  "), domain.ErrBadRequest)
	assert.NoError(t, domain.CheckRequired("name", " x "))

	assert.NoError(t, domain.CheckClock("time", "09:30"))
	for _, bad := range []string{"9:30", "24:00", "09:30:00", "ab:cd"} {
		assert.ErrorIs(t, domain.CheckClock("time", bad), domain.ErrBadRequest, bad)
	}
	_, err := domain.ParseDay("date", "2026-02-30")
	assert.ErrorIs(t, err, domain.ErrBadRequest)
	day, err := domain.ParseDay("date", "2026-10-18")
	assert.NoError(t, err)
	assert.Equal(t, 18, day.Day())
}
