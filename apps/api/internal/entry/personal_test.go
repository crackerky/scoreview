package entry_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

func TestCheckPersonal(t *testing.T) {
	today := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	both := entry.PersonalAsk{Gender: true, Birthdate: true}
	got, err := entry.CheckPersonal(both, "F", "2001-04-02", today)
	require.NoError(t, err)
	require.NotNil(t, got.Gender)
	assert.Equal(t, entry.GenderFemale, *got.Gender)
	assert.Equal(t, "2001-04-02", *got.Birthdate)

	for name, input := range map[string][2]string{
		"性別なし":      {"", "2001-04-02"},
		"性別が不明":     {"X", "2001-04-02"},
		"生年月日なし":    {"M", ""},
		"日付でない":     {"M", "2001/04/02"},
		"未来の日":      {"M", "2026-11-29"},
		"1900 年より前": {"M", "1899-12-31"},
	} {
		_, err := entry.CheckPersonal(both, input[0], input[1], today)
		require.ErrorIs(t, err, domain.ErrBadRequest, name)
	}

	none, err := entry.CheckPersonal(entry.PersonalAsk{}, "X", "bad", today)
	require.NoError(t, err, "聞かない項目の答えは確かめずに捨てる")
	assert.Nil(t, none.Gender)
	assert.Nil(t, none.Birthdate)
}

func TestPrefillPrefersThisTeamsAnswer(t *testing.T) {
	male, female := entry.GenderMale, entry.GenderFemale
	day := "1999-01-01"
	profile := &entry.Profile{Personal: entry.Personal{Gender: &male, Birthdate: &day}}
	assert.Equal(t, profile.Personal, entry.Prefill(nil, profile), "まだ登録していなければ前回の入力")
	mine := &entry.Member{Personal: entry.Personal{Gender: &female}}
	got := entry.Prefill(mine, profile)
	assert.Equal(t, &female, got.Gender, "このチームで答えた値が先")
	assert.Equal(t, &day, got.Birthdate, "このチームで答えていない項目は前回の入力")
	assert.Equal(t, entry.Personal{}, entry.Prefill(nil, nil))
}
