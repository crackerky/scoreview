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
	s.handle("GET /api/v1/tournaments/{id}/schedule", s.getTournamentSchedule)
	s.handle("GET /api/v1/groups/{id}", s.getGroup)
	s.handle("GET /api/v1/matches/{id}", s.getMatch)
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

func (s *Server) getTournamentSchedule(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	filter, err := parseScheduleFilter(r)
	if err != nil {
		return nil, err
	}
	page, err := store.GetTournamentSchedule(r.Context(), s.db, id, filter)
	if err != nil {
		return nil, err
	}
	return toSchedule(page, "tournament"), nil
}

// parseScheduleFilter はスケジュールのクエリを検証する。ymd はキーが無いこと(既定を使う)と
// 空(全日程)を区別する(docs/api.md)。dir は sort があるときだけ見る。
func parseScheduleFilter(r *http.Request) (store.ScheduleFilter, error) {
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
