package format_test

import (
	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

func intp(v int) *int { return &v }

// score は通常の終了試合の結果。
func score(left, right int) format.Result {
	return format.Result{Status: domain.MatchFinished, LeftScore: intp(left), RightScore: intp(right)}
}

// game はチーム left と right の終了試合。
func game(code string, left, right int64, ls, rs int) format.Game {
	r := score(ls, rs)
	return format.Game{Code: code, Left: left, Right: right, Result: &r}
}
