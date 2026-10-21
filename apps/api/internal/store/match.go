package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const sqlOrderPlayerID = "player_id"

// GetMatch は試合ページを返す。無ければ MATCH_ID の NotFound。
//
// 得点経過は経過時間の昇順(同時刻は seq 順)で、各時点の累計スコアを付ける。画面で新しい順に
// 並べ替えるのは web(scoreview-design 5)。出場選手は出場記録のある選手と、記録が無くても
// 得点経過に現れた選手を左右に分け、domain.ComparePlayers の順に並べる。
func GetMatch(ctx context.Context, db *gorm.DB, id int64) (MatchPage, error) {
	match, err := findByID[Match](ctx, db, id, domain.ResourceMatch)
	if err != nil {
		return MatchPage{}, err
	}
	division, err := findInternal[Division](ctx, db, match.DivisionID, "division")
	if err != nil {
		return MatchPage{}, err
	}
	scope, err := loadScopeOf(ctx, db, division.TournamentID)
	if err != nil {
		return MatchPage{}, err
	}
	var groupName *string
	if match.GroupID != nil {
		group, err := findInternal[Group](ctx, db, *match.GroupID, "group")
		if err != nil {
			return MatchPage{}, err
		}
		groupName = &group.Name
	}
	if match.GroupLabel != nil {
		// clone-spec D 2.2: 試合に持たせたグループ欄の文字がグループの名前より優先
		groupName = match.GroupLabel
	}
	teams, err := loadTeams(ctx, db, matchTeamIDs([]Match{match}))
	if err != nil {
		return MatchPage{}, err
	}
	tx := db.WithContext(ctx)
	var events []MatchEvent
	if err := tx.Where(sqlByMatch, id).Order(sqlOrderEvents).Find(&events).Error; err != nil {
		return MatchPage{}, fmt.Errorf("load events of match %d: %w", id, err)
	}
	var stats []MatchPlayerStat
	if err := tx.Where(sqlByMatch, id).Order(sqlOrderPlayerID).Find(&stats).Error; err != nil {
		return MatchPage{}, fmt.Errorf("load player stats of match %d: %w", id, err)
	}
	var playerIDs idSet
	for _, stat := range stats {
		playerIDs.add(stat.PlayerID)
	}
	for _, ev := range events {
		playerIDs.addPtr(ev.ScorerID)
		playerIDs.addPtr(ev.AssistID)
	}
	players, err := loadPlayers(ctx, db, playerIDs.list)
	if err != nil {
		return MatchPage{}, err
	}

	info, err := matchInfo(match, groupName, division, teams)
	if err != nil {
		return MatchPage{}, err
	}
	lines, err := eventLines(events, players)
	if err != nil {
		return MatchPage{}, err
	}
	return MatchPage{
		Frame:   scope.frame,
		Match:   info,
		Events:  lines,
		Lineups: buildLineups(stats, events, players),
		Stats:   domain.ComputeMatchStats(toDomainEvents(events)),
	}, nil
}

func matchInfo(
	match Match,
	groupName *string,
	division Division,
	teams map[int64]Team,
) (MatchInfo, error) {
	left, err := teamScore(teams, match.LeftTeamID, match.LeftScore)
	if err != nil {
		return MatchInfo{}, err
	}
	right, err := teamScore(teams, match.RightTeamID, match.RightScore)
	if err != nil {
		return MatchInfo{}, err
	}
	return MatchInfo{
		ID:         match.ID,
		Code:       match.Code,
		GroupName:  groupName,
		Date:       formatDate(match),
		Time:       formatClock(match),
		Field:      match.Field,
		Status:     match.Status,
		Left:       left,
		Right:      right,
		Note:       match.Note,
		HideBlocks: match.HideBlocks,
		HideStats:  match.HideStats,
		ResultType: match.ResultType,
		WinnerSide: toResult(match).Winner(),
		Breadcrumb: Breadcrumb{
			DivisionID: division.ID, DivisionName: i18n(division.NameJa, division.NameEn), GroupID: match.GroupID,
		},
	}, nil
}

// teamScore はスコアボードの片側(短縮名、正式名、スコア)を作る。
func teamScore(teams map[int64]Team, teamID *int64, score *int) (MatchTeamScore, error) {
	ref, err := teamRefByID(teams, teamID)
	if err != nil || ref == nil {
		return MatchTeamScore{Score: score}, err
	}
	team := teams[*teamID]
	fullName := i18n(team.NameFullJa, team.NameFullEn)
	return MatchTeamScore{Team: ref, FullName: &fullName, Score: score}, nil
}

func toDomainEvents(events []MatchEvent) []domain.Event {
	out := make([]domain.Event, len(events))
	for i, ev := range events {
		out[i] = domain.Event{
			ElapsedSec: ev.ElapsedSec, Kind: ev.Kind, Side: ev.Side, Callahan: ev.Callahan,
			ScorerID: ev.ScorerID, AssistID: ev.AssistID,
		}
	}
	return out
}

func eventLines(events []MatchEvent, players map[int64]Player) ([]EventLine, error) {
	scores := domain.RunningScores(toDomainEvents(events))
	lines := make([]EventLine, 0, len(events))
	for i, ev := range events {
		scorer, err := playerRefByID(players, ev.ScorerID)
		if err != nil {
			return nil, err
		}
		assist, err := playerRefByID(players, ev.AssistID)
		if err != nil {
			return nil, err
		}
		lines = append(lines, EventLine{
			ElapsedSec: ev.ElapsedSec, Kind: ev.Kind, Side: ev.Side, Scorer: scorer, Assist: assist,
			Callahan: ev.Callahan, LeftScore: scores[i].Left, RightScore: scores[i].Right,
		})
	}
	return lines, nil
}

// buildLineups は出場選手を左右に分ける。出場記録の side を使い、記録の無い選手は得点イベントの side に置く。
func buildLineups(stats []MatchPlayerStat, events []MatchEvent, players map[int64]Player) Lineups {
	type line struct {
		side domain.Side
		LineupLine
	}
	byPlayer := map[int64]*line{}
	var order []int64
	get := func(playerID int64, side domain.Side) *line {
		entry, ok := byPlayer[playerID]
		if !ok {
			entry = &line{side: side, LineupLine: LineupLine{Player: playerRef(players[playerID])}}
			byPlayer[playerID] = entry
			order = append(order, playerID)
		}
		return entry
	}
	for _, stat := range stats {
		get(stat.PlayerID, stat.Side).Blocks += stat.Blocks
	}
	for _, ev := range events {
		if ev.Kind != domain.EventGoal || ev.Side == nil {
			continue
		}
		if ev.ScorerID != nil {
			get(*ev.ScorerID, *ev.Side).Goals++
		}
		if ev.AssistID != nil {
			get(*ev.AssistID, *ev.Side).Assists++
		}
	}
	lineups := Lineups{Left: []LineupLine{}, Right: []LineupLine{}}
	for _, playerID := range order {
		entry := byPlayer[playerID]
		if entry.side == domain.SideLeft {
			lineups.Left = append(lineups.Left, entry.LineupLine)
		} else {
			lineups.Right = append(lineups.Right, entry.LineupLine)
		}
	}
	key := func(lineup LineupLine) domain.PlayerSortKey {
		return domain.PlayerSortKey{
			Goals:   lineup.Goals,
			Assists: lineup.Assists,
			Number:  lineup.Player.Number,
		}
	}
	domain.SortPlayers(lineups.Left, key)
	domain.SortPlayers(lineups.Right, key)
	return lineups
}
