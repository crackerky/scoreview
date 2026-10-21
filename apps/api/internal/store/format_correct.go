package store

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 結果の訂正と、その影響の事前表示(docs/format-engine.md「訂正の影響を先に見せる」)。

// CorrectionInput は試合の結果(得点経過の無い試合のスコアと、同点のときの勝者)。
type CorrectionInput struct {
	Score      domain.Score
	WinnerSide *domain.Side
}

// CorrectionImpact は訂正で変わるもの。Lines は運営者向けの文、Matches は対戦が変わる試合。
// NeedsApproval は既に結果のある試合の対戦が変わる(明示的な承認が要る)こと。
type CorrectionImpact struct {
	Lines         []string
	Matches       []ImpactMatch
	NeedsApproval bool
}

// ImpactMatch は対戦チームが変わる試合。
type ImpactMatch struct {
	MatchID     int64
	Code        string
	GroupName   string
	BeforeLeft  *domain.TeamRef
	BeforeRight *domain.TeamRef
	AfterLeft   *domain.TeamRef
	AfterRight  *domain.TeamRef
	HasResult   bool
}

func (in CorrectionInput) validate() error {
	if in.Score.Left < 0 || in.Score.Right < 0 {
		return domain.Invalid("score", "must be 0 or more")
	}
	return nil
}

// PreviewCorrection は試合の結果を input に変えたときの影響を返す(保存しない)。大会形式の無い
// ディビジョンの試合は影響なしとして返す。
func PreviewCorrection(
	ctx context.Context,
	db *gorm.DB,
	matchID int64,
	input CorrectionInput,
) (CorrectionImpact, error) {
	if err := input.validate(); err != nil {
		return CorrectionImpact{}, err
	}
	match, err := findByID[Match](ctx, db, matchID, domain.ResourceMatch)
	if err != nil {
		return CorrectionImpact{}, err
	}
	return correctionImpact(ctx, db, match, input)
}

func correctionImpact(ctx context.Context, db *gorm.DB, match Match, input CorrectionInput) (CorrectionImpact, error) {
	fs, ok, err := loadFormatState(ctx, db, match.DivisionID)
	if err != nil {
		return CorrectionImpact{}, err
	}
	if !ok {
		return CorrectionImpact{Lines: []string{"この部門には大会の形が無いので、順位と組み合わせは自動では変わりません"}}, nil
	}
	if _, inPlan := fs.plan.MatchByCode(match.Code); !inPlan {
		return CorrectionImpact{Lines: []string{"この試合は大会の形の外の試合です"}}, nil
	}
	left, right := input.Score.Left, input.Score.Right
	result := formatResultOf(match)
	result.Status, result.Type, result.LeftScore, result.RightScore, result.Winner =
		domain.MatchFinished, format.ResultNormal, &left, &right, input.WinnerSide
	impact, err := format.ImpactOfResult(fs.plan, fs.cfg, fs.state, match.Code, result)
	if err != nil {
		return CorrectionImpact{}, err
	}
	teams, err := fs.loadTeams(ctx, db)
	if err != nil {
		return CorrectionImpact{}, err
	}
	v := progressBuilder{fs: fs, teams: teams}
	return v.describeImpact(impact), nil
}

func (v progressBuilder) teamName(id int64) string {
	if id == 0 {
		return "未定"
	}
	return v.ref(id).Name.Ja
}

func (v progressBuilder) optRef(id int64) *domain.TeamRef {
	if id == 0 {
		return nil
	}
	ref := v.ref(id)
	return &ref
}

func (v progressBuilder) describeImpact(impact *format.Impact) CorrectionImpact {
	fs := v.fs
	out := CorrectionImpact{NeedsApproval: impact.NeedsApproval}
	byGroup := map[int][]format.RankChange{}
	var order []int
	for _, c := range impact.RankChanges {
		if c.Before == 0 || c.After == 0 {
			continue // 枠が埋まる前後の変化は順位の入れ替わりではない
		}
		if _, seen := byGroup[c.Group]; !seen {
			order = append(order, c.Group)
		}
		byGroup[c.Group] = append(byGroup[c.Group], c)
	}
	for _, g := range order {
		changes := byGroup[g]
		name := format.OperatorName(fs.plan.Groups[g])
		if len(changes) == 2 && changes[0].Before == changes[1].After && changes[0].After == changes[1].Before {
			low, high := min(changes[0].Before, changes[0].After), max(changes[0].Before, changes[0].After)
			out.Lines = append(out.Lines, fmt.Sprintf("%sの %d 位と %d 位が入れ替わります(%s と %s)", spaced(name), low,
				high, v.teamName(changes[0].TeamID), v.teamName(changes[1].TeamID)))
			continue
		}
		var parts []string
		for _, c := range changes {
			parts = append(parts, fmt.Sprintf("%s が %d 位 → %d 位", v.teamName(c.TeamID), c.Before, c.After))
		}
		out.Lines = append(out.Lines, fmt.Sprintf("%sの順位が変わります(%s)", spaced(name), strings.Join(parts, "、")))
	}
	for _, a := range impact.AffectedMatches {
		m, _ := fs.plan.MatchByCode(a.Code)
		group := fs.plan.Groups[m.Group]
		stored := fs.matches[a.Code]
		line := fmt.Sprintf("%sの %s の対戦が「%s 対 %s」から「%s 対 %s」に変わります",
			spaced(format.OperatorName(group)), a.Code,
			v.teamName(a.BeforeLeft), v.teamName(a.BeforeRight), v.teamName(a.AfterLeft), v.teamName(a.AfterRight))
		if a.HasResult {
			line += "(この試合は既に結果があります)"
		}
		out.Lines = append(out.Lines, line)
		out.Matches = append(out.Matches, ImpactMatch{MatchID: stored.ID, Code: a.Code, GroupName: group.Name,
			BeforeLeft: v.optRef(a.BeforeLeft), BeforeRight: v.optRef(a.BeforeRight),
			AfterLeft: v.optRef(a.AfterLeft), AfterRight: v.optRef(a.AfterRight), HasResult: a.HasResult})
	}
	if len(out.Lines) == 0 {
		out.Lines = []string{"順位と次の組み合わせは変わりません"}
	}
	return out
}

// ApplyCorrection は得点経過の無い試合の結果を input にする(未試合や試合中なら終了にし、終了済みなら
// 結果を直す)。既に結果のある下流の試合の対戦が変わるなら、downstream で下流の扱いを選んだときだけ
// 保存する(DownstreamNone なら ErrConflict。format_guard.go)。下流の試合のうち未試合のものは対戦が
// 自動で変わる。戻り値は保存前に計算した影響。
func ApplyCorrection(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	matchID int64,
	input CorrectionInput,
	downstream Downstream,
) (CorrectionImpact, error) {
	if err := input.validate(); err != nil {
		return CorrectionImpact{}, err
	}
	var impact CorrectionImpact
	_, err := runApprovedMatchOp(ctx, db, actor, matchID, nil, AuditMatchStatus, downstream,
		func(tx *gorm.DB, match Match) (matchOpChange, error) {
			events, err := loadEvents(ctx, tx, matchID)
			if err != nil {
				return matchOpChange{}, err
			}
			if len(events) > 0 {
				return matchOpChange{}, domain.Invalid("score",
					"得点経過のある試合は、スコア入力画面で得点経過を直してください")
			}
			if impact, err = correctionImpact(ctx, tx, match, input); err != nil {
				return matchOpChange{}, err
			}
			current := match
			if match.Status == domain.MatchFinished {
				if err := changeStatus(ctx, tx, actor, match, StatusInput{Status: domain.MatchLive}); err != nil {
					return matchOpChange{}, err
				}
				if current, err = findInternal[Match](ctx, tx, matchID, "match"); err != nil {
					return matchOpChange{}, err
				}
			}
			score := input.Score
			return matchOpChange{}, changeStatus(ctx, tx, actor, current,
				StatusInput{Status: domain.MatchFinished, WinnerSide: input.WinnerSide, Score: &score})
		})
	return impact, err
}
