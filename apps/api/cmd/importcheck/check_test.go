package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer"
	"scoreview/api/internal/importer/parse"
)

func TestGoalCounts(t *testing.T) {
	left, right := "left", "right"
	m := &parse.Match{Events: []parse.MatchEvent{
		{Kind: "pull", Side: &left},
		{Kind: "goal", Side: &left},
		{Kind: "goal", Side: &right},
		{Kind: "goal", Side: &left},
		{Kind: "end"},
	}}
	l, r := goalCounts(m)
	assert.Equal(t, 2, l)
	assert.Equal(t, 1, r)
}

func TestCheckStandingsComparesTopThree(t *testing.T) {
	rank := func(v int) *int { return &v }
	c := &checker{b: &importer.Bundle{Standings: map[int64]*parse.DivisionStandings{
		10: {Rows: []parse.StandingRow{
			{TeamID: 1, Rank: rank(1)}, {TeamID: 2, Rank: rank(2)}, {TeamID: 3, Rank: rank(3)},
			{TeamID: 4, Rank: rank(4)},
		}},
	}}}
	var api standingsDTO
	require.NoError(t, json.Unmarshal([]byte(`{"standings":[
		{"team":{"id":1},"rank":1},{"team":{"id":3},"rank":2},{"team":{"id":2},"rank":3},
		{"team":{"id":4},"rank":4}]}`), &api))
	c.checkStandings(map[int64]standingsDTO{10: api})
	require.Len(t, c.results, 1)
	assert.False(t, c.results[0].ok, "2 位と 3 位が入れ替わっている")
}

func TestFetchRecordsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Token") != "token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{}`)) // テスト用サーバーの書き込み失敗は検証対象外
	}))
	t.Cleanup(srv.Close)
	c := &checker{api: newAPIClient(srv.URL, "token")}
	assert.True(t, c.fetch(context.Background(), "/ok", &ignoredDTO{}))
	assert.False(t, c.fetch(context.Background(), "/missing", &ignoredDTO{}))
	assert.Equal(t, 2, c.requests)
	assert.Equal(t, []string{"/missing: HTTP 404"}, c.non200)
}
