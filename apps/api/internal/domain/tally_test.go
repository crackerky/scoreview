package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
)

func score(v int) *int { return &v }

func side(s domain.Side) *domain.Side { return &s }

func finished(left, right int64, ls, rs int) domain.Result {
	return domain.Result{
		LeftTeamID: left, RightTeamID: right,
		LeftScore: score(ls), RightScore: score(rs), Status: domain.MatchFinished,
	}
}

func TestTallyFor(t *testing.T) {
	const team = 1
	cases := []struct {
		name    string
		results []domain.Result
		want    domain.Tally
	}{
		{"no matches", nil, domain.Tally{}},
		{
			"win as left and loss as right",
			[]domain.Result{finished(team, 2, 9, 3), finished(3, team, 8, 2)},
			domain.Tally{Games: 2, Wins: 1, Losses: 1, GoalsFor: 11, GoalsAgainst: 11},
		},
		{
			"draw counts as draw, not win (clone-spec C 1.3)",
			[]domain.Result{finished(team, 2, 5, 5)},
			domain.Tally{Games: 1, Draws: 1, GoalsFor: 5, GoalsAgainst: 5},
		},
		{
			"explicit winner on tied score wins",
			[]domain.Result{func() domain.Result {
				r := finished(2, team, 4, 4)
				r.WinnerSide = side(domain.SideRight)
				return r
			}()},
			domain.Tally{Games: 1, Wins: 1, GoalsFor: 4, GoalsAgainst: 4},
		},
		{
			"scheduled and live matches are not counted",
			[]domain.Result{
				{LeftTeamID: team, RightTeamID: 2, Status: domain.MatchScheduled},
				{
					LeftTeamID:  team,
					RightTeamID: 2,
					LeftScore:   score(3),
					RightScore:  score(1),
					Status:      domain.MatchLive,
				},
			},
			domain.Tally{},
		},
		{
			"finished match without score is not counted",
			[]domain.Result{{LeftTeamID: team, RightTeamID: 2, Status: domain.MatchFinished}},
			domain.Tally{},
		},
		{
			"other teams' matches are ignored",
			[]domain.Result{finished(2, 3, 7, 0)},
			domain.Tally{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, domain.TallyFor(team, tc.results))
		})
	}
}

func TestTallyPointsAndDiff(t *testing.T) {
	tally := domain.Tally{Games: 3, Wins: 2, Draws: 1, GoalsFor: 30, GoalsAgainst: 12}
	assert.Equal(t, 7, tally.Points())
	assert.Equal(t, 18, tally.GoalDiff())
}
