package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const sqlMatchesOfTeam = "left_team_id = ? OR right_team_id = ?"

// GetTeam はチームページを返す。無ければ ENTRY_TEAM_ID の NotFound。
//
// 成績はディビジョンでの通算(終了した試合だけ)、順位は teams.final_rank(clone-spec D 1.1 の 3)。
// 対戦一覧は試合記録の左右のまま日付、時刻の昇順。選手はスタッフを除き domain.ComparePlayers の順、
// スタッフは ID 順。ディスクゴルフ(stats_kind=discgolf)のディビジョンは roster_only にする
// (clone-spec D 1.2)。
func GetTeam(ctx context.Context, db *gorm.DB, id int64) (TeamPage, error) {
	team, err := findByID[Team](ctx, db, id, domain.ResourceTeam)
	if err != nil {
		return TeamPage{}, err
	}
	division, err := findInternal[Division](ctx, db, team.DivisionID, "division")
	if err != nil {
		return TeamPage{}, err
	}
	scope, err := loadScopeOf(ctx, db, team.TournamentID)
	if err != nil {
		return TeamPage{}, err
	}
	tx := db.WithContext(ctx)
	var matches []Match
	if err := tx.Where(sqlMatchesOfTeam, id, id).Order(sqlOrderID).Find(&matches).Error; err != nil {
		return TeamPage{}, fmt.Errorf("load matches of team %d: %w", id, err)
	}
	sortMatches(matches)
	teams, err := loadTeams(ctx, db, matchTeamIDs(matches))
	if err != nil {
		return TeamPage{}, err
	}
	rows, err := toMatchRows(matches, teams)
	if err != nil {
		return TeamPage{}, err
	}
	var members []Player
	if err := tx.Where(sqlByTeam, id).Order(sqlOrderID).Find(&members).Error; err != nil {
		return TeamPage{}, fmt.Errorf("load players of team %d: %w", id, err)
	}
	memberIDs := make([]int64, len(members))
	for i, player := range members {
		memberIDs[i] = player.ID
	}
	aggs, err := loadPlayersAggs(ctx, db, memberIDs)
	if err != nil {
		return TeamPage{}, err
	}

	layout := TeamLayoutStandard
	if division.StatsKind == domain.StatsDiscgolf {
		layout = TeamLayoutRosterOnly
	}
	page := TeamPage{
		Frame: scope.frame,
		Team: TeamInfo{
			ID:        team.ID,
			FullName:  i18n(team.NameFullJa, team.NameFullEn),
			ShortName: i18n(team.NameShortJa, team.NameShortEn),
			Comment:   i18n(team.CommentJa, team.CommentEn),
			Layout:    layout,
		},
		Record:  domain.TallyFor(id, toResults(matches)),
		Rank:    team.FinalRank,
		Matches: rows,
		Players: []PlayerLine{},
		Staff:   []StaffLine{},
	}
	for _, player := range members {
		if player.IsStaff() {
			ref := playerRef(player)
			page.Staff = append(
				page.Staff,
				StaffLine{ID: player.ID, Role: *playerRole(player), Name: ref.Name, Gender: ref.Gender},
			)
			continue
		}
		agg := aggs.get(player.ID)
		page.Players = append(page.Players, PlayerLine{
			Player: playerRef(
				player,
			), Games: agg.games(), Assists: agg.Assists, Goals: agg.Goals, Blocks: agg.Blocks,
		})
	}
	domain.SortPlayers(page.Players, func(line PlayerLine) domain.PlayerSortKey {
		return domain.PlayerSortKey{Goals: line.Goals, Assists: line.Assists, Number: line.Player.Number}
	})
	return page, nil
}
