package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const (
	sqlListed = "listed = ?"
	// 区分内で開催日の新しい順(clone-spec A 3)。開催日未登録の位置は不明なので先頭にする(スキーマの索引と同じ)
	sqlOrderTournamentList = "start_date DESC NULLS FIRST, id DESC"
	sqlMemberCount         = "team_id, COUNT(*) AS members"
	sqlPlayersOnly         = "team_id IN ? AND role_ja IS NULL AND role_en IS NULL"
	sqlGroupByTeam         = "team_id"
)

// ListTournaments は一覧に載せる大会(listed=true)を区分ごとに返す。区分は entry_open,
// entry_closed, finished の順で、大会が 0 件の区分は含めない。
func ListTournaments(ctx context.Context, db *gorm.DB) ([]TournamentSection, error) {
	var rows []Tournament
	err := db.WithContext(ctx).Select("id", "name_ja", "name_en", "status", "start_date").
		Where(sqlListed, true).Order(sqlOrderTournamentList).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list tournaments: %w", err)
	}
	byStatus := map[domain.TournamentStatus][]TournamentSummary{}
	for _, tournament := range rows {
		byStatus[tournament.Status] = append(
			byStatus[tournament.Status],
			TournamentSummary{ID: tournament.ID, Name: i18n(tournament.NameJa, tournament.NameEn)},
		)
	}
	sections := []TournamentSection{}
	for _, status := range domain.TournamentStatusOrder() {
		if list := byStatus[status]; len(list) > 0 {
			sections = append(sections, TournamentSection{Status: status, Tournaments: list})
		}
	}
	return sections, nil
}

// GetTournamentTop は大会トップを返す。listed=false の大会も返す。無ければ TOURNAMENT_ID の NotFound。
func GetTournamentTop(ctx context.Context, db *gorm.DB, id int64) (TournamentTopPage, error) {
	scope, err := loadScopeByTournamentID(ctx, db, id)
	if err != nil {
		return TournamentTopPage{}, err
	}
	tournament := scope.tournament
	page := TournamentTopPage{
		Frame:     scope.frame,
		LogoURL:   tournament.LogoURL,
		TopHTML:   i18n(tournament.TopHTMLJa, tournament.TopHTMLEn),
		Divisions: make([]TopDivision, 0, len(scope.divisions)),
	}
	for _, division := range scope.divisions {
		page.Divisions = append(page.Divisions, TopDivision{
			ID: division.ID, Name: i18n(division.NameJa, division.NameEn),
			ScheduleLinked: division.ScheduleEnabled, StandingsLinked: division.StandingsPublished,
		})
	}
	return page, nil
}

// GetTeamList はチーム一覧を返す。チームはディビジョンごとに sort_order, id 順。
func GetTeamList(ctx context.Context, db *gorm.DB, id int64) (TeamListPage, error) {
	scope, err := loadScopeByTournamentID(ctx, db, id)
	if err != nil {
		return TeamListPage{}, err
	}
	var teams []Team
	if err := db.WithContext(ctx).Where(sqlByTournament, id).Order(sqlOrderSortID).Find(&teams).Error; err != nil {
		return TeamListPage{}, fmt.Errorf("load teams of tournament %d: %w", id, err)
	}
	members, err := countMembers(ctx, db, teams)
	if err != nil {
		return TeamListPage{}, err
	}

	byDivision := map[int64][]Team{}
	for _, team := range teams {
		byDivision[team.DivisionID] = append(byDivision[team.DivisionID], team)
	}
	page := TeamListPage{
		Frame:     scope.frame,
		Divisions: make([]TeamListDivision, 0, len(scope.divisions)),
	}
	for _, division := range scope.divisions {
		div := TeamListDivision{
			ID: division.ID, Name: i18n(division.NameJa, division.NameEn), Kind: division.Kind, EntryKind: division.EntryKind,
			Teams: make([]TeamListTeam, 0, len(byDivision[division.ID])),
		}
		for _, team := range byDivision[division.ID] {
			item := TeamListTeam{
				ID:       team.ID,
				Name:     i18n(team.NameFullJa, team.NameFullEn),
				BarStyle: team.BarStyle,
			}
			// 個人戦は人数を出さない(clone-spec E 7: 区切りセルが &nbsp;)
			if division.EntryKind != domain.EntryIndividual {
				count := members[team.ID]
				item.MemberCount = &count
			}
			div.Teams = append(div.Teams, item)
		}
		page.Divisions = append(page.Divisions, div)
	}
	return page, nil
}

// countMembers はチームごとのスタッフを除く選手数を 1 回の問い合わせで数える。
func countMembers(ctx context.Context, db *gorm.DB, teams []Team) (map[int64]int, error) {
	members := map[int64]int{}
	if len(teams) == 0 {
		return members, nil
	}
	ids := make([]int64, len(teams))
	for i, team := range teams {
		ids[i] = team.ID
	}
	var counts []struct {
		TeamID  int64
		Members int
	}
	err := db.WithContext(ctx).Model(&Player{}).Select(sqlMemberCount).
		Where(sqlPlayersOnly, ids).Group(sqlGroupByTeam).Scan(&counts).Error
	if err != nil {
		return nil, fmt.Errorf("count team members: %w", err)
	}
	for _, count := range counts {
		members[count.TeamID] = count.Members
	}
	return members, nil
}
