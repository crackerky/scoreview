package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 閲覧用エンドポイント(docs/api.md)。ハンドラはパラメータの検証、store の呼び出し、DTO への変換だけを行う。

func (s *Server) browseRoutes() {
	s.handle("GET /api/v1/tournaments", s.listTournaments)
	s.handle("GET /api/v1/tournaments/{id}", s.getTournamentTop)
	s.handle("GET /api/v1/tournaments/{id}/teams", s.getTeamList)
	s.handle("GET /api/v1/tournaments/{id}/schedule", s.getTournamentSchedule)
	s.handle("GET /api/v1/divisions/{id}/schedule", s.getDivisionSchedule)
	s.handle("GET /api/v1/divisions/{id}/standings", s.getStandings)
	s.handle("GET /api/v1/groups/{id}", s.getGroup)
	s.handle("GET /api/v1/teams/{id}", s.getTeam)
	s.handle("GET /api/v1/matches/{id}", s.getMatch)
	s.handle("GET /api/v1/players/{id}", s.getPlayer)
	s.handle("GET /api/v1/galleries/{id}", s.getGallery)
}

// handle は (応答, エラー) を返す関数をルートに登録する。エラーは writeError が HTTP に変換する。
func (s *Server) handle(pattern string, fn func(*http.Request) (any, error)) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		body, err := fn(r)
		if err != nil {
			writeError(w, s.log, err)
			return
		}
		writeJSON(w, http.StatusOK, body)
	})
}

// pathID はパス変数 {id} を正の整数として読む。
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.Invalid("id", "must be a positive integer")
	}
	return id, nil
}

func (s *Server) listTournaments(r *http.Request) (any, error) {
	sections, err := store.ListTournaments(r.Context(), s.db, s.now())
	if err != nil {
		return nil, err
	}
	return toTournamentList(sections), nil
}

func (s *Server) getTournamentTop(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetTournamentTop(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	out := tournamentTopDTO{
		Frame:   toFrame(page.Frame),
		LogoURL: page.LogoURL,
		// 自由記述 HTML は無害化してから返す(scoreview-design 7)
		TopHTML:   i18nDTO{Ja: s.sanitizer.HTML(page.TopHTML.Ja), En: s.sanitizer.HTML(page.TopHTML.En)},
		Divisions: make([]topDivisionDTO, 0, len(page.Divisions)),
	}
	for _, division := range page.Divisions {
		out.Divisions = append(out.Divisions, topDivisionDTO{
			ID: division.ID, Name: toI18n(division.Name),
			ScheduleLinked: division.ScheduleLinked, StandingsLinked: division.StandingsLinked,
		})
	}
	return out, nil
}

func (s *Server) getTeamList(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetTeamList(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	return toTeamList(page), nil
}

func (s *Server) getTournamentSchedule(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	filter, err := parseScheduleFilter(r, false)
	if err != nil {
		return nil, err
	}
	page, err := store.GetTournamentSchedule(r.Context(), s.db, id, filter)
	if err != nil {
		return nil, err
	}
	return toSchedule(page, "tournament"), nil
}

func (s *Server) getDivisionSchedule(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	filter, err := parseScheduleFilter(r, true)
	if err != nil {
		return nil, err
	}
	page, err := store.GetDivisionSchedule(r.Context(), s.db, id, filter)
	if err != nil {
		return nil, err
	}
	return toSchedule(page, "division"), nil
}

// parseScheduleFilter はスケジュールのクエリを検証する。ymd はキーが無いこと(既定を使う)と
// 空(全日程)を区別する(docs/api.md)。teamIsID ならチームはチーム ID(ディビジョン別)。
// dir は sort があるときだけ見る。
func parseScheduleFilter(r *http.Request, teamIsID bool) (store.ScheduleFilter, error) {
	query := r.URL.Query()
	var filter store.ScheduleFilter
	if values, ok := query["ymd"]; ok {
		ymd := values[0]
		if ymd != "" {
			if _, err := time.Parse("20060102", ymd); err != nil || len(ymd) != len("20060102") {
				return store.ScheduleFilter{}, domain.Invalid("ymd", "must be YYYYMMDD or empty")
			}
		}
		filter.YMD = &ymd
	}
	filter.Team = query.Get("team")
	if teamIsID && filter.Team != "" {
		if id, err := strconv.ParseInt(filter.Team, 10, 64); err != nil || id <= 0 {
			return store.ScheduleFilter{}, domain.Invalid("team", "must be a team id")
		}
	}
	// コートは数字とは限らない("A" など)ので文字列の完全一致で絞る
	filter.Field = query.Get("field")
	if key := query.Get("sort"); key != "" {
		if !domain.IsScheduleSortKey(key) {
			return store.ScheduleFilter{}, domain.Invalid("sort", "unknown sort key")
		}
		dir := domain.SortDir(query.Get("dir"))
		switch dir {
		case "":
			dir = domain.SortAsc
		case domain.SortAsc, domain.SortDesc:
		default:
			return store.ScheduleFilter{}, domain.Invalid("dir", "must be asc or desc")
		}
		filter.Sort = &domain.ScheduleSort{Key: domain.ScheduleSortKey(key), Dir: dir}
	}
	return filter, nil
}

func (s *Server) getStandings(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetDivisionStandings(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	return toStandings(page), nil
}

func (s *Server) getGroup(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetGroup(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	return toGroup(page), nil
}

func (s *Server) getTeam(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetTeam(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	return toTeamPage(page), nil
}

func (s *Server) getMatch(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetMatch(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	return toMatchPage(page), nil
}

func (s *Server) getPlayer(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetPlayer(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	return toPlayerPage(page), nil
}

func (s *Server) getGallery(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetGallery(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	return toGalleryPage(page), nil
}
