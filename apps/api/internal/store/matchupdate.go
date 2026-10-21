package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
)

// 試合の更新関数群。試合のスコアと状態(matches.left_score / right_score / status / winner_side)を
// 変えるのはこのファイルの関数だけ(scoreview-design 8、docs/line-integration.md)。どれも 1 つの
// トランザクションで、得点経過の変更、スコアの再計算、状態の変更、後続試合の未確定チームの解決、
// トーナメント表の格子の更新までを行う。
//
// 取込ツールとシードはこの関数を通らず、行を作るときの初期値としてスコアと状態を書く(過去の大会を
// 一括で入れるので通知を積まない経路。LINE 連携で通知を足すときもこの区別を保つ)。

const (
	sqlMaxSeq            = "COALESCE(MAX(seq), 0)"
	sqlEventOfMatch      = "id = ? AND match_id = ?"
	sqlEndEventsOfMatch  = "match_id = ? AND kind = ?"
	sqlOtherEndEvents    = "match_id = ? AND kind = ? AND id <> ?"
	sqlDependentMatches  = "division_id = ? AND id <> ? AND (left_placeholder_code = ? OR right_placeholder_code = ?)"
	sqlSlotsOfMatches    = "match_id IN ?"
	sqlOrderSlots        = "row_index, col_index"
	sqlDistinctSlotGroup = "group_id"
)

// EventInput は追加する得点経過。ElapsedSec が nil なら、最初のイベントを記録した時刻からの経過秒を
// 使う(スコア入力画面で 1 タップで記録するため)。WinnerSide は kind=end で同点のトーナメントの試合を
// 終えるときの勝者。
type EventInput struct {
	domain.NewEvent
	ElapsedSec *int
	WinnerSide *domain.Side
}

// StatusInput は状態の変更。Score は得点経過の無い試合(詳細なしの試合)の結果を直接入れるときだけ使い、
// 得点経過がある試合ではスコアは経過から決まるので指定できない。
type StatusInput struct {
	Status     domain.MatchStatus
	WinnerSide *domain.Side
	Score      *domain.Score
}

// lockMatch は試合の行を FOR UPDATE で読む(同じ試合への同時の入力で seq とスコアが食い違わないように)。
func lockMatch(ctx context.Context, tx *gorm.DB, id int64) (Match, error) {
	var rows []Match
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(sqlByID, id).
		Limit(1).
		Find(&rows).
		Error
	if err != nil {
		return Match{}, fmt.Errorf("lock match %d: %w", id, err)
	}
	if len(rows) == 0 {
		return Match{}, domain.NotFound(domain.ResourceMatch, id)
	}
	return rows[0], nil
}

// AddMatchEvent は得点経過を 1 行足し、スコアと状態を更新する。未試合の試合は試合中になり、
// end を足すと終了になる。終了した試合には足せない。
func AddMatchEvent(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	matchID int64,
	input EventInput,
	now time.Time,
) (MatchDetail, error) {
	if err := input.Validate(); err != nil {
		return MatchDetail{}, err
	}
	if input.ElapsedSec != nil && *input.ElapsedSec < 0 {
		return MatchDetail{}, domain.Invalid("elapsedSec", "must be 0 or more")
	}
	if input.WinnerSide != nil && input.Kind != domain.EventEnd {
		return MatchDetail{}, domain.Invalid("winnerSide", "only for end")
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		match, err := lockMatch(ctx, tx, matchID)
		if err != nil {
			return err
		}
		next, err := domain.StatusAfterEvent(match.Status, input.Kind)
		if err != nil {
			return err
		}
		if next != match.Status {
			if err := domain.ValidTransition(match.Status, next, actor.Role); err != nil {
				return err
			}
		}
		if input.Kind == domain.EventGoal {
			var players []int64
			for _, id := range []*int64{input.ScorerID, input.AssistID} {
				if id != nil {
					players = append(players, *id)
				}
			}
			if err := checkPlayersOnSide(
				ctx,
				tx,
				match,
				*input.Side,
				"scorerId",
				players,
			); err != nil {
				return err
			}
		}
		event, err := newEventRow(ctx, tx, match.ID, input, now)
		if err != nil {
			return err
		}
		if err := tx.Create(&event).Error; err != nil {
			return writeError("add match event", err)
		}
		return recomputeMatch(ctx, tx, match, next, input.WinnerSide)
	})
	if err != nil {
		return MatchDetail{}, err
	}
	return GetMatchDetail(ctx, db, matchID)
}

func newEventRow(
	ctx context.Context,
	tx *gorm.DB,
	matchID int64,
	input EventInput,
	now time.Time,
) (MatchEvent, error) {
	var seq int
	if err := tx.WithContext(ctx).Model(&MatchEvent{}).Select(sqlMaxSeq).Where(sqlByMatch, matchID).
		Scan(&seq).Error; err != nil {
		return MatchEvent{}, fmt.Errorf("next seq of match %d: %w", matchID, err)
	}
	elapsed := 0
	if input.ElapsedSec != nil {
		elapsed = *input.ElapsedSec
	} else {
		var first []MatchEvent
		err := tx.WithContext(ctx).
			Where(sqlByMatch, matchID).
			Order(sqlOrderEvents).
			Limit(1).
			Find(&first).
			Error
		if err != nil {
			return MatchEvent{}, fmt.Errorf("first event of match %d: %w", matchID, err)
		}
		if len(first) > 0 {
			elapsed = first[0].ElapsedSec + domain.ElapsedSince(first[0].CreatedAt, now)
		}
	}
	return MatchEvent{
		MatchID:    matchID,
		Seq:        seq + 1,
		ElapsedSec: elapsed,
		Kind:       input.Kind,
		Side:       input.Side,
		ScorerID:   input.ScorerID,
		AssistID:   input.AssistID,
		Callahan:   input.Callahan,
		CreatedAt:  now,
	}, nil
}

// RemoveMatchEvent は得点経過を 1 行取り消し、スコアと状態を更新する。終了した試合で取り消せるのは
// end だけで、取り消すと試合中へ戻る(finished → live なので admin だけ)。
func RemoveMatchEvent(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	matchID, eventID int64,
) (MatchDetail, error) {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		match, err := lockMatch(ctx, tx, matchID)
		if err != nil {
			return err
		}
		var events []MatchEvent
		if err := tx.Where(sqlEventOfMatch, eventID, matchID).
			Limit(1).
			Find(&events).
			Error; err != nil {
			return fmt.Errorf("find event %d: %w", eventID, err)
		}
		if len(events) == 0 {
			return domain.NotFound(domain.ResourceEvent, eventID)
		}
		var otherEnds int64
		err = tx.Model(&MatchEvent{}).
			Where(sqlOtherEndEvents, matchID, domain.EventEnd, eventID).
			Count(&otherEnds).
			Error
		if err != nil {
			return fmt.Errorf("count end events of match %d: %w", matchID, err)
		}
		next, err := domain.StatusAfterEventRemoval(match.Status, events[0].Kind, otherEnds > 0)
		if err != nil {
			return err
		}
		if next != match.Status {
			if err := domain.ValidTransition(match.Status, next, actor.Role); err != nil {
				return err
			}
		}
		if err := tx.Where(sqlByID, eventID).Delete(&MatchEvent{}).Error; err != nil {
			return fmt.Errorf("delete event %d: %w", eventID, err)
		}
		return recomputeMatch(ctx, tx, match, next, match.WinnerSide)
	})
	if err != nil {
		return MatchDetail{}, err
	}
	return GetMatchDetail(ctx, db, matchID)
}

// ChangeMatchStatus は試合の状態を変える。終了にするときは勝者を決め(同点のトーナメントの試合は
// WinnerSide が必須)、終了から試合中へ戻すときは end の得点経過を消す(終了の取り消し)。
func ChangeMatchStatus(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	matchID int64,
	input StatusInput,
) (MatchDetail, error) {
	if input.Score != nil && (input.Score.Left < 0 || input.Score.Right < 0) {
		return MatchDetail{}, domain.Invalid("score", "must be 0 or more")
	}
	if input.WinnerSide != nil && input.Status != domain.MatchFinished {
		return MatchDetail{}, domain.Invalid("winnerSide", "only when finishing")
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		match, err := lockMatch(ctx, tx, matchID)
		if err != nil {
			return err
		}
		if err := domain.ValidTransition(match.Status, input.Status, actor.Role); err != nil {
			return err
		}
		if match.Status == domain.MatchFinished {
			if err := tx.Where(sqlEndEventsOfMatch, matchID, domain.EventEnd).
				Delete(&MatchEvent{}).
				Error; err != nil {
				return fmt.Errorf("remove end events of match %d: %w", matchID, err)
			}
		}
		events, err := loadEvents(ctx, tx, matchID)
		if err != nil {
			return err
		}
		if len(events) > 0 {
			if input.Score != nil {
				return domain.Invalid("score", "is derived from the events of this match")
			}
			return recomputeMatch(ctx, tx, match, input.Status, input.WinnerSide)
		}
		score := input.Score
		if score == nil && match.LeftScore != nil && match.RightScore != nil {
			score = &domain.Score{Left: *match.LeftScore, Right: *match.RightScore}
		}
		if score == nil {
			if input.Status == domain.MatchFinished {
				return domain.Invalid("score", "is required to finish a match without events")
			}
			score = &domain.Score{}
		}
		return setMatchState(ctx, tx, match, input.Status, *score, input.WinnerSide)
	})
	if err != nil {
		return MatchDetail{}, err
	}
	return GetMatchDetail(ctx, db, matchID)
}

// recomputeMatch は得点経過からスコアを計算し直して状態を書く。
func recomputeMatch(
	ctx context.Context,
	tx *gorm.DB,
	before Match,
	status domain.MatchStatus,
	winner *domain.Side,
) error {
	events, err := loadEvents(ctx, tx, before.ID)
	if err != nil {
		return err
	}
	return setMatchState(ctx, tx, before, status, domain.FinalScore(toDomainEvents(events)), winner)
}

// setMatchState は試合のスコア、状態、勝者を書き、結果に依存する後続の処理を同じトランザクションで行う。
// requestedWinner は終了にするときだけ使う(domain.DecideWinner)。
func setMatchState(
	ctx context.Context,
	tx *gorm.DB,
	before Match,
	status domain.MatchStatus,
	score domain.Score,
	requestedWinner *domain.Side,
) error {
	var winner *domain.Side
	if status == domain.MatchFinished {
		bracket, err := inBracket(ctx, tx, before)
		if err != nil {
			return err
		}
		if winner, err = domain.DecideWinner(score, bracket, requestedWinner); err != nil {
			return err
		}
	}
	after := before
	left, right := score.Left, score.Right
	after.LeftScore, after.RightScore, after.Status, after.WinnerSide = &left, &right, status, winner
	err := tx.WithContext(ctx).Model(&Match{}).Where(sqlByID, before.ID).Updates(map[string]any{
		"left_score": after.LeftScore, "right_score": after.RightScore,
		"status": after.Status, "winner_side": after.WinnerSide, "updated_at": time.Now(),
	}).Error
	if err != nil {
		return writeError("update match state", err)
	}

	changed, err := advanceFromResult(ctx, tx, after)
	if err != nil {
		return err
	}
	if err := syncBracketSlots(ctx, tx, append(changed, after.ID)); err != nil {
		return err
	}

	// ここで通知ジョブを積む(LINE 連携。docs/line-integration.md)。before.Status != after.Status の
	// とき(試合開始、試合終了)に、同じトランザクションで notification_jobs へ INSERT する。
	// トランザクションの外で配信すると、ロールバックした変更を通知してしまうため。
	return nil
}

// advanceFromResult は試合の結果を、それを参照する後続試合へ反映する。勝者や敗者を次の試合へ進める
// 処理はここに集める(後で「プール順位の確定 → 次のグループの枠を埋める」もここに足す)。
//
// 同じディビジョンで、左右の未確定チームの参照(placeholder_code)がこの試合の CD を指す試合の
// チームを、この試合の勝者または敗者にする。試合が終了していなければ(終了の取り消しを含む)空に戻す。
// 後続試合が既に始まっていれば変えない(試合中や終了した試合の対戦相手を差し替えない)。
// 戻り値はチームを変えた試合の ID。
func advanceFromResult(ctx context.Context, tx *gorm.DB, match Match) ([]int64, error) {
	if match.Code == "" {
		return nil, nil
	}
	var dependents []Match
	err := tx.WithContext(ctx).
		Where(sqlDependentMatches, match.DivisionID, match.ID, match.Code, match.Code).
		Order(sqlOrderID).
		Find(&dependents).
		Error
	if err != nil {
		return nil, fmt.Errorf("find matches waiting for %s: %w", match.Code, err)
	}
	result := toResult(match)
	var changed []int64
	for _, dep := range dependents {
		if dep.Status != domain.MatchScheduled {
			continue
		}
		updates := map[string]any{}
		resolve := func(column string, code *string, outcome *domain.Outcome, current *int64) {
			if code == nil || outcome == nil || *code != match.Code {
				return
			}
			team := domain.OutcomeTeam(result, *outcome)
			if !sameID(team, current) {
				updates[column] = team
			}
		}
		resolve("left_team_id", dep.LeftPlaceholderCode, dep.LeftPlaceholderOutcome, dep.LeftTeamID)
		resolve(
			"right_team_id",
			dep.RightPlaceholderCode,
			dep.RightPlaceholderOutcome,
			dep.RightTeamID,
		)
		if len(updates) == 0 {
			continue
		}
		if err := tx.WithContext(ctx).
			Model(&Match{}).
			Where(sqlByID, dep.ID).
			Updates(updates).
			Error; err != nil {
			return nil, writeError(fmt.Sprintf("advance %s to match %d", match.Code, dep.ID), err)
		}
		changed = append(changed, dep.ID)
	}
	return changed, nil
}

func sameID(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// syncBracketSlots は matchIDs の試合を参照するトーナメント表の格子のチーム名を書き直す。
func syncBracketSlots(ctx context.Context, tx *gorm.DB, matchIDs []int64) error {
	var groupIDs []int64
	err := tx.WithContext(ctx).Model(&BracketSlot{}).Distinct(sqlDistinctSlotGroup).
		Where(sqlSlotsOfMatches, matchIDs).Pluck("group_id", &groupIDs).Error
	if err != nil {
		return fmt.Errorf("find bracket groups of matches: %w", err)
	}
	for _, groupID := range groupIDs {
		if err := syncBracketGroup(ctx, tx, groupID); err != nil {
			return err
		}
	}
	return nil
}

// errSlotOutOfLayout は bracket_slots が格子の外を指していることを表す(生成後に格子を差し替えた等の不整合)。
var errSlotOutOfLayout = errors.New("bracket slot is outside of the layout")

// syncBracketGroup はグループの格子の、bracket_slots があるセルのチームを、試合の現在のチームと勝敗で
// 書き直す(left / right は試合のチーム、winner / loser は終了した試合の勝者と敗者)。
func syncBracketGroup(ctx context.Context, tx *gorm.DB, groupID int64) error {
	var slots []BracketSlot
	if err := tx.WithContext(ctx).
		Where(sqlByGroup, groupID).
		Order(sqlOrderSlots).
		Find(&slots).
		Error; err != nil {
		return fmt.Errorf("load bracket slots of group %d: %w", groupID, err)
	}
	if len(slots) == 0 {
		return nil
	}
	group, err := findInternal[Group](ctx, tx, groupID, "group")
	if err != nil {
		return err
	}
	layout, err := domain.ParseBracketLayout(group.BracketLayout)
	if err != nil {
		return fmt.Errorf("group %d: %w", groupID, err)
	}
	var ids idSet
	for _, slot := range slots {
		ids.add(slot.MatchID)
	}
	var matches []Match
	if err := tx.WithContext(ctx).Where(sqlByIDs, ids.list).Find(&matches).Error; err != nil {
		return fmt.Errorf("load matches of bracket %d: %w", groupID, err)
	}
	byID := make(map[int64]Match, len(matches))
	for _, match := range matches {
		byID[match.ID] = match
	}
	for _, slot := range slots {
		if slot.RowIndex >= len(layout.Rows) || slot.ColIndex >= layout.Columns {
			return fmt.Errorf(
				"group %d (%d,%d): %w",
				groupID,
				slot.RowIndex,
				slot.ColIndex,
				errSlotOutOfLayout,
			)
		}
		match := byID[slot.MatchID]
		var team *int64
		switch slot.Role {
		case domain.SlotLeft:
			team = match.LeftTeamID
		case domain.SlotRight:
			team = match.RightTeamID
		case domain.SlotWinner:
			team = domain.OutcomeTeam(toResult(match), domain.OutcomeWin)
		case domain.SlotLoser:
			team = domain.OutcomeTeam(toResult(match), domain.OutcomeLose)
		}
		layout.Rows[slot.RowIndex][slot.ColIndex].TeamID = team
	}
	data, err := domain.MarshalBracketLayout(layout)
	if err != nil {
		return fmt.Errorf("group %d: %w", groupID, err)
	}
	if err := tx.WithContext(ctx).
		Model(&Group{}).
		Where(sqlByID, groupID).
		Update("bracket_layout", data).
		Error; err != nil {
		return fmt.Errorf("save bracket layout of group %d: %w", groupID, err)
	}
	return nil
}
