package store

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const (
	// leadersLimit は得点ランキングの件数(docs/api.md。参照サイトの 949 は 30 行。clone-spec C 1.5)
	leadersLimit = 30
	// blockLeadersLimit はブロックランキングの件数(docs/api.md。参照サイトの 949 は 10 行)
	blockLeadersLimit = 10
)

// GetDivisionStandings はディビジョン成績を返す。無ければ EVENT_DIVISION_ID の NotFound。
//
// 順位は teams.final_rank の値で、勝率から計算しない(clone-spec C 1.4)。ディビジョンの順位が
// 未公開(standings_published=false)の間は全チームの順位を nil にする(publishedRank)。行は順位の
// 昇順(未確定は末尾)、同順位は compareStandingRows の順。未公開の間は順位で並べないので、行の順から
// 順位は分からない。成績はディビジョンの終了した全試合の合算。
func GetDivisionStandings(ctx context.Context, db *gorm.DB, id int64) (StandingsPage, error) {
	division, err := findByID[Division](ctx, db, id, domain.ResourceDivision)
	if err != nil {
		return StandingsPage{}, err
	}
	scope, err := loadScopeOf(ctx, db, division.TournamentID)
	if err != nil {
		return StandingsPage{}, err
	}
	tx := db.WithContext(ctx)
	var teams []Team
	if err := tx.Where(sqlByDivision, id).Order(sqlOrderSortID).Find(&teams).Error; err != nil {
		return StandingsPage{}, fmt.Errorf("load teams of division %d: %w", id, err)
	}
	var matches []Match
	if err := tx.Where(sqlByDivision, id).Find(&matches).Error; err != nil {
		return StandingsPage{}, fmt.Errorf("load matches of division %d: %w", id, err)
	}
	teamIDs := make([]int64, len(teams))
	for i, team := range teams {
		teamIDs[i] = team.ID
	}
	var players []Player
	if len(teamIDs) > 0 {
		if err := tx.Where(sqlByTeams, teamIDs).Order(sqlOrderID).Find(&players).Error; err != nil {
			return StandingsPage{}, fmt.Errorf("load players of division %d: %w", id, err)
		}
	}
	aggs, err := loadDivisionPlayerAggs(ctx, db, id)
	if err != nil {
		return StandingsPage{}, err
	}

	results := toResults(matches)
	page := StandingsPage{
		Frame:    scope.frame,
		Division: DivisionRef{ID: division.ID, Name: i18n(division.NameJa, division.NameEn)},
	}
	page.Standings = make([]StandingRow, 0, len(teams))
	for _, team := range teams {
		page.Standings = append(page.Standings, StandingRow{
			Team: teamRef(team), Tally: domain.TallyFor(team.ID, results),
			Rank: publishedRank(division, team),
		})
	}
	slices.SortStableFunc(page.Standings, compareStandingRows)
	teamGames := make(map[int64]int, len(page.Standings))
	for _, row := range page.Standings {
		teamGames[row.Team.ID] = row.Tally.Games
	}
	page.Leaders, page.BlockLeaders = buildLeaders(teams, players, aggs, teamGames)
	return page, nil
}

// publishedRank はチームの最終順位を、ディビジョンの順位を公開している(standings_published)ときだけ
// 返す。ページ自体は参照サイトと同じく ID で直接開けるが、本部が確定させて公開する前の順位は
// 閲覧用に出さない(docs/requirements.md「権限モデル」。セキュリティ点検 L-6)。
func publishedRank(division Division, team Team) *int {
	if !division.StandingsPublished {
		return nil
	}
	return team.FinalRank
}

// compareStandingRows はディビジョン成績の行の順。順位の昇順(未確定は末尾)、同順位は勝ち数の降順、
// 得失差の降順(代表 16 大会のディビジョン成績 11 ページの同順位 86 組すべてで一致)。それでも
// 同じならチームの並び(呼び出し側の安定ソートの元の順)。
func compareStandingRows(a, b StandingRow) int {
	switch {
	case a.Rank == nil && b.Rank == nil:
		return 0 // 未確定どうしはチームの並びのまま
	case a.Rank == nil:
		return 1
	case b.Rank == nil:
		return -1
	case *a.Rank != *b.Rank:
		return cmp.Compare(*a.Rank, *b.Rank)
	}
	if c := cmp.Compare(b.Tally.Wins, a.Tally.Wins); c != 0 {
		return c
	}
	return cmp.Compare(b.Tally.GoalDiff(), a.Tally.GoalDiff())
}

// buildLeaders は得点ランキングとブロックランキングを作る。どちらも記録が 0 の選手は除く
// (参照サイトはデータがある節だけを出す。clone-spec C 1.1)。得点ランキングの並びは選手の並び順
// (domain.ComparePlayers。同点の順は参照サイトで未検証)、ブロックはブロック数の降順。
// それでも同じなら、チームの並び、選手 ID 順。試合数は選手のチームの試合数 teamGames(チームページの
// 選手一覧と同じ。代表 16 大会の得点 288 行、ブロック 80 行すべてで一致)。
func buildLeaders(
	teams []Team, players []Player, aggs playerAggs, teamGames map[int64]int,
) ([]LeaderRow, []BlockLeaderRow) {
	teamsByID := map[int64]Team{}
	teamOrder := map[int64]int{}
	for i, team := range teams {
		teamsByID[team.ID] = team
		teamOrder[team.ID] = i
	}
	sorted := slices.Clone(players)
	slices.SortStableFunc(sorted, func(a, b Player) int {
		if c := cmp.Compare(teamOrder[a.TeamID], teamOrder[b.TeamID]); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})

	leaders := []LeaderRow{}
	blocks := []BlockLeaderRow{}
	for _, player := range sorted {
		if player.IsStaff() {
			continue
		}
		agg := aggs.get(player.ID)
		team := teamRef(teamsByID[player.TeamID])
		if agg.Goals+agg.Assists > 0 {
			leaders = append(leaders, LeaderRow{
				Team: team, Player: playerRef(player), Games: teamGames[player.TeamID],
				Assists: agg.Assists, Goals: agg.Goals,
			})
		}
		if agg.Blocks > 0 {
			blocks = append(
				blocks,
				BlockLeaderRow{
					Team: team, Player: playerRef(player), Games: teamGames[player.TeamID], Blocks: agg.Blocks,
				},
			)
		}
	}
	domain.SortPlayers(leaders, func(leader LeaderRow) domain.PlayerSortKey {
		return domain.PlayerSortKey{
			Goals:   leader.Goals,
			Assists: leader.Assists,
			Number:  leader.Player.Number,
		}
	})
	slices.SortStableFunc(
		blocks,
		func(a, b BlockLeaderRow) int { return cmp.Compare(b.Blocks, a.Blocks) },
	)
	if len(leaders) > leadersLimit {
		leaders = leaders[:leadersLimit]
	}
	if len(blocks) > blockLeadersLimit {
		blocks = blocks[:blockLeadersLimit]
	}
	return leaders, blocks
}
