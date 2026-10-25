package store

import (
	"context"
	"fmt"
	"time"

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
	sqlTournamentIn        = "tournament_id IN ?"
	sqlDivisionIn          = "division_id IN ?"
	sqlGroupByDivision     = "division_id"
	sqlGroupByTournament   = "tournament_id"
	// 試合の状態の値は引数で渡す(finished, live の順)
	sqlMatchProgress = "division_id, COUNT(*) AS matches, " +
		"COUNT(*) FILTER (WHERE status = ?) AS finished, COUNT(*) FILTER (WHERE status = ?) AS live, " +
		"MIN(match_date) AS first_day, MAX(match_date) AS last_day"
	sqlTeamCount      = "tournament_id, COUNT(*) AS teams"
	sqlEnteredTeamsIn = "tournament_id IN ? AND id NOT IN (SELECT team_id FROM team_withdrawals)"
)

// ListTournaments は一覧に載せる大会(listed=true)を区分ごとに返す。区分は
// domain.ListSectionOrder の順で、大会が 0 件の区分は含めない。区分は大会の状態と試合の
// 進み具合から domain.ListSectionOf で決め、その「今日」は now の会場の暦の日にする。
func ListTournaments(ctx context.Context, db *gorm.DB, now time.Time) ([]TournamentSection, error) {
	var rows []Tournament
	err := db.WithContext(ctx).Select("id", "name_ja", "name_en", "status", "start_date").
		Where(sqlListed, true).Order(sqlOrderTournamentList).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list tournaments: %w", err)
	}
	ids := make([]int64, 0, len(rows))
	for _, tournament := range rows {
		ids = append(ids, tournament.ID)
	}
	progress, err := playProgress(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	entries, err := entryCounts(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	today := now.In(venueZone())
	bySection := map[domain.ListSection][]TournamentSummary{}
	for _, tournament := range rows {
		section := domain.ListSectionOf(tournament.Status, progress[tournament.ID], today)
		bySection[section] = append(bySection[section], TournamentSummary{
			ID: tournament.ID, Name: i18n(tournament.NameJa, tournament.NameEn),
			StartDate: tournament.StartDate, TeamCount: entries[tournament.ID],
		})
	}
	sections := []TournamentSection{}
	for _, section := range domain.ListSectionOrder() {
		if list := bySection[section]; len(list) > 0 {
			sections = append(sections, TournamentSection{Section: section, Tournaments: list})
		}
	}
	return sections, nil
}

// divisionProgress は 1 ディビジョンの試合の集計(playProgress の途中の形)。
type divisionProgress struct {
	DivisionID int64      `gorm:"column:division_id"`
	Matches    int        `gorm:"column:matches"`
	Finished   int        `gorm:"column:finished"`
	Live       int        `gorm:"column:live"`
	FirstDay   *time.Time `gorm:"column:first_day"`
	LastDay    *time.Time `gorm:"column:last_day"`
}

// playProgress は大会ごとの試合の進み具合を返す。試合の無い大会は含めない(ゼロ値として扱える)。
func playProgress(ctx context.Context, db *gorm.DB, tournamentIDs []int64) (map[int64]domain.PlayProgress, error) {
	out := map[int64]domain.PlayProgress{}
	if len(tournamentIDs) == 0 {
		return out, nil
	}
	var divisions []Division
	err := db.WithContext(ctx).Select("id", "tournament_id").
		Where(sqlTournamentIn, tournamentIDs).Find(&divisions).Error
	if err != nil {
		return nil, fmt.Errorf("load divisions of listed tournaments: %w", err)
	}
	tournamentOf := map[int64]int64{}
	divisionIDs := make([]int64, 0, len(divisions))
	for _, division := range divisions {
		tournamentOf[division.ID] = division.TournamentID
		divisionIDs = append(divisionIDs, division.ID)
	}
	if len(divisionIDs) == 0 {
		return out, nil
	}
	var rows []divisionProgress
	err = db.WithContext(ctx).Model(&Match{}).
		Select(sqlMatchProgress, domain.MatchFinished, domain.MatchLive).
		Where(sqlDivisionIn, divisionIDs).Group(sqlGroupByDivision).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("count matches of listed tournaments: %w", err)
	}
	for _, row := range rows {
		id := tournamentOf[row.DivisionID]
		sum := out[id]
		sum.Matches += row.Matches
		sum.Finished += row.Finished
		sum.Live += row.Live
		sum.FirstDay = earlier(sum.FirstDay, row.FirstDay)
		sum.LastDay = later(sum.LastDay, row.LastDay)
		out[id] = sum
	}
	return out, nil
}

func earlier(current, candidate *time.Time) *time.Time {
	if current == nil || (candidate != nil && candidate.Before(*current)) {
		return candidate
	}
	return current
}

func later(current, candidate *time.Time) *time.Time {
	if current == nil || (candidate != nil && candidate.After(*current)) {
		return candidate
	}
	return current
}

// entryCounts は大会ごとのエントリー数(棄権したチームを除くチーム数)を返す。
func entryCounts(ctx context.Context, db *gorm.DB, tournamentIDs []int64) (map[int64]int, error) {
	out := map[int64]int{}
	if len(tournamentIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		TournamentID int64 `gorm:"column:tournament_id"`
		Teams        int   `gorm:"column:teams"`
	}
	err := db.WithContext(ctx).Model(&Team{}).Select(sqlTeamCount).
		Where(sqlEnteredTeamsIn, tournamentIDs).Group(sqlGroupByTournament).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("count teams of listed tournaments: %w", err)
	}
	for _, row := range rows {
		out[row.TournamentID] = row.Teams
	}
	return out, nil
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
