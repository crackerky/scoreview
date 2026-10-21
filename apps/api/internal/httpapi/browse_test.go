package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/httpapi"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

// updateGoldenEnv が 1 なら testdata/*.json を現在の応答で書き直す。
// 書き直したら差分が docs/api.md と合っているかを必ず目で確かめること。
const updateGoldenEnv = "UPDATE_GOLDEN"

type fixture struct {
	srv *httpapi.Server
	sc  storetest.Scenario
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testdb.New(t)
	sc := storetest.BuildScenario(t, storetest.New(t, db))
	return fixture{srv: httpapi.NewServer(db, testToken), sc: sc}
}

func (f fixture) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	req.Header.Set("X-Internal-Token", testToken)
	rec := httptest.NewRecorder()
	f.srv.ServeHTTP(rec, req)
	return rec
}

// goldenCases は閲覧用の全エンドポイントの正常系。名前は testdata/{name}.json。
func goldenCases(sc storetest.Scenario) []struct{ name, path string } {
	return []struct{ name, path string }{
		{"tournaments", "/api/v1/tournaments"},
		{"tournament_top", fmt.Sprintf("/api/v1/tournaments/%d", sc.Tournament.ID)},
		{"tournament_teams", fmt.Sprintf("/api/v1/tournaments/%d/teams", sc.Tournament.ID)},
		{"tournament_schedule", fmt.Sprintf("/api/v1/tournaments/%d/schedule", sc.Tournament.ID)},
		{
			"division_schedule",
			fmt.Sprintf("/api/v1/divisions/%d/schedule?sort=time&dir=desc", sc.Mixed.ID),
		},
		{"division_standings", fmt.Sprintf("/api/v1/divisions/%d/standings", sc.Mixed.ID)},
		{"group_pool", fmt.Sprintf("/api/v1/groups/%d", sc.Pool.ID)},
		{"group_bracket", fmt.Sprintf("/api/v1/groups/%d", sc.Bracket.ID)},
		{"group_image", fmt.Sprintf("/api/v1/groups/%d", sc.Image.ID)},
		{"team", fmt.Sprintf("/api/v1/teams/%d", sc.TeamA.ID)},
		{"match", fmt.Sprintf("/api/v1/matches/%d", sc.MatchAB.ID)},
		{"player", fmt.Sprintf("/api/v1/players/%d", sc.PlayerA7.ID)},
		{"player_staff", fmt.Sprintf("/api/v1/players/%d", sc.StaffA.ID)},
		{"gallery", fmt.Sprintf("/api/v1/galleries/%d", sc.Gallery.ID)},
	}
}

// TestBrowseGolden は応答 JSON 全体(キー、null、値)をゴールデンファイルと比べる。
// ゴールデンファイルは api.md の型をそのまま写した形で、契約の変化を差分として見せる。
func TestBrowseGolden(t *testing.T) {
	f := newFixture(t)
	update := os.Getenv(updateGoldenEnv) == "1"
	for _, tc := range goldenCases(f.sc) {
		t.Run(tc.name, func(t *testing.T) {
			rec := f.get(t, tc.path)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
			path := filepath.Join("testdata", tc.name+".json")
			if update {
				var pretty bytes.Buffer
				require.NoError(t, json.Indent(&pretty, rec.Body.Bytes(), "", "  "))
				require.NoError(t, os.WriteFile(path, pretty.Bytes(), 0o600))
				return
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err, "run with %s=1 to create the golden file", updateGoldenEnv)
			assert.JSONEq(t, string(want), rec.Body.String())
		})
	}
}

// TestGoldenKeysAppearInAPIDoc はゴールデンファイルに出るキーが全て docs/api.md の型定義に
// 書かれていることを機械的に確かめる(DTO の json タグの綴り違いや api.md に無いキーを検出する)。
func TestGoldenKeysAppearInAPIDoc(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "api.md"))
	require.NoError(t, err)
	documented := map[string]bool{}
	tsBlock := regexp.MustCompile("(?s)```ts\n(.*?)```")
	key := regexp.MustCompile(`([A-Za-z]+)\??:`)
	for _, block := range tsBlock.FindAllSubmatch(doc, -1) {
		for _, match := range key.FindAllSubmatch(block[1], -1) {
			documented[string(match[1])] = true
		}
	}
	require.NotEmpty(t, documented)

	files, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		var value any
		require.NoError(t, json.Unmarshal(data, &value))
		var missing []string
		walkKeys(value, func(k string) {
			if !documented[k] {
				missing = append(missing, k)
			}
		})
		sort.Strings(missing)
		assert.Empty(t, missing, "%s has keys not in docs/api.md", file)
	}
}

func walkKeys(value any, visit func(string)) {
	switch typed := value.(type) {
	case map[string]any:
		for k, child := range typed {
			visit(k)
			walkKeys(child, visit)
		}
	case []any:
		for _, child := range typed {
			walkKeys(child, visit)
		}
	}
}

func TestBrowseNotFound(t *testing.T) {
	f := newFixture(t)
	cases := []struct{ path, message string }{
		{"/api/v1/tournaments/999999", "TOURNAMENT_ID [999999]"},
		{"/api/v1/tournaments/999999/teams", "TOURNAMENT_ID [999999]"},
		{"/api/v1/tournaments/999999/schedule", "TOURNAMENT_ID [999999]"},
		{"/api/v1/divisions/999999/schedule", "EVENT_DIVISION_ID [999999]"},
		{"/api/v1/divisions/999999/standings", "EVENT_DIVISION_ID [999999]"},
		{"/api/v1/groups/999999", "GROUP_ID [999999]"},
		{"/api/v1/teams/999999", "ENTRY_TEAM_ID [999999]"},
		{"/api/v1/matches/999999", "MATCH_ID [999999]"},
		{"/api/v1/players/999999", "ENTRY_PLAYER_ID [999999]"},
		{"/api/v1/galleries/999999", "TGALLERY_ID [999999]"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rec := f.get(t, tc.path)
			assert.Equal(t, http.StatusNotFound, rec.Code)
			assert.JSONEq(
				t,
				fmt.Sprintf(`{"error":{"code":"not_found","message":%q}}`, tc.message),
				rec.Body.String(),
			)
		})
	}
}

func TestBrowseBadRequest(t *testing.T) {
	f := newFixture(t)
	tour := fmt.Sprintf("/api/v1/tournaments/%d/schedule", f.sc.Tournament.ID)
	div := fmt.Sprintf("/api/v1/divisions/%d/schedule", f.sc.Mixed.ID)
	cases := []struct{ path, field string }{
		{"/api/v1/tournaments/abc", "id"},
		{"/api/v1/tournaments/0", "id"},
		{"/api/v1/teams/-1", "id"},
		{"/api/v1/matches/99999999999999999999", "id"},
		{tour + "?ymd=2026-10-18", "ymd"},
		{tour + "?ymd=20261332", "ymd"},
		{tour + "?sort=status", "sort"},
		{tour + "?sort=time&dir=up", "dir"},
		{div + "?team=abc", "team"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rec := f.get(t, tc.path)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			var body struct {
				Error struct{ Code, Message string } `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "bad_request", body.Error.Code)
			assert.True(t, strings.HasPrefix(body.Error.Message, tc.field+":"), body.Error.Message)
		})
	}
}

// scheduleResponse はスケジュールの応答のうちテストで見る部分。
type scheduleResponse struct {
	Filters struct {
		Selected struct{ Ymd, Team, Field string } `json:"selected"`
	} `json:"filters"`
	Sort *struct{ Key, Dir string } `json:"sort"`
	Rows []struct {
		MatchID int64 `json:"matchId"`
	} `json:"rows"`
}

func (f fixture) schedule(t *testing.T, path string) scheduleResponse {
	t.Helper()
	rec := f.get(t, path)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var out scheduleResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func TestScheduleYMDKeyAbsentVersusEmpty(t *testing.T) {
	f := newFixture(t)
	base := fmt.Sprintf("/api/v1/tournaments/%d/schedule", f.sc.Tournament.ID)

	absent := f.schedule(t, base)
	assert.Equal(t, "20261018", absent.Filters.Selected.Ymd, "no ymd key means the last day")
	assert.Len(t, absent.Rows, 3)

	empty := f.schedule(t, base+"?ymd=")
	assert.Equal(t, "", empty.Filters.Selected.Ymd, "empty ymd means all days")
	assert.Len(t, empty.Rows, 5)

	day := f.schedule(t, base+"?ymd=20261017&field=2")
	assert.Equal(t, "20261017", day.Filters.Selected.Ymd)
	assert.Equal(t, "2", day.Filters.Selected.Field)
	require.Len(t, day.Rows, 1)
	assert.Equal(t, f.sc.MatchCA.ID, day.Rows[0].MatchID)
	assert.Nil(t, day.Sort)

	// コートは文字列の完全一致("B" のような数字でないコートもある)
	letter := f.schedule(t, base+"?ymd=&field=B")
	assert.Equal(t, "B", letter.Filters.Selected.Field)
	require.Len(t, letter.Rows, 1)
	assert.Equal(t, f.sc.MatchWomen.ID, letter.Rows[0].MatchID)

	// dir は sort があるときだけ有効
	sorted := f.schedule(t, base+"?ymd=&dir=desc")
	assert.Nil(t, sorted.Sort)
	sorted = f.schedule(t, base+"?ymd=&sort=leftTeam")
	require.NotNil(t, sorted.Sort)
	assert.Equal(t, "asc", sorted.Sort.Dir)
}

func TestTournamentTopSanitizesHTML(t *testing.T) {
	f := newFixture(t)
	rec := f.get(t, fmt.Sprintf("/api/v1/tournaments/%d", f.sc.Tournament.ID))
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		TopHTML struct{ Ja, En string } `json:"topHtml"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.NotContains(t, body.TopHTML.Ja, "script")
	assert.Contains(t, body.TopHTML.Ja, `<span class="SCV_MIDA_M">【Result】</span>`)
	assert.Equal(t, "", body.TopHTML.En)
}
