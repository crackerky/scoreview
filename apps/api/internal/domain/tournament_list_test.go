package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
)

func day(tb testing.TB, value string) *time.Time {
	tb.Helper()
	parsed, err := time.Parse(time.DateOnly, value)
	require.NoError(tb, err)
	return &parsed
}

func TestListSectionOf(t *testing.T) {
	today := time.Date(2026, 10, 25, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	twoDays := domain.PlayProgress{Matches: 10, Finished: 3, FirstDay: day(t, "2026-10-25"), LastDay: day(t, "2026-10-26")}
	cases := []struct {
		name     string
		status   domain.TournamentStatus
		progress domain.PlayProgress
		want     domain.ListSection
	}{
		{"試合の無い受付中", domain.TournamentEntryOpen, domain.PlayProgress{}, domain.SectionEntryOpen},
		{"試合の無い締切り", domain.TournamentEntryClosed, domain.PlayProgress{}, domain.SectionEntryClosed},
		{
			"初日の前は管理者の決めた状態のまま", domain.TournamentEntryClosed,
			domain.PlayProgress{Matches: 4, FirstDay: day(t, "2026-11-03"), LastDay: day(t, "2026-11-03")},
			domain.SectionEntryClosed,
		},
		{"初日は開催中", domain.TournamentEntryClosed, twoDays, domain.SectionInProgress},
		{"受付中のままでも試合の日なら開催中", domain.TournamentEntryOpen, twoDays, domain.SectionInProgress},
		{
			"日付が先でも試合中の試合があれば開催中", domain.TournamentEntryClosed,
			domain.PlayProgress{Matches: 4, Live: 1, FirstDay: day(t, "2026-11-03"), LastDay: day(t, "2026-11-03")},
			domain.SectionInProgress,
		},
		{
			"全試合が終われば最終日の前でも終了", domain.TournamentEntryClosed,
			domain.PlayProgress{Matches: 10, Finished: 10, FirstDay: day(t, "2026-10-25"), LastDay: day(t, "2026-10-26")},
			domain.SectionFinished,
		},
		{
			"最終日を過ぎたら残りの試合があっても終了", domain.TournamentEntryOpen,
			domain.PlayProgress{Matches: 24, Finished: 15, FirstDay: day(t, "2026-10-21"), LastDay: day(t, "2026-10-21")},
			domain.SectionFinished,
		},
		{
			"管理者が終了にした大会は試合中の試合があっても終了", domain.TournamentFinished,
			domain.PlayProgress{Matches: 29, Finished: 25, Live: 2, FirstDay: day(t, "2026-10-25"), LastDay: day(t, "2026-10-25")},
			domain.SectionFinished,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, domain.ListSectionOf(tc.status, tc.progress, today))
		})
	}
}

func TestListSectionOfUsesTheCalendarDayOfToday(t *testing.T) {
	// 会場の暦で 10/25 の 0 時 30 分(UTC ではまだ 10/24)。今日は 10/25 として扱う
	justAfterMidnight := time.Date(2026, 10, 25, 0, 30, 0, 0, time.FixedZone("JST", 9*60*60))
	progress := domain.PlayProgress{Matches: 2, FirstDay: day(t, "2026-10-25"), LastDay: day(t, "2026-10-25")}
	assert.Equal(
		t, domain.SectionInProgress,
		domain.ListSectionOf(domain.TournamentEntryClosed, progress, justAfterMidnight),
	)
}
