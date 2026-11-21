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
