package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/viewing"
)

const sqlByGroups = "group_id IN ?"

// ViewingSource は viewing.Source を GORM で実装する(公開のスコアサイトの記録の読み方)。
type ViewingSource struct {
	db *gorm.DB
}

var _ viewing.Source = (*ViewingSource)(nil)

// NewViewingSource は db から読む ViewingSource を返す。
func NewViewingSource(db *gorm.DB) *ViewingSource {
	return &ViewingSource{db: db}
}

// Snapshot は viewing.Source.Snapshot。問い合わせは試合、組、チームの数によらず 14 回(大会と範囲 6、
// 試合 1、組 1、組のチーム 1、チーム 1、試合の名前のための大会の形 1、表の見せ方と形 3)。
func (s *ViewingSource) Snapshot(ctx context.Context, tournamentID int64) (viewing.Snapshot, error) {
	scope, err := loadScopeByTournamentID(ctx, s.db, tournamentID)
	if err != nil {
		return viewing.Snapshot{}, err
	}
	divisionIDs := make([]int64, len(scope.divisions))
	for i, division := range scope.divisions {
		divisionIDs[i] = division.ID
	}
	tx := s.db.WithContext(ctx)
	var matches []Match
	var groups []Group
	if len(divisionIDs) > 0 {
		if err := tx.Where(sqlByDivisions, divisionIDs).Find(&matches).Error; err != nil {
			return viewing.Snapshot{}, fmt.Errorf("load matches of tournament %d: %w", tournamentID, err)
		}
		// bracket_layout は大きく、掲示板では使わない
		err := tx.Select("id", "division_id", "name", "name_en", "kind", "linked").
			Where(sqlPublishedGroups, divisionIDs, true).Order(sqlOrderSortID).Find(&groups).Error
		if err != nil {
			return viewing.Snapshot{}, fmt.Errorf("load groups of tournament %d: %w", tournamentID, err)
		}
	}
	// 名前だけを出す組(linked が偽)には詳しい画面が無いので、案内しない
	shown := groups[:0]
	var poolIDs []int64
	for _, group := range groups {
		if !group.Linked {
			continue
		}
		shown = append(shown, group)
		if group.Kind == domain.GroupPool {
			poolIDs = append(poolIDs, group.ID)
		}
	}
	var members []GroupTeam
	if len(poolIDs) > 0 {
		if err := tx.Where(sqlByGroups, poolIDs).Order(sqlOrderGroupTeams).Find(&members).Error; err != nil {
			return viewing.Snapshot{}, fmt.Errorf("load teams of groups of tournament %d: %w", tournamentID, err)
		}
	}
	var teamIDs idSet
	for _, id := range matchTeamIDs(matches) {
		teamIDs.add(id)
	}
	for _, member := range members {
		teamIDs.add(member.TeamID)
	}
	teams, err := loadTeams(ctx, s.db, teamIDs.list)
	if err != nil {
		return viewing.Snapshot{}, err
	}
	stages, err := stageNames(ctx, s.db, matches)
	if err != nil {
		return viewing.Snapshot{}, err
	}
	tables, err := loadLiveTables(ctx, s.db, scope)
	if err != nil {
		return viewing.Snapshot{}, err
	}

	out := viewing.Snapshot{
		Tournament: viewing.Tournament{ID: scope.tournament.ID, Name: scope.frame.Name},
		Divisions:  make([]viewing.Division, 0, len(scope.divisions)),
		Groups:     make([]viewing.Group, 0, len(shown)),
		Matches:    make([]viewing.Match, 0, len(matches)),
		Brackets:   []viewing.Bracket{},
		Tables:     viewingTablesConfig(tables.Config),
	}
	if len(scope.frame.Banners) > 0 {
		banner := scope.frame.Banners[0]
		out.Tournament.Banner = &viewing.Banner{ImageURL: banner.ImageURL, LinkURL: banner.LinkURL}
	}
	for _, division := range scope.divisions {
		out.Divisions = append(out.Divisions, viewing.Division{ID: division.ID,
			Name: i18n(division.NameJa, division.NameEn)})
	}
	if out.Groups, err = viewingGroups(shown, members, teams, tables); err != nil {
		return viewing.Snapshot{}, err
	}
	shownIDs := map[int64]bool{}
	for _, group := range shown {
		shownIDs[group.ID] = true
	}
	divisions := scope.scheduleDivisions()
	for _, match := range matches {
		row, err := toMatchRow(match, teams)
		if err != nil {
			return viewing.Snapshot{}, err
		}
		division, ok := divisions[match.DivisionID]
		if !ok {
			return viewing.Snapshot{}, fmt.Errorf("match %d belongs to unknown division %d", match.ID, match.DivisionID)
		}
		item := viewing.Match{ScheduleRow: domain.ScheduleRow{MatchRow: row, Division: division,
			Stage: stages[match.DivisionID][match.Code]}}
		if match.GroupID != nil && shownIDs[*match.GroupID] {
			id := *match.GroupID
			item.GroupID = &id
		}
		out.Matches = append(out.Matches, item)
	}
	for _, division := range tables.Divisions {
		for _, bracket := range division.Brackets {
			out.Brackets = append(out.Brackets, viewingBracket(division.ID, bracket))
		}
	}
	return out, nil
}

// viewingGroups は案内する組に、総当たりの組の登録順のチームと、優勝を争うグループへ進む数を付ける。
func viewingGroups(
	groups []Group,
	members []GroupTeam,
	teams map[int64]Team,
	tables LiveTables,
) ([]viewing.Group, error) {
	advance := map[int64]int{}
	for _, division := range tables.Divisions {
		for _, pool := range division.Pools {
			advance[pool.GroupID] = pool.Advance
		}
	}
	entries := map[int64][]viewing.Entry{}
	for _, member := range members {
		team, ok := teams[member.TeamID]
		if !ok {
			return nil, fmt.Errorf("team %d of group %d is not loaded", member.TeamID, member.GroupID)
		}
		entries[member.GroupID] = append(entries[member.GroupID],
			viewing.Entry{Team: teamRef(team), RankOverride: member.RankOverride})
	}
	out := make([]viewing.Group, 0, len(groups))
	for _, group := range groups {
		out = append(out, viewing.Group{ID: group.ID, DivisionID: group.DivisionID,
			Name: i18n(group.Name, group.NameEn), Kind: group.Kind, Advance: advance[group.ID],
			Entries: entries[group.ID]})
	}
	return out, nil
}

func viewingBracket(divisionID int64, bracket LiveBracket) viewing.Bracket {
	side := func(side LiveBracketSide) viewing.BracketSide {
		return viewing.BracketSide{From: side.From, Seed: side.Seed}
	}
	out := viewing.Bracket{DivisionID: divisionID, GroupID: bracket.GroupID, Name: bracket.Name,
		Matches: make([]viewing.BracketMatch, 0, len(bracket.Matches))}
	for _, m := range bracket.Matches {
		out.Matches = append(out.Matches, viewing.BracketMatch{Code: m.Code, Round: m.Round,
			WinnerPlace: m.WinnerPlace, LoserPlace: m.LoserPlace, Left: side(m.Left), Right: side(m.Right)})
	}
	return out
}

func viewingTablesConfig(c LiveTablesConfig) viewing.TablesConfig {
	return viewing.TablesConfig{Shown: c.Shown, PoolShown: c.PoolShown, BracketShown: c.BracketShown,
		Columns: viewing.TableColumns{WDL: c.Columns.WDL, Goals: c.Columns.Goals, Diff: c.Columns.Diff,
			Points: c.Columns.Points, Rank: c.Columns.Rank},
		Marks: c.Marks, Order: c.Order, Live: c.Live, Highlight: c.Highlight}
}
