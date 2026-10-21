package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
)

func withType(r domain.Result, kind domain.ResultType, winner *domain.Side) domain.Result {
	r.ResultType, r.WinnerSide = kind, winner
	return r
}

func noScore(left, right int64) domain.Result {
	return domain.Result{LeftTeamID: left, RightTeamID: right, Status: domain.MatchFinished}
}

// 結果の種類ごとの集計(docs/format-engine.md「試合の結果の種類」)。
func TestTallyByResultType(t *testing.T) {
	const team, other = 1, 2
	cases := []struct {
		name     string
		result   domain.Result
		own, opp domain.Tally
	}{
		{
			"forfeit is a win and a loss with the forfeit score",
			withType(finished(team, other, 15, 0), domain.ResultForfeit, side(domain.SideLeft)),
			domain.Tally{Games: 1, Wins: 1, GoalsFor: 15},
			domain.Tally{Games: 1, Losses: 1, GoalsAgainst: 15},
		},
		{
			"forfeit without a score counts only the winner and loser",
			withType(noScore(team, other), domain.ResultForfeit, side(domain.SideLeft)),
			domain.Tally{Games: 1, Wins: 1},
			domain.Tally{Games: 1, Losses: 1},
		},
		{
			"double forfeit is a loss for both teams without a score",
			withType(noScore(team, other), domain.ResultDoubleForfeit, nil),
			domain.Tally{Games: 1, Losses: 1},
			domain.Tally{Games: 1, Losses: 1},
		},
		{
			"no contest counts only the winner and loser",
			withType(noScore(team, other), domain.ResultNoContest, side(domain.SideRight)),
			domain.Tally{Games: 1, Losses: 1},
			domain.Tally{Games: 1, Wins: 1},
		},
		{
			"abandoned counts the score at that time",
			withType(finished(team, other, 6, 4), domain.ResultAbandoned, nil),
			domain.Tally{Games: 1, Wins: 1, GoalsFor: 6, GoalsAgainst: 4},
			domain.Tally{Games: 1, Losses: 1, GoalsFor: 4, GoalsAgainst: 6},
		},
		{
			"unplayed is not counted even with a score",
			withType(finished(team, other, 3, 2), domain.ResultUnplayed, nil),
			domain.Tally{},
			domain.Tally{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := []domain.Result{tc.result}
			assert.Equal(t, tc.own, domain.TallyFor(team, results))
			assert.Equal(t, tc.opp, domain.TallyFor(other, results))
		})
	}
}

func TestWinnerByResultType(t *testing.T) {
	assert.Nil(t, withType(noScore(1, 2), domain.ResultDoubleForfeit, nil).Winner(),
		"両チーム不戦敗は勝者がいないので後続試合を埋めない")
	assert.Equal(t, side(domain.SideRight),
		withType(noScore(1, 2), domain.ResultNoContest, side(domain.SideRight)).Winner())
	assert.Nil(t, withType(finished(1, 2, 3, 1), domain.ResultUnplayed, nil).Winner())
}

func TestDecideResult(t *testing.T) {
	forfeit := &domain.ForfeitScore{Winner: 15, Loser: 0}
	cases := []struct {
		name      string
		in        domain.ExceptionalResult
		inBracket bool
		want      domain.ResultOutcome
		wantErr   bool
	}{
		{
			name: "forfeit to the right puts the winner score on the right",
			in: domain.ExceptionalResult{
				Type: domain.ResultForfeit, WinnerSide: side(domain.SideRight), Forfeit: forfeit,
			},
			want: domain.ResultOutcome{
				Score: &domain.Score{Left: 0, Right: 15}, Winner: side(domain.SideRight),
			},
		},
		{
			name:    "forfeit without a winner",
			in:      domain.ExceptionalResult{Type: domain.ResultForfeit, Forfeit: forfeit},
			wantErr: true,
		},
		{
			name: "forfeit without a forfeit score records only the winner",
			in: domain.ExceptionalResult{
				Type: domain.ResultForfeit, WinnerSide: side(domain.SideLeft),
			},
			want: domain.ResultOutcome{Winner: side(domain.SideLeft)},
		},
		{
			name: "forfeit score must have a winner greater than the loser",
			in: domain.ExceptionalResult{
				Type: domain.ResultForfeit, WinnerSide: side(domain.SideLeft),
				Forfeit: &domain.ForfeitScore{Winner: 0, Loser: 0},
			},
			wantErr: true,
		},
		{
			name: "no contest records only the winner",
			in: domain.ExceptionalResult{
				Type: domain.ResultNoContest, WinnerSide: side(domain.SideLeft),
			},
			want: domain.ResultOutcome{Winner: side(domain.SideLeft)},
		},
		{
			name: "double forfeit has no winner",
			in: domain.ExceptionalResult{
				Type: domain.ResultDoubleForfeit, WinnerSide: side(domain.SideLeft),
			},
			wantErr: true,
		},
		{
			name: "unplayed keeps nothing",
			in:   domain.ExceptionalResult{Type: domain.ResultUnplayed},
			want: domain.ResultOutcome{},
		},
		{
			name: "abandoned keeps the current score",
			in: domain.ExceptionalResult{
				Type: domain.ResultAbandoned, Current: domain.Score{Left: 5, Right: 3},
			},
			want: domain.ResultOutcome{Score: &domain.Score{Left: 5, Right: 3}},
		},
		{
			name: "abandoned with a tie in a bracket needs a winner",
			in: domain.ExceptionalResult{
				Type: domain.ResultAbandoned, Current: domain.Score{Left: 4, Right: 4},
			},
			inBracket: true,
			wantErr:   true,
		},
		{
			name:    "normal is not an exceptional result",
			in:      domain.ExceptionalResult{Type: domain.ResultNormal},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.DecideResult(tc.in, tc.inBracket)
			if tc.wantErr {
				require.ErrorIs(t, err, domain.ErrBadRequest)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
