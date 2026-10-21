package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
	"scoreview/api/internal/store"
)

// 大会の形と進行の管理用エンドポイント(docs/api.md「大会の形と進行」)。進行状況の読み取りは
// 記録係にも許し、それ以外は管理者だけ。

func (s *Server) formatRoutes() {
	const base = "/api/v1/admin"
	ok, noContent := http.StatusOK, http.StatusNoContent

	s.admin("POST "+base+"/divisions/{id}/format/proposals", adminOnly(), ok, withPathID(s.proposeFormats))
	s.admin("POST "+base+"/divisions/{id}/format/preview", adminOnly(), ok, withPathID(s.previewFormat))
	s.admin("GET "+base+"/divisions/{id}/format", adminOnly(), ok, withPathID(s.getFormat))
	s.admin("PUT "+base+"/divisions/{id}/format", adminOnly(), ok, withPathID(s.applyFormat))
	s.admin("GET "+base+"/formats", adminOnly(), ok, s.listFormats)
	s.admin("GET "+base+"/divisions/{id}/progress", staffOfDivision(), ok, withPathID(s.getProgress))
	s.admin("GET "+base+"/divisions/{id}/format/matches", staffOfDivision(), ok,
		withPathID(s.listFormatMatches))
	s.admin("POST "+base+"/groups/{id}/confirm", adminOnly(), noContent, withPathID(s.confirmGroup))
	s.admin("POST "+base+"/stages/{id}/confirm", adminOnly(), noContent, withPathID(s.confirmStage))
	s.admin("POST "+base+"/groups/{id}/rank-decisions", adminOnly(), noContent, withPathID(s.decideRank))
	s.admin("POST "+base+"/stages/{id}/rank-decisions", adminOnly(), noContent,
		withPathID(s.decideBandRank))
	s.admin("POST "+base+"/matches/{id}/correction-impact", adminOnly(), ok,
		withPathID(s.correctionImpact))
	s.admin("POST "+base+"/matches/{id}/correction", adminOnly(), ok, withPathID(s.applyCorrection))
	s.admin("POST "+base+"/matches/{id}/align-teams", adminOnly(), noContent, withPathID(s.alignTeams))
	s.admin("GET "+base+"/divisions/{id}/schedule-check", adminOnly(), ok, withPathID(s.checkSchedule))
	s.admin("POST "+base+"/divisions/{id}/schedule-shift", adminOnly(), ok, withPathID(s.shiftSchedule))
	s.admin("POST "+base+"/matches/{id}/swap-schedule", adminOnly(), ok, withPathID(s.swapSchedule))
	s.setupRoutes()
}

func (s *Server) proposeFormats(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	if _, err := store.GetDivision(ctx, s.db, id); err != nil {
		return nil, err
	}
	var in proposalInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	if in.Schedule != nil {
		cfg, err := in.Schedule.toSetting().Config()
		if err != nil {
			return nil, err
		}
		return proposalList(format.ProposalInput{Teams: in.Teams, Sport: format.Sport(in.Sport), Schedule: &cfg})
	}
	day, err := domain.ParseDay("date", in.Date)
	if err != nil {
		return nil, err
	}
	if err := domain.CheckClock("start", in.Start); err != nil {
		return nil, err
	}
	clock, err := time.Parse("15:04", in.Start)
	if err != nil {
		return nil, domain.Invalid("start", "must be HH:MM")
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, time.UTC)
	return proposalList(format.ProposalInput{Teams: in.Teams, Courts: in.Courts,
		SlotMinutes: in.SlotMinutes, Start: start, Sport: format.Sport(in.Sport)})
}

func proposalList(input format.ProposalInput) (any, error) {
	proposals, err := format.Propose(input)
	if err != nil {
		return nil, err
	}
	out := proposalListDTO{Proposals: []proposalDTO{}}
	for _, p := range proposals {
		dto, err := toProposal(p)
		if err != nil {
			return nil, err
		}
		out.Proposals = append(out.Proposals, dto)
	}
	return out, nil
}

func readFormatInput(r *http.Request) (store.FormatInput, error) {
	var in formatInputDTO
	if err := decodeBody(r, &in); err != nil {
		return store.FormatInput{}, err
	}
	return in.toInput()
}

func (s *Server) previewFormat(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	input, err := readFormatInput(r)
	if err != nil {
		return nil, err
	}
	preview, err := store.PreviewFormat(ctx, s.db, id, input)
	if err != nil {
		return nil, err
	}
	return toFormatPreview(preview), nil
}

func (s *Server) applyFormat(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	input, err := readFormatInput(r)
	if err != nil {
		return nil, err
	}
	preview, err := store.ApplyFormat(ctx, s.db, actorOf(user), id, input)
	if err != nil {
		return nil, err
	}
	return toFormatPreview(preview), nil
}

func (s *Server) getFormat(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	saved, ok, err := store.GetSavedFormat(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if ok {
		dto, err := toSavedFormat(saved)
		if err != nil {
			return nil, err
		}
		return savedFormatOneDTO{Format: &dto}, nil
	}
	inherited, ok, err := store.GetInheritedFormat(ctx, s.db, id)
	if err != nil || !ok {
		return savedFormatOneDTO{}, err
	}
	dto, err := toSavedFormat(inherited)
	if err != nil {
		return nil, err
	}
	return savedFormatOneDTO{Inherited: &dto}, nil
}

func (s *Server) listFormats(r *http.Request, _ domain.User) (any, error) {
	var exclude int64
	if raw := r.URL.Query().Get("exclude"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return nil, domain.Invalid("exclude", "must be a positive integer")
		}
		exclude = value
	}
	formats, err := store.ListSavedFormats(r.Context(), s.db, exclude)
	if err != nil {
		return nil, err
	}
	out := savedFormatListDTO{Formats: []savedFormatDTO{}}
	for _, f := range formats {
		dto, err := toSavedFormat(f)
		if err != nil {
			return nil, err
		}
		out.Formats = append(out.Formats, dto)
	}
	return out, nil
}

func (s *Server) getProgress(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	now := s.now()
	progress, err := store.GetFormatProgress(ctx, s.db, id, now)
	if err != nil {
		return nil, err
	}
	return toProgress(progress, now), nil
}

func (s *Server) listFormatMatches(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	matches, err := store.ListFormatMatches(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return progressMatchListDTO{Matches: toProgressMatches(matches)}, nil
}

func (s *Server) confirmGroup(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var in confirmInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	return nil, store.ConfirmGroup(ctx, s.db, actorOf(user), id, in.Unplayed, s.now())
}

func (s *Server) confirmStage(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var in confirmInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	return nil, store.ConfirmStage(ctx, s.db, actorOf(user), id, in.Unplayed, s.now())
}

func (s *Server) decideRank(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var in rankDecisionInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	return nil, store.DecideRank(ctx, s.db, actorOf(user), id,
		store.RankDecisionInput{Teams: in.TeamIDs, Reason: in.Reason})
}

func (s *Server) decideBandRank(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var in bandDecisionInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	return nil, store.DecideBandRank(ctx, s.db, actorOf(user), id, in.Rank,
		store.RankDecisionInput{Teams: in.TeamIDs, Reason: in.Reason})
}

func readCorrection(r *http.Request) (store.CorrectionInput, store.Downstream, error) {
	var in correctionInputDTO
	if err := decodeBody(r, &in); err != nil {
		return store.CorrectionInput{}, store.DownstreamNone, err
	}
	input, err := in.toInput()
	if err != nil {
		return store.CorrectionInput{}, store.DownstreamNone, err
	}
	downstream, err := parseDownstream(in.Downstream)
	return input, downstream, err
}

// parseDownstream は API の downstream(null、"keep"、"reset")を読む。
func parseDownstream(value *string) (store.Downstream, error) {
	if value == nil {
		return store.DownstreamNone, nil
	}
	if *value == "" {
		return store.DownstreamNone, domain.Invalid("downstream", "must be keep, reset or null")
	}
	return store.ParseDownstream("downstream", *value)
}

func (s *Server) correctionImpact(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	input, _, err := readCorrection(r)
	if err != nil {
		return nil, err
	}
	impact, err := store.PreviewCorrection(ctx, s.db, id, input)
	if err != nil {
		return nil, err
	}
	return toImpact(impact), nil
}

func (s *Server) applyCorrection(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	input, downstream, err := readCorrection(r)
	if err != nil {
		return nil, err
	}
	impact, err := store.ApplyCorrection(ctx, s.db, actorOf(user), id, input, downstream)
	if err != nil {
		return nil, err
	}
	return toImpact(impact), nil
}

func (s *Server) alignTeams(ctx context.Context, id int64, _ *http.Request, user domain.User) (any, error) {
	return nil, store.AlignMatchTeams(ctx, s.db, actorOf(user), id)
}

func (s *Server) checkSchedule(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	violations, err := store.CheckFormatSchedule(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return violationListDTO{Violations: toViolations(violations)}, nil
}

func (s *Server) shiftSchedule(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	var in shiftInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	moved, err := store.CountShift(ctx, s.db, id, in.Court, in.Minutes)
	if err != nil {
		return nil, err
	}
	if in.DryRun {
		return shiftResultDTO{Violations: []violationDTO{}, Moved: moved}, nil
	}
	violations, err := store.ShiftFormatSchedule(ctx, s.db, id, in.Court, in.Minutes)
	if err != nil {
		return nil, err
	}
	return shiftResultDTO{Violations: toViolations(violations), Moved: moved}, nil
}

func (s *Server) swapSchedule(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	var in swapInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	violations, err := store.SwapMatchSchedule(ctx, s.db, id, in.OtherMatchID)
	if err != nil {
		return nil, err
	}
	return violationListDTO{Violations: toViolations(violations)}, nil
}
