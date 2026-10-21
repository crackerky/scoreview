package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
)

func TestCheckExpectedScore(t *testing.T) {
	actual := domain.Score{Left: 2, Right: 3}
	require.NoError(t, domain.CheckExpectedScore(nil, actual), "確認なしの操作は確かめない")
	require.NoError(t, domain.CheckExpectedScore(&domain.Score{Left: 2, Right: 3}, actual))

	err := domain.CheckExpectedScore(&domain.Score{Left: 2, Right: 0}, actual)
	require.ErrorIs(t, err, domain.ErrConflict)
	var mismatch *domain.ScoreMismatchError
	require.True(t, errors.As(err, &mismatch))
	assert.Equal(t, actual, mismatch.Actual)
	assert.Contains(t, err.Error(), "2-3")
}

func TestFindConcurrentRecords(t *testing.T) {
	start := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	ev := func(id string, kind domain.EventKind, user int64, sec int) domain.RecordedEvent {
		return domain.RecordedEvent{
			ClientID: id, Kind: kind, UserID: user,
			RecordedAt: start.Add(time.Duration(sec) * time.Second),
		}
	}
	goal, pull := domain.EventGoal, domain.EventPull
	cases := []struct {
		name   string
		events []domain.RecordedEvent
		want   []string // 後から記録した方の ClientID
	}{
		{
			"two scorers record the same goal within 5 seconds",
			[]domain.RecordedEvent{ev("a", goal, 1, 0), ev("b", goal, 2, 3)},
			[]string{"b"},
		},
		{
			"two pulls from different users",
			[]domain.RecordedEvent{ev("a", pull, 1, 10), ev("b", pull, 2, 11)},
			[]string{"b"},
		},
		{
			"the same user may record quickly",
			[]domain.RecordedEvent{ev("a", goal, 1, 0), ev("b", goal, 1, 1)},
			[]string{},
		},
		{
			"more than 5 seconds apart is another point",
			[]domain.RecordedEvent{ev("a", goal, 1, 0), ev("b", goal, 2, 6)},
			[]string{},
		},
		{
			"different kinds are not compared",
			[]domain.RecordedEvent{ev("a", pull, 1, 0), ev("b", goal, 2, 1)},
			[]string{},
		},
		{
			"an unknown recorder is not compared",
			[]domain.RecordedEvent{ev("a", goal, 0, 0), ev("b", goal, 2, 1)},
			[]string{},
		},
		{
			"timeouts are not checked",
			[]domain.RecordedEvent{ev("a", domain.EventTimeout, 1, 0), ev("b", domain.EventTimeout, 2, 1)},
			[]string{},
		},
		{
			"order is by the recorded time",
			[]domain.RecordedEvent{ev("b", goal, 2, 4), ev("a", goal, 1, 0)},
			[]string{"b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := []string{}
			for _, record := range domain.FindConcurrentRecords(tc.events) {
				got = append(got, record.Second.ClientID)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
