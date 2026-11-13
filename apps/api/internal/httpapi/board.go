package httpapi

import (
	"net/http"

	"scoreview/api/internal/viewing"
)

// 公開のスコアサイトの掲示板(docs/api.md「GET /api/v1/tournaments/{id}/board」)。ハンドラは ID の
// 検証と応答の形への変換だけを行い、組み立ては viewing、記録の読み方は store に任せる。

func (s *Server) boardRoutes() {
	s.handle("GET /api/v1/tournaments/{id}/board", s.getBoard)
}

func (s *Server) getBoard(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	board, err := s.boards.Board(r.Context(), id)
	if err != nil {
		return nil, err
	}
	return toBoard(board), nil
}

type boardDTO struct {
	Tournament boardTournamentDTO  `json:"tournament"`
	Divisions  []boardDivisionDTO  `json:"divisions"`
	Matches    []boardMatchDTO     `json:"matches"`
	Tables     liveTablesConfigDTO `json:"tables"`
}

type boardTournamentDTO struct {
	ID     int64      `json:"id"`
	Name   i18nDTO    `json:"name"`
	Banner *bannerDTO `json:"banner"`
}

type boardDivisionDTO struct {
	ID       int64            `json:"id"`
	Name     i18nDTO          `json:"name"`
	Pools    []boardPoolDTO   `json:"pools"`
	Brackets []liveBracketDTO `json:"brackets"`
}

type boardPoolDTO struct {
	GroupID   int64           `json:"groupId"`
	Name      i18nDTO         `json:"name"`
	Done      bool            `json:"done"`
	Advance   int             `json:"advance"`
	Standings []poolResultDTO `json:"standings"`
}

type boardMatchDTO struct {
	scheduleRowDTO
	Group *boardMatchGroupDTO `json:"group"`
}

type boardMatchGroupDTO struct {
	ID   int64   `json:"id"`
	Name i18nDTO `json:"name"`
	Pool bool    `json:"pool"`
}

func toBoard(board viewing.Board) boardDTO {
	out := boardDTO{
		Tournament: boardTournamentDTO{ID: board.Tournament.ID, Name: toI18n(board.Tournament.Name)},
		Divisions:  make([]boardDivisionDTO, 0, len(board.Divisions)),
		Matches:    make([]boardMatchDTO, 0, len(board.Matches)),
		Tables:     toBoardTablesConfig(board.Tables),
	}
	if banner := board.Tournament.Banner; banner != nil {
		out.Tournament.Banner = &bannerDTO{ImageURL: banner.ImageURL, LinkURL: banner.LinkURL}
	}
	for _, division := range board.Divisions {
		item := boardDivisionDTO{ID: division.ID, Name: toI18n(division.Name),
			Pools:    make([]boardPoolDTO, 0, len(division.Pools)),
			Brackets: make([]liveBracketDTO, 0, len(division.Brackets))}
		for _, pool := range division.Pools {
			standings := make([]poolResultDTO, 0, len(pool.Standings))
			for _, row := range pool.Standings {
				standings = append(standings, poolResultDTO{tallyDTO: toTally(row.Tally), Team: toTeamRef(row.Team),
					Points: row.Points, Rank: row.Rank})
			}
			item.Pools = append(item.Pools, boardPoolDTO{GroupID: pool.GroupID, Name: toI18n(pool.Name),
				Done: pool.Done, Advance: pool.Advance, Standings: standings})
		}
		for _, bracket := range division.Brackets {
			item.Brackets = append(item.Brackets, toBoardBracket(bracket))
		}
		out.Divisions = append(out.Divisions, item)
	}
	for _, match := range board.Matches {
		item := boardMatchDTO{scheduleRowDTO: toScheduleRow(match.ScheduleRow)}
		if group := match.Group; group != nil {
			item.Group = &boardMatchGroupDTO{ID: group.ID, Name: toI18n(group.Name), Pool: group.Pool}
		}
		out.Matches = append(out.Matches, item)
	}
	return out
}

func toBoardBracket(bracket viewing.Bracket) liveBracketDTO {
	side := func(side viewing.BracketSide) liveBracketSideDTO {
		if side.From != nil {
			return liveBracketSideDTO{From: &placeholderDTO{Code: side.From.Code, Outcome: string(side.From.Outcome)}}
		}
		seed := side.Seed
		return liveBracketSideDTO{Seed: &seed}
	}
	out := liveBracketDTO{GroupID: bracket.GroupID, Name: bracket.Name,
		Matches: make([]liveBracketMatchDTO, 0, len(bracket.Matches))}
	for _, m := range bracket.Matches {
		out.Matches = append(out.Matches, liveBracketMatchDTO{Code: m.Code, Round: m.Round,
			WinnerPlace: m.WinnerPlace, LoserPlace: m.LoserPlace, Left: side(m.Left), Right: side(m.Right)})
	}
	return out
}

func toBoardTablesConfig(c viewing.TablesConfig) liveTablesConfigDTO {
	return liveTablesConfigDTO{Shown: c.Shown, PoolShown: c.PoolShown, BracketShown: c.BracketShown,
		Columns: liveTableColumnsDTO{WDL: c.Columns.WDL, Goals: c.Columns.Goals, Diff: c.Columns.Diff,
			Points: c.Columns.Points, Rank: c.Columns.Rank},
		Marks: c.Marks, Order: c.Order, Live: c.Live, Highlight: c.Highlight}
}
