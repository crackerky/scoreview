package httpapi

import (
	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// このファイルは store のビュー型を DTO に写すだけで、計算をしない(計算は domain と store)。

func toI18n(text domain.I18nText) i18nDTO { return i18nDTO{Ja: text.Ja, En: text.En} }

func toIDI18n(id int64, name domain.I18nText) idI18nDTO {
	return idI18nDTO{ID: id, Name: toI18n(name)}
}

func toFrame(frame store.Frame) frameDTO {
	out := frameDTO{
		ID:           frame.ID,
		Name:         toI18n(frame.Name),
		ShortName:    frame.ShortName,
		Labels:       labelsDTO{TeamList: frame.TeamListLabel, Groups: frame.GroupsLabel},
		ShowSchedule: frame.ShowSchedule,
		Divisions:    make([]frameDivisionDTO, 0, len(frame.Divisions)),
		Galleries:    make([]frameGalleryDTO, 0, len(frame.Galleries)),
		Banners:      make([]bannerDTO, 0, len(frame.Banners)),
	}
	for _, division := range frame.Divisions {
		groups := make([]frameGroupDTO, 0, len(division.Groups))
		for _, group := range division.Groups {
			groups = append(
				groups,
				frameGroupDTO{ID: group.ID, Name: toI18n(group.Name), Linked: group.Linked},
			)
		}
		out.Divisions = append(out.Divisions, frameDivisionDTO{
			ID: division.ID, Name: toI18n(division.Name), ScheduleEnabled: division.ScheduleEnabled,
			Groups: groups,
		})
	}
	for _, gallery := range frame.Galleries {
		out.Galleries = append(
			out.Galleries,
			frameGalleryDTO{ID: gallery.ID, Name: toI18n(gallery.Name)},
		)
	}
	for _, banner := range frame.Banners {
		out.Banners = append(out.Banners, bannerDTO{ImageURL: banner.ImageURL, LinkURL: banner.LinkURL})
	}
	return out
}

func toTeamRef(team domain.TeamRef) teamRefDTO { return toIDI18n(team.ID, team.Name) }

func toTeamRefPtr(team *domain.TeamRef) *teamRefDTO {
	if team == nil {
		return nil
	}
	out := toTeamRef(*team)
	return &out
}

func toPlayerRef(player domain.PlayerRef) playerRefDTO {
	out := playerRefDTO{ID: player.ID, Number: player.Number, Name: toI18n(player.Name)}
	if player.Gender != nil {
		gender := string(*player.Gender)
		out.Gender = &gender
	}
	return out
}

func toPlayerRefPtr(player *domain.PlayerRef) *playerRefDTO {
	if player == nil {
		return nil
	}
	out := toPlayerRef(*player)
	return &out
}

func toPlayerWithRole(player store.PlayerWithRole) playerWithRoleDTO {
	out := playerWithRoleDTO{playerRefDTO: toPlayerRef(player.Player)}
	if player.Role != nil {
		role := toI18n(*player.Role)
		out.Role = &role
	}
	return out
}

func toMatchSide(side domain.MatchSide) matchSideDTO {
	out := matchSideDTO{Team: toTeamRefPtr(side.Team), Score: side.Score, LabelEn: side.LabelEn}
	if side.Placeholder != nil {
		out.Placeholder = &placeholderDTO{
			Code:    side.Placeholder.Code,
			Outcome: string(side.Placeholder.Outcome),
		}
	}
	return out
}

func toMatchRow(row domain.MatchRow) matchRowDTO {
	return matchRowDTO{
		MatchID:   row.ID,
		HasDetail: row.HasDetail,
		Code:      row.Code,
		Date:      row.Date,
		Time:      row.Time,
		Field:     row.Field,
		Status:    string(row.Status),
		Left:      toMatchSide(row.Left),
		Right:     toMatchSide(row.Right),
	}
}

func toMatchRows(rows []domain.MatchRow) []matchRowDTO {
	out := make([]matchRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, toMatchRow(row))
	}
	return out
}

func toTally(tally domain.Tally) tallyDTO {
	return tallyDTO{
		Games: tally.Games, Wins: tally.Wins, Draws: tally.Draws, Losses: tally.Losses,
		GoalsFor: tally.GoalsFor, GoalsAgainst: tally.GoalsAgainst,
	}
}

func toTournamentList(sections []store.TournamentSection) tournamentListDTO {
	out := tournamentListDTO{Sections: make([]tournamentSectionDTO, 0, len(sections))}
	for _, section := range sections {
		list := make([]idI18nDTO, 0, len(section.Tournaments))
		for _, tournament := range section.Tournaments {
			list = append(list, toIDI18n(tournament.ID, tournament.Name))
		}
		out.Sections = append(
			out.Sections,
			tournamentSectionDTO{Status: string(section.Status), Tournaments: list},
		)
	}
	return out
}

func toTeamList(page store.TeamListPage) teamListDTO {
	out := teamListDTO{
		Frame:     toFrame(page.Frame),
		Divisions: make([]teamListDivisionDTO, 0, len(page.Divisions)),
	}
	for _, division := range page.Divisions {
		teams := make([]teamListTeamDTO, 0, len(division.Teams))
		for _, team := range division.Teams {
			teams = append(teams, teamListTeamDTO{
				ID: team.ID, Name: toI18n(team.Name), BarStyle: string(team.BarStyle), MemberCount: team.MemberCount,
			})
		}
		out.Divisions = append(out.Divisions, teamListDivisionDTO{
			ID: division.ID, Name: toI18n(division.Name), Kind: string(division.Kind),
			EntryKind: string(division.EntryKind), Teams: teams,
		})
	}
	return out
}

func toSchedule(page store.SchedulePage, scope string) scheduleDTO {
	out := scheduleDTO{
		Frame: toFrame(page.Frame),
		Scope: scope,
		Filters: scheduleFiltersDTO{
			Dates:  page.Dates,
			Teams:  make([]scheduleTeamDTO, 0, len(page.Teams)),
			Fields: page.Fields,
			Selected: scheduleSelectedDTO{
				YMD:   page.Selected.YMD,
				Team:  page.Selected.Team,
				Field: page.Selected.Field,
			},
		},
		Rows: make([]scheduleRowDTO, 0, len(page.Rows)),
	}
	if page.Division != nil {
		division := toIDI18n(page.Division.ID, page.Division.Name)
		out.Division = &division
	}
	for _, team := range page.Teams {
		out.Filters.Teams = append(
			out.Filters.Teams,
			scheduleTeamDTO{Value: team.Value, Name: toI18n(team.Name)},
		)
	}
	if page.Sort != nil {
		out.Sort = &scheduleSortDTO{Key: string(page.Sort.Key), Dir: string(page.Sort.Dir)}
	}
	for _, row := range page.Rows {
		out.Rows = append(out.Rows, scheduleRowDTO{
			matchRowDTO: toMatchRow(row.MatchRow),
			Division: scheduleDivisionDTO{
				ID: row.Division.ID, Name: toI18n(row.Division.Name), Kind: string(row.Division.Kind),
			},
		})
	}
	return out
}

func toStandings(page store.StandingsPage) standingsDTO {
	out := standingsDTO{
		Frame:        toFrame(page.Frame),
		Division:     toIDI18n(page.Division.ID, page.Division.Name),
		Standings:    make([]standingRowDTO, 0, len(page.Standings)),
		Leaders:      make([]leaderRowDTO, 0, len(page.Leaders)),
		BlockLeaders: make([]blockLeaderRowDTO, 0, len(page.BlockLeaders)),
	}
	for _, row := range page.Standings {
		out.Standings = append(out.Standings, standingRowDTO{
			tallyDTO: toTally(row.Tally), Team: toTeamRef(row.Team), Rank: row.Rank,
		})
	}
	for _, row := range page.Leaders {
		out.Leaders = append(out.Leaders, leaderRowDTO{
			Team: toTeamRef(row.Team), Player: toPlayerRef(row.Player),
			Games: row.Games, Assists: row.Assists, Goals: row.Goals,
		})
	}
	for _, row := range page.BlockLeaders {
		out.BlockLeaders = append(out.BlockLeaders, blockLeaderRowDTO{
			Team: toTeamRef(row.Team), Player: toPlayerRef(row.Player), Games: row.Games, Blocks: row.Blocks,
		})
	}
	return out
}

func toGroup(page store.GroupPage) groupDTO {
	out := groupDTO{
		Frame: toFrame(page.Frame),
		Group: groupInfoDTO{
			ID: page.Group.ID, Name: page.Group.Name, Kind: string(page.Group.Kind),
			Division: toIDI18n(page.Group.Division.ID, page.Group.Division.Name),
		},
		Matches: toMatchRows(page.Matches),
	}
	if page.Pool != nil {
		pool := toPool(*page.Pool)
		out.Pool = &pool
	}
	if page.Bracket != nil {
		bracket := toBracket(*page.Bracket)
		out.Bracket = &bracket
	}
	if page.Image != nil {
		out.Image = &imageDTO{URL: page.Image.URL, Width: page.Image.Width, Height: page.Image.Height}
	}
	return out
}

func toPool(pool store.PoolView) poolDTO {
	out := poolDTO{
		Teams:   make([]teamRefDTO, 0, len(pool.Teams)),
		Matrix:  make([][]matrixCellDTO, 0, len(pool.Matrix)),
		Results: make([]poolResultDTO, 0, len(pool.Results)),
	}
	for _, team := range pool.Teams {
		out.Teams = append(out.Teams, toTeamRef(team))
	}
	for _, row := range pool.Matrix {
		cells := make([]matrixCellDTO, 0, len(row))
		for _, cell := range row {
			dto := matrixCellDTO{Kind: string(cell.Kind)}
			if cell.Kind == store.MatrixMatch {
				matchID, status, text := cell.MatchID, string(cell.Status), cell.Text
				dto.MatchID, dto.Status, dto.Text = &matchID, &status, &text
			}
			cells = append(cells, dto)
		}
		out.Matrix = append(out.Matrix, cells)
	}
	for _, row := range pool.Results {
		out.Results = append(out.Results, poolResultDTO{
			tallyDTO: toTally(row.Tally), Team: toTeamRef(row.Team), Points: row.Points, Rank: row.Rank,
		})
	}
	return out
}

func toBracket(bracket store.BracketView) bracketDTO {
	out := bracketDTO{
		Pattern: bracket.Pattern, PtnName: bracket.PtnName, Columns: bracket.Columns,
		Rows: make([][]bracketCellDTO, 0, len(bracket.Rows)),
	}
	for _, row := range bracket.Rows {
		cells := make([]bracketCellDTO, 0, len(row))
		for _, cell := range row {
			dto := bracketCellDTO{
				Borders: bordersDTO{
					Top: cell.Borders.Top, Bottom: cell.Borders.Bottom, Left: cell.Borders.Left, Right: cell.Borders.Right,
				},
				Kind:    string(cell.Kind),
				Text:    cell.Text,
				Team:    toTeamRefPtr(cell.Team),
				MatchID: cell.MatchID,
			}
			if cell.Status != nil {
				status := string(*cell.Status)
				dto.Status = &status
			}
			cells = append(cells, dto)
		}
		out.Rows = append(out.Rows, cells)
	}
	return out
}

func toTeamPage(page store.TeamPage) teamPageDTO {
	out := teamPageDTO{
		Frame: toFrame(page.Frame),
		Team: teamInfoDTO{
			ID: page.Team.ID, FullName: toI18n(page.Team.FullName), ShortName: toI18n(page.Team.ShortName),
			Comment: toI18n(page.Team.Comment), Layout: string(page.Team.Layout),
		},
		Record:  teamRecordDTO{tallyDTO: toTally(page.Record), Rank: page.Rank},
		Matches: toMatchRows(page.Matches),
		Players: make([]playerLineDTO, 0, len(page.Players)),
		Staff:   make([]staffLineDTO, 0, len(page.Staff)),
	}
	for _, line := range page.Players {
		out.Players = append(out.Players, playerLineDTO{
			Player: toPlayerRef(line.Player), Games: line.Games, Assists: line.Assists, Goals: line.Goals,
			Blocks: line.Blocks, Absent: line.Absent,
		})
	}
	for _, staff := range page.Staff {
		line := staffLineDTO{ID: staff.ID, Role: toI18n(staff.Role), Name: toI18n(staff.Name)}
		if staff.Gender != nil {
			gender := string(*staff.Gender)
			line.Gender = &gender
		}
		out.Staff = append(out.Staff, line)
	}
	return out
}

func toMatchPage(page store.MatchPage) matchPageDTO {
	match := page.Match
	out := matchPageDTO{
		Frame: toFrame(page.Frame),
		Match: matchInfoDTO{
			ID: match.ID, Code: match.Code, GroupName: match.GroupName,
			Date: match.Date, Time: match.Time, Field: match.Field, Status: string(match.Status),
			Left:  toMatchTeamScore(match.Left),
			Right: toMatchTeamScore(match.Right),
			Breadcrumb: breadcrumbDTO{
				DivisionID:   match.Breadcrumb.DivisionID,
				DivisionName: toI18n(match.Breadcrumb.DivisionName),
				GroupID:      match.Breadcrumb.GroupID,
			},
		},
		Events:  make([]eventLineDTO, 0, len(page.Events)),
		Lineups: lineupsDTO{Left: toLineup(page.Lineups.Left), Right: toLineup(page.Lineups.Right)},
		Stats:   matchStatsDTO{Left: toSideStats(page.Stats.Left), Right: toSideStats(page.Stats.Right)},
	}
	for _, event := range page.Events {
		line := eventLineDTO{
			ElapsedSec: event.ElapsedSec, Kind: string(event.Kind),
			Scorer: toPlayerRefPtr(event.Scorer), Assist: toPlayerRefPtr(event.Assist),
			Callahan: event.Callahan, LeftScore: event.LeftScore, RightScore: event.RightScore,
		}
		if event.Side != nil {
			side := string(*event.Side)
			line.Side = &side
		}
		out.Events = append(out.Events, line)
	}
	return out
}

func toMatchTeamScore(side store.MatchTeamScore) matchTeamScoreDTO {
	out := matchTeamScoreDTO{Team: toTeamRefPtr(side.Team), Score: side.Score}
	if side.FullName != nil {
		fullName := toI18n(*side.FullName)
		out.FullName = &fullName
	}
	return out
}

func toLineup(lines []store.LineupLine) []lineupLineDTO {
	out := make([]lineupLineDTO, 0, len(lines))
	for _, line := range lines {
		out = append(out, lineupLineDTO{
			Player: toPlayerRef(line.Player), Assists: line.Assists, Goals: line.Goals, Blocks: line.Blocks,
		})
	}
	return out
}

func toSideStats(stats domain.SideStats) sideStatsDTO {
	return sideStatsDTO{
		Goals: stats.Goals, FromOffense: stats.FromOffense, FromDefense: stats.FromDefense,
		FromTurnover: stats.FromTurnover, Timeouts: stats.Timeouts,
	}
}

func toPlayerPage(page store.PlayerPage) playerPageDTO {
	out := playerPageDTO{
		Frame:     toFrame(page.Frame),
		Player:    toPlayerWithRole(page.Player),
		Team:      toTeamRef(page.Team),
		Prev:      toPlayerWithRole(page.Prev),
		Next:      toPlayerWithRole(page.Next),
		StatsKind: string(page.StatsKind),
		Stats: playerStatsDTO{
			Games: page.Stats.Games, Assists: page.Stats.Assists, Goals: page.Stats.Goals,
			Callahans: page.Stats.Callahans, Greatests: page.Stats.Greatests, Blocks: page.Stats.Blocks,
		},
		Events: make([]playerEventLineDTO, 0, len(page.Events)),
	}
	for _, event := range page.Events {
		out.Events = append(out.Events, playerEventLineDTO{
			Code: event.Code, Date: event.Date, Opponent: toTeamRefPtr(event.Opponent), ElapsedSec: event.ElapsedSec,
			Callahan: event.Callahan, Scorer: toPlayerRefPtr(event.Scorer), Assist: toPlayerRefPtr(event.Assist),
		})
	}
	return out
}

func toGalleryPage(page store.GalleryPage) galleryPageDTO {
	photos := make([]photoDTO, 0, len(page.Photos))
	for _, photo := range page.Photos {
		photos = append(photos, photoDTO{ThumbURL: photo.ThumbURL, FullURL: photo.FullURL})
	}
	return galleryPageDTO{
		Frame:   toFrame(page.Frame),
		Gallery: galleryDTO{ID: page.ID, Name: page.Name, Photos: photos},
	}
}
