package httpapi_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 本部スタッフ(hq)の権限(docs/requirements.md「権限モデル」)。担当の大会の当日の進行はできるが、
// 大会と部門とチームの作成、設定、削除はできない。記録は試合ごとの割り当てなしでできる。
func TestHQAuthorization(t *testing.T) {
	f := newAdminFixture(t)
	s := f.setupScoring(t)
	hash, err := auth.HashPassword(adminPassword)
	require.NoError(t, err)
	hq, err := store.CreateUser(
		context.Background(),
		f.db,
		store.UserFields{Email: "hq@example.com", Role: domain.RoleHQ},
		hash,
	)
	require.NoError(t, err)
	f.call(t, http.StatusOK, http.MethodPut, fmt.Sprintf("/api/v1/admin/users/%d/tournaments", hq.ID),
		map[string]any{"tournamentIds": []int64{s.own}})
	token := f.login(t, "hq@example.com")

	// 担当の大会だけが一覧に出る
	list := f.callAs(t, token, http.StatusOK, http.MethodGet, "/api/v1/admin/tournaments", nil)
	require.Len(t, list["tournaments"], 1)

	// 当日の進行: 中断と再開、記録係の割り当て、部門のチームと組(担当の大会だけ)
	f.callAs(t, token, http.StatusOK, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/suspension", s.own),
		map[string]any{"reason": "thunder", "note": "", "resumeAt": nil})
	f.callAs(t, token, http.StatusOK, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/resume", s.own), map[string]any{"minutes": nil})
	f.callAs(t, token, http.StatusForbidden, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/suspension", s.other),
		map[string]any{"reason": "thunder", "note": "", "resumeAt": nil})
	f.callAs(t, token, http.StatusOK, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/assignments", s.own), nil)
	f.callAs(t, token, http.StatusForbidden, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/tournaments/%d/assignments", s.other), nil)
	f.callAs(t, token, http.StatusOK, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/divisions/%d/teams", s.division), nil)
	f.callAs(t, token, http.StatusForbidden, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/divisions/%d/teams", s.otherDivision), nil)
	// 記録係を選ぶためのユーザー一覧は読める
	f.callAs(t, token, http.StatusOK, http.MethodGet, "/api/v1/admin/users", nil)

	// 記録は割り当てのない試合でもできる(記録係は割り当てのない試合を記録できない)
	// 割り当てのない試合(A2)は選手のチームが右側
	goal := map[string]any{"kind": "goal", "side": "right", "scorerId": s.scorer}
	f.callAs(t, f.staffToken, http.StatusForbidden, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/events", s.unassigned), goal)
	f.callAs(t, token, http.StatusCreated, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/events", s.unassigned), goal)
	f.callAs(t, token, http.StatusForbidden, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/matches/%d/events", s.otherMatch), goal)

	// 作成、設定、削除はできない
	f.callAs(t, token, http.StatusForbidden, http.MethodPost, "/api/v1/admin/tournaments",
		map[string]any{"name": map[string]string{"ja": "新しい大会", "en": ""}})
	f.callAs(t, token, http.StatusForbidden, http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/tournaments/%d", s.own), map[string]any{"listed": true})
	f.callAs(t, token, http.StatusForbidden, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/divisions/%d/teams", s.division),
		map[string]any{"fullName": map[string]string{"ja": "追加"}, "shortName": map[string]string{"ja": "追加"}})
	f.callAs(t, token, http.StatusForbidden, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/matches/%d", s.match), nil)
	f.callAs(t, token, http.StatusForbidden, http.MethodPost, "/api/v1/admin/users",
		map[string]any{"email": "new@example.com", "role": "staff", "password": "long enough password 1"})
}

func TestHQCanReopenFinishedMatch(t *testing.T) {
	require.NoError(t, domain.ValidTransition(domain.MatchFinished, domain.MatchLive, domain.RoleHQ))
	require.ErrorIs(
		t,
		domain.ValidTransition(domain.MatchFinished, domain.MatchLive, domain.RoleStaff),
		domain.ErrForbidden,
	)
}
