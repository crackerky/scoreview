package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/httpapi"
	"scoreview/api/internal/store"
	"scoreview/api/internal/testdb"
)

const adminPassword = "admin password 0123"

// adminFixture は管理者 1 人と記録係 1 人がいる空の DB の api。
type adminFixture struct {
	db         *gorm.DB
	srv        *httpapi.Server
	now        time.Time
	adminToken string
	staffToken string
	staffID    int64
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	db := testdb.New(t)
	f := &adminFixture{db: db, now: time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)}
	f.srv = httpapi.NewServer(db, testToken, httpapi.WithClock(func() time.Time { return f.now }))
	hash, err := auth.HashPassword(adminPassword)
	require.NoError(t, err)
	ctx := context.Background()
	_, err = store.CreateUser(
		ctx,
		db,
		store.UserFields{Email: "admin@example.com", Role: domain.RoleAdmin},
		hash,
	)
	require.NoError(t, err)
	staff, err := store.CreateUser(
		ctx,
		db,
		store.UserFields{Email: "staff@example.com", Role: domain.RoleStaff},
		hash,
	)
	require.NoError(t, err)
	f.staffID = staff.ID
	f.adminToken = f.login(t, "admin@example.com")
	f.staffToken = f.login(t, "staff@example.com")
	return f
}

func (f *adminFixture) login(t *testing.T, email string) string {
	t.Helper()
	rec := f.do(
		t,
		"",
		http.MethodPost,
		"/api/v1/auth/login",
		map[string]string{"email": email, "password": adminPassword},
	)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct{ Token string }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Token
}

func (f *adminFixture) do(
	t *testing.T,
	token, method, path string,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else if raw, ok := body.(string); ok {
		reader = bytes.NewReader([]byte(raw))
	} else {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, reader)
	req.Header.Set("X-Internal-Token", testToken)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.srv.ServeHTTP(rec, req)
	return rec
}

// call は管理者として呼び、want のステータスを確かめて応答を map で返す。
func (f *adminFixture) call(t *testing.T, want int, method, path string, body any) map[string]any {
	t.Helper()
	return f.callAs(t, f.adminToken, want, method, path, body)
}

func (f *adminFixture) callAs(
	t *testing.T,
	token string,
	want int,
	method, path string,
	body any,
) map[string]any {
	t.Helper()
	rec := f.do(t, token, method, path, body)
	require.Equal(t, want, rec.Code, "%s %s: %s", method, path, rec.Body.String())
	if rec.Code == http.StatusNoContent {
		assert.Empty(t, rec.Body.String())
		return nil
	}
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func id(value any) int64 {
	number, ok := value.(float64)
	if !ok {
		panic(fmt.Sprintf("not a number: %v", value)) // テストの前提(応答の id は数値)が崩れた
	}
	return int64(number)
}

func TestAuthEndpoints(t *testing.T) {
	f := newAdminFixture(t)

	me := f.call(t, http.StatusOK, http.MethodGet, "/api/v1/auth/me", nil)
	user := me["user"].(map[string]any)
	assert.Equal(t, "admin@example.com", user["email"])
	assert.Equal(t, "admin", user["role"])
	assert.Equal(t, []any{}, user["tournamentIds"])

	// 失敗の文面はメールアドレスの有無で変わらない
	wrong := f.do(t, "", http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "admin@example.com", "password": "wrong password!"})
	unknown := f.do(t, "", http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "nobody@example.com", "password": "wrong password!"})
	assert.Equal(t, http.StatusUnauthorized, wrong.Code)
	assert.Equal(t, wrong.Body.String(), unknown.Body.String())

	f.call(t, http.StatusNoContent, http.MethodPost, "/api/v1/auth/logout", nil)
	assert.Equal(
		t,
		http.StatusUnauthorized,
		f.do(t, f.adminToken, http.MethodGet, "/api/v1/auth/me", nil).Code,
	)
}

func TestLoginRateLimitResponds429(t *testing.T) {
	f := newAdminFixture(t)
	for range auth.FreeLoginFailures {
		f.do(t, "", http.MethodPost, "/api/v1/auth/login",
			map[string]string{"email": "staff@example.com", "password": "wrong password!"})
	}
	rec := f.do(t, "", http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "staff@example.com", "password": adminPassword})
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "1", rec.Header().Get("Retry-After"))
	assert.JSONEq(
		t,
		`{"error":{"code":"too_many_requests","message":"too many login attempts; try again later"}}`,
		rec.Body.String(),
	)
	// 間隔を空ければ正しいパスワードで入れる(締め出さない)
	f.now = f.now.Add(time.Second)
	f.login(t, "staff@example.com")
}

// loginFrom は X-Client-IP つきでログインを送る。
func (f *adminFixture) loginFrom(t *testing.T, ip, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("X-Internal-Token", testToken)
	req.Header.Set("X-Client-IP", ip)
	rec := httptest.NewRecorder()
	f.srv.ServeHTTP(rec, req)
	return rec
}

func TestLoginLimitUsesClientIP(t *testing.T) {
	f := newAdminFixture(t)
	for i := range auth.MaxLoginFailuresPerIP {
		// IPv6 は /64 の範囲で数える(範囲の中でアドレスを変えても同じ利用者)
		ip := fmt.Sprintf("2001:db8:1:2::%x", i+1)
		rec := f.loginFrom(t, ip, fmt.Sprintf("u%d@example.com", i), "wrong password!")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}
	rec := f.loginFrom(t, "2001:db8:1:2:ffff::1", "admin@example.com", adminPassword)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "900", rec.Header().Get("Retry-After"))
	// 別の /64 や IPv4 は数えない
	assert.Equal(t, http.StatusOK, f.loginFrom(t, "2001:db8:1:3::1", "admin@example.com", adminPassword).Code)
	assert.Equal(t, http.StatusOK, f.loginFrom(t, "192.0.2.1", "admin@example.com", adminPassword).Code)
}

func TestLoginRespondsUnavailableWhenPasswordChecksAreBusy(t *testing.T) {
	db := testdb.New(t)
	gate := auth.NewPasswordGate(1, 0)
	srv := httpapi.NewServer(db, testToken, httpapi.WithPasswordGate(gate))
	// 照合中の席がふさがっているうちにログインを送る
	leave, err := gate.Enter(context.Background())
	require.NoError(t, err)
	defer leave()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"wrong password!"}`))
	req.Header.Set("X-Internal-Token", testToken)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "1", rec.Header().Get("Retry-After"))
	assert.JSONEq(t, `{"error":{"code":"unavailable","message":"server is busy; try again shortly"}}`,
		rec.Body.String())
}

func TestSessionExpiresOverHTTP(t *testing.T) {
	f := newAdminFixture(t)
	f.now = f.now.Add(auth.SessionTTL)
	assert.Equal(
		t,
		http.StatusUnauthorized,
		f.do(t, f.adminToken, http.MethodGet, "/api/v1/auth/me", nil).Code,
	)
}

func TestAdminRequiresSession(t *testing.T) {
	f := newAdminFixture(t)
	for _, path := range []string{"/api/v1/admin/tournaments", "/api/v1/admin/users", "/api/v1/auth/me"} {
		rec := f.do(t, "", http.MethodGet, path, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, path)
		rec = f.do(t, "not-a-token", http.MethodGet, path, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, path)
	}
}

// scoringSetup は大会 2 つ(記録係はその 1 つの担当)と、担当大会の試合を作る。
type scoringSetup struct {
	own, other        int64 // 大会 ID
	division          int64
	otherDivision     int64
	match, otherMatch int64
	unassigned        int64 // 担当大会の試合のうち、記録係に割り当てていない試合
	left, right       int64 // チーム ID
	scorer            int64 // 左チームの選手
}

func (f *adminFixture) setupScoring(t *testing.T) scoringSetup {
	t.Helper()
	var s scoringSetup
	s.own = id(f.call(t, http.StatusCreated, http.MethodPost, "/api/v1/admin/tournaments",
		map[string]any{"name": map[string]string{"ja": "担当大会", "en": "Own"}})["id"])
	s.other = id(f.call(t, http.StatusCreated, http.MethodPost, "/api/v1/admin/tournaments",
		map[string]any{"name": map[string]string{"ja": "他の大会", "en": ""}})["id"])
	f.call(
		t,
		http.StatusOK,
		http.MethodPut,
		fmt.Sprintf("/api/v1/admin/users/%d/tournaments", f.staffID),
		map[string]any{"tournamentIds": []int64{s.own}},
	)

	makeMatch := func(tournament int64) (int64, int64, int64, int64) {
		division := id(f.call(t, http.StatusCreated, http.MethodPost,
			fmt.Sprintf("/api/v1/admin/tournaments/%d/divisions", tournament),
			map[string]any{"name": map[string]string{"ja": "オープン", "en": "Open"}})["id"])
		team := func(name string) int64 {
			return id(
				f.call(t, http.StatusCreated, http.MethodPost, fmt.Sprintf("/api/v1/admin/divisions/%d/teams", division),
					map[string]any{"fullName": map[string]string{"ja": name}, "shortName": map[string]string{"ja": name}})["id"],
			)
		}
		left, right := team("左"), team("右")
		match := id(
			f.call(t, http.StatusCreated, http.MethodPost, fmt.Sprintf("/api/v1/admin/divisions/%d/matches", division),
				map[string]any{"code": "A1", "left": map[string]any{"teamId": left}, "right": map[string]any{"teamId": right}})["id"],
		)
		return division, match, left, right
	}
	s.division, s.match, s.left, s.right = makeMatch(s.own)
	s.otherDivision, s.otherMatch, _, _ = makeMatch(s.other)
	s.unassigned = id(f.call(t, http.StatusCreated, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/matches", s.division),
		map[string]any{"code": "A2", "left": map[string]any{"teamId": s.right},
			"right": map[string]any{"teamId": s.left}})["id"])
	f.call(t, http.StatusCreated, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/assignments", s.own),
		map[string]any{"userId": f.staffID, "matchId": s.match})
	s.scorer = id(
		f.call(t, http.StatusCreated, http.MethodPost, fmt.Sprintf("/api/v1/admin/teams/%d/players", s.left),
			map[string]any{"number": "7", "name": map[string]string{"ja": "選手", "en": "Player"}, "gender": "F"})["id"],
	)
	return s
}

func TestStaffAuthorization(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	staff := f.staffToken

	// 担当大会だけが一覧に出る
	list := f.callAs(t, staff, http.StatusOK, http.MethodGet, "/api/v1/admin/tournaments", nil)
	tournaments := list["tournaments"].([]any)
	require.Len(t, tournaments, 1)
	assert.Equal(t, float64(s.own), tournaments[0].(map[string]any)["id"])

	// 担当大会の試合は読み書きできる
	f.callAs(
		t,
		staff,
		http.StatusOK,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/divisions", s.own),
		nil,
	)
	f.callAs(
		t,
		staff,
		http.StatusOK,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/divisions/%d/matches", s.division),
		nil,
	)
	f.callAs(
		t,
		staff,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/events", s.match),
		map[string]any{"kind": "goal", "side": "left", "scorerId": s.scorer},
	)
	f.callAs(
		t,
		staff,
		http.StatusOK,
		http.MethodPut,
		fmt.Sprintf("/api/v1/admin/matches/%d/player-stats", s.match),
		map[string]any{
			"stats": []map[string]any{
				{"playerId": s.scorer, "side": "left", "blocks": 2, "greatests": 0},
			},
		},
	)

	// 他の大会の試合は触れない
	forbidden := []struct{ method, path string }{
		{http.MethodGet, fmt.Sprintf("/api/v1/admin/tournaments/%d", s.other)},
		{http.MethodGet, fmt.Sprintf("/api/v1/admin/divisions/%d/matches", s.otherDivision)},
		{http.MethodGet, fmt.Sprintf("/api/v1/admin/matches/%d", s.otherMatch)},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/matches/%d/events", s.otherMatch)},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/matches/%d/status", s.otherMatch)},
		// 担当大会でも、割り当てのない試合は入力できない
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/matches/%d/events", s.unassigned)},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/matches/%d/result", s.unassigned)},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/tournaments/%d/assignments", s.own)},
		// 担当大会でも、試合以外の登録と編集は admin だけ
		{http.MethodPatch, fmt.Sprintf("/api/v1/admin/tournaments/%d", s.own)},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/divisions/%d/teams", s.division)},
		{http.MethodPatch, fmt.Sprintf("/api/v1/admin/matches/%d", s.match)},
		{http.MethodDelete, fmt.Sprintf("/api/v1/admin/matches/%d", s.match)},
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodPost, "/api/v1/admin/tournaments"},
	}
	for _, tc := range forbidden {
		rec := f.do(t, staff, tc.method, tc.path, `{"kind":"goal","side":"left","status":"live"}`)
		assert.Equal(
			t,
			http.StatusForbidden,
			rec.Code,
			"%s %s: %s",
			tc.method,
			tc.path,
			rec.Body.String(),
		)
	}
	// 存在しない試合は 404(権限より先に資源の有無が分かる)
	assert.Equal(
		t,
		http.StatusNotFound,
		f.do(t, staff, http.MethodGet, "/api/v1/admin/matches/999999", nil).Code,
	)
}

func TestScoringOverHTTP(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	staff := f.staffToken
	path := fmt.Sprintf("/api/v1/admin/matches/%d", s.match)

	f.callAs(
		t,
		staff,
		http.StatusCreated,
		http.MethodPost,
		path+"/events",
		map[string]any{"kind": "pull", "side": "right"},
	)
	f.now = f.now.Add(2 * time.Minute)
	detail := f.callAs(t, staff, http.StatusCreated, http.MethodPost, path+"/events",
		map[string]any{"kind": "goal", "side": "left", "scorerId": s.scorer})
	match := detail["match"].(map[string]any)
	assert.Equal(t, "live", match["status"])
	assert.Equal(t, float64(1), match["left"].(map[string]any)["score"])
	events := detail["events"].([]any)
	assert.Equal(t, float64(120), events[1].(map[string]any)["elapsedSec"], "最初のイベントからの経過秒")
	assert.Equal(t, "7", events[1].(map[string]any)["scorer"].(map[string]any)["number"])
	assert.Len(t, detail["rosters"].(map[string]any)["left"], 1)

	detail = f.callAs(
		t,
		staff,
		http.StatusCreated,
		http.MethodPost,
		path+"/events",
		map[string]any{"kind": "end"},
	)
	assert.Equal(t, "finished", detail["match"].(map[string]any)["status"])
	endID := id(detail["events"].([]any)[2].(map[string]any)["id"])

	// 終了の取り消しは admin だけ
	f.callAs(
		t,
		staff,
		http.StatusForbidden,
		http.MethodDelete,
		fmt.Sprintf("%s/events/%d", path, endID),
		nil,
	)
	detail = f.call(
		t,
		http.StatusOK,
		http.MethodDelete,
		fmt.Sprintf("%s/events/%d", path, endID),
		nil,
	)
	assert.Equal(t, "live", detail["match"].(map[string]any)["status"])

	// 閲覧用の試合ページに反映される
	rec := f.do(t, "", http.MethodGet, fmt.Sprintf("/api/v1/matches/%d", s.match), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var page struct {
		Match struct {
			Status string
			Left   struct{ Score *int }
		}
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	assert.Equal(t, "live", page.Match.Status)
	assert.Equal(t, 1, *page.Match.Left.Score)

	// 状態の変更(試合終了)。不正な値は 400
	f.callAs(
		t,
		staff,
		http.StatusBadRequest,
		http.MethodPost,
		path+"/status",
		map[string]any{"status": "paused"},
	)
	detail = f.callAs(
		t,
		staff,
		http.StatusOK,
		http.MethodPost,
		path+"/status",
		map[string]any{"status": "finished"},
	)
	assert.Equal(t, "finished", detail["match"].(map[string]any)["status"])
	f.callAs(
		t,
		staff,
		http.StatusConflict,
		http.MethodPost,
		path+"/events",
		map[string]any{"kind": "goal", "side": "left"},
	)
}

func TestTournamentCRUD(t *testing.T) {
	f := newAdminFixture(t)
	created := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		"/api/v1/admin/tournaments",
		map[string]any{
			"name":      map[string]string{"ja": "新しい大会", "en": "New"},
			"startDate": "2026-11-01",
			"topHtml": map[string]string{
				"ja": `<p onclick="x()">本文<script>alert(1)</script></p>`,
				"en": "",
			},
		},
	)
	tid := id(created["id"])
	// 既定値と、保存時の無害化
	assert.Equal(t, "entry_open", created["status"])
	assert.Equal(t, true, created["listed"])
	assert.Equal(t, map[string]any{"teamList": "teams", "groups": "groups"}, created["labels"])
	assert.Equal(t, "<p>本文</p>", created["topHtml"].(map[string]any)["ja"])

	// PATCH はボディにあるキーだけ変える
	patched := f.call(
		t,
		http.StatusOK,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/tournaments/%d", tid),
		map[string]any{
			"status":  "finished",
			"name":    map[string]string{"en": "Renamed"},
			"logoUrl": "/logo.png",
		},
	)
	assert.Equal(t, "finished", patched["status"])
	assert.Equal(t, map[string]any{"ja": "新しい大会", "en": "Renamed"}, patched["name"])
	assert.Equal(t, "2026-11-01", patched["startDate"])
	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/tournaments/%d", tid),
		map[string]any{"logoUrl": "javascript:alert(1)"},
	)
	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/tournaments/%d", tid),
		map[string]any{"unknownKey": 1},
	)

	banners := f.call(
		t,
		http.StatusOK,
		http.MethodPut,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/banners", tid),
		map[string]any{"banners": []map[string]any{
			{
				"imageUrl": "/b1.png",
				"linkUrl":  "https://example.com/",
			},
			{"imageUrl": "/b2.png", "linkUrl": nil},
		}},
	)
	assert.Len(t, banners["banners"], 2)
	assert.Len(
		t,
		f.call(t, http.StatusOK, http.MethodGet, fmt.Sprintf("/api/v1/admin/tournaments/%d/banners", tid), nil)["banners"],
		2,
	)

	// 閲覧用の大会トップに出る(一覧に載る)
	top := f.do(t, "", http.MethodGet, fmt.Sprintf("/api/v1/tournaments/%d", tid), nil)
	require.Equal(t, http.StatusOK, top.Code)
	assert.Contains(t, top.Body.String(), `"Renamed"`)
	assert.Contains(t, top.Body.String(), `"/b1.png"`)

	got := f.call(
		t,
		http.StatusOK,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/tournaments/%d", tid),
		nil,
	)
	assert.Equal(t, "/logo.png", got["logoUrl"])
	f.call(
		t,
		http.StatusNoContent,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/tournaments/%d", tid),
		nil,
	)
	f.call(
		t,
		http.StatusNotFound,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/tournaments/%d", tid),
		nil,
	)
}

func TestDivisionTeamPlayerCRUD(t *testing.T) {
	f := newAdminFixture(t)
	tid := id(f.call(t, http.StatusCreated, http.MethodPost, "/api/v1/admin/tournaments",
		map[string]any{"name": map[string]string{"ja": "大会"}})["id"])
	division := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/divisions", tid),
		map[string]any{"name": map[string]string{"ja": "ミックス", "en": "Mixed"}, "kind": "mixed"},
	)
	did := id(division["id"])
	assert.Equal(t, "team", division["entryKind"])
	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/divisions", tid),
		map[string]any{"name": map[string]string{"ja": "x"}, "kind": "coed"},
	)

	team := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/teams", did),
		map[string]any{
			"fullName":  map[string]string{"ja": " メッターによるお料理向上委員会 ", "en": ""},
			"shortName": map[string]string{"ja": "お料理向上委員会", "en": ""}, "barStyle": "mixed",
		},
	)
	teamID := id(team["id"])
	// 正式名は加工せずに保存する
	assert.Equal(t, " メッターによるお料理向上委員会 ", team["fullName"].(map[string]any)["ja"])
	assert.Nil(t, team["finalRank"])

	player := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/teams/%d/players", teamID),
		map[string]any{"number": "00", "name": map[string]string{"ja": "選手"}, "gender": "M"},
	)
	staffMember := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/teams/%d/players", teamID),
		map[string]any{
			"name": map[string]string{"ja": "監督"},
			"role": map[string]string{"ja": "マネージャ", "en": "Manager"},
		},
	)
	assert.Nil(t, player["role"])
	assert.Equal(t, "マネージャ", staffMember["role"].(map[string]any)["ja"])
	f.call(
		t,
		http.StatusOK,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/players/%d", id(player["id"])),
		map[string]any{"gender": nil, "number": "10"},
	)
	got := f.call(
		t,
		http.StatusOK,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/players/%d", id(player["id"])),
		nil,
	)
	assert.Equal(t, "10", got["number"])
	assert.Nil(t, got["gender"])
	players := f.call(
		t,
		http.StatusOK,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/teams/%d/players", teamID),
		nil,
	)
	assert.Len(t, players["players"], 2)

	// 最終順位の登録と公開
	ranks := f.call(
		t,
		http.StatusOK,
		http.MethodPut,
		fmt.Sprintf("/api/v1/admin/divisions/%d/final-ranks", did),
		[]map[string]any{{"teamId": teamID, "rank": 1}},
	)
	assert.Equal(t, float64(1), ranks["teams"].([]any)[0].(map[string]any)["finalRank"])
	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPut,
		fmt.Sprintf("/api/v1/admin/divisions/%d/final-ranks", did),
		[]map[string]any{{"teamId": 999999, "rank": 1}},
	)
	published := f.call(
		t,
		http.StatusOK,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/divisions/%d", did),
		map[string]any{"standingsPublished": true},
	)
	assert.Equal(t, true, published["standingsPublished"])
	assert.Equal(t, "mixed", published["kind"])
	standings := f.do(
		t,
		"",
		http.MethodGet,
		fmt.Sprintf("/api/v1/divisions/%d/standings", did),
		nil,
	)
	assert.Contains(t, standings.Body.String(), `"rank":1`)

	f.call(
		t,
		http.StatusNoContent,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/players/%d", id(staffMember["id"])),
		nil,
	)
	f.call(
		t,
		http.StatusNoContent,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/teams/%d", teamID),
		nil,
	)
	f.call(
		t,
		http.StatusNoContent,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/divisions/%d", did),
		nil,
	)
	f.call(
		t,
		http.StatusNotFound,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/divisions/%d", did),
		nil,
	)
}

func TestGroupsAndBracketGeneration(t *testing.T) {
	f := newAdminFixture(t)
	tid := id(f.call(t, http.StatusCreated, http.MethodPost, "/api/v1/admin/tournaments",
		map[string]any{"name": map[string]string{"ja": "大会"}})["id"])
	did := id(
		f.call(t, http.StatusCreated, http.MethodPost, fmt.Sprintf("/api/v1/admin/tournaments/%d/divisions", tid),
			map[string]any{"name": map[string]string{"ja": "オープン"}})["id"],
	)
	var seeds []map[string]any
	for i := range 4 {
		name := fmt.Sprintf("チーム%d", i+1)
		teamID := id(
			f.call(t, http.StatusCreated, http.MethodPost, fmt.Sprintf("/api/v1/admin/divisions/%d/teams", did),
				map[string]any{"fullName": map[string]string{"ja": name}, "shortName": map[string]string{"ja": name}})["id"],
		)
		seeds = append(seeds, map[string]any{"teamId": teamID})
	}

	pool := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/groups", did),
		map[string]any{"name": "Pool A"},
	)
	assert.Equal(t, "pool", pool["kind"])
	teams := make([]map[string]any, 0, len(seeds))
	for _, seed := range seeds {
		teams = append(teams, map[string]any{"teamId": seed["teamId"], "rankOverride": nil})
	}
	pool = f.call(
		t,
		http.StatusOK,
		http.MethodPut,
		fmt.Sprintf("/api/v1/admin/groups/%d/teams", id(pool["id"])),
		map[string]any{"teams": teams},
	)
	assert.Len(t, pool["teams"], 4)

	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/groups", did),
		map[string]any{"name": "Image", "kind": "image"},
	)
	image := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/groups", did),
		map[string]any{
			"name":  "Image",
			"kind":  "image",
			"image": map[string]any{"url": "/r.png", "width": 500, "height": nil},
		},
	)
	assert.Equal(t, "/r.png", image["image"].(map[string]any)["url"])

	templates := f.call(t, http.StatusOK, http.MethodGet, "/api/v1/admin/bracket-templates", nil)
	assert.Len(t, templates["templates"], 3)

	bracketGroup := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/groups", did),
		map[string]any{"name": "Final", "kind": "bracket"},
	)
	gid := id(bracketGroup["id"])
	assert.Nil(t, bracketGroup["bracket"])
	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/groups/%d/bracket", gid),
		map[string]any{"template": "single8", "codePrefix": "F", "seeds": seeds},
	)
	generated := f.call(
		t,
		http.StatusOK,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/groups/%d/bracket", gid),
		map[string]any{"template": "single4", "codePrefix": "F", "seeds": seeds},
	)
	assert.Equal(
		t,
		map[string]any{"pattern": "T4", "ptnName": "T4 =Final="},
		generated["bracket"],
	)

	matches := f.call(
		t,
		http.StatusOK,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/divisions/%d/matches", did),
		nil,
	)
	require.Len(t, matches["matches"], 4)
	// 閲覧用のグループ成績にトーナメント表が出る
	rec := f.do(t, "", http.MethodGet, fmt.Sprintf("/api/v1/groups/%d", gid), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var page struct {
		Bracket struct {
			Pattern string
			Columns int
			Rows    [][]struct {
				Kind string
				Text string
				Team *struct{ ID int64 }
			}
		}
		Matches []struct{ Code string }
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	assert.Equal(t, "T4", page.Bracket.Pattern)
	assert.Equal(t, 3, page.Bracket.Columns)
	assert.Equal(t, "team", page.Bracket.Rows[2][0].Kind)
	assert.Equal(t, seeds[0]["teamId"], page.Bracket.Rows[2][0].Team.ID)
	assert.Equal(t, "F2", page.Bracket.Rows[3][0].Text, "1 回戦の上は F2(決勝が F1。参照サイトと同じ)")
	assert.Len(t, page.Matches, 4)

	f.call(
		t,
		http.StatusConflict,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/groups/%d/bracket", gid),
		map[string]any{"template": "single4", "codePrefix": "G", "seeds": seeds},
	)
	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/groups/%d/bracket", gid),
		map[string]any{"template": "nope", "codePrefix": "G", "seeds": seeds},
	)
	f.call(
		t,
		http.StatusOK,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/groups/%d", gid),
		map[string]any{"published": false},
	)
	f.call(
		t,
		http.StatusNoContent,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/groups/%d", gid),
		nil,
	)
}

func TestMatchCRUDWithPlaceholder(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	created := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/matches", s.division),
		map[string]any{
			"code": "A2", "date": "2026-10-18", "time": "13:00", "field": "B",
			"left":  map[string]any{"teamId": s.left},
			"right": map[string]any{"placeholder": map[string]any{"code": "A1", "outcome": "win"}},
		},
	)
	mid := id(created["id"])
	assert.Equal(
		t,
		map[string]any{"code": "A1", "outcome": "win"},
		created["right"].(map[string]any)["placeholder"],
	)
	assert.Equal(t, "scheduled", created["status"])
	f.call(t, http.StatusBadRequest, http.MethodPatch, fmt.Sprintf("/api/v1/admin/matches/%d", mid),
		map[string]any{"time": "25:00"})
	patched := f.call(
		t,
		http.StatusOK,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/matches/%d", mid),
		map[string]any{"field": "3"},
	)
	assert.Equal(t, "13:00", patched["time"])
	assert.Equal(t, "3", patched["field"])

	// A1 が終わると A2 の右に勝者が入る
	f.call(
		t,
		http.StatusOK,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/status", s.match),
		map[string]any{"status": "finished", "score": map[string]int{"left": 2, "right": 5}},
	)
	got := f.call(
		t,
		http.StatusOK,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/matches/%d", mid),
		nil,
	)
	right := got["match"].(map[string]any)["right"].(map[string]any)
	assert.Equal(t, float64(s.right), right["team"].(map[string]any)["id"])

	f.call(
		t,
		http.StatusNoContent,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/matches/%d", mid),
		nil,
	)
}

func TestGalleryCRUD(t *testing.T) {
	f := newAdminFixture(t)
	tid := id(f.call(t, http.StatusCreated, http.MethodPost, "/api/v1/admin/tournaments",
		map[string]any{"name": map[string]string{"ja": "大会"}})["id"])
	gallery := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/galleries", tid),
		map[string]any{"name": "Day1"},
	)
	gid := id(gallery["id"])
	gallery = f.call(
		t,
		http.StatusOK,
		http.MethodPut,
		fmt.Sprintf("/api/v1/admin/galleries/%d/photos", gid),
		map[string]any{
			"photos": []map[string]string{
				{"thumbUrl": "/t1.jpg", "fullUrl": "https://example.com/1.jpg"},
			},
		},
	)
	assert.Len(t, gallery["photos"], 1)
	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPut,
		fmt.Sprintf("/api/v1/admin/galleries/%d/photos", gid),
		map[string]any{
			"photos": []map[string]string{
				{"thumbUrl": "data:image/png;base64,AA", "fullUrl": "/1.jpg"},
			},
		},
	)
	f.call(
		t,
		http.StatusOK,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/galleries/%d", gid),
		map[string]any{"name": "Day2"},
	)
	list := f.call(
		t,
		http.StatusOK,
		http.MethodGet,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/galleries", tid),
		nil,
	)
	assert.Equal(t, "Day2", list["galleries"].([]any)[0].(map[string]any)["name"])
	public := f.do(t, "", http.MethodGet, fmt.Sprintf("/api/v1/galleries/%d", gid), nil)
	assert.Contains(t, public.Body.String(), "/t1.jpg")
	f.call(
		t,
		http.StatusNoContent,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/galleries/%d", gid),
		nil,
	)
}

func TestUserCRUD(t *testing.T) {
	f := newAdminFixture(t)
	created := f.call(
		t,
		http.StatusCreated,
		http.MethodPost,
		"/api/v1/admin/users",
		map[string]any{
			"email":       " New@Example.com ",
			"displayName": "記録係",
			"password":    "long enough password",
		},
	)
	uid := id(created["id"])
	assert.Equal(t, "new@example.com", created["email"])
	assert.Equal(t, "staff", created["role"])
	assert.NotContains(t, fmt.Sprint(created), "argon2id", "ハッシュを応答に出さない")
	f.call(t, http.StatusBadRequest, http.MethodPost, "/api/v1/admin/users",
		map[string]any{"email": "short@example.com", "password": "short"})
	f.call(t, http.StatusConflict, http.MethodPost, "/api/v1/admin/users",
		map[string]any{"email": "new@example.com", "password": "long enough password"})

	token := f.login(t, "staff@example.com")
	// パスワードを変えるとそのユーザーのセッションは切れる
	f.call(t, http.StatusOK, http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d", f.staffID),
		map[string]any{"password": "another long password"})
	assert.Equal(
		t,
		http.StatusUnauthorized,
		f.do(t, token, http.MethodGet, "/api/v1/auth/me", nil).Code,
	)

	users := f.call(t, http.StatusOK, http.MethodGet, "/api/v1/admin/users", nil)
	assert.Len(t, users["users"], 3)

	me := f.call(t, http.StatusOK, http.MethodGet, "/api/v1/auth/me", nil)["user"].(map[string]any)
	f.call(
		t,
		http.StatusConflict,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/users/%d", id(me["id"])),
		nil,
	)
	f.call(
		t,
		http.StatusConflict,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/users/%d", id(me["id"])),
		map[string]any{"role": "staff"},
	)
	f.call(
		t,
		http.StatusNoContent,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/users/%d", uid),
		nil,
	)
	f.call(t, http.StatusNotFound, http.MethodGet, fmt.Sprintf("/api/v1/admin/users/%d", uid), nil)
}

func TestBodyLimits(t *testing.T) {
	f := newAdminFixture(t)
	huge := `{"name":{"ja":"` + strings.Repeat("a", 2<<20) + `"}}`
	f.call(t, http.StatusBadRequest, http.MethodPost, "/api/v1/admin/tournaments", huge)
	f.call(
		t,
		http.StatusBadRequest,
		http.MethodPost,
		"/api/v1/admin/tournaments",
		`{"name":{"ja":"a"}} {}`,
	)
}
