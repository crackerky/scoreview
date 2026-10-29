package httpapi

import (
	"context"
	"net/http"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 大会当日の本部の一括操作(docs/api.md「チームの棄権」「大会の締め」「大会の中断」)。管理者だけ。

func (s *Server) dayOfRoutes() {
	const base = "/api/v1/admin"
	ok := http.StatusOK
	s.admin("GET "+base+"/teams/{id}/withdrawal", hqOfTeam(), ok, withPathID(s.getWithdrawal))
	s.admin("POST "+base+"/teams/{id}/withdrawal", hqOfTeam(), ok, withPathID(s.withdrawTeam))
	s.admin("GET "+base+"/divisions/{id}/closing", hqOfDivision(), ok, withPathID(s.getClosing))
	s.admin("POST "+base+"/divisions/{id}/close", hqOfDivision(), ok, withPathID(s.closeDivision))
	s.admin("POST "+base+"/tournaments/{id}/suspension", hqOfTournament(), ok, withPathID(s.suspendTournament))
	s.admin("POST "+base+"/tournaments/{id}/resume", hqOfTournament(), ok, withPathID(s.resumeTournament))
}

type withdrawalMatchDTO struct {
	ID       int64       `json:"id"`
	Code     string      `json:"code"`
	Date     *string     `json:"date"`
	Time     *string     `json:"time"`
	Field    *string     `json:"field"`
	Status   string      `json:"status"`
	Opponent *teamRefDTO `json:"opponent"`
	Result   string      `json:"result"`
}

type withdrawalDTO struct {
	Team         teamRefDTO           `json:"team"`
	DivisionID   int64                `json:"divisionId"`
	WithdrawnAt  *string              `json:"withdrawnAt"`
	Targets      []withdrawalMatchDTO `json:"targets"`
	Waiting      []withdrawalMatchDTO `json:"waiting"`
	ForfeitScore *forfeitScoreDTO     `json:"forfeitScore"`
}

type matchIDsInputDTO struct {
	MatchIDs []int64 `json:"matchIds"`
}

func toWithdrawalMatches(rows []store.WithdrawalMatch) []withdrawalMatchDTO {
	out := make([]withdrawalMatchDTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, withdrawalMatchDTO{ID: m.ID, Code: m.Code, Date: m.Date, Time: m.Time, Field: m.Field,
			Status: string(m.Status), Opponent: toTeamRefPtr(m.Opponent), Result: string(m.Result)})
	}
	return out
}

func toWithdrawal(w store.Withdrawal) withdrawalDTO {
	out := withdrawalDTO{Team: toTeamRef(w.Team), DivisionID: w.DivisionID,
		Targets: toWithdrawalMatches(w.Targets), Waiting: toWithdrawalMatches(w.Waiting)}
	if w.WithdrawnAt != nil {
		at := formatInstant(*w.WithdrawnAt)
		out.WithdrawnAt = &at
	}
	if w.ForfeitScore != nil {
		out.ForfeitScore = &forfeitScoreDTO{Winner: w.ForfeitScore.Winner, Loser: w.ForfeitScore.Loser}
	}
	return out
}

func (s *Server) getWithdrawal(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	w, err := store.GetWithdrawal(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toWithdrawal(w), nil
}

func (s *Server) withdrawTeam(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var in matchIDsInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	if in.MatchIDs == nil {
		return nil, domain.Invalid("matchIds", "is required")
	}
	w, err := store.WithdrawTeam(ctx, s.db, actorOf(user), id, in.MatchIDs, s.now())
	if err != nil {
		return nil, err
	}
	return toWithdrawal(w), nil
}

type closingRankDTO struct {
	Rank     int        `json:"rank"`
	Team     teamRefDTO `json:"team"`
	Tied     bool       `json:"tied"`
	Proposed bool       `json:"proposed"`
}

type closingDTO struct {
	Remaining []progressMatchDTO `json:"remaining"`
	Ranks     []closingRankDTO   `json:"ranks"`
	Ties      []string           `json:"ties"`
	ClosedAt  *string            `json:"closedAt"`
	Published bool               `json:"published"`
}

type closeInputDTO struct {
	MatchIDs []int64 `json:"matchIds"`
	Publish  bool    `json:"publish"`
}

func toClosing(c store.Closing) closingDTO {
	out := closingDTO{Remaining: toProgressMatches(c.Remaining), Ranks: []closingRankDTO{},
		Ties: nonNil(c.Ties), Published: c.Published}
	for _, r := range c.Ranks {
		out.Ranks = append(out.Ranks, closingRankDTO{Rank: r.Rank, Team: toTeamRef(r.Team), Tied: r.Tied,
			Proposed: r.Proposed})
	}
	if c.ClosedAt != nil {
		at := formatInstant(*c.ClosedAt)
		out.ClosedAt = &at
	}
	return out
}

func (s *Server) getClosing(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	if _, err := store.GetDivision(ctx, s.db, id); err != nil {
		return nil, err
	}
	c, err := store.GetClosing(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toClosing(c), nil
}

func (s *Server) closeDivision(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	if _, err := store.GetDivision(ctx, s.db, id); err != nil {
		return nil, err
	}
	var in closeInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	if in.MatchIDs == nil {
		return nil, domain.Invalid("matchIds", "is required")
	}
	c, err := store.CloseDivision(ctx, s.db, actorOf(user), id, in.MatchIDs, in.Publish, s.now())
	if err != nil {
		return nil, err
	}
	return toClosing(c), nil
}

type suspensionDTO struct {
	Reason   string  `json:"reason"`
	Note     string  `json:"note"`
	ResumeAt *string `json:"resumeAt"`
	Since    string  `json:"since"`
}

func toSuspension(sp *store.Suspension) *suspensionDTO {
	if sp == nil {
		return nil
	}
	return &suspensionDTO{Reason: sp.Reason, Note: sp.Note, ResumeAt: sp.ResumeAt, Since: formatInstant(sp.Since)}
}

type suspendInputDTO struct {
	Reason   string  `json:"reason"`
	Note     string  `json:"note"`
	ResumeAt *string `json:"resumeAt"`
}

type resumeInputDTO struct {
	Minutes *int `json:"minutes"`
}

type overEndDTO struct {
	MatchID     int64  `json:"matchId"`
	Code        string `json:"code"`
	Division    string `json:"division"`
	Time        string `json:"time"`
	OverMinutes int    `json:"overMinutes"`
}

type resumeDTO struct {
	Minutes int          `json:"minutes"`
	Shifted int          `json:"shifted"`
	OverEnd []overEndDTO `json:"overEnd"`
}

func (s *Server) suspendTournament(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var in suspendInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	sp, err := store.SuspendTournament(ctx, s.db, actorOf(user), id,
		store.SuspensionInput{Reason: in.Reason, Note: in.Note, ResumeAt: in.ResumeAt}, s.now())
	if err != nil {
		return nil, err
	}
	return toSuspension(&sp), nil
}

func (s *Server) resumeTournament(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var in resumeInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	result, err := store.ResumeTournament(ctx, s.db, actorOf(user), id, in.Minutes, s.now())
	if err != nil {
		return nil, err
	}
	out := resumeDTO{Minutes: result.Minutes, Shifted: result.Shifted, OverEnd: []overEndDTO{}}
	for _, o := range result.OverEnd {
		out.OverEnd = append(out.OverEnd, overEndDTO(o))
	}
	return out, nil
}
