package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
)

func TestValidTransition(t *testing.T) {
	const (
		scheduled = domain.MatchScheduled
		live      = domain.MatchLive
		finished  = domain.MatchFinished
	)
	cases := []struct {
		from, to domain.MatchStatus
		role     domain.Role
		want     error
	}{
		{scheduled, live, domain.RoleStaff, nil},
		{live, finished, domain.RoleStaff, nil},
		{scheduled, finished, domain.RoleStaff, nil},
		{finished, live, domain.RoleAdmin, nil},
		{finished, live, domain.RoleStaff, domain.ErrForbidden},
		{live, scheduled, domain.RoleAdmin, domain.ErrConflict},
		{finished, scheduled, domain.RoleAdmin, domain.ErrConflict},
		{live, live, domain.RoleAdmin, domain.ErrConflict},
		{live, "paused", domain.RoleAdmin, domain.ErrBadRequest},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.to)+"/"+string(tc.role), func(t *testing.T) {
			err := domain.ValidTransition(tc.from, tc.to, tc.role)
			if tc.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestDecideWinner(t *testing.T) {
	left, right := domain.SideLeft, domain.SideRight
	cases := []struct {
		name      string
		score     domain.Score
		inBracket bool
		requested *domain.Side
		want      *domain.Side
		wantErr   bool
	}{
		{name: "score decides, nothing stored", score: domain.Score{Left: 7, Right: 5}},
		{
			name:      "matching request is accepted",
			score:     domain.Score{Left: 3, Right: 5},
			requested: &right,
		},
		{
			name:      "contradicting request",
			score:     domain.Score{Left: 7, Right: 5},
			requested: &right,
			wantErr:   true,
		},
		// clone-spec C 2.3: 4-4 でも勝者が付く
		{
			name:      "tie in bracket with winner",
			score:     domain.Score{Left: 4, Right: 4},
			inBracket: true,
			requested: &left,
			want:      &left,
		},
		{
			name:      "tie in bracket without winner",
			score:     domain.Score{Left: 4, Right: 4},
			inBracket: true,
			wantErr:   true,
		},
		{name: "tie in pool is a draw", score: domain.Score{Left: 5, Right: 5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.DecideWinner(tc.score, tc.inBracket, tc.requested)
			if tc.wantErr {
				assert.ErrorIs(t, err, domain.ErrBadRequest)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNewEventValidate(t *testing.T) {
	left := domain.SideLeft
	one, two := int64(1), int64(2)
	cases := []struct {
		name  string
		event domain.NewEvent
		ok    bool
	}{
		{
			"goal with scorer and assist",
			domain.NewEvent{Kind: domain.EventGoal, Side: &left, ScorerID: &one, AssistID: &two},
			true,
		},
		{"goal without players", domain.NewEvent{Kind: domain.EventGoal, Side: &left}, true},
		{
			"callahan without assist",
			domain.NewEvent{Kind: domain.EventGoal, Side: &left, ScorerID: &one, Callahan: true},
			true,
		},
		{
			"callahan with assist",
			domain.NewEvent{
				Kind:     domain.EventGoal,
				Side:     &left,
				ScorerID: &one,
				AssistID: &two,
				Callahan: true,
			},
			false,
		},
		{
			"same scorer and assist",
			domain.NewEvent{Kind: domain.EventGoal, Side: &left, ScorerID: &one, AssistID: &one},
			false,
		},
		{"goal without side", domain.NewEvent{Kind: domain.EventGoal}, false},
		{"timeout", domain.NewEvent{Kind: domain.EventTimeout, Side: &left}, true},
		{
			"timeout with scorer",
			domain.NewEvent{Kind: domain.EventTimeout, Side: &left, ScorerID: &one},
			false,
		},
		{"end", domain.NewEvent{Kind: domain.EventEnd}, true},
		{"end with side", domain.NewEvent{Kind: domain.EventEnd, Side: &left}, false},
		{"unknown kind", domain.NewEvent{Kind: "foul", Side: &left}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.event.Validate()
			if tc.ok {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, domain.ErrBadRequest)
			}
		})
	}
}

func TestStatusAfterEvent(t *testing.T) {
	got, err := domain.StatusAfterEvent(domain.MatchScheduled, domain.EventPull)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchLive, got)

	got, err = domain.StatusAfterEvent(domain.MatchLive, domain.EventEnd)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchFinished, got)

	_, err = domain.StatusAfterEvent(domain.MatchFinished, domain.EventGoal)
	assert.ErrorIs(t, err, domain.ErrConflict)
}

func TestStatusAfterEventRemoval(t *testing.T) {
	got, err := domain.StatusAfterEventRemoval(domain.MatchLive, domain.EventGoal, false)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchLive, got)

	got, err = domain.StatusAfterEventRemoval(domain.MatchFinished, domain.EventEnd, false)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchLive, got)

	got, err = domain.StatusAfterEventRemoval(domain.MatchFinished, domain.EventEnd, true)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchFinished, got)

	_, err = domain.StatusAfterEventRemoval(domain.MatchFinished, domain.EventGoal, false)
	assert.True(t, errors.Is(err, domain.ErrConflict))
}

func TestOutcomeTeam(t *testing.T) {
	score := func(left, right int) (*int, *int) { return &left, &right }
	l, r := score(9, 3)
	result := domain.Result{
		LeftTeamID:  10,
		RightTeamID: 20,
		LeftScore:   l,
		RightScore:  r,
		Status:      domain.MatchFinished,
	}
	assert.Equal(t, int64(10), *domain.OutcomeTeam(result, domain.OutcomeWin))
	assert.Equal(t, int64(20), *domain.OutcomeTeam(result, domain.OutcomeLose))

	// 4-4 で右が勝者(clone-spec C 2.3)
	tl, tr := score(4, 4)
	tied := domain.Result{
		LeftTeamID:  10,
		RightTeamID: 20,
		LeftScore:   tl,
		RightScore:  tr,
		Status:      domain.MatchFinished,
		WinnerSide:  side(domain.SideRight),
	}
	assert.Equal(t, int64(20), *domain.OutcomeTeam(tied, domain.OutcomeWin))

	draw := tied
	draw.WinnerSide = nil
	assert.Nil(t, domain.OutcomeTeam(draw, domain.OutcomeWin))

	live := result
	live.Status = domain.MatchLive
	assert.Nil(t, domain.OutcomeTeam(live, domain.OutcomeWin))
}

func TestFinalScoreAndElapsed(t *testing.T) {
	assert.Equal(t, domain.Score{}, domain.FinalScore(nil))
	events := []domain.Event{
		ev(domain.EventGoal, leftSide),
		ev(domain.EventGoal, rightSide),
		ev(domain.EventGoal, leftSide),
	}
	assert.Equal(t, domain.Score{Left: 2, Right: 1}, domain.FinalScore(events))

	start := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	assert.Equal(
		t,
		125,
		domain.ElapsedSince(start, start.Add(125*time.Second+400*time.Millisecond)),
	)
	assert.Equal(t, 0, domain.ElapsedSince(start, start.Add(-time.Minute)))
}
