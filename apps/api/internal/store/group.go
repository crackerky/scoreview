package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const (
	sqlByGroup         = "group_id = ?"
	sqlOrderGroupTeams = "sort_order, team_id"
)

// GetGroup はグループ成績を返す。無ければ GROUP_ID の NotFound。Kind に応じて Pool、Bracket、
// Image のどれか 1 つだけを埋める。Matches はグループの全試合を日付、時刻の昇順で返す。
func GetGroup(ctx context.Context, db *gorm.DB, id int64) (GroupPage, error) {
	group, err := findByID[Group](ctx, db, id, domain.ResourceGroup)
	if err != nil {
		return GroupPage{}, err
	}
	division, err := findInternal[Division](ctx, db, group.DivisionID, "division")
	if err != nil {
		return GroupPage{}, err
	}
	scope, err := loadScopeOf(ctx, db, division.TournamentID)
	if err != nil {
		return GroupPage{}, err
	}
	tx := db.WithContext(ctx)
	var matches []Match
	if err := tx.Where(sqlByGroup, id).Order(sqlOrderID).Find(&matches).Error; err != nil {
		return GroupPage{}, fmt.Errorf("load matches of group %d: %w", id, err)
	}
	// 結果一覧の並びと、対戦表で同じ 2 チームの試合が複数あるときに先の試合を選ぶために並べる
	sortMatches(matches)
	var members []GroupTeam
	if err := tx.Where(sqlByGroup, id).Order(sqlOrderGroupTeams).Find(&members).Error; err != nil {
		return GroupPage{}, fmt.Errorf("load teams of group %d: %w", id, err)
	}
	var layout domain.BracketLayout
	if group.Kind == domain.GroupBracket {
		if layout, err = domain.ParseBracketLayout(group.BracketLayout); err != nil {
			return GroupPage{}, fmt.Errorf("group %d: %w", id, err)
		}
	}
	matchesByID, err := matchesForLayout(ctx, db, matches, layout)
	if err != nil {
		return GroupPage{}, err
	}

	var teamIDs idSet
	for _, member := range members {
		teamIDs.add(member.TeamID)
	}
	for _, match := range matchesByID {
		teamIDs.addPtr(match.LeftTeamID)
		teamIDs.addPtr(match.RightTeamID)
	}
	for _, row := range layout.Rows {
		for _, cell := range row {
			teamIDs.addPtr(cell.TeamID)
		}
	}
	teams, err := loadTeams(ctx, db, teamIDs.list)
	if err != nil {
		return GroupPage{}, err
	}

	rows, err := toMatchRows(matches, teams)
	if err != nil {
		return GroupPage{}, err
	}
	page := GroupPage{
		Frame: scope.frame,
		Group: GroupInfo{
			ID: group.ID, Name: group.Name, Kind: group.Kind,
			Division: DivisionRef{ID: division.ID, Name: i18n(division.NameJa, division.NameEn)},
		},
		Matches: rows,
	}
	switch group.Kind {
	case domain.GroupPool:
		pool, err := buildPool(members, matches, teams)
		if err != nil {
			return GroupPage{}, fmt.Errorf("group %d: %w", id, err)
		}
		if pool.Notes, err = poolNotes(ctx, db, group, members, matches, teams); err != nil {
			return GroupPage{}, err
		}
		page.Pool = &pool
	case domain.GroupBracket:
		bracket, err := buildBracket(group, layout, matchesByID, teams)
		if err != nil {
			return GroupPage{}, fmt.Errorf("group %d: %w", id, err)
		}
		page.Bracket = &bracket
	case domain.GroupImage:
		page.Image = &ImageView{
			URL:    deref(group.ImageURL),
			Width:  group.ImageWidth,
			Height: group.ImageHeight,
		}
	}
	return page, nil
}

// matchesForLayout はグループの試合に、トーナメント表が参照する別グループの試合を足して ID で引けるようにする。
func matchesForLayout(
	ctx context.Context,
	db *gorm.DB,
	matches []Match,
	layout domain.BracketLayout,
) (
	map[int64]Match, error,
) {
	byID := make(map[int64]Match, len(matches))
	for _, match := range matches {
		byID[match.ID] = match
	}
	var missing idSet
	for _, row := range layout.Rows {
		for _, cell := range row {
			if cell.MatchID != nil {
				if _, ok := byID[*cell.MatchID]; !ok {
					missing.add(*cell.MatchID)
				}
			}
		}
	}
	if len(missing.list) == 0 {
		return byID, nil
	}
	var extra []Match
	if err := db.WithContext(ctx).Where(sqlByIDs, missing.list).Find(&extra).Error; err != nil {
		return nil, fmt.Errorf("load matches referenced by bracket: %w", err)
	}
	for _, match := range extra {
		byID[match.ID] = match
	}
	return byID, nil
}

// scoreText は "9-3" の形(試合記録の左右順。clone-spec C 2.1(a))。スコアが無ければ空文字。
func scoreText(match Match) string {
	if match.LeftScore == nil || match.RightScore == nil {
		return ""
	}
	return fmt.Sprintf("%d-%d", *match.LeftScore, *match.RightScore)
}

// buildPool は対戦表とリーグ結果を作る。チームは group_teams の登録順で、順位順に並べ替えない
// (clone-spec C 2.1(b))。対戦表は上三角だけに試合を置き、下三角に鏡像を出さない(C 2.1(a))。
func buildPool(members []GroupTeam, matches []Match, teams map[int64]Team) (PoolView, error) {
	pool := PoolView{
		Teams:   make([]domain.TeamRef, 0, len(members)),
		Matrix:  make([][]MatrixCell, 0, len(members)),
		Results: make([]PoolResultRow, 0, len(members)),
	}
	entries := make([]domain.PoolEntry, 0, len(members))
	for _, member := range members {
		team, ok := teams[member.TeamID]
		if !ok {
			return PoolView{}, fmt.Errorf("team %d of group is not loaded", member.TeamID)
		}
		pool.Teams = append(pool.Teams, teamRef(team))
		entries = append(
			entries,
			domain.PoolEntry{TeamID: member.TeamID, RankOverride: member.RankOverride},
		)
	}

	// 同じ 2 チームの試合が複数あれば、日付、時刻で先の試合を出す(matches はその順)
	type pair struct{ low, high int64 }
	byPair := map[pair]Match{}
	for _, match := range matches {
		if match.LeftTeamID == nil || match.RightTeamID == nil {
			continue
		}
		key := pair{
			min(*match.LeftTeamID, *match.RightTeamID),
			max(*match.LeftTeamID, *match.RightTeamID),
		}
		if _, ok := byPair[key]; !ok {
			byPair[key] = match
		}
	}
	for i, row := range members {
		cells := make([]MatrixCell, len(members))
		for j, col := range members {
			switch {
			case i == j:
				cells[j] = MatrixCell{Kind: MatrixSelf}
			case i > j:
				cells[j] = MatrixCell{Kind: MatrixLower}
			default:
				key := pair{min(row.TeamID, col.TeamID), max(row.TeamID, col.TeamID)}
				if match, ok := byPair[key]; ok {
					text := scoreText(match)
					if text == "" {
						// 未試合のセルは試合の CD を出す(527 のグループ 6631 で "A1" など)
						text = match.Code
					}
					cells[j] = MatrixCell{
						Kind:    MatrixMatch,
						MatchID: match.ID,
						Status:  match.Status,
						Text:    text,
					}
				} else {
					cells[j] = MatrixCell{Kind: MatrixEmpty}
				}
			}
		}
		pool.Matrix = append(pool.Matrix, cells)
	}

	for i, standing := range domain.RankPool(entries, toResults(matches)) {
		pool.Results = append(pool.Results, PoolResultRow{
			Team: pool.Teams[i], Tally: standing.Tally, Points: standing.Points, Rank: standing.Rank,
		})
	}
	return pool, nil
}

// buildBracket はトーナメント表の格子に、チーム名と試合のスコア、状態を差し込む。
// game セルの文字列は試合にスコアがあればそれ("上-下"。テンプレートは上のチームを試合の左に
// 登録する前提)、無ければ格子に保存した文字列。
func buildBracket(
	group Group,
	layout domain.BracketLayout,
	matches map[int64]Match,
	teams map[int64]Team,
) (
	BracketView, error,
) {
	view := BracketView{
		Pattern: deref(group.BracketPattern),
		PtnName: deref(group.BracketPtnName),
		Columns: layout.Columns,
		Rows:    make([][]BracketCellView, 0, len(layout.Rows)),
	}
	for _, row := range layout.Rows {
		cells := make([]BracketCellView, 0, len(row))
		for _, cell := range row {
			out := BracketCellView{
				Borders: cell.Borders,
				Kind:    cell.Kind,
				Text:    cell.Text,
				MatchID: cell.MatchID,
			}
			team, err := teamRefByID(teams, cell.TeamID)
			if err != nil {
				return BracketView{}, err
			}
			out.Team = team
			if cell.MatchID != nil {
				match, ok := matches[*cell.MatchID]
				if !ok {
					return BracketView{}, fmt.Errorf("bracket references missing match %d", *cell.MatchID)
				}
				status := match.Status
				out.Status = &status
				if text := scoreText(match); text != "" {
					out.Text = text
				}
			}
			cells = append(cells, out)
		}
		view.Rows = append(view.Rows, cells)
	}
	return view, nil
}
