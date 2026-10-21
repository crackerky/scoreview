package store

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 大会の締め(レビュー H10)。時間切れで残った試合をまとめて未消化にし、プールと段を確定し、決まら
// なかった順位(未消化で終わった順位決定戦、未決の同順位)を同順位で提案して、本部が確かめたうえで
// 最終順位に書き、閲覧ページに公開する。試合の結果は matchupdate.go の writeBulkResult だけが書く。

// 監査ログの操作の種類と補足。
const (
	AuditDivisionClose = "division.close"
	closingNote        = "大会の締め(未消化)"
	timeUpNote         = "時間切れで未消化のまま確定"
)

// DivisionClosing は division_closings の行。Ranks はエンジンが決めなかったチームに本部が承認した順位。
type DivisionClosing struct {
	DivisionID int64     `gorm:"column:division_id;primaryKey"`
	ClosedAt   time.Time `gorm:"column:closed_at"`
	ClosedBy   *int64    `gorm:"column:closed_by"`
	Ranks      []byte    `gorm:"column:ranks;type:jsonb"`
}

// TableName は GORM に表名を伝える。
func (DivisionClosing) TableName() string { return "division_closings" }

// closedRank は division_closings.ranks の 1 件。
type closedRank struct {
	Team int64 `json:"team"`
	Rank int   `json:"rank"`
}

// ClosingRank は締めた後の最終順位の 1 行。Proposed はエンジンでは決まらず、締めで同順位にした順位。
type ClosingRank struct {
	Rank     int
	Team     domain.TeamRef
	Tied     bool
	Proposed bool
}

// Closing は「大会を締める」の確認画面のデータ。Remaining は未消化にする試合(相手が未定の試合を
// 含む)、Ranks は締めた後の最終順位、Ties は同順位の説明(「9〜12 位は 9 位タイ(…)」)。
type Closing struct {
	Remaining []ProgressMatch
	Ranks     []ClosingRank
	Ties      []string
	ClosedAt  *time.Time
	Published bool
}

// remainingCodes は結果の無い(終了でも未消化でもない)形の試合の CD(時刻の順)。
func (fs *formatState) remainingMatches() []Match {
	var out []Match
	for _, m := range fs.matches {
		if !decisive(m) {
			out = append(out, m)
		}
	}
	slices.SortStableFunc(out, func(a, b Match) int {
		if c := compareMatchTime(a, b); c != 0 {
			return c
		}
		return cmp.Compare(a.Code, b.Code)
	})
	return out
}

// closingResolution は残りの試合をすべて未消化にし、全プールと全段を確定したときの解決結果
// (確認画面の提案。CloseDivision が実際に書いた後の状態と同じになる)。
func (fs *formatState) closingResolution() (*format.Resolution, error) {
	st := fs.buildState()
	for code, m := range fs.matches {
		if !decisive(m) {
			st.Results[code] = format.Result{Status: domain.MatchFinished, Type: format.ResultUnplayed}
		}
	}
	for i, g := range fs.plan.Groups {
		if g.Kind == domain.GroupPool {
			st.ConfirmedGroups[i] = true
		}
	}
	for i := range fs.plan.Stages {
		st.ConfirmedStages[i] = true
	}
	res, err := format.Resolve(fs.plan, fs.cfg, st)
	if err != nil {
		return nil, fmt.Errorf("resolve closing of division %d: %w", fs.division.ID, err)
	}
	return res, nil
}

type rankRow struct {
	team     int64
	rank     int
	tied     bool
	proposed bool
}

// closingRanks は解決結果から最終順位を作る。エンジンの順位があるチームはそのまま、未消化で終わった
// 順位決定戦のチームは、その試合から先で取りえた最も上の順位で並べる(ReachablePlaces)。どの
// 順位決定戦にも進めなかったチーム(未決の同順位で枠が埋まらなかった)は、最後の順位の次で並べる。
func (fs *formatState) closingRanks(res *format.Resolution) []rankRow {
	var rows []rankRow
	ranked := map[int64]bool{}
	for _, f := range res.Final {
		rows = append(rows, rankRow{team: f.TeamID, rank: f.Rank, tied: f.Tied})
		ranked[f.TeamID] = true
	}
	for _, g := range fs.plan.Groups {
		if !g.Final || g.Kind != domain.GroupBracket {
			continue
		}
		for _, slot := range g.Slots {
			st := res.Slots[format.SlotKey{Group: g.Index, Position: slot.Position}]
			if !st.Resolved() || ranked[st.TeamID] {
				continue
			}
			last := ""
			for _, m := range fs.plan.GroupMatches(g.Index) {
				mt := res.Matches[m.Code]
				if mt.Left.TeamID == st.TeamID || mt.Right.TeamID == st.TeamID {
					last = m.Code
				}
			}
			places := fs.plan.ReachablePlaces(last)
			if len(places) == 0 {
				continue
			}
			rows = append(rows, rankRow{team: st.TeamID, rank: g.RankOffset + places[0], proposed: true})
			ranked[st.TeamID] = true
		}
	}
	next := 1
	for _, r := range rows {
		next = max(next, r.rank+1)
	}
	byRank := map[int]int{}
	for _, r := range rows {
		byRank[r.rank]++
	}
	for rank, count := range byRank {
		next = max(next, rank+count)
	}
	for _, team := range fs.state.Seeds {
		if team != 0 && !ranked[team] {
			rows = append(rows, rankRow{team: team, rank: next, proposed: true})
		}
	}
	counts := map[int]int{}
	for _, r := range rows {
		counts[r.rank]++
	}
	for i := range rows {
		rows[i].tied = rows[i].tied || counts[rows[i].rank] > 1
	}
	slices.SortStableFunc(rows, func(a, b rankRow) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		return cmp.Compare(a.team, b.team)
	})
	return rows
}

// tieTexts は同順位の説明「9〜12 位は 9 位タイ(T03、T06、T09、T12)」。
func (v progressBuilder) tieTexts(rows []rankRow) []string {
	var out []string
	for i := 0; i < len(rows); {
		j := i
		for j < len(rows) && rows[j].rank == rows[i].rank {
			j++
		}
		if j-i > 1 {
			names := make([]string, 0, j-i)
			for _, r := range rows[i:j] {
				names = append(names, v.ref(r.team).Name.Ja)
			}
			out = append(out, fmt.Sprintf("%d〜%d 位は %d 位タイ(%s)", rows[i].rank, rows[i].rank+j-i-1,
				rows[i].rank, strings.Join(names, "、")))
		}
		i = j
	}
	return out
}

// GetClosing は「大会を締める」の確認画面のデータを返す(保存しない)。
func GetClosing(ctx context.Context, db *gorm.DB, divisionID int64) (Closing, error) {
	fs, ok, err := loadFormatState(ctx, db, divisionID)
	if err != nil {
		return Closing{}, err
	}
	if !ok {
		return Closing{}, fmt.Errorf("%w: この部門には大会の形が設定されていません", domain.ErrConflict)
	}
	return closingView(ctx, db, fs)
}

func closingView(ctx context.Context, db *gorm.DB, fs *formatState) (Closing, error) {
	teams, err := fs.loadTeams(ctx, db)
	if err != nil {
		return Closing{}, err
	}
	v := progressBuilder{fs: fs, teams: teams}
	res, err := fs.closingResolution()
	if err != nil {
		return Closing{}, err
	}
	out := Closing{Published: fs.division.StandingsPublished}
	for _, m := range fs.remainingMatches() {
		out.Remaining = append(out.Remaining, v.match(m))
	}
	rows := fs.closingRanks(res)
	for _, r := range rows {
		out.Ranks = append(out.Ranks, ClosingRank{Rank: r.rank, Team: v.ref(r.team), Tied: r.tied,
			Proposed: r.proposed})
	}
	out.Ties = v.tieTexts(rows)
	if fs.closing != nil {
		at := fs.closing.ClosedAt
		out.ClosedAt = &at
	}
	return out, nil
}

// CloseDivision は部門を締める。matchIDs は本部が確認画面で見た未消化にする試合で、今の残りの試合と
// 違えば ErrConflict で何もしない。段の順に、残りの試合を未消化にし(相手が未定の試合も)、プールと
// 段を確定する(未決の同順位があるプールも同順位のまま確定する)。最後に、エンジンが決めなかった
// チームの順位(確認画面の提案と同じ)を division_closings に保存して最終順位に書き、publish なら
// 閲覧ページに公開する(standings_published)。
func CloseDivision(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	divisionID int64,
	matchIDs []int64,
	publish bool,
	now time.Time,
) (Closing, error) {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockDivision(ctx, tx, divisionID); err != nil {
			return err
		}
		fs, err := mustFormatState(ctx, tx, divisionID)
		if err != nil {
			return err
		}
		var remaining []int64
		for _, m := range fs.remainingMatches() {
			remaining = append(remaining, m.ID)
		}
		if !sameIDSet(remaining, matchIDs) {
			return fmt.Errorf("%w: 未消化にする試合が、確かめた後に変わりました。画面を開き直してください",
				domain.ErrConflict)
		}
		for stage := range fs.plan.Stages {
			if err := closeStage(ctx, tx, actor, divisionID, stage, now); err != nil {
				return err
			}
		}
		if fs, err = mustFormatState(ctx, tx, divisionID); err != nil {
			return err
		}
		res, err := fs.closingResolution()
		if err != nil {
			return err
		}
		if err := saveClosing(ctx, tx, actor, divisionID, fs.closingRanks(res), now); err != nil {
			return err
		}
		changed, err := syncFormatProgress(ctx, tx, divisionID)
		if err != nil {
			return err
		}
		if err := syncBracketSlots(ctx, tx, changed); err != nil {
			return err
		}
		if publish {
			if err := tx.WithContext(ctx).Model(&Division{}).Where(sqlByID, divisionID).
				Update("standings_published", true).Error; err != nil {
				return writeError("publish standings", err)
			}
		}
		return auditFormatOp(ctx, tx, fs.division.TournamentID, actor, AuditDivisionClose, nil,
			AuditFormat{Unplayed: len(remaining) > 0, Published: publish})
	})
	if err != nil {
		return Closing{}, err
	}
	return GetClosing(ctx, db, divisionID)
}

func mustFormatState(ctx context.Context, tx *gorm.DB, divisionID int64) (*formatState, error) {
	fs, ok, err := loadFormatState(ctx, tx, divisionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: この部門には大会の形が設定されていません", domain.ErrConflict)
	}
	return fs, nil
}

// closeStage は段の残りの試合を未消化にし、段のプールと段を確定する。
func closeStage(
	ctx context.Context,
	tx *gorm.DB,
	actor domain.Actor,
	divisionID int64,
	stage int,
	now time.Time,
) error {
	fs, err := mustFormatState(ctx, tx, divisionID)
	if err != nil {
		return err
	}
	for _, g := range fs.plan.StageGroups(stage) {
		if err := unplayedGroupMatches(ctx, tx, fs, actor, g, closingNote); err != nil {
			return err
		}
	}
	if fs, err = mustFormatState(ctx, tx, divisionID); err != nil {
		return err
	}
	for _, g := range fs.plan.StageGroups(stage) {
		row := fs.groups[g]
		if fs.plan.Groups[g].Kind != domain.GroupPool || row.ConfirmedAt != nil {
			continue
		}
		if err := tx.WithContext(ctx).Model(&formatGroupRow{}).Where(sqlByID, row.ID).
			Updates(map[string]any{"confirmed_at": now, "confirmed_unplayed": true}).Error; err != nil {
			return writeError("confirm group on closing", err)
		}
	}
	if row, ok := fs.stages[stage]; ok && row.ConfirmedAt == nil {
		if err := tx.WithContext(ctx).Model(&Stage{}).Where(sqlByID, row.ID).
			Updates(map[string]any{"confirmed_at": now, "confirmed_unplayed": true}).Error; err != nil {
			return writeError("confirm stage on closing", err)
		}
	}
	changed, err := syncFormatProgress(ctx, tx, divisionID)
	if err != nil {
		return err
	}
	return syncBracketSlots(ctx, tx, changed)
}

// unplayedGroupMatches はグループの結果の無い試合を、上の回戦から順に未消化にする。
func unplayedGroupMatches(
	ctx context.Context,
	tx *gorm.DB,
	fs *formatState,
	actor domain.Actor,
	group int,
	note string,
) error {
	for _, m := range fs.plan.GroupMatches(group) {
		stored, ok := fs.matches[m.Code]
		if !ok || decisive(stored) {
			continue
		}
		_, err := writeBulkResult(ctx, tx, actor, stored.ID, bulkResult{resultType: domain.ResultUnplayed,
			action: AuditMatchResult, note: note})
		if err != nil {
			return err
		}
	}
	return nil
}

func saveClosing(
	ctx context.Context,
	tx *gorm.DB,
	actor domain.Actor,
	divisionID int64,
	rows []rankRow,
	now time.Time,
) error {
	ranks := []closedRank{}
	for _, r := range rows {
		if r.proposed {
			ranks = append(ranks, closedRank{Team: r.team, Rank: r.rank})
		}
	}
	data, err := json.Marshal(ranks)
	if err != nil {
		return fmt.Errorf("encode closing ranks: %w", err)
	}
	by := actor.UserID
	row := DivisionClosing{DivisionID: divisionID, ClosedAt: now, ClosedBy: &by, Ranks: data}
	err = tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "division_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"closed_at", "closed_by", "ranks"}),
	}).Create(&row).Error
	if err != nil {
		return writeError("save division closing", err)
	}
	return nil
}

// loadClosing は部門の締めの記録と、締めで決めた順位(チーム → 順位)。締めていなければ nil。
func loadClosing(ctx context.Context, db *gorm.DB, divisionID int64) (*DivisionClosing, map[int64]int, error) {
	var rows []DivisionClosing
	if err := db.WithContext(ctx).Where(sqlByDivision, divisionID).Limit(1).Find(&rows).Error; err != nil {
		return nil, nil, fmt.Errorf("load closing of division %d: %w", divisionID, err)
	}
	if len(rows) == 0 {
		return nil, map[int64]int{}, nil
	}
	var ranks []closedRank
	// 書き込みは saveClosing だけで、JSON の形は決まっている
	if err := json.Unmarshal(rows[0].Ranks, &ranks); err != nil {
		return nil, nil, fmt.Errorf("decode closing of division %d: %w", divisionID, err)
	}
	out := make(map[int64]int, len(ranks))
	for _, r := range ranks {
		out[r.Team] = r.Rank
	}
	return &rows[0], out, nil
}
