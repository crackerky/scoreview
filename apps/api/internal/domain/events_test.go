package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/domain"
)

func ev(kind domain.EventKind, s domain.Side) domain.Event {
	return domain.Event{Kind: kind, Side: side(s)}
}

const (
	leftSide  = domain.SideLeft
	rightSide = domain.SideRight
)

func TestRunningScores(t *testing.T) {
	events := []domain.Event{
		ev(domain.EventPull, leftSide),
		ev(domain.EventGoal, rightSide),
		ev(domain.EventTimeout, leftSide),
		ev(domain.EventGoal, leftSide),
		ev(domain.EventGoal, leftSide),
		{Kind: domain.EventEnd},
	}
	want := []domain.Score{{0, 0}, {0, 1}, {0, 1}, {1, 1}, {2, 1}, {2, 1}}
	assert.Equal(t, want, domain.RunningScores(events))
	assert.Empty(t, domain.RunningScores(nil))
}

func TestComputeMatchStats(t *testing.T) {
	cases := []struct {
		name   string
		events []domain.Event
		want   domain.MatchStats
	}{
		{
			name:   "no events",
			events: nil,
			want:   domain.MatchStats{},
		},
		{
			// 左がプル → 右がオフェンスで始めて得点(右 O)。右が次にプル → 左が得点(左 O)。
			// 左が次にプル → 左が得点(ブレイク、左 D。ターンオーバーからの得点も D と同じ数)。
			name: "pull side starts on defense and scorer pulls next",
			events: []domain.Event{
				ev(domain.EventPull, leftSide),
				ev(domain.EventGoal, rightSide),
				ev(domain.EventGoal, leftSide),
				ev(domain.EventGoal, leftSide),
				ev(domain.EventTimeout, rightSide),
				ev(domain.EventTimeout, rightSide),
				{Kind: domain.EventEnd},
			},
			want: domain.MatchStats{
				Left:  domain.SideStats{Goals: 2, FromOffense: 1, FromDefense: 1, FromTurnover: 1},
				Right: domain.SideStats{Goals: 1, FromOffense: 1, Timeouts: 2},
			},
		},
		{
			name: "a later pull (second half) overrides the pulling side",
			events: []domain.Event{
				ev(domain.EventPull, rightSide),
				ev(domain.EventGoal, leftSide),  // 左 O
				ev(domain.EventPull, leftSide),  // 後半: 左がプル
				ev(domain.EventGoal, rightSide), // 右 O
			},
			want: domain.MatchStats{
				Left:  domain.SideStats{Goals: 1, FromOffense: 1},
				Right: domain.SideStats{Goals: 1, FromOffense: 1},
			},
		},
		{
			name: "goals before the first pull count as neither offense nor defense",
			events: []domain.Event{
				ev(domain.EventGoal, leftSide),
				ev(domain.EventGoal, leftSide), // 左が得点後にプルしているので左 D
			},
			want: domain.MatchStats{Left: domain.SideStats{Goals: 2, FromDefense: 1, FromTurnover: 1}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, domain.ComputeMatchStats(tc.events))
		})
	}
}
