package httpapi_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/httpapi"
	"scoreview/api/internal/notify"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
)

// multicasts は公式アカウントから送ったものを覚える notify.Sender。
type multicasts struct {
	to    [][]string
	texts []string
}

func (m *multicasts) Multicast(_ context.Context, to []string, text, _ string) error {
	m.to, m.texts = append(m.to, to), append(m.texts, text)
	return nil
}

func (m *multicasts) Quota(context.Context) (notify.Quota, error) {
	limit := int64(200)
	return notify.Quota{Limit: &limit, Used: 12}, nil
}

// 管理者が宛先を確かめてから送り、送った記録を読む。同じ画面からの 2 回目は送らない。
func TestNotifications(t *testing.T) {
	f := newAdminFixture(t)
	b := storetest.New(t, f.db)
	tour := b.Tournament(func(row *store.Tournament) { row.NameJa = "ルーキーズファイナル" })
	division := b.Division(tour.ID)
	rep := store.LineUser{LineUserID: "Urep", Friend: true, Name: "山田 太郎", CreatedAt: f.now, UpdatedAt: f.now}
	require.NoError(t, f.db.Create(&rep).Error)
	entryRow := store.TeamEntry{TournamentID: tour.ID, DivisionID: division.ID, TeamNameFull: "熱海ディスクス",
		TeamNameShort: "熱海", RepName: "山田 太郎", RepLineUserID: &rep.ID, Channel: "line", Status: "pending",
		InviteCode: "code-1", SubmittedAt: f.now, CreatedAt: f.now, UpdatedAt: f.now}
	require.NoError(t, f.db.Create(&entryRow).Error)

	sent := &multicasts{}
	f.srv = httpapi.NewServer(f.db, testToken, httpapi.WithClock(func() time.Time { return f.now }),
		httpapi.WithNotifications(notify.NewService(store.NewNotifyStore(f.db),
			notify.WithSender(sent, func(err error) { t.Error(err) }))))
	base := fmt.Sprintf("/api/v1/admin/tournaments/%d/notifications", tour.ID)

	preview := f.call(t, http.StatusOK, http.MethodGet,
		fmt.Sprintf("%s/preview?kind=reps&status=pending&divisionId=%d", base, division.ID), nil)
	assert.Equal(t, "ルーキーズファイナル", preview["heading"])
	assert.InDelta(t, 1, preview["reachable"], 0)
	assert.Equal(t, map[string]any{"limit": float64(200), "used": float64(12)}, preview["quota"])
	assert.Equal(t, true, preview["canSend"])
	assert.Equal(t, []any{map[string]any{"name": "山田 太郎", "teams": []any{"熱海ディスクス"}, "reachable": true}},
		preview["recipients"])
	f.call(t, http.StatusBadRequest, http.MethodGet, base+"/preview?kind=everyone", nil)
	f.call(t, http.StatusBadRequest, http.MethodGet, base+"/preview?kind=reps&divisionId=x", nil)

	send := map[string]any{"audience": map[string]any{"kind": "reps", "status": "", "divisionId": nil},
		"body": "明日は 8 時集合です", "requestKey": "0f8fad5b-d9cb-469f-a165-70867728950e"}
	created := f.call(t, http.StatusCreated, http.MethodPost, base, send)
	assert.Equal(t, "ルーキーズファイナル\n\n明日は 8 時集合です", created["body"])
	assert.Equal(t, map[string]any{"pending": float64(0), "sent": float64(1), "skipped": float64(0),
		"failed": float64(0)}, created["counts"])
	assert.Equal(t, map[string]any{"kind": "reps", "status": "", "divisionId": nil}, created["audience"])
	assert.Equal(t, "admin@example.com", created["sentBy"])
	require.Equal(t, [][]string{{"Urep"}}, sent.to)
	again := f.call(t, http.StatusCreated, http.MethodPost, base, send)
	assert.Equal(t, created["id"], again["id"])
	assert.Len(t, sent.to, 1, "the same request is not sent twice")
	send["body"] = " "
	send["requestKey"] = "other"
	f.call(t, http.StatusBadRequest, http.MethodPost, base, send)

	list := f.call(t, http.StatusOK, http.MethodGet, base, nil)
	assert.Equal(t, true, list["canSend"])
	assert.Len(t, list["notifications"], 1)
	detail := f.call(t, http.StatusOK, http.MethodGet, fmt.Sprintf("/api/v1/admin/notifications/%v", created["id"]), nil)
	jobs, ok := detail["jobs"].([]any)
	require.True(t, ok)
	require.Len(t, jobs, 1)
	job, ok := jobs[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "sent", job["status"])
	assert.Equal(t, "熱海ディスクス", job["teams"])

	// 宛先は申込の代表なので、記録係には読ませない
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodGet, base, nil)
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodGet, base+"/preview?kind=reps", nil)
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodPost, base, send)
	f.call(t, http.StatusNotFound, http.MethodGet, "/api/v1/admin/notifications/999999", nil)
	f.call(t, http.StatusNotFound, http.MethodGet, "/api/v1/admin/tournaments/999999/notifications", nil)
}

// LINE の設定が無ければ、宛先は確かめられるが送れない。
func TestNotificationsWithoutLine(t *testing.T) {
	f := newAdminFixture(t)
	tour := storetest.New(t, f.db).Tournament()
	base := fmt.Sprintf("/api/v1/admin/tournaments/%d/notifications", tour.ID)
	preview := f.call(t, http.StatusOK, http.MethodGet, base+"/preview?kind=past_reps", nil)
	assert.Equal(t, false, preview["canSend"])
	assert.Nil(t, preview["quota"])
	assert.Equal(t, []any{}, preview["recipients"])
	body := f.call(t, http.StatusConflict, http.MethodPost, base, map[string]any{
		"audience": map[string]any{"kind": "past_reps", "status": "", "divisionId": nil},
		"body":     "受付を始めました", "requestKey": "k"})
	problem, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "conflict", problem["code"])
	list := f.call(t, http.StatusOK, http.MethodGet, base, nil)
	assert.Equal(t, false, list["canSend"])
	assert.Equal(t, []any{}, list["notifications"])
}
