package httpapi

import (
	"context"
	"net/http"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 管理用エンドポイントのルートとハンドラ。ハンドラはボディの読み込み、store の呼び出し、DTO への
// 変換だけを行う。記録係(staff)に許すのは、担当大会の読み取りと、試合の状態、得点経過、選手記録の
// 変更だけ(docs/requirements.md 権限モデル)。試合を変えられるのは、記録係の割り当てが覆う試合だけ。

func (s *Server) adminRoutes() {
	const base = "/api/v1/admin"
	ok, created, noContent := http.StatusOK, http.StatusCreated, http.StatusNoContent

	s.admin("GET "+base+"/tournaments", anyUser(), ok, s.listTournamentsAdmin)
	s.admin("POST "+base+"/tournaments", adminOnly(), created, s.createTournament)
	s.admin(
		"GET "+base+"/tournaments/{id}",
		staffOfTournament(),
		ok,
		withPathID(s.getTournamentAdmin),
	)
	s.admin("PATCH "+base+"/tournaments/{id}", adminOnly(), ok, withPathID(s.patchTournament))
	s.admin(
		"DELETE "+base+"/tournaments/{id}",
		adminOnly(),
		noContent,
		withPathID(s.deleteTournament),
	)
	s.admin("GET "+base+"/tournaments/{id}/banners", adminOnly(), ok, withPathID(s.getBanners))
	s.admin("PUT "+base+"/tournaments/{id}/banners", adminOnly(), ok, withPathID(s.putBanners))

	s.admin(
		"GET "+base+"/tournaments/{id}/divisions",
		staffOfTournament(),
		ok,
		withPathID(s.listDivisions),
	)
	s.admin(
		"POST "+base+"/tournaments/{id}/divisions",
		adminOnly(),
		created,
		withPathID(s.createDivision),
	)
	s.admin("GET "+base+"/divisions/{id}", staffOfDivision(), ok, withPathID(s.getDivision))
	s.admin("PATCH "+base+"/divisions/{id}", adminOnly(), ok, withPathID(s.patchDivision))
	s.admin("DELETE "+base+"/divisions/{id}", adminOnly(), noContent, withPathID(s.deleteDivision))
	s.admin("PUT "+base+"/divisions/{id}/final-ranks", adminOnly(), ok, withPathID(s.putFinalRanks))

	s.admin("GET "+base+"/divisions/{id}/groups", adminOnly(), ok, withPathID(s.listGroups))
	s.admin("POST "+base+"/divisions/{id}/groups", adminOnly(), created, withPathID(s.createGroup))
	s.admin("GET "+base+"/groups/{id}", adminOnly(), ok, withPathID(s.getGroupAdmin))
	s.admin("PATCH "+base+"/groups/{id}", adminOnly(), ok, withPathID(s.patchGroup))
	s.admin("DELETE "+base+"/groups/{id}", adminOnly(), noContent, withPathID(s.deleteGroup))
	s.admin("PUT "+base+"/groups/{id}/teams", adminOnly(), ok, withPathID(s.putGroupTeams))
	s.admin("POST "+base+"/groups/{id}/bracket", adminOnly(), ok, withPathID(s.generateBracket))
	s.admin("GET "+base+"/bracket-templates", adminOnly(), ok, s.listBracketTemplates)

	s.admin("GET "+base+"/divisions/{id}/teams", adminOnly(), ok, withPathID(s.listTeams))
	s.admin("POST "+base+"/divisions/{id}/teams", adminOnly(), created, withPathID(s.createTeam))
	s.admin("GET "+base+"/teams/{id}", adminOnly(), ok, withPathID(s.getTeamAdmin))
	s.admin("PATCH "+base+"/teams/{id}", adminOnly(), ok, withPathID(s.patchTeam))
	s.admin("DELETE "+base+"/teams/{id}", adminOnly(), noContent, withPathID(s.deleteTeam))

	s.admin("GET "+base+"/teams/{id}/players", adminOnly(), ok, withPathID(s.listPlayers))
	s.admin("POST "+base+"/teams/{id}/players", adminOnly(), created, withPathID(s.createPlayer))
	s.admin("GET "+base+"/players/{id}", adminOnly(), ok, withPathID(s.getPlayerAdmin))
	s.admin("PATCH "+base+"/players/{id}", adminOnly(), ok, withPathID(s.patchPlayer))
	s.admin("DELETE "+base+"/players/{id}", adminOnly(), noContent, withPathID(s.deletePlayer))

	s.admin("GET "+base+"/divisions/{id}/matches", staffOfDivision(), ok, withPathID(s.listMatches))
	s.admin("POST "+base+"/divisions/{id}/matches", adminOnly(), created, withPathID(s.createMatch))
	s.admin("GET "+base+"/matches/{id}", staffOfMatch(), ok, withPathID(s.getMatchAdmin))
	s.admin("PATCH "+base+"/matches/{id}", adminOnly(), ok, withPathID(s.patchMatch))
	s.admin("DELETE "+base+"/matches/{id}", adminOnly(), noContent, withPathID(s.deleteMatch))
	s.admin("POST "+base+"/matches/{id}/events", scorerOfMatch(), created, withPathID(s.addEvent))
	s.admin(
		"DELETE "+base+"/matches/{id}/events/{eventId}",
		scorerOfMatch(),
		ok,
		withPathID(s.removeEvent),
	)
	s.admin(
		"PATCH "+base+"/matches/{id}/events/{eventId}",
		scorerOfMatch(),
		ok,
		withPathID(s.patchEvent),
	)
	s.admin(
		"PUT "+base+"/matches/{id}/player-stats",
		scorerOfMatch(),
		ok,
		withPathID(s.putPlayerStats),
	)
	s.admin("POST "+base+"/matches/{id}/status", scorerOfMatch(), ok, withPathID(s.changeStatus))
	s.admin("POST "+base+"/matches/{id}/result", scorerOfMatch(), ok, withPathID(s.setResult))
	s.admin("GET "+base+"/matches/{id}/audit", staffOfMatch(), ok, withPathID(s.matchAudit))
	s.admin(
		"PUT "+base+"/matches/{id}/events/{eventId}/side",
		scorerOfMatch(),
		ok,
		withPathID(s.moveGoal),
	)
	s.admin("POST "+base+"/matches/{id}/scorer", scorerOfMatch(), ok, withPathID(s.claimScorer))
	s.admin(
		"DELETE "+base+"/matches/{id}/scorer",
		scorerOfMatch(),
		noContent,
		withPathID(s.releaseScorer),
	)

	s.admin("GET "+base+"/me/matches", anyUser(), ok, s.myMatches)
	s.admin(
		"GET "+base+"/tournaments/{id}/assignments",
		adminOnly(),
		ok,
		withPathID(s.listAssignments),
	)
	s.admin(
		"POST "+base+"/tournaments/{id}/assignments",
		adminOnly(),
		created,
		withPathID(s.createAssignment),
	)
	s.admin(
		"DELETE "+base+"/assignments/{id}",
		adminOnly(),
		noContent,
		withPathID(s.deleteAssignment),
	)

	s.admin("GET "+base+"/tournaments/{id}/galleries", adminOnly(), ok, withPathID(s.listGalleries))
	s.admin(
		"POST "+base+"/tournaments/{id}/galleries",
		adminOnly(),
		created,
		withPathID(s.createGallery),
	)
	s.admin("GET "+base+"/galleries/{id}", adminOnly(), ok, withPathID(s.getGalleryAdmin))
	s.admin("PATCH "+base+"/galleries/{id}", adminOnly(), ok, withPathID(s.patchGallery))
	s.admin("DELETE "+base+"/galleries/{id}", adminOnly(), noContent, withPathID(s.deleteGallery))
	s.admin("PUT "+base+"/galleries/{id}/photos", adminOnly(), ok, withPathID(s.putPhotos))

	s.admin("GET "+base+"/users", adminOnly(), ok, s.listUsers)
	s.admin("POST "+base+"/users", adminOnly(), created, s.createUser)
	s.admin("GET "+base+"/users/{id}", adminOnly(), ok, withPathID(s.getUser))
	s.admin("PATCH "+base+"/users/{id}", adminOnly(), ok, withPathID(s.patchUser))
	s.admin("DELETE "+base+"/users/{id}", adminOnly(), noContent, withPathID(s.deleteUser))
	s.admin(
		"PUT "+base+"/users/{id}/tournaments",
		adminOnly(),
		ok,
		withPathID(s.putUserTournaments),
	)
}

// 大会

func (s *Server) listTournamentsAdmin(r *http.Request, user domain.User) (any, error) {
	var only []int64
	if user.Role != domain.RoleAdmin {
		ids, err := store.StaffTournamentIDs(r.Context(), s.db, user.ID)
		if err != nil {
			return nil, err
		}
		only = ids
	}
	records, err := store.ListAdminTournaments(r.Context(), s.db, only)
	if err != nil {
		return nil, err
	}
	out := adminTournamentListDTO{Tournaments: make([]adminTournamentDTO, len(records))}
	for i, record := range records {
		out.Tournaments[i] = toAdminTournament(record)
	}
	return out, nil
}

// tournamentFields は自由記述 HTML を無害化してから store に渡す(保存時に無害化する。scoreview-design 7)。
func (s *Server) tournamentFields(in tournamentInputDTO) store.TournamentFields {
	fields := in.fields()
	fields.TopHTML.Ja = s.sanitizer.HTML(fields.TopHTML.Ja)
	fields.TopHTML.En = s.sanitizer.HTML(fields.TopHTML.En)
	return fields
}

func (s *Server) createTournament(r *http.Request, _ domain.User) (any, error) {
	in := newTournamentInput()
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.CreateTournament(r.Context(), s.db, s.tournamentFields(in))
	if err != nil {
		return nil, err
	}
	return toAdminTournament(record), nil
}

func (s *Server) getTournamentAdmin(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	record, err := store.GetAdminTournament(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAdminTournament(record), nil
}

func (s *Server) patchTournament(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	current, err := store.GetAdminTournament(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := toTournamentInput(current)
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.UpdateTournament(ctx, s.db, id, s.tournamentFields(in))
	if err != nil {
		return nil, err
	}
	return toAdminTournament(record), nil
}

func (s *Server) deleteTournament(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	return nil, store.DeleteTournament(ctx, s.db, id)
}

func (s *Server) getBanners(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	banners, err := store.ListBanners(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toBanners(banners), nil
}

func (s *Server) putBanners(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var body bannersDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	fields := make([]store.BannerFields, len(body.Banners))
	for i, banner := range body.Banners {
		fields[i] = store.BannerFields{ImageURL: banner.ImageURL, LinkURL: banner.LinkURL}
	}
	banners, err := store.ReplaceBanners(ctx, s.db, id, fields)
	if err != nil {
		return nil, err
	}
	return toBanners(banners), nil
}

// ディビジョン

func (s *Server) listDivisions(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	records, err := store.ListDivisions(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	out := adminDivisionListDTO{Divisions: make([]adminDivisionDTO, len(records))}
	for i, record := range records {
		out.Divisions[i] = toAdminDivision(record)
	}
	return out, nil
}

func (s *Server) createDivision(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	in := newDivisionInput()
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.CreateDivision(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminDivision(record), nil
}

func (s *Server) getDivision(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	record, err := store.GetDivision(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAdminDivision(record), nil
}

func (s *Server) patchDivision(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	current, err := store.GetDivision(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := toDivisionInput(current)
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.UpdateDivision(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminDivision(record), nil
}

func (s *Server) deleteDivision(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	return nil, store.DeleteDivision(ctx, s.db, id)
}

func (s *Server) putFinalRanks(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	var body []finalRankDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	ranks := make([]store.FinalRank, len(body))
	for i, rank := range body {
		ranks[i] = store.FinalRank{TeamID: rank.TeamID, Rank: rank.Rank}
	}
	records, err := store.SetFinalRanks(ctx, s.db, actorOf(user), id, ranks)
	if err != nil {
		return nil, err
	}
	return toAdminTeams(records), nil
}

// グループ

func (s *Server) listGroups(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	records, err := store.ListGroups(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	out := adminGroupListDTO{Groups: make([]adminGroupDTO, len(records))}
	for i, record := range records {
		out.Groups[i] = toAdminGroup(record)
	}
	return out, nil
}

func (s *Server) createGroup(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	in := newGroupInput()
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.CreateGroup(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminGroup(record), nil
}

func (s *Server) getGroupAdmin(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	record, err := store.GetAdminGroup(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAdminGroup(record), nil
}

func (s *Server) patchGroup(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	current, err := store.GetAdminGroup(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := toGroupInput(current)
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.UpdateGroup(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminGroup(record), nil
}

func (s *Server) deleteGroup(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	return nil, store.DeleteGroup(ctx, s.db, id)
}

func (s *Server) putGroupTeams(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var body groupTeamsBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	teams := make([]store.GroupTeamFields, len(body.Teams))
	for i, team := range body.Teams {
		teams[i] = store.GroupTeamFields{TeamID: team.TeamID, RankOverride: team.RankOverride}
	}
	record, err := store.ReplaceGroupTeams(ctx, s.db, id, teams)
	if err != nil {
		return nil, err
	}
	return toAdminGroup(record), nil
}

func (s *Server) generateBracket(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var body bracketBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	template, found := s.brackets.Get(body.Template)
	if !found {
		return nil, domain.Invalid("template", "unknown template")
	}
	seeds := make([]store.MatchSideFields, len(body.Seeds))
	for i, seed := range body.Seeds {
		seeds[i] = seed.fields()
	}
	record, err := store.ApplyBracketPlan(ctx, s.db, id, template.Plan(), seeds, body.CodePrefix)
	if err != nil {
		return nil, err
	}
	return toAdminGroup(record), nil
}

func (s *Server) listBracketTemplates(_ *http.Request, _ domain.User) (any, error) {
	templates := s.brackets.List()
	out := bracketTemplateListDTO{Templates: make([]bracketTemplateDTO, len(templates))}
	for i, template := range templates {
		out.Templates[i] = bracketTemplateDTO{
			ID:    template.ID,
			Name:  toI18n(template.Name),
			Teams: template.Teams,
		}
	}
	return out, nil
}

// チームと選手

func (s *Server) listTeams(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	records, err := store.ListTeams(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAdminTeams(records), nil
}

func (s *Server) createTeam(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	in := newTeamInput()
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.CreateTeam(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminTeam(record), nil
}

func (s *Server) getTeamAdmin(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	record, err := store.GetAdminTeam(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAdminTeam(record), nil
}

func (s *Server) patchTeam(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	current, err := store.GetAdminTeam(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := toTeamInput(current)
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.UpdateTeam(ctx, s.db, actorOf(user), id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminTeam(record), nil
}

func (s *Server) deleteTeam(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	return nil, store.DeleteTeam(ctx, s.db, id)
}

func (s *Server) listPlayers(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	records, err := store.ListPlayers(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return adminPlayerListDTO{Players: toAdminPlayers(records)}, nil
}

func (s *Server) createPlayer(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var in playerInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.CreatePlayer(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminPlayer(record), nil
}

func (s *Server) getPlayerAdmin(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	record, err := store.GetAdminPlayer(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAdminPlayer(record), nil
}

func (s *Server) patchPlayer(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	current, err := store.GetAdminPlayer(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := toPlayerInput(current)
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.UpdatePlayer(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminPlayer(record), nil
}

func (s *Server) deletePlayer(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	return nil, store.DeletePlayer(ctx, s.db, id)
}

// 試合

func (s *Server) listMatches(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	records, err := store.ListMatches(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAdminMatches(records), nil
}

func (s *Server) createMatch(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	in := newMatchInput()
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.CreateMatch(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminMatch(record), nil
}

func (s *Server) getMatchAdmin(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	detail, err := store.GetMatchDetail(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toMatchDetail(detail), nil
}

func (s *Server) patchMatch(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	current, err := store.GetAdminMatch(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := toMatchInput(current)
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.UpdateMatch(ctx, s.db, id, in.fields())
	if err != nil {
		return nil, err
	}
	return toAdminMatch(record), nil
}

func (s *Server) deleteMatch(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	return nil, store.DeleteMatch(ctx, s.db, id)
}

func (s *Server) addEvent(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	var body eventBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	input := store.EventInput{
		NewEvent: domain.NewEvent{
			Kind: domain.EventKind(body.Kind), Side: toSidePtr(body.Side),
			ScorerID: body.ScorerID, AssistID: body.AssistID, Callahan: body.Callahan,
		},
		ElapsedSec: body.ElapsedSec, WinnerSide: toSidePtr(body.WinnerSide),
		ClientID: body.ClientID, ExpectedScore: body.ExpectedScore.toDomain(),
	}
	detail, err := store.AddMatchEvent(ctx, s.db, actorOf(user), id, input, s.now())
	if err != nil {
		return nil, err
	}
	return toMatchDetail(detail), nil
}

func (s *Server) removeEvent(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	ref, err := eventRef(r)
	if err != nil {
		return nil, err
	}
	var clientID *string
	if value := r.URL.Query().Get("clientId"); value != "" {
		clientID = &value
	}
	detail, err := store.RemoveMatchEventRef(ctx, s.db, actorOf(user), id, ref, clientID)
	if err != nil {
		return nil, err
	}
	return toMatchDetail(detail), nil
}

func (s *Server) putPlayerStats(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var body playerStatsBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	stats := make([]store.PlayerStatFields, len(body.Stats))
	for i, stat := range body.Stats {
		stats[i] = store.PlayerStatFields{
			PlayerID:  stat.PlayerID,
			Side:      domain.Side(stat.Side),
			Blocks:    stat.Blocks,
			Greatests: stat.Greatests,
		}
	}
	detail, err := store.ReplaceMatchPlayerStats(ctx, s.db, id, stats)
	if err != nil {
		return nil, err
	}
	return toMatchDetail(detail), nil
}

func (s *Server) changeStatus(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	var body statusBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	input := store.StatusInput{
		Status:        domain.MatchStatus(body.Status),
		WinnerSide:    toSidePtr(body.WinnerSide),
		ClientID:      body.ClientID,
		ExpectedScore: body.ExpectedScore.toDomain(),
	}
	if body.Score != nil {
		input.Score = &domain.Score{Left: body.Score.Left, Right: body.Score.Right}
	}
	downstream, err := parseDownstream(body.Downstream)
	if err != nil {
		return nil, err
	}
	input.Downstream = downstream
	detail, err := store.ChangeMatchStatus(ctx, s.db, actorOf(user), id, input)
	if err != nil {
		return nil, err
	}
	return toMatchDetail(detail), nil
}

// ギャラリー

func (s *Server) listGalleries(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	records, err := store.ListGalleries(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	out := adminGalleryListDTO{Galleries: make([]adminGalleryDTO, len(records))}
	for i, record := range records {
		out.Galleries[i] = toAdminGallery(record)
	}
	return out, nil
}

func (s *Server) createGallery(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var in galleryInputDTO
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.CreateGallery(
		ctx,
		s.db,
		id,
		store.GalleryFields{Name: in.Name, SortOrder: in.SortOrder},
	)
	if err != nil {
		return nil, err
	}
	return toAdminGallery(record), nil
}

func (s *Server) getGalleryAdmin(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	record, err := store.GetAdminGallery(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toAdminGallery(record), nil
}

func (s *Server) patchGallery(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	current, err := store.GetAdminGallery(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := galleryInputDTO{Name: current.Name, SortOrder: current.SortOrder}
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	record, err := store.UpdateGallery(
		ctx,
		s.db,
		id,
		store.GalleryFields{Name: in.Name, SortOrder: in.SortOrder},
	)
	if err != nil {
		return nil, err
	}
	return toAdminGallery(record), nil
}

func (s *Server) deleteGallery(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	return nil, store.DeleteGallery(ctx, s.db, id)
}

func (s *Server) putPhotos(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var body photosBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	photos := make([]store.PhotoFields, len(body.Photos))
	for i, photo := range body.Photos {
		photos[i] = store.PhotoFields{ThumbURL: photo.ThumbURL, FullURL: photo.FullURL}
	}
	record, err := store.ReplacePhotos(ctx, s.db, id, photos)
	if err != nil {
		return nil, err
	}
	return toAdminGallery(record), nil
}

// ユーザー

func toUserDTO(record store.UserRecord) adminUserDTO {
	return toAdminUser(record.User, record.TournamentIDs)
}

func (s *Server) listUsers(r *http.Request, _ domain.User) (any, error) {
	records, err := store.ListUsers(r.Context(), s.db)
	if err != nil {
		return nil, err
	}
	out := userListDTO{Users: make([]adminUserDTO, len(records))}
	for i, record := range records {
		out.Users[i] = toUserDTO(record)
	}
	return out, nil
}

// userFields はメールアドレスを正規化し、パスワードがあれば長さを確かめてハッシュにする。
func userFields(in userInputDTO) (store.UserFields, *string, error) {
	fields := store.UserFields{
		Email: auth.NormalizeEmail(
			in.Email,
		),
		DisplayName: in.DisplayName,
		Role:        domain.Role(in.Role),
	}
	if in.Password == nil {
		return fields, nil, nil
	}
	if err := auth.ValidatePassword(*in.Password); err != nil {
		return store.UserFields{}, nil, err
	}
	hash, err := auth.HashPassword(*in.Password)
	if err != nil {
		return store.UserFields{}, nil, err
	}
	return fields, &hash, nil
}

func (s *Server) createUser(r *http.Request, _ domain.User) (any, error) {
	in := userInputDTO{Role: string(domain.RoleStaff)}
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	if in.Password == nil {
		return nil, domain.Invalid("password", "is required")
	}
	fields, hash, err := userFields(in)
	if err != nil {
		return nil, err
	}
	record, err := store.CreateUser(r.Context(), s.db, fields, *hash)
	if err != nil {
		return nil, err
	}
	return toUserDTO(record), nil
}

func (s *Server) getUser(
	ctx context.Context,
	id int64,
	_ *http.Request,
	_ domain.User,
) (any, error) {
	record, err := store.GetUserRecord(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toUserDTO(record), nil
}

func (s *Server) patchUser(
	ctx context.Context,
	id int64,
	r *http.Request,
	user domain.User,
) (any, error) {
	current, err := store.GetUserRecord(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := userInputDTO{
		Email:       current.Email,
		DisplayName: current.DisplayName,
		Role:        string(current.Role),
	}
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	fields, hash, err := userFields(in)
	if err != nil {
		return nil, err
	}
	record, err := store.UpdateUser(ctx, s.db, actorOf(user), id, fields, hash)
	if err != nil {
		return nil, err
	}
	return toUserDTO(record), nil
}

func (s *Server) deleteUser(
	ctx context.Context,
	id int64,
	_ *http.Request,
	user domain.User,
) (any, error) {
	return nil, store.DeleteUser(ctx, s.db, actorOf(user), id)
}

func (s *Server) putUserTournaments(
	ctx context.Context,
	id int64,
	r *http.Request,
	_ domain.User,
) (any, error) {
	var body userTournamentsBodyDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	record, err := store.SetUserTournaments(ctx, s.db, id, body.TournamentIDs)
	if err != nil {
		return nil, err
	}
	return toUserDTO(record), nil
}
