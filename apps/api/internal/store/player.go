package store

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const sqlByPlayer = "player_id = ?"

// GetPlayer は選手ページ(スタッフも同じ)を返す。無ければ ENTRY_PLAYER_ID の NotFound。
//
// 前後の選手は同じチームの ID 順(スタッフを含む)で、先頭の前は末尾、末尾の次は先頭に循環する
// (clone-spec D 3.1)。イベントは本人が得点かアシストをした得点だけを、試合の日時順、
// 試合内は経過時間順で返す。
func GetPlayer(ctx context.Context, db *gorm.DB, id int64) (PlayerPage, error) {
	player, err := findByID[Player](ctx, db, id, domain.ResourcePlayer)
	if err != nil {
		return PlayerPage{}, err
	}
	team, err := findInternal[Team](ctx, db, player.TeamID, "team")
	if err != nil {
		return PlayerPage{}, err
	}
	division, err := findInternal[Division](ctx, db, team.DivisionID, "division")
	if err != nil {
		return PlayerPage{}, err
	}
	scope, err := loadScopeOf(ctx, db, team.TournamentID)
	if err != nil {
		return PlayerPage{}, err
	}
	tx := db.WithContext(ctx)
	var teammates []Player
	if err := tx.Where(sqlByTeam, team.ID).Order(sqlOrderID).Find(&teammates).Error; err != nil {
		return PlayerPage{}, fmt.Errorf("load players of team %d: %w", team.ID, err)
	}
	var stats []MatchPlayerStat
	if err := tx.Where(sqlByPlayer, id).Find(&stats).Error; err != nil {
		return PlayerPage{}, fmt.Errorf("load stats of player %d: %w", id, err)
	}
	var goals []MatchEvent
	err = tx.Where(sqlGoalKind, domain.EventGoal).Where(sqlEventsOfPlayer, id, id).Find(&goals).Error
	if err != nil {
		return PlayerPage{}, fmt.Errorf("load goals of player %d: %w", id, err)
	}
	events, err := playerEvents(ctx, db, team.ID, goals)
	if err != nil {
		return PlayerPage{}, err
	}

	prev, next := neighbors(teammates, id)
	agg := aggregatePlayers(stats, goals).get(id)
	return PlayerPage{
		Frame:     scope.frame,
		Player:    withRole(player),
		Team:      teamRef(team),
		Prev:      withRole(prev),
		Next:      withRole(next),
		StatsKind: division.StatsKind,
		Stats: PlayerStats{
			Games: agg.games(), Assists: agg.Assists, Goals: agg.Goals,
			Callahans: agg.Callahans, Greatests: agg.Greatests, Blocks: agg.Blocks,
		},
		Events: events,
	}, nil
}

func withRole(player Player) PlayerWithRole {
	return PlayerWithRole{Player: playerRef(player), Role: playerRole(player)}
}

// neighbors は ID 順に並んだ teammates の中で id の前後を循環で返す。1 人なら本人を返す。
// teammates は id の選手を含むこと(同じチームから読んでいるので必ず含む)。
func neighbors(teammates []Player, id int64) (Player, Player) {
	idx := slices.IndexFunc(teammates, func(player Player) bool { return player.ID == id })
	count := len(teammates)
	return teammates[(idx-1+count)%count], teammates[(idx+1)%count]
}

// playerEvents は得点イベントに試合の情報(CD、日付、相手)と選手名を付けて並べる。
func playerEvents(
	ctx context.Context,
	db *gorm.DB,
	teamID int64,
	goals []MatchEvent,
) ([]PlayerEventLine, error) {
	if len(goals) == 0 {
		return []PlayerEventLine{}, nil
	}
	var matchIDs, playerIDs idSet
	for _, ev := range goals {
		matchIDs.add(ev.MatchID)
		playerIDs.addPtr(ev.ScorerID)
		playerIDs.addPtr(ev.AssistID)
	}
	var matches []Match
	if err := db.WithContext(ctx).Where(sqlByIDs, matchIDs.list).Find(&matches).Error; err != nil {
		return nil, fmt.Errorf("load matches of player events: %w", err)
	}
	matchByID := make(map[int64]Match, len(matches))
	for _, match := range matches {
		matchByID[match.ID] = match
	}
	teams, err := loadTeams(ctx, db, matchTeamIDs(matches))
	if err != nil {
		return nil, err
	}
	players, err := loadPlayers(ctx, db, playerIDs.list)
	if err != nil {
		return nil, err
	}

	sorted := slices.Clone(goals)
	slices.SortStableFunc(sorted, func(a, b MatchEvent) int {
		if c := compareMatchTime(matchByID[a.MatchID], matchByID[b.MatchID]); c != 0 {
			return c
		}
		if c := cmp.Compare(a.ElapsedSec, b.ElapsedSec); c != 0 {
			return c
		}
		return cmp.Compare(a.Seq, b.Seq)
	})
	lines := make([]PlayerEventLine, 0, len(sorted))
	for _, ev := range sorted {
		match := matchByID[ev.MatchID]
		opponentID := match.LeftTeamID
		if match.LeftTeamID != nil && *match.LeftTeamID == teamID {
			opponentID = match.RightTeamID
		}
		opponent, err := teamRefByID(teams, opponentID)
		if err != nil {
			return nil, err
		}
		scorer, err := playerRefByID(players, ev.ScorerID)
		if err != nil {
			return nil, err
		}
		assist, err := playerRefByID(players, ev.AssistID)
		if err != nil {
			return nil, err
		}
		lines = append(lines, PlayerEventLine{
			Code: match.Code, Date: formatDate(match), Opponent: opponent, ElapsedSec: ev.ElapsedSec,
			Callahan: ev.Callahan, Scorer: scorer, Assist: assist,
		})
	}
	return lines, nil
}
