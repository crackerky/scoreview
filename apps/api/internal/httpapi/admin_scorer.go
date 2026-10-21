package httpapi

import (
	"context"
	"net/http"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 記録画面を複数の端末で開いたときのためのエンドポイント(docs/api.md「試合の記録担当」)。
// 試合の記録担当の引き受けと延長、解放、得点のチームの付け替え。

type scorerClaimDTO struct {
	UserID    int64  `json:"userId"`
	Name      string `json:"name"`
	DeviceID  string `json:"deviceId"`
	ExpiresAt string `json:"expiresAt"`
}

type concurrentRecordDTO struct {
	Kind           string `json:"kind"`
	FirstClientID  string `json:"firstClientId"`
	SecondClientID string `json:"secondClientId"`
	FirstUser      string `json:"firstUser"`
	SecondUser     string `json:"secondUser"`
	SecondsApart   int    `json:"secondsApart"`
}

type scorerBodyDTO struct {
	DeviceID string `json:"deviceId"`
	Takeover bool   `json:"takeover"`
}

type scorerResultDTO struct {
	Scorer scorerClaimDTO `json:"scorer"`
	Mine   bool           `json:"mine"`
}

type goalSideBodyDTO struct {
	ClientID *string `json:"clientId"`
	Side     string  `json:"side"`
}

func toScorerClaimDTO(claim *store.ScorerClaim) *scorerClaimDTO {
	if claim == nil {
		return nil
	}
	return &scorerClaimDTO{
		UserID: claim.UserID, Name: claim.UserName, DeviceID: claim.DeviceID,
		ExpiresAt: formatInstant(claim.ExpiresAt),
	}
}

func toConcurrentRecordDTOs(records []store.ConcurrentRecordView) []concurrentRecordDTO {
	out := make([]concurrentRecordDTO, len(records))
	for i, record := range records {
		out[i] = concurrentRecordDTO{
			Kind: string(record.Kind), FirstClientID: record.FirstClientID,
			SecondClientID: record.SecondClientID, FirstUser: record.FirstUser,
			SecondUser: record.SecondUser, SecondsApart: record.SecondsApart,
		}
	}
	return out
}

func (s *Server) claimScorer(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	var body scorerBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	result, err := store.ClaimScorer(
		ctx, s.db, actorOf(user), id, body.DeviceID, body.Takeover, s.now(),
	)
	if err != nil {
		return nil, err
	}
	return scorerResultDTO{Scorer: *toScorerClaimDTO(&result.Scorer), Mine: result.Mine}, nil
}

func (s *Server) releaseScorer(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	return nil, store.ReleaseScorer(ctx, s.db, actorOf(user), id, r.URL.Query().Get("deviceId"))
}

func (s *Server) moveGoal(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	ref, err := eventRef(r)
	if err != nil {
		return nil, err
	}
	var body goalSideBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	detail, err := store.MoveGoalToSide(
		ctx, s.db, actorOf(user), id, ref, domain.Side(body.Side), body.ClientID,
	)
	if err != nil {
		return nil, err
	}
	return toMatchDetail(detail), nil
}
