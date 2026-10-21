package store

import (
	"context"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
)

// チームの棄権の一括処理(レビュー H3)。大会の途中でチームが帰ったとき、本部は残りの試合を 1 試合
// ずつ不戦勝にせず、「残り 2 試合(C3、G3)を相手の不戦勝にします」を確かめて一括で記録する。
// 試合の結果は matchupdate.go の writeBulkResult だけが書く(scoreview-design 8)。

// AuditMatchWithdraw は棄権の一括処理で記録した試合の監査ログの操作の種類。
const AuditMatchWithdraw = "match.withdraw"

// withdrawalNote は棄権の一括処理の監査ログの補足。
const withdrawalNote = "棄権の一括処理"

const (
	sqlTeamMatches     = "division_id = ? AND (left_team_id = ? OR right_team_id = ?)"
	sqlWithdrawalTeams = "team_id IN ?"
)

// TeamWithdrawal は team_withdrawals の行。
type TeamWithdrawal struct {
	TeamID      int64     `gorm:"column:team_id;primaryKey"`
	WithdrawnAt time.Time `gorm:"column:withdrawn_at"`
	WithdrawnBy *int64    `gorm:"column:withdrawn_by"`
}

// TableName は GORM に表名を伝える。
func (TeamWithdrawal) TableName() string { return "team_withdrawals" }

// WithdrawalMatch は棄権したチームの残りの試合。Result は記録する結果(相手の不戦勝。相手も棄権して
// いれば両チーム不戦敗)。Opponent が nil なら相手が決まっていない。
type WithdrawalMatch struct {
	ID       int64
	Code     string
	Date     *string
	Time     *string
	Field    *string
	Status   domain.MatchStatus
	Opponent *domain.TeamRef
	Result   domain.ResultType
	// winner は不戦勝で勝つ側(棄権したチームの相手の側)
	winner domain.Side
}

// Withdrawal はチームの棄権の確認画面のデータ。Targets は一括で記録する試合(試合前で相手が決まって
// いる)、Waiting は今は記録しない試合(相手が未定、試合中)。ForfeitScore は不戦勝の規定スコア
// (無ければ勝敗だけを記録する)。
type Withdrawal struct {
	Team         domain.TeamRef
	DivisionID   int64
	WithdrawnAt  *time.Time
	Targets      []WithdrawalMatch
	Waiting      []WithdrawalMatch
	ForfeitScore *domain.ForfeitScore
}

// GetWithdrawal はチームを棄権にしたときに記録する試合を返す(保存しない)。
func GetWithdrawal(ctx context.Context, db *gorm.DB, teamID int64) (Withdrawal, error) {
	team, err := findByID[Team](ctx, db, teamID, domain.ResourceTeam)
	if err != nil {
		return Withdrawal{}, err
	}
	return loadWithdrawal(ctx, db, team)
}

func loadWithdrawal(ctx context.Context, db *gorm.DB, team Team) (Withdrawal, error) {
	out := Withdrawal{Team: teamRef(team), DivisionID: team.DivisionID}
	withdrawn, err := withdrawnTeams(ctx, db, team.DivisionID)
	if err != nil {
		return Withdrawal{}, err
	}
	if at, ok := withdrawn[team.ID]; ok {
		out.WithdrawnAt = &at
	}
	division, err := findInternal[Division](ctx, db, team.DivisionID, "division")
	if err != nil {
		return Withdrawal{}, err
	}
	if out.ForfeitScore, err = regulationForfeitScore(ctx, db, division); err != nil {
		return Withdrawal{}, err
	}
	var matches []Match
	if err := db.WithContext(ctx).Where(sqlTeamMatches, team.DivisionID, team.ID, team.ID).
		Order(sqlOrderID).Find(&matches).Error; err != nil {
		return Withdrawal{}, fmt.Errorf("load matches of team %d: %w", team.ID, err)
	}
	slices.SortStableFunc(matches, compareMatchTime)
	teams, err := loadTeamsOfMatches(ctx, db, matches)
	if err != nil {
		return Withdrawal{}, err
	}
	for _, m := range matches {
		if decisive(m) {
			continue
		}
		opponent, winner := m.RightTeamID, domain.SideRight
		if m.RightTeamID != nil && *m.RightTeamID == team.ID {
			opponent, winner = m.LeftTeamID, domain.SideLeft
		}
		row := WithdrawalMatch{ID: m.ID, Code: m.Code, Date: formatDate(m), Time: formatClock(m), Field: m.Field,
			Status: m.Status, Result: domain.ResultForfeit, winner: winner}
		if opponent != nil {
			ref := teamRef(teams[*opponent])
			row.Opponent = &ref
			if _, both := withdrawn[*opponent]; both {
				row.Result = domain.ResultDoubleForfeit
			}
		}
		if m.Status == domain.MatchScheduled && opponent != nil {
			out.Targets = append(out.Targets, row)
		} else {
			out.Waiting = append(out.Waiting, row)
		}
	}
	return out, nil
}

func loadTeamsOfMatches(ctx context.Context, db *gorm.DB, matches []Match) (map[int64]Team, error) {
	var ids idSet
	for _, m := range matches {
		ids.addPtr(m.LeftTeamID)
		ids.addPtr(m.RightTeamID)
	}
	out := map[int64]Team{}
	if len(ids.list) == 0 {
		return out, nil
	}
	var rows []Team
	if err := db.WithContext(ctx).Where(sqlByIDs, ids.list).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load teams of matches: %w", err)
	}
	for _, t := range rows {
		out[t.ID] = t
	}
	return out, nil
}

// withdrawnTeams はディビジョンの棄権したチームと棄権した時刻。
func withdrawnTeams(ctx context.Context, db *gorm.DB, divisionID int64) (map[int64]time.Time, error) {
	var teamIDs []int64
	if err := db.WithContext(ctx).Model(&Team{}).Where(sqlByDivision, divisionID).
		Pluck("id", &teamIDs).Error; err != nil {
		return nil, fmt.Errorf("list teams of division %d: %w", divisionID, err)
	}
	out := map[int64]time.Time{}
	if len(teamIDs) == 0 {
		return out, nil
	}
	var rows []TeamWithdrawal
	if err := db.WithContext(ctx).Where(sqlWithdrawalTeams, teamIDs).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load withdrawals of division %d: %w", divisionID, err)
	}
	for _, row := range rows {
		out[row.TeamID] = row.WithdrawnAt
	}
	return out, nil
}

// WithdrawTeam はチームを棄権にし、残りの試合(GetWithdrawal の Targets)を相手の不戦勝として一括で
// 記録する。matchIDs は本部が確認画面で見た試合で、今の対象と違えば(この間に結果が入った、相手が
// 決まった)ErrConflict で何も記録しない。各試合は監査ログに「棄権の一括処理」として残る。
// 規定スコアが無ければ勝敗だけを記録する(スコアは本部が後で決める)。
func WithdrawTeam(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	teamID int64,
	matchIDs []int64,
	now time.Time,
) (Withdrawal, error) {
	team, err := findByID[Team](ctx, db, teamID, domain.ResourceTeam)
	if err != nil {
		return Withdrawal{}, err
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockDivision(ctx, tx, team.DivisionID); err != nil {
			return err
		}
		plan, err := loadWithdrawal(ctx, tx, team)
		if err != nil {
			return err
		}
		var want []int64
		for _, m := range plan.Targets {
			want = append(want, m.ID)
		}
		if !sameIDSet(want, matchIDs) {
			return fmt.Errorf("%w: 棄権で記録する試合が、確かめた後に変わりました。画面を開き直してください",
				domain.ErrConflict)
		}
		by := actor.UserID
		row := TeamWithdrawal{TeamID: team.ID, WithdrawnAt: now, WithdrawnBy: &by}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return writeError("save team withdrawal", err)
		}
		id := team.ID
		for _, m := range plan.Targets {
			result := bulkResult{resultType: m.Result, teamID: &id, action: AuditMatchWithdraw,
				note: withdrawalNote + "(" + team.NameShortJa + ")"}
			if m.Result == domain.ResultForfeit {
				winner := m.winner
				result.winner = &winner
			}
			if _, err := writeBulkResult(ctx, tx, actor, m.ID, result); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Withdrawal{}, err
	}
	return loadWithdrawal(ctx, db, team)
}

// sameIDSet は a と b が重複を除いて同じ ID の集まりかどうか。
func sameIDSet(a, b []int64) bool {
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(slices.Compact(x), slices.Compact(y))
}

// withdrawnTodo は、棄権したチームが入った試合前の試合(順位決定戦に後から入ったものなど)。
func (v progressBuilder) withdrawnTodo() []TodoItem {
	var out []TodoItem
	for _, m := range v.sortedMatches() {
		if m.Status != domain.MatchScheduled || m.LeftTeamID == nil || m.RightTeamID == nil {
			continue
		}
		for _, team := range []int64{*m.LeftTeamID, *m.RightTeamID} {
			if _, ok := v.withdrawn[team]; !ok {
				continue
			}
			id, teamID := m.ID, team
			out = append(out, TodoItem{Kind: TodoWithdrawn, MatchID: &id, TeamID: &teamID,
				Text: fmt.Sprintf("棄権した %s が %s(%s %s)に入っています。相手の不戦勝にしてください",
					v.ref(team).Name.Ja, m.Code, clockOf(m), courtLabel(m.Field))})
		}
	}
	return out
}
