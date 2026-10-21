package store

import (
	"context"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 閲覧ページのグループ成績(プール)に出す本サイト独自の補足。参照サイトには無い表示なので、
// docs/requirements.md の「意図的な差分」に書いてある。

// PoolNotes はプールの補足。Reasons は同点を分けた根拠(「勝ち点同数、得失点差で上位」)、
// Conditions は進行中のプールの勝ち上がり条件(「次の A3 に勝てば 1 位確定」)。
type PoolNotes struct {
	Reasons    []TeamNote
	Conditions []TeamNote
}

// TeamNote はチームについての 1 文。
type TeamNote struct {
	Team domain.TeamRef
	Text string
}

// poolNotes はプールの補足を作る。大会形式のプールは形のタイブレークと運営者の指定で、それ以外は
// 現行方式(clone-spec C 2.1(b) の順位と同じ)で根拠を求める。順位を手で上書きしたプールには
// 根拠を出さない(計算の根拠と表示の順位が食い違うため)。
func poolNotes(
	ctx context.Context,
	db *gorm.DB,
	group Group,
	members []GroupTeam,
	matches []Match,
	teams map[int64]Team,
) (PoolNotes, error) {
	row, err := findInternal[formatGroupRow](ctx, db, group.ID, "group")
	if err != nil {
		return PoolNotes{}, err
	}
	if row.FormatIndex != nil {
		return formatPoolNotes(ctx, db, group.DivisionID, *row.FormatIndex)
	}
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if m.RankOverride != nil {
			return PoolNotes{}, nil
		}
		ids = append(ids, m.TeamID)
	}
	var games []format.Game
	remaining, played := 0, 0
	for _, m := range matches {
		if m.LeftTeamID == nil || m.RightTeamID == nil {
			continue
		}
		game := format.Game{Code: m.Code, Left: *m.LeftTeamID, Right: *m.RightTeamID}
		if m.Status == domain.MatchFinished {
			result := formatResultOf(m)
			game.Result = &result
			played++
		} else {
			remaining++
		}
		games = append(games, game)
	}
	ref := func(id int64) domain.TeamRef { return teamRef(teams[id]) }
	var out PoolNotes
	rule := format.CurrentRule(true)
	// 形の無いディビジョンの不戦勝は記録したスコアだけで数える(規定スコアで補わない)
	ranking := format.RankPool(format.RankInput{Teams: ids, Games: games, Rule: rule})
	if played > 0 {
		for _, r := range ranking.Rows {
			if text := r.Reason.Text(); text != "" {
				out.Reasons = append(out.Reasons, TeamNote{Team: ref(r.TeamID), Text: text})
			}
		}
	}
	if played > 0 && remaining > 0 {
		if q, err := format.Qualify(ids, games, rule); err == nil {
			for _, look := range q.Teams {
				if text := outlookText(look); text != "" {
					out.Conditions = append(out.Conditions, TeamNote{Team: ref(look.TeamID), Text: text})
				}
			}
		}
	}
	return out, nil
}

func formatPoolNotes(ctx context.Context, db *gorm.DB, divisionID int64, index int) (PoolNotes, error) {
	fs, ok, err := loadFormatState(ctx, db, divisionID)
	if err != nil || !ok {
		return PoolNotes{}, err
	}
	teams, err := fs.loadTeams(ctx, db)
	if err != nil {
		return PoolNotes{}, err
	}
	v := progressBuilder{fs: fs, teams: teams}
	gp := v.group(fs.plan.Groups[index])
	var out PoolNotes
	for _, r := range gp.Rows {
		if r.Reason != "" && r.Tally.Games > 0 {
			out.Reasons = append(out.Reasons, TeamNote{Team: r.Team, Text: r.Reason})
		}
	}
	for _, c := range gp.Conditions {
		out.Conditions = append(out.Conditions, TeamNote(c))
	}
	return out, nil
}
