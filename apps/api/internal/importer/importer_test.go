package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/fetch"
	"scoreview/api/internal/importer/parse"
	"scoreview/api/internal/importer/site"
)

// fileFetcher は URL に対応する parse/testdata のファイルを返すフェイク。対応のない URL は
// 参照サイトと同じく「存在しない」(500)を返し、取得した URL を記録する。
type fileFetcher struct {
	files     map[string]string
	requested []string
}

func (f *fileFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.requested = append(f.requested, url)
	name, ok := f.files[url]
	if !ok {
		return nil, &fetch.NotFoundError{URL: url, Status: http.StatusInternalServerError}
	}
	return os.ReadFile(filepath.Join("parse", "testdata", name))
}

func fixtures525() map[string]string {
	jp, en := site.LangJP, site.LangEN
	return map[string]string{
		site.URL(site.PageTournamentList, 0, jp):  "list_jp.html",
		site.URL(site.PageTournamentList, 0, en):  "list_en.html",
		site.URL(site.PageTournamentTop, 525, jp): "top_525_jp.html",
		site.URL(site.PageTournamentTop, 525, en): "top_525_en.html",
		site.URL(site.PageTeamList, 525, jp):      "teamlist_525_jp.html",
		site.URL(site.PageTeamList, 525, en):      "teamlist_525_en.html",
		site.URL(site.PageScheduleDiv, 949, jp):   "schedule_div949_jp.html",
		site.URL(site.PageDivision, 949, jp):      "division_949_jp.html",
		site.URL(site.PageGroup, 6623, jp):        "group_6623_jp.html",
		site.URL(site.PageGroup, 6627, jp):        "group_6627_jp.html",
		site.URL(site.PageMatch, 45277, jp):       "match_45277_jp.html",
		site.URL(site.PageGallery, 362, jp):       "gallery_362_jp.html",
	}
}

func TestCollect(t *testing.T) {
	ff := &fileFetcher{files: fixtures525()}
	im, err := New(WithFetcher(ff))
	require.NoError(t, err)

	b, err := im.Collect(context.Background(), 525)
	require.NoError(t, err)

	assert.Equal(t, int64(525), b.TournamentID)
	require.NotNil(t, b.ListEntry)
	assert.Equal(t, parse.StatusFinished, b.ListEntry.Status)
	assert.Equal(t, "2026 Atami Beach Ultimate Beach Line Tournament", b.Top.Name.EN)
	require.NotNil(t, b.TeamList)
	assert.Equal(t, "Beach Mixed", b.TeamList.Divisions[0].Name.EN)

	// スケジュールは英語版がないので日本語版だけ。Missing に記録する
	require.Contains(t, b.Schedules, int64(949))
	assert.Equal(t, "", b.Schedules[949].Rows[0].Left.ShortName.EN)
	assert.Contains(t, b.Missing, site.URL(site.PageScheduleDiv, 949, site.LangEN))

	require.Contains(t, b.Standings, int64(949))
	assert.Len(t, b.Groups, 2, "フィクスチャのある 2 グループ")
	assert.Contains(t, b.Missing, site.URL(site.PageGroup, 6624, site.LangJP))
	require.Contains(t, b.Matches, int64(45277))
	assert.Equal(t, "A1", b.Matches[45277].Code)
	require.Contains(t, b.Galleries, int64(362))
	assert.Empty(t, b.Players, "選手ページは既定では取らない")

	// スケジュール・グループに出る試合とチームを取りに行っている
	assert.Contains(t, ff.requested, site.URL(site.PageMatch, 45280, site.LangJP))
	assert.Contains(t, ff.requested, site.URL(site.PageTeam, 17322, site.LangJP))
	assert.Contains(t, ff.requested, site.URL(site.PageTeam, 17396, site.LangJP), "順位表のチーム")
	assert.NotContains(t, ff.requested, site.URL(site.PageMatch, 45277, site.LangEN), "試合は日本語版だけ")

	var buf bytes.Buffer
	require.NoError(t, JSONSink{W: &buf}.Write(context.Background(), b))
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Contains(t, decoded, "schedules")
	assert.Contains(t, buf.String(), "<p", "自由記述の HTML をエスケープしない")
}

func TestCollectTournamentNotFound(t *testing.T) {
	im, err := New(WithFetcher(&fileFetcher{files: map[string]string{}}))
	require.NoError(t, err)
	_, err = im.Collect(context.Background(), 999999)
	assert.ErrorIs(t, err, fetch.ErrNotFound)
}

func TestCollectReportsParseError(t *testing.T) {
	files := fixtures525()
	// 大会トップの URL にチーム一覧の HTML を返すと、ナビがないのでパースに失敗する
	files[site.URL(site.PageTournamentTop, 525, site.LangJP)] = "teamlist_525_jp.html"
	im, err := New(WithFetcher(&fileFetcher{files: files}))
	require.NoError(t, err)
	_, err = im.Collect(context.Background(), 525)
	assert.ErrorIs(t, err, parse.ErrUnexpected)
}

func TestNewRequiresFetcher(t *testing.T) {
	_, err := New()
	assert.Error(t, err)
}

func TestDivisionKindFromSchedule(t *testing.T) {
	rows := func(kinds ...string) *parse.Schedule {
		s := &parse.Schedule{}
		for _, k := range kinds {
			s.Rows = append(s.Rows, parse.MatchRow{DivisionKind: k})
		}
		return s
	}
	cases := []struct {
		name   string
		in     *parse.Schedule
		want   string
		wantOK bool
	}{
		{"mixed", rows("mixed", "mixed"), "mixed", true},
		{"none はオープンや 4 人制", rows("none"), "none", true},
		{"食い違い", rows("men", "women"), "", false},
		{"行なし", rows(), "", false},
		{"nil", nil, "", false},
	}
	for _, tc := range cases {
		got, ok := DivisionKindFromSchedule(tc.in)
		assert.Equal(t, tc.want, got, tc.name)
		assert.Equal(t, tc.wantOK, ok, tc.name)
	}
}

func TestDivisionKindFromTeamList(t *testing.T) {
	div := func(styles ...string) parse.TeamListDivision {
		d := parse.TeamListDivision{}
		for _, s := range styles {
			d.Entries = append(d.Entries, parse.TeamListEntry{BarStyle: s})
		}
		return d
	}
	cases := []struct {
		name   string
		in     parse.TeamListDivision
		want   string
		wantOK bool
	}{
		{"mixed", div("mixed", "mixed"), "mixed", true},
		{"women", div("women"), "women", true},
		{"univ は数えない", div("univ", "women"), "women", true},
		{"FTitle はメンか none か決められない", div("default", "default"), "", false},
		{"食い違い", div("mixed", "women"), "", false},
		{"チームなし", div(), "", false},
	}
	for _, tc := range cases {
		got, ok := DivisionKindFromTeamList(tc.in)
		assert.Equal(t, tc.want, got, tc.name)
		assert.Equal(t, tc.wantOK, ok, tc.name)
	}
}
