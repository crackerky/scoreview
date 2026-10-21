package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickDate(t *testing.T) {
	two := []string{"2017-08-10", "2018-08-10"}
	cases := []struct {
		name       string
		prev, next *string
		want       string
		wantOK     bool
	}{
		{"前の行と同じ日", ptr("2018-08-10"), nil, "2018-08-10", true},
		{"次の行と同じ日", nil, ptr("2017-08-10"), "2017-08-10", true},
		{"前の行の翌日", ptr("2018-08-09"), nil, "2018-08-10", true},
		{"次の行の前日", nil, ptr("2017-08-11"), "2017-08-10", true},
		{"前後とも離れている", ptr("2016-01-01"), ptr("2019-01-01"), "2017-08-10", false},
		{"前後がない", nil, nil, "2017-08-10", false},
	}
	for _, tc := range cases {
		got, ok := pickDate(two, tc.prev, tc.next)
		assert.Equal(t, tc.want, got, tc.name)
		assert.Equal(t, tc.wantOK, ok, tc.name)
	}
}

func TestResolveDateSlots(t *testing.T) {
	// 大会 172 の形: 08/10 が 2017 と 2018 の両方にあり、前後の行の日付から年を決める
	two := []string{"2017-08-10", "2018-08-10"}
	slots := []dateSlot{
		{Date: ptr("2018-08-09")},
		{Candidates: two},
		{Candidates: []string{"2018-08-11"}},
		{},
		{Candidates: []string{"2020-01-01", "2021-01-01"}},
	}
	dates, ambiguous := resolveDateSlots(slots)
	require.Len(t, dates, len(slots))
	assert.Equal(t, "2018-08-10", *dates[1])
	assert.Equal(t, "2018-08-11", *dates[2], "候補が 1 つならそれ")
	assert.Nil(t, dates[3], "日付の手がかりがない行")
	assert.Equal(t, "2020-01-01", *dates[4], "決められなければ先頭")
	assert.Equal(t, []int{4}, ambiguous)
}
