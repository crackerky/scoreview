package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 記録係の画面のためのエンドポイント(docs/api.md「試合」「監査ログ」「記録係の割り当て」)。
// 得点経過の付け直し、通常以外の結果、試合の監査ログ、自分の担当試合、割り当ての登録。

// eventRef はパスの {eventId} を、正の整数なら得点経過の id、UUID なら端末が付けた clientId として読む
// (通信が切れている間に記録した得点経過は、端末が api の id を知らないため)。
func eventRef(r *http.Request) (store.EventRef, error) {
	value := r.PathValue("eventId")
	if domain.IsClientID(value) {
		return store.EventRef{ClientID: value}, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return store.EventRef{}, domain.Invalid("eventId", "must be a positive integer or a UUID")
	}
	return store.EventRef{ID: id}, nil
}

func (s *Server) patchEvent(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	ref, err := eventRef(r)
	if err != nil {
		return nil, err
	}
	var body eventPlayersBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	detail, err := store.UpdateMatchEvent(ctx, s.db, actorOf(user), id, ref, store.EventPlayersInput{
		ScorerID: body.ScorerID, AssistID: body.AssistID, Callahan: body.Callahan,
		ClientID: body.ClientID,
	})
	if err != nil {
		return nil, err
	}
	return toMatchDetail(detail), nil
}

func (s *Server) setResult(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	var body resultBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	detail, err := store.SetMatchResult(ctx, s.db, actorOf(user), id, store.ResultInput{
		Type:          domain.ResultType(body.ResultType),
		WinnerSide:    toSidePtr(body.WinnerSide),
		ForfeitScore:  body.ForfeitScore.toDomain(),
		ClientID:      body.ClientID,
		ExpectedScore: body.ExpectedScore.toDomain(),
	})
	if err != nil {
		return nil, err
	}
	return toMatchDetail(detail), nil
}

// 監査ログ

type auditMatchStateDTO struct {
	Status     string  `json:"status"`
	ResultType string  `json:"resultType"`
	LeftScore  *int    `json:"leftScore"`
	RightScore *int    `json:"rightScore"`
	WinnerSide *string `json:"winnerSide"`
}

type auditPlayerDTO struct {
	ID     int64  `json:"id"`
	Number string `json:"number"`
	Name   string `json:"name"`
}

type auditEventDTO struct {
	ClientID   string          `json:"clientId"`
	Kind       string          `json:"kind"`
	Side       *string         `json:"side"`
	ElapsedSec int             `json:"elapsedSec"`
	Scorer     *auditPlayerDTO `json:"scorer"`
	Assist     *auditPlayerDTO `json:"assist"`
	Callahan   bool            `json:"callahan"`
}

type auditSnapshotDTO struct {
	Match     *auditMatchStateDTO `json:"match"`
	Event     *auditEventDTO      `json:"event"`
	FinalRank *int                `json:"finalRank"`
}

type auditEntryDTO struct {
	ID      int64            `json:"id"`
	At      string           `json:"at"`
	User    *auditUserDTO    `json:"user"`
	Action  string           `json:"action"`
	MatchID *int64           `json:"matchId"`
	TeamID  *int64           `json:"teamId"`
	Before  auditSnapshotDTO `json:"before"`
	After   auditSnapshotDTO `json:"after"`
}

type auditUserDTO struct {
	ID   int64   `json:"id"`
	Name string  `json:"name"`
	Role *string `json:"role"`
}

type auditListDTO struct {
	Entries []auditEntryDTO `json:"entries"`
}

func toAuditPlayer(player *store.AuditPlayer) *auditPlayerDTO {
	if player == nil {
		return nil
	}
	return &auditPlayerDTO{ID: player.ID, Number: player.Number, Name: player.Name}
}

func toAuditSnapshot(snapshot store.AuditSnapshot) auditSnapshotDTO {
	out := auditSnapshotDTO{FinalRank: snapshot.FinalRank}
	if state := snapshot.Match; state != nil {
		out.Match = &auditMatchStateDTO{
			Status: string(state.Status), ResultType: resultTypeString(state.ResultType),
			LeftScore: state.LeftScore, RightScore: state.RightScore,
			WinnerSide: sidePtrString(state.WinnerSide),
		}
	}
	if event := snapshot.Event; event != nil {
		out.Event = &auditEventDTO{
			ClientID: event.ClientID, Kind: string(event.Kind), Side: sidePtrString(event.Side),
			ElapsedSec: event.ElapsedSec, Scorer: toAuditPlayer(event.Scorer),
			Assist: toAuditPlayer(event.Assist), Callahan: event.Callahan,
		}
	}
	return out
}

func toAuditList(records []store.AuditRecord) auditListDTO {
	out := auditListDTO{Entries: make([]auditEntryDTO, len(records))}
	for i, record := range records {
		entry := auditEntryDTO{
			ID: record.ID, At: formatInstant(record.At), Action: record.Action,
			MatchID: record.MatchID, TeamID: record.TeamID,
			Before: toAuditSnapshot(record.Before), After: toAuditSnapshot(record.After),
		}
		if record.UserID != nil {
			entry.User = &auditUserDTO{ID: *record.UserID, Name: record.UserName}
			if record.UserRole != "" {
				role := string(record.UserRole)
				entry.User.Role = &role
			}
		}
		out.Entries[i] = entry
	}
	return out
}

func (s *Server) matchAudit(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	records, err := store.ListMatchAudit(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAuditList(records), nil
}

// 記録係の割り当て

type assignmentInputDTO struct {
	UserID  int64   `json:"userId"`
	MatchID *int64  `json:"matchId"`
	Field   *string `json:"field"`
	Date    *string `json:"date"`
	From    *string `json:"from"`
	To      *string `json:"to"`
}

type assignmentDTO struct {
	ID        int64   `json:"id"`
	UserName  string  `json:"userName"`
	MatchCode *string `json:"matchCode"`
	assignmentInputDTO
}

type assignmentListDTO struct {
	Assignments []assignmentDTO `json:"assignments"`
}

func toAssignmentDTO(record store.AssignmentRecord) assignmentDTO {
	return assignmentDTO{
		ID: record.ID, UserName: record.UserName, MatchCode: record.MatchCode,
		assignmentInputDTO: assignmentInputDTO{
			UserID: record.UserID, MatchID: record.MatchID, Field: record.Field,
			Date: record.Date, From: record.From, To: record.To,
		},
	}
}

func (s *Server) listAssignments(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	records, err := store.ListAssignments(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	out := assignmentListDTO{Assignments: make([]assignmentDTO, len(records))}
	for i, record := range records {
		out.Assignments[i] = toAssignmentDTO(record)
	}
	return out, nil
}

func (s *Server) createAssignment(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var body assignmentInputDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	record, err := store.CreateAssignment(ctx, s.db, id, store.AssignmentFields{
		UserID: body.UserID,
		Assignment: domain.Assignment{
			MatchID: body.MatchID, Field: body.Field, Date: body.Date, From: body.From, To: body.To,
		},
	})
	if err != nil {
		return nil, err
	}
	return toAssignmentDTO(record), nil
}

func (s *Server) deleteAssignment(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	return nil, store.DeleteAssignment(ctx, s.db, id)
}

type assignedMatchDTO struct {
	adminMatchDTO
	TournamentID   int64  `json:"tournamentId"`
	TournamentName string `json:"tournamentName"`
	DivisionName   string `json:"divisionName"`
}

type assignedMatchListDTO struct {
	Matches []assignedMatchDTO `json:"matches"`
}

func (s *Server) myMatches(r *http.Request, user domain.User) (any, error) {
	records, err := store.ListAssignedMatches(r.Context(), s.db, user.ID)
	if err != nil {
		return nil, err
	}
	out := assignedMatchListDTO{Matches: make([]assignedMatchDTO, len(records))}
	for i, record := range records {
		out.Matches[i] = assignedMatchDTO{
			adminMatchDTO: toAdminMatch(record.Match), TournamentID: record.TournamentID,
			TournamentName: record.TournamentName, DivisionName: record.DivisionName,
		}
	}
	return out, nil
}
