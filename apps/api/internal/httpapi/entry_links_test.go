package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/httpapi"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

func send(t *testing.T, srv http.Handler, method, path string, body []byte, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("X-Internal-Token", testToken)
	for key, value := range header {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

// 本人用のリンクの口。リンクは公式 LINE の受け答え(talk)が発行するものを、ここでは entry から直接発行する。
func TestEntryLinkRoutes(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	tour := b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen })
	division := b.Division(tour.ID)
	entries := entry.NewService(store.NewEntryStore(db),
		entry.WithInviteURL(func(code string) string { return "invite:" + code }))
	srv := httpapi.NewServer(db, testToken, httpapi.WithEntryDesk(entries))
	ctx := context.Background()
	lineUser := func(id string) int64 {
		row := store.LineUser{LineUserID: id, Friend: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, db.Create(&row).Error)
		return row.ID
	}

	links, err := entries.IssueTeamLinks(ctx, lineUser("Urep"))
	require.NoError(t, err)
	require.Len(t, links, 1)
	token := links[0].Token
	rec := send(t, srv, http.MethodGet, "/api/v1/entry-links/"+token, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"purpose":"team"`)
	assert.Contains(t, rec.Body.String(), `"profile":null`, "nothing entered yet")
	assert.Contains(t, rec.Body.String(), `"pastTeams":[]`)

	draft, _ := json.Marshal(map[string]any{"divisionId": division.ID, "fullName": "熱海ディスクス", // 固定の値
		"shortName": "熱海", "repName": "山田 太郎", "repPhone": "", "repEmail": "",
		"members": []any{map[string]any{"name": "佐藤 花子", "number": "12"}}, "answers": []any{}})
	rec = send(t, srv, http.MethodPost, "/api/v1/entry-links/"+token+"/team", draft, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var submitted struct {
		ID         int64  `json:"id"`
		Channel    string `json:"channel"`
		InviteCode string `json:"inviteCode"`
		InviteURL  string `json:"inviteUrl"`
		MemberList []struct {
			ID int64 `json:"id"`
		} `json:"memberList"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &submitted))
	assert.Equal(t, "line", submitted.Channel)
	assert.Equal(t, "invite:"+submitted.InviteCode, submitted.InviteURL)

	member, err := entries.IssueMemberLink(ctx, lineUser("Umember"), submitted.InviteCode)
	require.NoError(t, err)
	register, _ := json.Marshal(map[string]any{"memberId": submitted.MemberList[0].ID, // 固定の値
		"name": "佐藤 花子", "number": "12", "answers": []any{}})
	rec = send(t, srv, http.MethodPost, "/api/v1/entry-links/"+member.Token+"/member", register, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"mine":{`)
	assert.Contains(t, rec.Body.String(), `"profileName":"佐藤 花子"`, "the member's input is kept for next time")
	rec = send(t, srv, http.MethodGet, "/api/v1/entry-links/"+token, nil, nil)
	assert.Contains(t, rec.Body.String(), `"profile":{"name":"山田 太郎","phone":"","email":""}`)

	// 合言葉の違うリンク、目的の違う送信は見つからない
	assert.Equal(t, http.StatusNotFound, send(t, srv, http.MethodGet, "/api/v1/entry-links/nope", nil, nil).Code)
	assert.Equal(t, http.StatusNotFound,
		send(t, srv, http.MethodPost, "/api/v1/entry-links/"+member.Token+"/team", draft, nil).Code)

	// 運営が差し戻した申込を、代表が同じリンクから直して送り直す
	_, err = entries.Decide(ctx, submitted.ID, entry.ActionReturn, "短縮名を直してください", domain.Actor{})
	require.NoError(t, err)
	resubmit := fmt.Sprintf("/api/v1/entry-links/%s/team/%d", token, submitted.ID)
	fixed, _ := json.Marshal(map[string]any{"divisionId": division.ID, "fullName": "熱海ディスクス", // 固定の値
		"shortName": "熱海DC", "repName": "山田 太郎", "repPhone": "", "repEmail": "", "members": []any{},
		"answers": []any{}, "registeredIds": []int64{submitted.MemberList[0].ID}})
	assert.Equal(t, http.StatusNotFound,
		send(t, srv, http.MethodPost, fmt.Sprintf("/api/v1/entry-links/%s/team/%d", member.Token, submitted.ID),
			fixed, nil).Code)
	assert.Equal(t, http.StatusConflict, send(t, srv, http.MethodPost, resubmit, draft, nil).Code,
		"佐藤 registered, but the form did not know it")
	rec = send(t, srv, http.MethodPost, resubmit, fixed, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"status":"pending"`)
	assert.Contains(t, rec.Body.String(), `"returnNote":""`)
	assert.Equal(t, http.StatusConflict, send(t, srv, http.MethodPost, resubmit, fixed, nil).Code,
		"a pending entry is not resubmitted again")
}

// 名簿の代表の行は、招待のリンクを持つほかの人には出さず、選ばせない。代表本人は選べる。
func TestEntryLinkRepRowIsTheRepresentatives(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	tour := b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen })
	division := b.Division(tour.ID)
	entries := entry.NewService(store.NewEntryStore(db))
	srv := httpapi.NewServer(db, testToken, httpapi.WithEntryDesk(entries))
	ctx := context.Background()
	lineUser := func(id string) int64 {
		row := store.LineUser{LineUserID: id, Friend: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, db.Create(&row).Error)
		return row.ID
	}
	rep := lineUser("Urep")
	links, err := entries.IssueTeamLinks(ctx, rep)
	require.NoError(t, err)
	draft, _ := json.Marshal(map[string]any{"divisionId": division.ID, "fullName": "熱海ディスクス", // 固定の値
		"shortName": "熱海", "repName": "山田 太郎", "repPhone": "", "repEmail": "",
		"members": []any{map[string]any{"name": "山田太郎", "number": "1"},
			map[string]any{"name": "佐藤 花子", "number": "12"}}, "answers": []any{}})
	rec := send(t, srv, http.MethodPost, "/api/v1/entry-links/"+links[0].Token+"/team", draft, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var submitted struct {
		InviteCode string `json:"inviteCode"`
		MemberList []struct {
			ID int64 `json:"id"`
		} `json:"memberList"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &submitted))
	repRow := submitted.MemberList[0].ID

	other, err := entries.IssueMemberLink(ctx, lineUser("Uother"), submitted.InviteCode)
	require.NoError(t, err)
	rec = send(t, srv, http.MethodGet, "/api/v1/entry-links/"+other.Token, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), `"name":"山田太郎"`, "the rep's row is not offered")
	assert.Contains(t, rec.Body.String(), `"name":"佐藤 花子"`)
	take, _ := json.Marshal(map[string]any{"memberId": repRow, "name": "山田太郎", "number": "1", // 固定の値
		"answers": []any{}})
	assert.Equal(t, http.StatusConflict,
		send(t, srv, http.MethodPost, "/api/v1/entry-links/"+other.Token+"/member", take, nil).Code)

	own, err := entries.IssueMemberLink(ctx, rep, submitted.InviteCode)
	require.NoError(t, err)
	rec = send(t, srv, http.MethodPost, "/api/v1/entry-links/"+own.Token+"/member", take, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), fmt.Sprintf(`"mine":{"id":%d`, repRow))
}

// 大会が性別と生年月日を聞くと、メンバー登録で答えが要り、運営の画面と書き出しに出て、参加を決めると選手の
// 性別になる。答えは前回の入力として残り、別のチームの登録の画面に前もって入る。
func TestEntryPersonalItems(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	tour := b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen })
	division := b.Division(tour.ID)
	entries := entry.NewService(store.NewEntryStore(f.db))
	ctx := context.Background()
	lineUser := func(id string) int64 {
		row := store.LineUser{LineUserID: id, Friend: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, f.db.Create(&row).Error)
		return row.ID
	}
	questionsURL := fmt.Sprintf("/api/v1/admin/tournaments/%d/entry-questions", tour.ID)
	page := f.call(t, http.StatusOK, http.MethodGet, questionsURL, nil)
	assert.Equal(t, map[string]any{"gender": false, "birthdate": false}, page["personal"])
	saved := f.call(t, http.StatusOK, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/entry-personal", tour.ID),
		map[string]any{"gender": true, "birthdate": true})
	assert.Equal(t, map[string]any{"gender": true, "birthdate": true}, saved)

	submit := func(rep, team string) (int64, string) {
		links, err := entries.IssueTeamLinks(ctx, lineUser(rep))
		require.NoError(t, err)
		draft, _ := json.Marshal(map[string]any{"divisionId": division.ID, "fullName": team, "shortName": team,
			"repName": "山田 太郎", "repPhone": "", "repEmail": "", "members": []any{}, "answers": []any{}})
		rec := send(t, f.srv, http.MethodPost, "/api/v1/entry-links/"+links[0].Token+"/team", draft, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var out struct {
			ID         int64  `json:"id"`
			InviteCode string `json:"inviteCode"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		return out.ID, out.InviteCode
	}
	firstID, firstInvite := submit("Urep1", "熱海")
	member := lineUser("Umember")
	link, err := entries.IssueMemberLink(ctx, member, firstInvite)
	require.NoError(t, err)
	rec := send(t, f.srv, http.MethodGet, "/api/v1/entry-links/"+link.Token, nil, nil)
	assert.Contains(t, rec.Body.String(), `"personal":{"gender":true,"birthdate":true}`)
	assert.Contains(t, rec.Body.String(), `"minePersonal":{"gender":null,"birthdate":null}`)

	join := func(gender, birthdate string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"name": "佐藤 花子", "number": "12", "answers": []any{},
			"gender": gender, "birthdate": birthdate})
		return send(t, f.srv, http.MethodPost, "/api/v1/entry-links/"+link.Token+"/member", body, nil)
	}
	assert.Equal(t, http.StatusBadRequest, join("", "2001-04-02").Code, "聞く項目は答えが要る")
	rec = join("F", "2001-04-02")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"minePersonal":{"gender":"F","birthdate":"2001-04-02"}`)
	assert.NotContains(t, rec.Body.String(), `"memberList"`, "本人用のリンクはほかの人の答えを返さない")

	detail := f.call(t, http.StatusOK, http.MethodGet, fmt.Sprintf("/api/v1/admin/entries/%d", firstID), nil)
	members, ok := detail["memberList"].([]any)
	require.True(t, ok)
	require.Len(t, members, 1)
	first, ok := members[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "F", first["gender"])
	assert.Equal(t, "2001-04-02", first["birthdate"])
	export := f.call(t, http.StatusOK, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/entries/export", tour.ID), nil)
	assert.Equal(t, map[string]any{"gender": true, "birthdate": true}, export["personal"])

	approved := f.call(t, http.StatusOK, http.MethodPost, fmt.Sprintf("/api/v1/admin/entries/%d/decision", firstID),
		map[string]any{"action": "approve", "note": ""})
	approvedMembers, ok := approved["memberList"].([]any)
	require.True(t, ok)
	approvedFirst, ok := approvedMembers[0].(map[string]any)
	require.True(t, ok)
	playerID, ok := approvedFirst["playerId"].(float64)
	require.True(t, ok)
	var player store.Player
	require.NoError(t, f.db.First(&player, int64(playerID)).Error)
	require.NotNil(t, player.Gender)
	assert.Equal(t, "F", *player.Gender, "参加を決めると、答えた性別が選手の性別になる")

	_, secondInvite := submit("Urep2", "伊東")
	second, err := entries.IssueMemberLink(ctx, member, secondInvite)
	require.NoError(t, err)
	rec = send(t, f.srv, http.MethodGet, "/api/v1/entry-links/"+second.Token, nil, nil)
	assert.Contains(t, rec.Body.String(), `"minePersonal":{"gender":"F","birthdate":"2001-04-02"}`,
		"別のチームの登録には前回の答えが入っている")
}
