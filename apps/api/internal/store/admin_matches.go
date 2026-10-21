package store

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// 管理画面の試合の登録と読み取り。スコアと状態(left_score / right_score / status / winner_side)は
// ここでは変えず、matchupdate.go の試合更新関数だけが変える(scoreview-design 8)。

// MatchSideFields は試合の片側の登録値。TeamID が確定したチーム、Placeholder が未確定の参照
// ("[A1-勝ち]" は Code "A1"、Outcome win)。両方あってもよい(参照が解決済みの状態)。
type MatchSideFields struct {
	TeamID      *int64
	Placeholder *domain.Placeholder
}

func (f MatchSideFields) validate(field string) error {
	if f.Placeholder == nil {
		return nil
	}
	if err := domain.CheckRequired(field+".placeholder.code", f.Placeholder.Code); err != nil {
		return err
	}
	if f.Placeholder.Outcome != domain.OutcomeWin && f.Placeholder.Outcome != domain.OutcomeLose {
		return domain.Invalid(field+".placeholder.outcome", "must be win or lose")
	}
	return nil
}

// MatchFields は試合の登録値。Date は "YYYY-MM-DD"、Time は "HH:MM"、Field はコート(数字とは限らない)。
type MatchFields struct {
	GroupID   *int64
	Code      string
	Date      *string
	Time      *string
	Field     *string
	HasDetail bool
	Left      MatchSideFields
	Right     MatchSideFields
}

// MatchRecord は管理画面の試合。Left と Right は閲覧用の日程の行と同じ形(チームは短縮名)。
type MatchRecord struct {
	ID         int64
	DivisionID int64
	GroupID    *int64
	Code       string
	Date       *string
	Time       *string
	Field      *string
	HasDetail  bool
	Status     domain.MatchStatus
	Left       domain.MatchSide
	Right      domain.MatchSide
	WinnerSide *domain.Side
	ResultType domain.ResultType
}

func (f MatchFields) validate() error {
	if f.Date != nil {
		if _, err := domain.ParseDay("date", *f.Date); err != nil {
			return err
		}
	}
	if f.Time != nil {
		if err := domain.CheckClock("time", *f.Time); err != nil {
			return err
		}
	}
	if f.Field != nil && strings.TrimSpace(*f.Field) == "" {
		return domain.Invalid("field", "must be null or non-empty")
	}
	if err := f.Left.validate("left"); err != nil {
		return err
	}
	if err := f.Right.validate("right"); err != nil {
		return err
	}
	if f.Left.TeamID != nil && f.Right.TeamID != nil && *f.Left.TeamID == *f.Right.TeamID {
		return domain.Invalid("right.teamId", "must differ from left.teamId")
	}
	return nil
}

func (f MatchFields) apply(row *Match) error {
	row.GroupID, row.Code, row.Field, row.HasDetail = f.GroupID, f.Code, f.Field, f.HasDetail
	row.MatchDate, row.StartTime = nil, f.Time
	if f.Date != nil {
		day, err := domain.ParseDay("date", *f.Date)
		if err != nil {
			return err
		}
		row.MatchDate = &day
	}
	row.LeftTeamID, row.RightTeamID = f.Left.TeamID, f.Right.TeamID
	row.LeftPlaceholderCode, row.LeftPlaceholderOutcome = placeholderColumns(f.Left.Placeholder)
	row.RightPlaceholderCode, row.RightPlaceholderOutcome = placeholderColumns(f.Right.Placeholder)
	return nil
}

func placeholderColumns(p *domain.Placeholder) (*string, *domain.Outcome) {
	if p == nil {
		return nil, nil
	}
	code, outcome := p.Code, p.Outcome
	return &code, &outcome
}

// checkMatchRefs は試合のチームとグループが同じディビジョンのものであることを確かめる。
func checkMatchRefs(ctx context.Context, db *gorm.DB, divisionID int64, fields MatchFields) error {
	var teamIDs []int64
	for _, id := range []*int64{fields.Left.TeamID, fields.Right.TeamID} {
		if id != nil {
			teamIDs = append(teamIDs, *id)
		}
	}
	if err := checkTeamsInDivision(ctx, db, teamIDs, divisionID); err != nil {
		return err
	}
	if fields.GroupID == nil {
		return nil
	}
	var groups []Group
	err := db.WithContext(ctx).
		Select("id", "division_id").
		Where(sqlByID, *fields.GroupID).
		Limit(1).
		Find(&groups).
		Error
	if err != nil {
		return fmt.Errorf("load group %d: %w", *fields.GroupID, err)
	}
	if len(groups) == 0 || groups[0].DivisionID != divisionID {
		return domain.Invalid("groupId", "must be a group of the same division")
	}
	return nil
}

func toMatchRecord(match Match, teams map[int64]Team) (MatchRecord, error) {
	row, err := toMatchRow(match, teams)
	if err != nil {
		return MatchRecord{}, err
	}
	// 閲覧用の行は team があると placeholder を出さないが、管理画面では解決済みの参照も見せる
	row.Left.Placeholder = placeholderOf(match.LeftPlaceholderCode, match.LeftPlaceholderOutcome)
	row.Right.Placeholder = placeholderOf(match.RightPlaceholderCode, match.RightPlaceholderOutcome)
	return MatchRecord{
		ID:         match.ID,
		DivisionID: match.DivisionID,
		GroupID:    match.GroupID,
		Code:       match.Code,
		Date:       row.Date,
		Time:       row.Time,
		Field:      match.Field,
		HasDetail:  match.HasDetail,
		Status:     match.Status,
		Left:       row.Left,
		Right:      row.Right,
		WinnerSide: match.WinnerSide,
		ResultType: match.ResultType,
	}, nil
}

func placeholderOf(code *string, outcome *domain.Outcome) *domain.Placeholder {
	if code == nil || outcome == nil {
		return nil
	}
	return &domain.Placeholder{Code: *code, Outcome: *outcome}
}

func toMatchRecords(ctx context.Context, db *gorm.DB, matches []Match) ([]MatchRecord, error) {
	teams, err := loadTeams(ctx, db, matchTeamIDs(matches))
	if err != nil {
		return nil, err
	}
	out := make([]MatchRecord, 0, len(matches))
	for _, match := range matches {
		record, err := toMatchRecord(match, teams)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

// ListMatches はディビジョンの試合を日程の既定の並び(日付、時刻、コートの自然順)で返す。
func ListMatches(ctx context.Context, db *gorm.DB, divisionID int64) ([]MatchRecord, error) {
	if _, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision); err != nil {
		return nil, err
	}
	var matches []Match
	if err := db.WithContext(ctx).
		Where(sqlByDivision, divisionID).
		Find(&matches).
		Error; err != nil {
		return nil, fmt.Errorf("list matches of division %d: %w", divisionID, err)
	}
	sortMatches(matches)
	return toMatchRecords(ctx, db, matches)
}

// CreateMatch はディビジョンに未試合の試合を作る。
func CreateMatch(
	ctx context.Context,
	db *gorm.DB,
	divisionID int64,
	fields MatchFields,
) (MatchRecord, error) {
	if err := fields.validate(); err != nil {
		return MatchRecord{}, err
	}
	if _, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision); err != nil {
		return MatchRecord{}, err
	}
	if err := checkMatchRefs(ctx, db, divisionID, fields); err != nil {
		return MatchRecord{}, err
	}
	row := Match{DivisionID: divisionID, Status: domain.MatchScheduled}
	if err := fields.apply(&row); err != nil {
		return MatchRecord{}, err
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return MatchRecord{}, writeError("create match", err)
	}
	records, err := toMatchRecords(ctx, db, []Match{row})
	if err != nil {
		return MatchRecord{}, err
	}
	return records[0], nil
}

// UpdateMatch は試合の登録値(日時、コート、CD、対戦チーム)を fields で置き換える。スコアと状態は
// 変えない。チームが変わればトーナメント表の格子のチーム名も書き直す。
func UpdateMatch(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	fields MatchFields,
) (MatchRecord, error) {
	if err := fields.validate(); err != nil {
		return MatchRecord{}, err
	}
	var row Match
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		row, err = lockMatchInDivision(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := checkMatchRefs(ctx, tx, row.DivisionID, fields); err != nil {
			return err
		}
		if err := fields.apply(&row); err != nil {
			return err
		}
		// スコアと状態の列は書かない(試合更新関数だけが書く)
		columns := matchRegistrationColumns()
		err = tx.WithContext(ctx).Model(&row).Select(columns).Updates(&row).Error
		if err != nil {
			return writeError("update match", err)
		}
		return syncBracketSlots(ctx, tx, []int64{id})
	})
	if err != nil {
		return MatchRecord{}, err
	}
	return GetAdminMatch(ctx, db, id)
}

// matchRegistrationColumns は管理画面の試合の登録で書く列。スコアと状態(left_score / right_score /
// status / winner_side)は含めない(試合更新関数だけが書く。scoreview-design 8)。
func matchRegistrationColumns() []string {
	return []string{
		"group_id", "code", "match_date", "start_time", "field", "has_detail",
		"left_team_id", "right_team_id", "left_placeholder_code", "left_placeholder_outcome",
		"right_placeholder_code", "right_placeholder_outcome", "updated_at",
	}
}

// DeleteMatch は試合と得点経過を消す。
func DeleteMatch(ctx context.Context, db *gorm.DB, id int64) error {
	return deleteByID[Match](ctx, db, id, domain.ResourceMatch)
}

// GetAdminMatch は試合を返す。無ければ MATCH_ID の NotFound。
func GetAdminMatch(ctx context.Context, db *gorm.DB, id int64) (MatchRecord, error) {
	match, err := findByID[Match](ctx, db, id, domain.ResourceMatch)
	if err != nil {
		return MatchRecord{}, err
	}
	records, err := toMatchRecords(ctx, db, []Match{match})
	if err != nil {
		return MatchRecord{}, err
	}
	return records[0], nil
}

// EventRecord は管理画面の得点経過の 1 行。LeftScore と RightScore はその時点の累計。
type EventRecord struct {
	ID         int64
	ClientID   string
	Seq        int
	ElapsedSec int
	Kind       domain.EventKind
	Side       *domain.Side
	Scorer     *domain.PlayerRef
	Assist     *domain.PlayerRef
	Callahan   bool
	LeftScore  int
	RightScore int
	CreatedAt  time.Time
}

// PlayerStatFields は試合ごとの選手記録(出場、ブロック、グレイテスト)。行があることが出場を表す。
type PlayerStatFields struct {
	PlayerID  int64
	Side      domain.Side
	Blocks    int
	Greatests int
}

// MatchDetail はスコア入力画面の全データ。Rosters は左右のチームの選手とスタッフ(チームが未確定なら空)。
// StartedAt は経過時間が 0 秒だった時刻(最初の得点経過の記録時刻からその経過秒を引いたもの。
// 得点経過が無ければ nil)。ForfeitScore は大会の不戦勝の規定スコア(未設定なら nil)。
type MatchDetail struct {
	Match          MatchRecord
	InBracket      bool
	TournamentID   int64
	TournamentName string
	DivisionName   string
	GroupName      *string
	ForfeitScore   *domain.ForfeitScore
	StartedAt      *time.Time
	Events         []EventRecord
	PlayerStats    []PlayerStatFields
	Rosters        struct{ Left, Right []PlayerRecord }
}

// GetMatchDetail はスコア入力画面のデータを返す。無ければ MATCH_ID の NotFound。得点経過は時系列順。
func GetMatchDetail(ctx context.Context, db *gorm.DB, id int64) (MatchDetail, error) {
	match, err := findByID[Match](ctx, db, id, domain.ResourceMatch)
	if err != nil {
		return MatchDetail{}, err
	}
	records, err := toMatchRecords(ctx, db, []Match{match})
	if err != nil {
		return MatchDetail{}, err
	}
	detail := MatchDetail{Match: records[0], PlayerStats: []PlayerStatFields{}}
	if detail.InBracket, err = inBracket(ctx, db, match); err != nil {
		return MatchDetail{}, err
	}
	if err := fillMatchContext(ctx, db, match, &detail); err != nil {
		return MatchDetail{}, err
	}
	events, err := loadEvents(ctx, db, id)
	if err != nil {
		return MatchDetail{}, err
	}
	var stats []MatchPlayerStat
	if err := db.WithContext(ctx).
		Where(sqlByMatch, id).
		Order(sqlOrderPlayerID).
		Find(&stats).
		Error; err != nil {
		return MatchDetail{}, fmt.Errorf("load player stats of match %d: %w", id, err)
	}
	for _, stat := range stats {
		detail.PlayerStats = append(detail.PlayerStats, PlayerStatFields{
			PlayerID:  stat.PlayerID,
			Side:      stat.Side,
			Blocks:    stat.Blocks,
			Greatests: stat.Greatests,
		})
	}
	var playerIDs idSet
	for _, ev := range events {
		playerIDs.addPtr(ev.ScorerID)
		playerIDs.addPtr(ev.AssistID)
	}
	players, err := loadPlayers(ctx, db, playerIDs.list)
	if err != nil {
		return MatchDetail{}, err
	}
	scores := domain.RunningScores(toDomainEvents(events))
	detail.Events = make([]EventRecord, 0, len(events))
	for i, ev := range events {
		scorer, err := playerRefByID(players, ev.ScorerID)
		if err != nil {
			return MatchDetail{}, err
		}
		assist, err := playerRefByID(players, ev.AssistID)
		if err != nil {
			return MatchDetail{}, err
		}
		detail.Events = append(detail.Events, EventRecord{
			ID: ev.ID, ClientID: ev.ClientID, Seq: ev.Seq, ElapsedSec: ev.ElapsedSec, Kind: ev.Kind, Side: ev.Side,
			Scorer: scorer, Assist: assist, Callahan: ev.Callahan,
			LeftScore: scores[i].Left, RightScore: scores[i].Right, CreatedAt: ev.CreatedAt,
		})
	}
	if len(events) > 0 {
		started := events[0].CreatedAt.Add(-time.Duration(events[0].ElapsedSec) * time.Second)
		detail.StartedAt = &started
	}
	detail.Rosters.Left, detail.Rosters.Right = []PlayerRecord{}, []PlayerRecord{}
	if match.LeftTeamID != nil {
		if detail.Rosters.Left, err = listPlayersOf(ctx, db, *match.LeftTeamID); err != nil {
			return MatchDetail{}, err
		}
	}
	if match.RightTeamID != nil {
		if detail.Rosters.Right, err = listPlayersOf(ctx, db, *match.RightTeamID); err != nil {
			return MatchDetail{}, err
		}
	}
	return detail, nil
}

// fillMatchContext は試合の大会、ディビジョン、グループの名前と、不戦勝の規定スコア(大会、無ければ
// 大会の形の設定。regulationForfeitScore)を入れる(スコア入力画面とスコアシートの見出しに出す)。
func fillMatchContext(ctx context.Context, db *gorm.DB, match Match, detail *MatchDetail) error {
	division, err := findInternal[Division](ctx, db, match.DivisionID, "division")
	if err != nil {
		return err
	}
	tournament, err := findInternal[Tournament](ctx, db, division.TournamentID, "tournament")
	if err != nil {
		return err
	}
	detail.TournamentID, detail.TournamentName = tournament.ID, tournament.NameJa
	detail.DivisionName = division.NameJa
	if detail.ForfeitScore, err = regulationForfeitScore(ctx, db, division); err != nil {
		return err
	}
	if match.GroupID != nil {
		group, err := findInternal[Group](ctx, db, *match.GroupID, "group")
		if err != nil {
			return err
		}
		detail.GroupName = &group.Name
	}
	return nil
}

// loadEvents は試合の得点経過を時系列順(経過時間、同時刻は seq)で読む。
func loadEvents(ctx context.Context, db *gorm.DB, matchID int64) ([]MatchEvent, error) {
	var events []MatchEvent
	if err := db.WithContext(ctx).
		Where(sqlByMatch, matchID).
		Order(sqlOrderEvents).
		Find(&events).
		Error; err != nil {
		return nil, fmt.Errorf("load events of match %d: %w", matchID, err)
	}
	return events, nil
}

// inBracket は試合がトーナメント(bracket グループ)の試合かどうか。
func inBracket(ctx context.Context, db *gorm.DB, match Match) (bool, error) {
	if match.GroupID == nil {
		return false, nil
	}
	group, err := findInternal[Group](ctx, db, *match.GroupID, "group")
	if err != nil {
		return false, err
	}
	return group.Kind == domain.GroupBracket, nil
}

// sideTeam は試合の side のチーム ID。
func sideTeam(match Match, side domain.Side) *int64 {
	if side == domain.SideLeft {
		return match.LeftTeamID
	}
	return match.RightTeamID
}

// checkPlayersOnSide は players がすべて試合の side のチームの選手であることを確かめる。
func checkPlayersOnSide(
	ctx context.Context,
	db *gorm.DB,
	match Match,
	side domain.Side,
	field string,
	players []int64,
) error {
	if len(players) == 0 {
		return nil
	}
	teamID := sideTeam(match, side)
	if teamID == nil {
		return domain.Invalid(field, "the team of this side is not decided yet")
	}
	rows, err := loadPlayers(ctx, db, players)
	if err != nil {
		return err
	}
	for _, id := range players {
		if player, ok := rows[id]; !ok || player.TeamID != *teamID {
			return domain.Invalid(field, fmt.Sprintf("player %d is not on the %s team", id, side))
		}
	}
	return nil
}

// ReplaceMatchPlayerStats は試合の選手記録を stats で置き換える。選手は side のチームに属すること。
func ReplaceMatchPlayerStats(
	ctx context.Context,
	db *gorm.DB,
	matchID int64,
	stats []PlayerStatFields,
) (MatchDetail, error) {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		match, err := findByID[Match](ctx, tx, matchID, domain.ResourceMatch)
		if err != nil {
			return err
		}
		bySide := map[domain.Side][]int64{}
		for i, stat := range stats {
			field := fmt.Sprintf("stats[%d]", i)
			if stat.Side != domain.SideLeft && stat.Side != domain.SideRight {
				return domain.Invalid(field+".side", "must be left or right")
			}
			if stat.Blocks < 0 || stat.Greatests < 0 {
				return domain.Invalid(field, "blocks and greatests must be 0 or more")
			}
			if slices.ContainsFunc(
				stats[:i],
				func(prev PlayerStatFields) bool { return prev.PlayerID == stat.PlayerID },
			) {
				return domain.Invalid(field+".playerId", "is duplicated")
			}
			bySide[stat.Side] = append(bySide[stat.Side], stat.PlayerID)
		}
		for _, side := range []domain.Side{domain.SideLeft, domain.SideRight} {
			if err := checkPlayersOnSide(ctx, tx, match, side, "stats", bySide[side]); err != nil {
				return err
			}
		}
		if err := tx.Where(sqlByMatch, matchID).Delete(&MatchPlayerStat{}).Error; err != nil {
			return fmt.Errorf("clear player stats of match %d: %w", matchID, err)
		}
		if len(stats) == 0 {
			return nil
		}
		rows := make([]MatchPlayerStat, len(stats))
		for i, stat := range stats {
			rows[i] = MatchPlayerStat{
				MatchID:   matchID,
				PlayerID:  stat.PlayerID,
				Side:      stat.Side,
				Blocks:    stat.Blocks,
				Greatests: stat.Greatests,
			}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return writeError("save player stats", err)
		}
		return nil
	})
	if err != nil {
		return MatchDetail{}, err
	}
	return GetMatchDetail(ctx, db, matchID)
}
