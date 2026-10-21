package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
)

func id(v int64) *int64 { return &v }

func TestBracketLayoutRoundTrip(t *testing.T) {
	layout := domain.BracketLayout{
		Columns: 2,
		Rows: [][]domain.BracketCell{
			{
				{Kind: domain.BracketTxtYen, Text: "FINAL"},
				{Kind: domain.BracketTxtSharp, Text: "Placement"},
			},
			{
				{Kind: domain.BracketTeam, TeamID: id(10), Borders: domain.BracketBorders{Bottom: true}},
				{},
			},
			{
				{
					Kind:    domain.BracketGame,
					MatchID: id(99),
					Borders: domain.BracketBorders{Top: true, Right: true},
				},
				{
					Kind:    domain.BracketWin,
					TeamID:  id(10),
					Borders: domain.BracketBorders{Left: true, Bottom: true},
				},
			},
		},
	}
	data, err := domain.MarshalBracketLayout(layout)
	require.NoError(t, err)
	assert.JSONEq(t, `{"columns":2,"rows":[
		[{"borders":{"top":false,"bottom":false,"left":false,"right":false},"kind":"txt_yen","text":"FINAL","teamId":null,"matchId":null},
		 {"borders":{"top":false,"bottom":false,"left":false,"right":false},"kind":"txt_sharp","text":"Placement","teamId":null,"matchId":null}],
		[{"borders":{"top":false,"bottom":true,"left":false,"right":false},"kind":"team","text":"","teamId":10,"matchId":null},
		 {"borders":{"top":false,"bottom":false,"left":false,"right":false},"kind":"","text":"","teamId":null,"matchId":null}],
		[{"borders":{"top":true,"bottom":false,"left":false,"right":true},"kind":"game","text":"","teamId":null,"matchId":99},
		 {"borders":{"top":false,"bottom":true,"left":true,"right":false},"kind":"win","text":"","teamId":10,"matchId":null}]
	]}`, string(data))

	parsed, err := domain.ParseBracketLayout(data)
	require.NoError(t, err)
	assert.Equal(t, layout, parsed)
}

func TestParseBracketLayoutRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"not json", `{`},
		{"zero columns", `{"columns":0,"rows":[]}`},
		{"ragged row", `{"columns":2,"rows":[[{"kind":"team"}]]}`},
		{"unknown kind", `{"columns":1,"rows":[[{"kind":"bogus"}]]}`},
		{"unknown key", `{"columns":1,"rows":[],"extra":1}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.ParseBracketLayout([]byte(tc.json))
			assert.ErrorIs(t, err, domain.ErrInvalidBracketLayout)
		})
	}
}

func TestMarshalBracketLayoutValidates(t *testing.T) {
	_, err := domain.MarshalBracketLayout(
		domain.BracketLayout{Columns: 1, Rows: [][]domain.BracketCell{{}, {{}, {}}}},
	)
	assert.ErrorIs(t, err, domain.ErrInvalidBracketLayout)
}
