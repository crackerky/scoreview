package httpapi

import (
	"context"
	"net/http"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 観戦アプリの「表で見る」(docs/api.md「観戦アプリの表」)。見せ方を決めるのは管理者だけ。

func (s *Server) liveTablesRoutes() {
	ok := http.StatusOK
	s.handle("GET /api/v1/tournaments/{id}/live-tables", s.getLiveTables)
	s.admin("GET /api/v1/admin/tournaments/{id}/live-tables", adminOnly(), ok, withPathID(s.getLiveTablesConfig))
	s.admin("PUT /api/v1/admin/tournaments/{id}/live-tables", adminOnly(), ok, withPathID(s.putLiveTablesConfig))
}

type liveTableColumnsDTO struct {
	WDL    bool `json:"wdl"`
	Goals  bool `json:"goals"`
	Diff   bool `json:"diff"`
	Points bool `json:"points"`
	Rank   bool `json:"rank"`
}

type liveTablesConfigDTO struct {
	Shown        bool                `json:"shown"`
	PoolShown    bool                `json:"poolShown"`
	BracketShown bool                `json:"bracketShown"`
	Columns      liveTableColumnsDTO `json:"columns"`
	Marks        string              `json:"marks"`
	Order        string              `json:"order"`
	Live         string              `json:"live"`
	Highlight    bool                `json:"highlight"`
}

type liveBracketSideDTO struct {
	From *placeholderDTO `json:"from"`
	Seed *string         `json:"seed"`
}

type liveBracketMatchDTO struct {
	Code        string             `json:"code"`
	Round       int                `json:"round"`
	WinnerPlace int                `json:"winnerPlace"`
	LoserPlace  int                `json:"loserPlace"`
	Left        liveBracketSideDTO `json:"left"`
	Right       liveBracketSideDTO `json:"right"`
}

type liveBracketDTO struct {
	GroupID int64                 `json:"groupId"`
	Name    string                `json:"name"`
	Matches []liveBracketMatchDTO `json:"matches"`
}

type liveTablePoolDTO struct {
	GroupID int64 `json:"groupId"`
	Advance int   `json:"advance"`
}

type liveTableDivisionDTO struct {
	ID       int64              `json:"id"`
	Pools    []liveTablePoolDTO `json:"pools"`
	Brackets []liveBracketDTO   `json:"brackets"`
}

type liveTablesDTO struct {
	Config    liveTablesConfigDTO    `json:"config"`
	Divisions []liveTableDivisionDTO `json:"divisions"`
}

func toLiveTablesConfig(c store.LiveTablesConfig) liveTablesConfigDTO {
	return liveTablesConfigDTO{Shown: c.Shown, PoolShown: c.PoolShown, BracketShown: c.BracketShown,
		Columns: liveTableColumnsDTO{WDL: c.Columns.WDL, Goals: c.Columns.Goals, Diff: c.Columns.Diff,
			Points: c.Columns.Points, Rank: c.Columns.Rank},
		Marks: c.Marks, Order: c.Order, Live: c.Live, Highlight: c.Highlight}
}

func (c liveTablesConfigDTO) toStore() store.LiveTablesConfig {
	return store.LiveTablesConfig{Shown: c.Shown, PoolShown: c.PoolShown, BracketShown: c.BracketShown,
		Columns: store.LiveTableColumns{WDL: c.Columns.WDL, Goals: c.Columns.Goals, Diff: c.Columns.Diff,
			Points: c.Columns.Points, Rank: c.Columns.Rank},
		Marks: c.Marks, Order: c.Order, Live: c.Live, Highlight: c.Highlight}
}

func toLiveBracketSide(side store.LiveBracketSide) liveBracketSideDTO {
	if side.From != nil {
		return liveBracketSideDTO{From: &placeholderDTO{Code: side.From.Code, Outcome: string(side.From.Outcome)}}
	}
	seed := side.Seed
	return liveBracketSideDTO{Seed: &seed}
}

func toLiveTables(t store.LiveTables) liveTablesDTO {
	out := liveTablesDTO{Config: toLiveTablesConfig(t.Config), Divisions: []liveTableDivisionDTO{}}
	for _, d := range t.Divisions {
		division := liveTableDivisionDTO{ID: d.ID, Pools: []liveTablePoolDTO{}, Brackets: []liveBracketDTO{}}
		for _, p := range d.Pools {
			division.Pools = append(division.Pools, liveTablePoolDTO{GroupID: p.GroupID, Advance: p.Advance})
		}
		for _, b := range d.Brackets {
			bracket := liveBracketDTO{GroupID: b.GroupID, Name: b.Name, Matches: []liveBracketMatchDTO{}}
			for _, m := range b.Matches {
				bracket.Matches = append(bracket.Matches, liveBracketMatchDTO{Code: m.Code, Round: m.Round,
					WinnerPlace: m.WinnerPlace, LoserPlace: m.LoserPlace,
					Left: toLiveBracketSide(m.Left), Right: toLiveBracketSide(m.Right)})
			}
			division.Brackets = append(division.Brackets, bracket)
		}
		out.Divisions = append(out.Divisions, division)
	}
	return out
}

func (s *Server) getLiveTables(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	tables, err := store.GetLiveTables(r.Context(), s.db, id)
	if err != nil {
		return nil, err
	}
	return toLiveTables(tables), nil
}

func (s *Server) getLiveTablesConfig(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	config, err := store.GetLiveTablesConfig(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	return toLiveTablesConfig(config), nil
}

// putLiveTablesConfig はボディに無いキーを今の見せ方のままにする。
func (s *Server) putLiveTablesConfig(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	current, err := store.GetLiveTablesConfig(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := toLiveTablesConfig(current)
	if err := decodeBody(r, &in); err != nil {
		return nil, err
	}
	config, err := store.SaveLiveTablesConfig(ctx, s.db, actorOf(user), id, in.toStore(), s.now())
	if err != nil {
		return nil, err
	}
	return toLiveTablesConfig(config), nil
}
