package store

import (
	"context"
	"fmt"
	"slices"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const sqlMatchesOfTeam = "left_team_id = ? OR right_team_id = ?"

// GetTeam はチームページを返す。無ければ ENTRY_TEAM_ID の NotFound。
//
// 成績はディビジョンでの通算(終了した試合だけ)、順位は teams.final_rank(clone-spec D 1.1 の 3。
// ディビジョンの順位が未公開なら nil。publishedRank)。
// 対戦一覧は試合記録の左右のまま日付、時刻の昇順。選手はスタッフを除き domain.ComparePlayers の順、
// スタッフは役割の順(domain.CompareStaffRoles)、同じ役割は ID 順。選手の試合数はチームの試合数。ディスクゴルフ(stats_kind=discgolf)のディビジョンは roster_only にする
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
		Rank:    publishedRank(division, team),
		Matches: rows,
		Players: []PlayerLine{},
		Staff:   []StaffLine{},
		// clone-spec D 1.1 の 6
		ShowSOTG: division.ShowSOTG,
	}
	if page.NextMatchID, page.CurrentGroup, err = teamWhereabouts(
		ctx, db, scope, team, matches, rows,
	); err != nil {
		return TeamPage{}, err
	}
	// 出場記録のある選手が 1 人でもいれば、出場記録の取られているチーム。そのチームで記録のない選手は
	// 欠場として灰色(ABS)にする。試合詳細のない大会のチームは全員記録がないので欠場にしない
	// (参照サイトの全チームページで、ABS の行は出場記録のない選手と一致した)
	teamHasStats := false
	for _, player := range members {
		if agg := aggs.get(player.ID); agg.games() > 0 {
			teamHasStats = true
		}
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
		// 試合数はチームの試合数(終了した試合)。参照サイトのチームページでは、出場記録の数や欠場に
		// かかわらず全選手がチームの試合数を出す(取込済みの 7,060 行すべてで一致)
		page.Players = append(page.Players, PlayerLine{
			Player: playerRef(player), Games: page.Record.Games,
			Assists: agg.Assists, Goals: agg.Goals, Blocks: agg.Blocks,
			Absent: teamHasStats && agg.games() == 0,
		})
	}
	domain.SortPlayers(page.Players, func(line PlayerLine) domain.PlayerSortKey {
		return domain.PlayerSortKey{Goals: line.Goals, Assists: line.Assists, Number: line.Player.Number}
	})
	// members は ID 順なので、安定ソートで同じ役割の中は ID 順のまま
	slices.SortStableFunc(page.Staff, func(a, b StaffLine) int {
		return domain.CompareStaffRoles(a.Role.Ja, b.Role.Ja)
	})
	return page, nil
}

// teamWhereabouts はチームの次の試合と、いま見るべきグループを返す(本サイト独自の追記)。
// matches と rows は同じ並び(既定の並び)であること。
//
// 次の試合は domain.NextMatch で、終了した大会では出さない。グループは公開中(ナビに出る)のものに限り、
// 次の試合のグループ → 最後に終えた試合のグループ → group_teams の登録(ナビの並びで最初)の順で選ぶ。
func teamWhereabouts(
	ctx context.Context,
	db *gorm.DB,
	scope tournamentScope,
	team Team,
	matches []Match,
	rows []domain.MatchRow,
) (*int64, *FrameGroup, error) {
	published := map[int64]FrameGroup{}
	var order []int64
	for _, division := range scope.frame.Divisions {
		if division.ID != team.DivisionID {
			continue
		}
		for _, group := range division.Groups {
			published[group.ID] = group
			order = append(order, group.ID)
		}
	}
	pick := func(groupID *int64) *FrameGroup {
		if groupID == nil {
			return nil
		}
		if group, ok := published[*groupID]; ok {
			return &group
		}
		return nil
	}

	var nextID *int64
	var current *FrameGroup
	if index := domain.NextMatch(rows); index >= 0 && scope.tournament.Status != domain.TournamentFinished {
		nextID = &rows[index].ID
		current = pick(matches[index].GroupID)
	}
	for i := len(matches) - 1; current == nil && i >= 0; i-- {
		if matches[i].Status == domain.MatchFinished {
			current = pick(matches[i].GroupID)
		}
	}
	if current != nil || len(order) == 0 {
		return nextID, current, nil
	}
	var members []GroupTeam
	err := db.WithContext(ctx).Where(sqlByTeam, team.ID).Find(&members).Error
	if err != nil {
		return nil, nil, fmt.Errorf("load groups of team %d: %w", team.ID, err)
	}
	joined := map[int64]bool{}
	for _, member := range members {
		joined[member.GroupID] = true
	}
	for _, groupID := range order {
		if joined[groupID] {
			return nextID, pick(&groupID), nil
		}
	}
	return nextID, nil, nil
}
