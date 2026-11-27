package httpapi_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/httpapi"
	"scoreview/api/internal/line"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/talk"
	"scoreview/api/internal/testdb"
)

const lineSecret = "line-channel-secret"

// replies は公式アカウントから送ったものを覚える。texts は返信、pushes は宛先ごとの知らせ。
type replies struct {
	texts  []string
	pushes map[string][]string
}

func (r *replies) Push(_ context.Context, to string, messages []talk.Message) error {
	if r.pushes == nil {
		r.pushes = map[string][]string{}
	}
	for _, message := range messages {
		r.pushes[to] = append(r.pushes[to], message.Text)
	}
	return nil
}

func (r *replies) Reply(_ context.Context, _ string, messages []talk.Message) error {
	for _, message := range messages {
		r.texts = append(r.texts, message.Text)
	}
	return nil
}

func lineEvent(id, userID, text string) []byte {
	body, _ := json.Marshal(map[string]any{"destination": "Ubot", "events": []any{map[string]any{ // 固定の値の JSON は失敗しない
		"type": "message", "mode": "active", "timestamp": 1763600000000, "webhookEventId": id, "replyToken": "r" + id,
		"source":  map[string]any{"type": "user", "userId": userID},
		"message": map[string]any{"id": id, "type": "text", "text": text}}}})
	return body
}

func signature(body []byte) map[string]string {
	mac := hmac.New(sha256.New, []byte(lineSecret))
	mac.Write(body)
	return map[string]string{"X-Line-Signature": base64.StdEncoding.EncodeToString(mac.Sum(nil))}
}

// linkToken は返信の文から本人用のリンクの合言葉を取り出す。
func linkToken(t *testing.T, text string) string {
	t.Helper()
	found := regexp.MustCompile(`https://score\.example/entry/([A-Za-z0-9_-]+)`).FindStringSubmatch(text)
	require.NotNil(t, found, text)
	return found[1]
}

// 公式 LINE で「エントリー」と送った代表が、代表を選んで返ってきた本人用のリンクで申し込み、招待リンクで
// メンバーが「参加」を送って登録するまで。
func TestLineEntryFlow(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	tour := b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen })
	division := b.Division(tour.ID)
	sent := &replies{}
	lines := store.NewLineStore(db)
	entries := entry.NewService(store.NewEntryStore(db), entry.WithInviteURL(line.InviteURL("@ultimate")),
		entry.WithNotifier(talk.NewNotifier(lines, sent, "https://score.example"), func(err error) { t.Error(err) }))
	bot := talk.NewBot(lines, entries, sent, "https://score.example")
	srv := httpapi.NewServer(db, testToken, httpapi.WithEntryDesk(entries),
		httpapi.WithLineWebhook(line.NewWebhook(lineSecret, bot)))

	body := lineEvent("01A", "Urep", "エントリー")
	rec := send(t, srv, http.MethodPost, "/api/v1/line/webhook", body, map[string]string{"X-Line-Signature": "bad"})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, sent.texts)

	rec = send(t, srv, http.MethodPost, "/api/v1/line/webhook", body, signature(body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, []string{talk.TextEntryChoice}, sent.texts, "first, the rep or the member is asked")
	sent.texts = nil

	// 「代表として申し込む」(ボタンを押さずに打った言い方)で、大会の申込のリンクが返る
	body = lineEvent("01C", "Urep", "代表として申し込む")
	rec = send(t, srv, http.MethodPost, "/api/v1/line/webhook", body, signature(body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, sent.texts, 1)
	token := linkToken(t, sent.texts[0])

	rec = send(t, srv, http.MethodGet, "/api/v1/entry-links/"+token, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var view map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
	assert.Equal(t, "team", view["purpose"])

	draft, _ := json.Marshal(map[string]any{"divisionId": division.ID, "fullName": "熱海ディスクス", // 固定の値
		"shortName": "熱海", "repName": "山田 太郎", "repPhone": "", "repEmail": "",
		"members": []any{map[string]any{"name": "佐藤 花子", "number": "12"}}, "answers": []any{}})
	rec = send(t, srv, http.MethodPost, "/api/v1/entry-links/"+token+"/team", draft, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var submitted map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &submitted))
	assert.Equal(t, "line", submitted["channel"])
	// 代表には、公式アカウントから受付の知らせと、転送できるメンバーへの登録のお願いが届く
	require.Len(t, sent.pushes["Urep"], 2)
	assert.Contains(t, sent.pushes["Urep"][0], "「熱海ディスクス」の申込を受け付けました")
	assert.Contains(t, sent.pushes["Urep"][1], "https://line.me/R/oaMessage/%40ultimate/")
	invite, ok := submitted["inviteUrl"].(string)
	require.True(t, ok)
	assert.Contains(t, invite, "https://line.me/R/oaMessage/%40ultimate/?%E5%8F%82%E5%8A%A0%20")

	// 招待リンクで開いたトークの「参加 招待コード」
	code, ok := submitted["inviteCode"].(string)
	require.True(t, ok)
	join := lineEvent("01B", "Umember", "参加 "+code)
	rec = send(t, srv, http.MethodPost, "/api/v1/line/webhook", join, signature(join))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, sent.texts, 2)
	assert.Contains(t, sent.texts[1], "「熱海ディスクス」のメンバー登録")
	memberToken := linkToken(t, sent.texts[1])

	rec = send(t, srv, http.MethodGet, "/api/v1/entry-links/"+memberToken, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var memberView struct {
		Member struct {
			Unclaimed []struct {
				ID int64 `json:"id"`
			} `json:"unclaimed"`
		} `json:"member"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &memberView))
	require.Len(t, memberView.Member.Unclaimed, 1)
	register, _ := json.Marshal(map[string]any{"memberId": memberView.Member.Unclaimed[0].ID, // 固定の値
		"name": "佐藤 花子", "number": "12", "answers": []any{}})
	rec = send(t, srv, http.MethodPost, "/api/v1/entry-links/"+memberToken+"/member", register, nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"mine":{`)

	// 同じイベントがもう一度届いても、2 回は返さない
	rec = send(t, srv, http.MethodPost, "/api/v1/line/webhook", join, signature(join))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, sent.texts, 2)
}

func TestLineWebhookIsOffWithoutSettings(t *testing.T) {
	srv := httpapi.NewServer(testdb.New(t), testToken)
	body := lineEvent("01A", "U1", "エントリー")
	assert.Equal(t, http.StatusNotFound,
		send(t, srv, http.MethodPost, "/api/v1/line/webhook", body, signature(body)).Code)
}
