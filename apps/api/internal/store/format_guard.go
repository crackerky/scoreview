package store

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 試合の結果を変える全経路(得点経過、状態、結果の種類、訂正)に共通の承認の検査
// (docs/format-engine.md「訂正の影響を先に見せる」)。既に結果のある下流の試合の対戦が変わる変更は、
// 運営者が下流の扱いを選んで承認しない限り保存しない。

// Downstream は、結果のある下流の試合の対戦が変わる変更を承認するときの下流の扱い。
type Downstream string

// Downstream の値。
const (
	// DownstreamNone は承認しない。下流に影響があれば ErrConflict
	DownstreamNone Downstream = ""
	// DownstreamKeep は下流の結果を残す。その試合は「食い違い」として進行画面に出て、結果は下流と
	// 最終順位に使わない。本部が対戦チームを差し替える(AlignMatchTeams)と、結果は新しいチームの
	// ものとして数える
	DownstreamKeep Downstream = "keep"
	// DownstreamReset は下流の試合の結果と得点経過を消して未試合に戻す。対戦チームは訂正後の順位で
	// 入れ直り、試合をやり直す
	DownstreamReset Downstream = "reset"
)

// ParseDownstream は API の値を Downstream にする。空は DownstreamNone。
func ParseDownstream(field, value string) (Downstream, error) {
	switch Downstream(value) {
	case DownstreamNone, DownstreamKeep, DownstreamReset:
		return Downstream(value), nil
	}
	return DownstreamNone, domain.Invalid(field, "must be keep or reset")
}

// decisive は試合の行がエンジンの数える結果を持つかどうか(終了か未消化)。
func decisive(m Match) bool {
	return m.Status == domain.MatchFinished || m.ResultType == domain.ResultUnplayed
}

// guardDownstream は試合の変更(before → after。after は保存済み)が、結果のある下流の試合の対戦を
// 新たに変えるかを調べ、変えるなら downstream に従う(None なら ErrConflict でロールバックさせる)。
// 呼ぶ前にディビジョンをロックしていること。
func guardDownstream(
	ctx context.Context,
	tx *gorm.DB,
	actor domain.Actor,
	before, after Match,
	downstream Downstream,
) error {
	if !decisive(before) && !decisive(after) {
		return nil // 試合中の得点などは下流を変えない
	}
	fs, ok, err := loadFormatState(ctx, tx, after.DivisionID)
	if err != nil || !ok {
		return err
	}
	if _, inPlan := fs.plan.MatchByCode(after.Code); !inPlan {
		return nil
	}
	var prev *format.Result
	if before.Status != domain.MatchScheduled {
		result := formatResultOf(before)
		prev = &result
	}
	impact, err := format.ComputeImpact(fs.plan, fs.cfg, fs.state.WithResult(after.Code, prev), fs.state)
	if err != nil {
		return err
	}
	if !impact.NeedsApproval {
		return nil
	}
	var codes []string
	for _, a := range impact.AffectedMatches {
		if a.NeedsApproval {
			codes = append(codes, a.Code)
		}
	}
	switch downstream {
	case DownstreamKeep:
		return nil
	case DownstreamReset:
		for _, code := range codes {
			if err := resetMatchResult(ctx, tx, actor, fs.matches[code].ID); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("%w: 既に結果のある試合(%s)の対戦が変わります。影響を確かめ、下流の結果の扱い"+
		"(未試合に戻す / 食い違いとして残す)を選んで承認してください", domain.ErrConflict, strings.Join(codes, "、"))
}
