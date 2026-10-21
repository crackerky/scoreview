package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const (
	sqlMatchesIn       = "match_id IN (?)"
	sqlGoalsOfPlayers  = "kind = ? AND (scorer_id IN ? OR assist_id IN ?)"
	sqlPlayersIn       = "player_id IN ?"
	sqlSelectMatchID   = "id"
	sqlGoalsInMatchSet = "kind = ? AND match_id IN (?)"
)

// playerAgg は選手 1 人の集計。試合数は出場記録(match_player_stats)か得点経過に現れた試合の数。
// 試合の状態は問わず、試合中の記録も数える(参照サイトで試合中を観測できず推測)。
type playerAgg struct {
	matches   map[int64]bool
	Goals     int
	Assists   int
	Blocks    int
	Callahans int
	Greatests int
}

func (a *playerAgg) games() int { return len(a.matches) }

type playerAggs map[int64]*playerAgg

func (p playerAggs) of(playerID int64) *playerAgg {
	agg, ok := p[playerID]
	if !ok {
		agg = &playerAgg{matches: map[int64]bool{}}
		p[playerID] = agg
	}
	return agg
}

// get は集計を返す。記録の無い選手はゼロの集計。
func (p playerAggs) get(playerID int64) playerAgg {
	if agg, ok := p[playerID]; ok {
		return *agg
	}
	return playerAgg{matches: map[int64]bool{}}
}

// aggregatePlayers は出場記録と得点イベント(kind=goal)から選手ごとの集計を作る。
func aggregatePlayers(stats []MatchPlayerStat, goals []MatchEvent) playerAggs {
	aggs := playerAggs{}
	for _, stat := range stats {
		agg := aggs.of(stat.PlayerID)
		agg.matches[stat.MatchID] = true
		agg.Blocks += stat.Blocks
		agg.Greatests += stat.Greatests
	}
	for _, ev := range goals {
		if ev.Kind != domain.EventGoal {
			continue
		}
		if ev.ScorerID != nil {
			agg := aggs.of(*ev.ScorerID)
			agg.matches[ev.MatchID] = true
			agg.Goals++
			// キャラハン数は得点者としてのキャラハンゴールの数(migrations/00002)
			if ev.Callahan {
				agg.Callahans++
			}
		}
		if ev.AssistID != nil {
			agg := aggs.of(*ev.AssistID)
			agg.matches[ev.MatchID] = true
			agg.Assists++
		}
	}
	return aggs
}

// loadDivisionPlayerAggs はディビジョンの全試合の記録を選手ごとに集計する(問い合わせ 2 回)。
func loadDivisionPlayerAggs(
	ctx context.Context,
	db *gorm.DB,
	divisionID int64,
) (playerAggs, error) {
	tx := db.WithContext(ctx)
	matchIDs := tx.Model(&Match{}).Select(sqlSelectMatchID).Where(sqlByDivision, divisionID)
	var stats []MatchPlayerStat
	if err := tx.Where(sqlMatchesIn, matchIDs).Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("load player stats of division %d: %w", divisionID, err)
	}
	var goals []MatchEvent
	if err := tx.Where(sqlGoalsInMatchSet, domain.EventGoal, matchIDs).Find(&goals).Error; err != nil {
		return nil, fmt.Errorf("load goals of division %d: %w", divisionID, err)
	}
	return aggregatePlayers(stats, goals), nil
}

// loadPlayersAggs は指定した選手の全試合の記録を集計する(問い合わせ 2 回)。
func loadPlayersAggs(ctx context.Context, db *gorm.DB, playerIDs []int64) (playerAggs, error) {
	if len(playerIDs) == 0 {
		return playerAggs{}, nil
	}
	tx := db.WithContext(ctx)
	var stats []MatchPlayerStat
	if err := tx.Where(sqlPlayersIn, playerIDs).Find(&stats).Error; err != nil {
		return nil, fmt.Errorf("load player stats: %w", err)
	}
	var goals []MatchEvent
	if err := tx.Where(sqlGoalsOfPlayers, domain.EventGoal, playerIDs, playerIDs).Find(&goals).Error; err != nil {
		return nil, fmt.Errorf("load goals of players: %w", err)
	}
	return aggregatePlayers(stats, goals), nil
}
