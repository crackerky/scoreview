package store

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 大会形式の進行(プールの確定、段の確定、同順位の指定)と、確定や結果の変更で埋まった枠を
// 下流の試合へ反映する処理。試合のスコアと状態は変えない(それは matchupdate.go の関数だけが行う)。
// 変えるのは枠のチーム、試合の左右のチーム、グループのチームとプールの順位、チームの最終順位。

// withFormatProgress は advanceFromResult から呼ぶ。試合の結果が変わったディビジョンに大会形式が
// あれば枠と下流の試合を解決し直し、changed にチームを変えた試合の ID を足して返す。
func withFormatProgress(
	ctx context.Context,
	tx *gorm.DB,
	divisionID int64,
	changed []int64,
) ([]int64, error) {
	more, err := syncFormatProgress(ctx, tx, divisionID)
	if err != nil {
		return nil, err
	}
	return append(changed, more...), nil
}

// syncFormatProgress はディビジョンの大会形式を解決し、結果を行に書く。形の指定が無ければ何もしない。
// 既に始まった試合(試合中、終了)の左右は変えない(advanceFromResult と同じ方針。食い違いは進行画面で
// 「対戦チームが順位と食い違っている試合」として出す)。戻り値はチームを変えた試合の ID。
func syncFormatProgress(ctx context.Context, tx *gorm.DB, divisionID int64) ([]int64, error) {
	fs, ok, err := loadFormatState(ctx, tx, divisionID)
	if err != nil || !ok {
		return nil, err
	}
	if err := fs.writeSlots(ctx, tx); err != nil {
		return nil, err
	}
	changed, err := fs.writeMatchTeams(ctx, tx, false, nil)
	if err != nil {
		return nil, err
	}
	if err := fs.writeGroupTeams(ctx, tx); err != nil {
		return nil, err
	}
	if err := fs.writeFinalRanks(ctx, tx); err != nil {
		return nil, err
	}
	return changed, nil
}

// writeSlots は枠のチームを解決したチームにする。行は (グループ、位置) の順に書く(書く順序を
// 決めておき、行ロックの取り方を毎回同じにするため)。
func (fs *formatState) writeSlots(ctx context.Context, tx *gorm.DB) error {
	keys := make([]format.SlotKey, 0, len(fs.slots))
	for key := range fs.slots {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b format.SlotKey) int {
		if a.Group != b.Group {
			return a.Group - b.Group
		}
		return a.Position - b.Position
	})
	for _, key := range keys {
		slot := fs.slots[key]
		var team *int64
		if status := fs.res.Slots[key]; status.Resolved() {
			id := status.TeamID
			team = &id
		}
		if sameID(team, slot.TeamID) {
			continue
		}
		if err := tx.WithContext(ctx).Model(&GroupSlot{}).Where(sqlByID, slot.ID).
			Update("team_id", team).Error; err != nil {
			return writeError("update group slot", err)
		}
	}
	return nil
}

// sideTeams は試合の左右の計算上のチーム(枠のチームか、前の試合の勝者・敗者。決まらなければ nil)。
// 形の外の試合なら hasLeft と hasRight が偽。
func (fs *formatState) sideTeams(m Match) (left, right *int64, hasLeft, hasRight bool) {
	mt, ok := fs.res.Matches[m.Code]
	if !ok {
		return nil, nil, false, false
	}
	pick := func(status format.TeamStatus) *int64 {
		if !status.Resolved() {
			return nil
		}
		id := status.TeamID
		return &id
	}
	return pick(mt.Left), pick(mt.Right), true, true
}

// writeMatchTeams は試合のチームを計算上のチーム(sideTeams)にする。force が偽なら未試合の試合だけ
// (読んだ後に始まった試合も変えないよう、UPDATE に status の条件を付けて変えた行数を確かめる)、
// 真なら only の試合を状態によらず変える(運営者が食い違いを直すとき)。
func (fs *formatState) writeMatchTeams(
	ctx context.Context,
	tx *gorm.DB,
	force bool,
	only []int64,
) ([]int64, error) {
	var changed []int64
	codes := make([]string, 0, len(fs.matches))
	for code := range fs.matches {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	for _, code := range codes {
		m := fs.matches[code]
		if force && !slices.Contains(only, m.ID) {
			continue
		}
		if !force && m.Status != domain.MatchScheduled {
			continue
		}
		left, right, hasLeft, hasRight := fs.sideTeams(m)
		updates := map[string]any{}
		if hasLeft && !sameID(left, m.LeftTeamID) {
			updates["left_team_id"] = left
		}
		if hasRight && !sameID(right, m.RightTeamID) {
			updates["right_team_id"] = right
		}
		if len(updates) == 0 {
			continue
		}
		query := tx.WithContext(ctx).Model(&Match{})
		if force {
			query = query.Where(sqlByID, m.ID)
		} else {
			query = query.Where(sqlScheduledMatch, m.ID, domain.MatchScheduled)
		}
		written := query.Updates(updates)
		if written.Error != nil {
			return nil, writeError(fmt.Sprintf("set teams of match %s", code), written.Error)
		}
		if written.RowsAffected == 0 {
			continue // 読んだ後に始まった(食い違いとして進行画面に出る)
		}
		changed = append(changed, m.ID)
	}
	return changed, nil
}

// writeGroupTeams はグループのチーム(group_teams)を、埋まった枠のチームの位置順にする。プールは
// 1 試合以上の結果があれば、エンジンの順位(タイブレークの設定と運営者の指定を反映したもの)を
// rank_override に入れ、閲覧ページの順位表に同じ順位を出す。
func (fs *formatState) writeGroupTeams(ctx context.Context, tx *gorm.DB) error {
	for index := range fs.plan.Groups {
		g, ok := fs.groups[index]
		if !ok {
			continue
		}
		ranking := fs.res.Groups[index].Ranking
		played := false
		if ranking != nil {
			for _, row := range ranking.Rows {
				if row.Stats.Games > 0 {
					played = true
				}
			}
		}
		var want []GroupTeamFields
		for position := 1; position <= fs.plan.Groups[index].Size(); position++ {
			status := fs.res.Slots[format.SlotKey{Group: index, Position: position}]
			if !status.Resolved() {
				continue
			}
			member := GroupTeamFields{TeamID: status.TeamID}
			if played {
				if row, ok := ranking.RankOf(status.TeamID); ok {
					rank := row.Rank
					member.RankOverride = &rank
				}
			}
			want = append(want, member)
		}
		current, err := groupMembers(ctx, tx, g.ID)
		if err != nil {
			return err
		}
		if sameMembers(current, want) {
			continue
		}
		if err := replaceGroupTeams(ctx, tx, g.ID, want); err != nil {
			return err
		}
	}
	return nil
}

func sameMembers(current []GroupTeam, want []GroupTeamFields) bool {
	if len(current) != len(want) {
		return false
	}
	for i := range current {
		if current[i].TeamID != want[i].TeamID {
			return false
		}
		a, b := current[i].RankOverride, want[i].RankOverride
		if (a == nil) != (b == nil) || (a != nil && *a != *b) {
			return false
		}
	}
	return true
}

// writeFinalRanks はチームの最終順位(teams.final_rank)を、エンジンが決めた順位にする。決まって
// いないチームは空にする(大会形式のあるディビジョンでは最終順位をエンジンが持つ)。
func (fs *formatState) writeFinalRanks(ctx context.Context, tx *gorm.DB) error {
	ranks := map[int64]int{}
	for _, f := range fs.res.Final {
		ranks[f.TeamID] = f.Rank
	}
	var teams []Team
	if err := tx.WithContext(ctx).Where(sqlByDivision, fs.division.ID).Order(sqlOrderID).
		Find(&teams).Error; err != nil {
		return fmt.Errorf("load teams of division %d: %w", fs.division.ID, err)
	}
	for _, team := range teams {
		var want *int
		if rank, ok := ranks[team.ID]; ok {
			want = &rank
		}
		if (want == nil) == (team.FinalRank == nil) && (want == nil || *want == *team.FinalRank) {
			continue
		}
		if err := tx.WithContext(ctx).Model(&Team{}).Where(sqlByID, team.ID).
			Update("final_rank", want).Error; err != nil {
			return writeError("set final rank", err)
		}
	}
	return nil
}

// runFormatOp は大会形式の操作を 1 つのトランザクションで行い、続けて枠と下流の試合とトーナメント表を
// 更新する。先にディビジョンをロックする(locks.go。試合の結果の更新と直列にする)。
func runFormatOp(
	ctx context.Context,
	db *gorm.DB,
	divisionID int64,
	op func(tx *gorm.DB, fs *formatState) error,
) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockDivision(ctx, tx, divisionID); err != nil {
			return err
		}
		fs, ok, err := loadFormatState(ctx, tx, divisionID)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: この部門には大会の形が設定されていません", domain.ErrConflict)
		}
		if err := op(tx, fs); err != nil {
			return err
		}
		changed, err := syncFormatProgress(ctx, tx, divisionID)
		if err != nil {
			return err
		}
		return syncBracketSlots(ctx, tx, changed)
	})
}

// formatGroupOf はグループの行と、そのディビジョンを返す。大会形式のグループでなければ ErrConflict。
func formatGroupOf(ctx context.Context, db *gorm.DB, groupID int64) (formatGroupRow, error) {
	row, err := findByID[formatGroupRow](ctx, db, groupID, domain.ResourceGroup)
	if err != nil {
		return formatGroupRow{}, err
	}
	if row.FormatIndex == nil {
		return formatGroupRow{}, fmt.Errorf("%w: このグループは大会の形で作られていません", domain.ErrConflict)
	}
	return row, nil
}

// ConfirmGroup はプールの順位を確定し、その順位を出どころにする枠を埋める。結果待ちの試合が
// あれば、unplayed(未消化のまま確定)が真のときだけ確定する。未決の同順位があれば確定しない
// (運営者が DecideRank で順位を決めてから)。
func ConfirmGroup(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	groupID int64,
	unplayed bool,
	now time.Time,
) error {
	row, err := formatGroupOf(ctx, db, groupID)
	if err != nil {
		return err
	}
	return runFormatOp(ctx, db, row.DivisionID, func(tx *gorm.DB, fs *formatState) error {
		return confirmGroupIn(ctx, tx, fs, actor, *row.FormatIndex, unplayed, now)
	})
}

func confirmGroupIn(
	ctx context.Context,
	tx *gorm.DB,
	fs *formatState,
	actor domain.Actor,
	index int,
	unplayed bool,
	now time.Time,
) error {
	group := fs.plan.Groups[index]
	status := fs.res.Groups[index]
	if group.Kind != domain.GroupPool {
		return fmt.Errorf("%w: トーナメントは試合の終了で自動に進むので確定は要りません", domain.ErrConflict)
	}
	if len(status.Mismatches) > 0 && status.State != format.GroupConfirmed {
		return fmt.Errorf("%w: %s に対戦チームが順位と食い違っている試合(%s)があります。先に差し替えてください",
			domain.ErrConflict, group.Name, strings.Join(status.Mismatches, "、"))
	}
	switch status.State {
	case format.GroupConfirmed:
		return nil
	case format.GroupWaiting:
		return fmt.Errorf("%w: %s はまだチームがそろっていません", domain.ErrConflict, group.Name)
	case format.GroupTied:
		return fmt.Errorf("%w: %s に未決の同順位があります。先に順位を指定してください",
			domain.ErrConflict, group.Name)
	case format.GroupPlaying:
		if !unplayed {
			return fmt.Errorf("%w: %s に結果待ちの試合があります", domain.ErrConflict, group.Name)
		}
		if status.Ranking != nil && len(status.Ranking.Unresolved) > 0 {
			return fmt.Errorf("%w: %s で並んだチームがあります。残りの試合を未消化にしてから順位を指定して"+
				"ください", domain.ErrConflict, group.Name)
		}
	}
	err := tx.WithContext(ctx).Model(&formatGroupRow{}).Where(sqlByID, fs.groups[index].ID).
		Updates(map[string]any{"confirmed_at": now, "confirmed_unplayed": status.State == format.GroupPlaying}).Error
	if err != nil {
		return writeError("confirm group", err)
	}
	return auditFormatOp(ctx, tx, fs.division.TournamentID, actor, AuditGroupConfirm, nil,
		AuditFormat{Group: group.Name, Unplayed: status.State == format.GroupPlaying})
}

// ConfirmStage は段のプールをすべて確定し、段の全体順位(プールの大きさが不揃いなときの band_rank)を
// 使えるようにする。unplayed は ConfirmGroup と同じ。
func ConfirmStage(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	stageID int64,
	unplayed bool,
	now time.Time,
) error {
	stage, err := findByID[Stage](ctx, db, stageID, domain.ResourceGroup)
	if err != nil {
		return err
	}
	return runFormatOp(ctx, db, stage.DivisionID, func(tx *gorm.DB, fs *formatState) error {
		played := false
		for _, g := range fs.plan.StageGroups(stage.StageIndex) {
			if fs.plan.Groups[g].Kind != domain.GroupPool {
				continue
			}
			// 先に「未消化のまま確定」したプールも、段の未消化の印に数える
			if fs.res.Groups[g].State == format.GroupPlaying || fs.groups[g].ConfirmedUnplayed {
				played = true
			}
			if err := confirmGroupIn(ctx, tx, fs, actor, g, unplayed, now); err != nil {
				return err
			}
		}
		err := tx.WithContext(ctx).Model(&Stage{}).Where(sqlByID, stage.ID).
			Updates(map[string]any{"confirmed_at": now, "confirmed_unplayed": played}).Error
		if err != nil {
			return writeError("confirm stage", err)
		}
		return auditFormatOp(ctx, tx, fs.division.TournamentID, actor, AuditStageConfirm, nil,
			AuditFormat{Stage: stage.Name, Unplayed: played})
	})
}

// RankDecisionInput は未決の同順位の運営者の指定。Teams は未決の組のチームを上位から並べたもの、
// Reason は決め方(「ブリックマーク投げ」など)。
type RankDecisionInput struct {
	Teams  []int64
	Reason string
}

// DecideRank はプールの未決の同順位を運営者の指定で決める。指定はプールの全試合が終わった後だけ
// 受け付け(試合の途中の同点は未決ではない)、指定した時点の成績に結び付けて保存する(後で結果が
// 変われば使わない)。Teams が未決の組 1 つとちょうど同じチームでなければ検証エラー。
func DecideRank(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	groupID int64,
	input RankDecisionInput,
) error {
	if err := domain.CheckRequired("reason", strings.TrimSpace(input.Reason)); err != nil {
		return err
	}
	row, err := formatGroupOf(ctx, db, groupID)
	if err != nil {
		return err
	}
	return runFormatOp(ctx, db, row.DivisionID, func(tx *gorm.DB, fs *formatState) error {
		status := fs.res.Groups[*row.FormatIndex]
		if !status.Done || status.Ranking == nil {
			return fmt.Errorf("%w: %s の全試合が終わるまで順位は指定できません(残りの試合は結果を入れるか"+
				"未消化にしてください)", domain.ErrConflict, row.Name)
		}
		tie := tieIndex(status.Ranking.Unresolved, input.Teams)
		if tie < 0 {
			return domain.Invalid("teams", "must list exactly the teams of one unresolved tie")
		}
		err := saveDecision(ctx, tx, GroupRankDecision{GroupID: &row.ID}, actor, input,
			status.Ranking.UnresolvedBasis[tie])
		if err != nil {
			return err
		}
		return auditFormatOp(ctx, tx, fs.division.TournamentID, actor, AuditGroupRankDecision, nil,
			AuditFormat{Group: row.Name, Teams: input.Teams, Reason: input.Reason})
	})
}

// DecideBandRank は段の全体順位(rank 位どうしの並び)の未決を運営者の指定で決める。
func DecideBandRank(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	stageID int64,
	rank int,
	input RankDecisionInput,
) error {
	if err := domain.CheckRequired("reason", strings.TrimSpace(input.Reason)); err != nil {
		return err
	}
	if rank < 1 {
		return domain.Invalid("rank", "must be 1 or more")
	}
	stage, err := findByID[Stage](ctx, db, stageID, domain.ResourceGroup)
	if err != nil {
		return err
	}
	return runFormatOp(ctx, db, stage.DivisionID, func(tx *gorm.DB, fs *formatState) error {
		for _, g := range fs.plan.StageGroups(stage.StageIndex) {
			if fs.plan.Groups[g].Kind == domain.GroupPool && !fs.res.Groups[g].Done {
				return fmt.Errorf("%w: %s の全プールの試合が終わるまで全体順位は指定できません",
					domain.ErrConflict, stage.Name)
			}
		}
		tie, basis := fs.bandTie(stage.StageIndex, rank)
		if tieIndex([][]int64{tie}, input.Teams) < 0 {
			return domain.Invalid("teams", "must list exactly the tied teams of the stage rank")
		}
		err := saveDecision(ctx, tx, GroupRankDecision{StageID: &stage.ID, BandRank: &rank}, actor, input,
			basis)
		if err != nil {
			return err
		}
		return auditFormatOp(ctx, tx, fs.division.TournamentID, actor, AuditStageRankDecision, nil,
			AuditFormat{Stage: stage.Name, Rank: rank, Teams: input.Teams, Reason: input.Reason})
	})
}

func saveDecision(
	ctx context.Context,
	tx *gorm.DB,
	row GroupRankDecision,
	actor domain.Actor,
	input RankDecisionInput,
	basis []format.TeamValues,
) error {
	data, err := json.Marshal(input.Teams)
	if err != nil {
		return fmt.Errorf("encode decision: %w", err)
	}
	if row.Basis, err = json.Marshal(basis); err != nil {
		return fmt.Errorf("encode decision basis: %w", err)
	}
	user := actor.UserID
	row.TeamIDs, row.Reason, row.DecidedBy = data, strings.TrimSpace(input.Reason), &user
	if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
		return writeError("save rank decision", err)
	}
	return nil
}

// tieIndex は teams と重複なく同じチームの未決の組の添字。無ければ -1。
func tieIndex(ties [][]int64, teams []int64) int {
	for i, tie := range ties {
		if len(tie) != len(teams) {
			continue
		}
		seen := map[int64]bool{}
		all := true
		for _, team := range teams {
			if seen[team] || !slices.Contains(tie, team) {
				all = false
			}
			seen[team] = true
		}
		if all {
			return i
		}
	}
	return -1
}

// bandTie は段の全体順位で rank 位どうしを並べたときに未決のまま残るチームと、その成績(指定の保存用)。
func (fs *formatState) bandTie(stage, rank int) ([]int64, []format.TeamValues) {
	var entries []format.BandEntry
	for _, g := range fs.plan.StageGroups(stage) {
		ranking := fs.res.Groups[g].Ranking
		if ranking == nil {
			continue
		}
		if team, ok := ranking.TeamAt(rank); ok {
			row, _ := ranking.RankOf(team)
			entries = append(entries, format.BandEntry{TeamID: team, Group: g, Stats: row.Stats})
		}
	}
	key := format.BandKey{Stage: stage, Rank: rank}
	ranking := format.RankBand(entries, fs.cfg.Band, fs.state.BandDecisions[key])
	if len(ranking.Unresolved) == 0 {
		return nil, nil
	}
	return ranking.Unresolved[0], ranking.UnresolvedBasis[0]
}

// AlignMatchTeams は既に始まった試合の左右を、計算上のチームに合わせる(訂正で順位が変わり、
// 運営者が対戦チームの差し替えを選んだとき)。スコアと状態は変えない。試合の結果は新しいチームの
// ものになるので、勝者と敗者を次の試合へ流し直す(advanceFromResult)。
func AlignMatchTeams(ctx context.Context, db *gorm.DB, actor domain.Actor, matchID int64) error {
	match, err := findByID[Match](ctx, db, matchID, domain.ResourceMatch)
	if err != nil {
		return err
	}
	return runFormatOp(ctx, db, match.DivisionID, func(tx *gorm.DB, fs *formatState) error {
		changed, err := fs.writeMatchTeams(ctx, tx, true, []int64{matchID})
		if err != nil {
			return err
		}
		aligned, err := findInternal[Match](ctx, tx, matchID, "match")
		if err != nil {
			return err
		}
		advanced, err := advanceFromResult(ctx, tx, aligned)
		if err != nil {
			return err
		}
		if err := syncBracketSlots(ctx, tx, append(changed, advanced...)); err != nil {
			return err
		}
		id := matchID
		return auditFormatOp(ctx, tx, fs.division.TournamentID, actor, AuditMatchAlignTeams, &id,
			AuditFormat{})
	})
}
