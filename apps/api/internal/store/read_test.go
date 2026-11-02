package store_test

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

func setup(t *testing.T) (*gorm.DB, *storetest.Builder, storetest.Scenario) {
	t.Helper()
	db := testdb.New(t)
	b := storetest.New(t, db)
	return db, b, storetest.BuildScenario(t, b)
}

func ptr[T any](v T) *T { return &v }

// listNow は大会一覧の「今日」。シナリオの大会の区分が日付で変わらない日にする
func listNow() time.Time { return time.Date(2026, 10, 25, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60)) }

func TestListTournaments(t *testing.T) {
	db, b, sc := setup(t)
	undated := b.Tournament(func(row *store.Tournament) { row.NameJa = "日付未定の大会" })

	sections, err := store.ListTournaments(context.Background(), db, listNow())
	require.NoError(t, err)
	require.Len(t, sections, 2, "in_progress and entry_closed have no tournament and are omitted")
	assert.Equal(t, domain.SectionEntryOpen, sections[0].Section)
	require.Len(t, sections[0].Tournaments, 1)
	upcoming := sections[0].Tournaments[0]
	assert.Equal(t, sc.Upcoming.ID, upcoming.ID)
	assert.Equal(t, domain.I18nText{Ja: "2027 架空カップ", En: "2027 Fictional Cup"}, upcoming.Name)
	assert.Equal(t, "2027-05-01", upcoming.StartDate.Format(time.DateOnly))
	assert.Equal(t, domain.SectionFinished, sections[1].Section)
	// 開催日の新しい順で、未定は先頭。一覧に載せない大会は出ない
	var ids []int64
	for _, tour := range sections[1].Tournaments {
		ids = append(ids, tour.ID)
	}
	assert.Equal(t, []int64{undated.ID, sc.Tournament.ID}, ids)
	assert.Nil(t, sections[1].Tournaments[0].StartDate)
}

func TestListTournamentsDecidesSectionsFromMatchesAndCountsEntries(t *testing.T) {
	db, b, _ := setup(t)
	night := b.Tournament(func(row *store.Tournament) {
		row.NameJa, row.Status = "架空ナイト", domain.TournamentEntryClosed
		row.StartDate = storetest.Date(t, "2026-10-25")
	})
	division := b.Division(night.ID)
	left, right, withdrawn := b.Team(division), b.Team(division), b.Team(division)
	require.NoError(t, db.Create(&store.TeamWithdrawal{TeamID: withdrawn.ID}).Error)
	b.Match(division, &left, &right, storetest.At(t, "2026-10-25", "19:00", "1"), storetest.Finished(3, 2))
	b.Match(division, &left, &withdrawn, storetest.At(t, "2026-10-26", "19:00", "1"))

	sectionOf := func(now time.Time) (domain.ListSection, store.TournamentSummary) {
		t.Helper()
		sections, err := store.ListTournaments(context.Background(), db, now)
		require.NoError(t, err)
		for _, section := range sections {
			for _, tour := range section.Tournaments {
				if tour.ID == night.ID {
					return section.Section, tour
				}
			}
		}
		require.FailNow(t, "the tournament is not listed")
		return "", store.TournamentSummary{}
	}

	before, summary := sectionOf(listNow().AddDate(0, 0, -1))
	assert.Equal(t, domain.SectionEntryClosed, before)
	assert.Equal(t, 2, summary.TeamCount, "the withdrawn team is not counted")
	during, _ := sectionOf(listNow())
	assert.Equal(t, domain.SectionInProgress, during)
	after, _ := sectionOf(listNow().AddDate(0, 0, 2))
	assert.Equal(t, domain.SectionFinished, after)
	sections, err := store.ListTournaments(context.Background(), db, listNow())
	require.NoError(t, err)
	assert.Equal(t, domain.SectionInProgress, sections[0].Section, "in_progress comes first")
}

func TestNotFound(t *testing.T) {
	db, _, _ := setup(t)
	ctx := context.Background()
	const missing = 999999
	cases := []struct {
		resource domain.Resource
		call     func() error
	}{
		{domain.ResourceTournament, func() error {
			_, err := store.GetTournamentSchedule(ctx, db, missing, store.ScheduleFilter{})
			return err
		}},
		{domain.ResourceGroup, func() error { _, err := store.GetGroup(ctx, db, missing); return err }},
		{domain.ResourceMatch, func() error { _, err := store.GetMatch(ctx, db, missing); return err }},
	}
	for _, tc := range cases {
		t.Run(string(tc.resource), func(t *testing.T) {
			err := tc.call()
			var notFound *domain.NotFoundError
			require.ErrorAs(t, err, &notFound)
			assert.Equal(t, tc.resource, notFound.Resource)
			assert.Equal(t, int64(missing), notFound.ID)
			assert.True(t, errors.Is(err, domain.ErrNotFound))
		})
	}
}

func scheduleIDs(page store.SchedulePage) []int64 {
	ids := []int64{}
	for _, row := range page.Rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func TestGetTournamentSchedule(t *testing.T) {
	db, _, sc := setup(t)
	ctx := context.Background()
	all := ""
	cases := []struct {
		name     string
		filter   store.ScheduleFilter
		want     []int64
		selected store.ScheduleSelected
	}{
		{
			name:     "absent ymd defaults to the last day",
			filter:   store.ScheduleFilter{},
			want:     []int64{sc.MatchBC.ID, sc.MatchWomen.ID, sc.MatchFinal.ID},
			selected: store.ScheduleSelected{YMD: "20261018"},
		},
		{
			name:   "empty ymd means all days",
			filter: store.ScheduleFilter{YMD: &all},
			want: []int64{
				sc.MatchAB.ID, sc.MatchCA.ID, sc.MatchBC.ID, sc.MatchWomen.ID, sc.MatchFinal.ID,
			},
		},
		{
			name:     "team by full Japanese name on either side",
			filter:   store.ScheduleFilter{YMD: &all, Team: "アルファ・スロー"},
			want:     []int64{sc.MatchAB.ID, sc.MatchCA.ID},
			selected: store.ScheduleSelected{Team: "アルファ・スロー"},
		},
		{
			name:     "field filter",
			filter:   store.ScheduleFilter{YMD: &all, Field: "2"},
			want:     []int64{sc.MatchCA.ID, sc.MatchFinal.ID},
			selected: store.ScheduleSelected{Field: "2"},
		},
		{
			name: "sort by field descending keeps date and time order inside a field",
			filter: store.ScheduleFilter{YMD: &all, Sort: &domain.ScheduleSort{
				Key: domain.SortByField, Dir: domain.SortDesc,
			}},
			want: []int64{
				sc.MatchWomen.ID, sc.MatchCA.ID, sc.MatchFinal.ID, sc.MatchAB.ID, sc.MatchBC.ID,
			},
		},
		{
			name: "sort by division",
			filter: store.ScheduleFilter{YMD: ptr("20261018"), Sort: &domain.ScheduleSort{
				Key: domain.SortByDivision, Dir: domain.SortDesc,
			}},
			want:     []int64{sc.MatchWomen.ID, sc.MatchBC.ID, sc.MatchFinal.ID},
			selected: store.ScheduleSelected{YMD: "20261018"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, err := store.GetTournamentSchedule(ctx, db, sc.Tournament.ID, tc.filter)
			require.NoError(t, err)
			assert.Equal(t, tc.want, scheduleIDs(page))
			assert.Equal(t, tc.selected, page.Selected)
			assert.Nil(t, page.Division)
			assert.Equal(t, []string{"2026-10-17", "2026-10-18"}, page.Dates)
			assert.Equal(t, []string{"1", "2", "B"}, page.Fields, "distinct fields in natural order")
		})
	}

	page, err := store.GetTournamentSchedule(ctx, db, sc.Tournament.ID, store.ScheduleFilter{})
	require.NoError(t, err)
	var values []string
	for _, option := range page.Teams {
		values = append(values, option.Value)
	}
	assert.Equal(t, []string{"アルファ・スロー", "ブラボー", "チャーリー", "デルタ", "ウィスキー", "ヤンキー"}, values)
	final := page.Rows[2]
	assert.Equal(t, domain.MatchLive, final.Status)
	assert.Nil(t, final.Right.Team)
	assert.Equal(
		t,
		&domain.Placeholder{Code: "A1", Outcome: domain.OutcomeWin},
		final.Right.Placeholder,
	)
	assert.Equal(t, ptr("10:00"), final.Time)
	assert.Equal(t, domain.ScheduleDivision{
		ID: sc.Mixed.ID, Name: domain.I18nText{Ja: "ビーチミックス", En: "Beach Mixed"}, Kind: domain.DivisionMixed,
	}, final.Division)
}

// countQueries は db で実行された SELECT の数を数える。
func countQueries(t *testing.T, db *gorm.DB) *atomic.Int64 {
	t.Helper()
	var count atomic.Int64
	err := db.Callback().
		Query().
		After("gorm:query").
		Register("test:count", func(*gorm.DB) { count.Add(1) })
	require.NoError(t, err)
	return &count
}

func TestScheduleQueryCountDoesNotGrowWithMatches(t *testing.T) {
	db, b, sc := setup(t)
	count := countQueries(t, db)
	ctx := context.Background()
	all := ""

	count.Store(0)
	_, err := store.GetTournamentSchedule(ctx, db, sc.Tournament.ID, store.ScheduleFilter{YMD: &all})
	require.NoError(t, err)
	before := count.Load()

	// 参照サイトの最大規模(282 試合)を超える数の試合を足しても問い合わせ数は変わらない
	for i := range 300 {
		b.Match(sc.Mixed, &sc.TeamA, &sc.TeamD, storetest.Finished(i%7, 3),
			storetest.At(t, "2026-10-19", "08:00", strconv.Itoa(1+i%10)))
	}
	count.Store(0)
	page, err := store.GetTournamentSchedule(
		ctx,
		db,
		sc.Tournament.ID,
		store.ScheduleFilter{YMD: &all},
	)
	require.NoError(t, err)
	assert.Len(t, page.Rows, 305)
	assert.Equal(t, before, count.Load())
	// 7 回に大会の中断(閲覧ページの帯)の 1 回と、試合の名前(何位決定戦か)のための大会の形の 1 回を足した
	assert.LessOrEqual(t, before, int64(9))
}

func TestGetGroupPool(t *testing.T) {
	db, _, sc := setup(t)
	page, err := store.GetGroup(context.Background(), db, sc.Pool.ID)
	require.NoError(t, err)
	assert.Nil(t, page.Bracket)
	assert.Nil(t, page.Image)
	require.NotNil(t, page.Pool)
	pool := page.Pool

	assert.Equal(t, store.MatrixCell{Kind: store.MatrixSelf}, pool.Matrix[0][0])
	assert.Equal(t, store.MatrixCell{Kind: store.MatrixLower}, pool.Matrix[1][0])
	assert.Equal(t, store.MatrixCell{
		Kind: store.MatrixMatch, MatchID: sc.MatchAB.ID, Status: domain.MatchFinished, Text: "9-3",
	}, pool.Matrix[0][1])
	// C が左の試合も記録の左右順のまま出す
	assert.Equal(t, "5-5", pool.Matrix[0][2].Text)
	// 未試合は試合の CD
	assert.Equal(t, store.MatrixCell{
		Kind: store.MatrixMatch, MatchID: sc.MatchBC.ID, Status: domain.MatchScheduled, Text: "A3",
	}, pool.Matrix[1][2])

	var ranks []int
	for _, row := range pool.Results {
		ranks = append(ranks, *row.Rank)
	}
	assert.Equal(t, []int{1, 3, 2}, ranks, "results keep registration order")
	assert.Equal(t, 4, pool.Results[0].Points)

	var matchIDs []int64
	for _, row := range page.Matches {
		matchIDs = append(matchIDs, row.ID)
	}
	assert.Equal(t, []int64{sc.MatchAB.ID, sc.MatchCA.ID, sc.MatchBC.ID}, matchIDs)
}

func TestGetGroupBracketAndImage(t *testing.T) {
	db, _, sc := setup(t)
	ctx := context.Background()
	page, err := store.GetGroup(ctx, db, sc.Bracket.ID)
	require.NoError(t, err)
	assert.Nil(t, page.Pool)
	require.NotNil(t, page.Bracket)
	assert.Equal(t, "T2A", page.Bracket.Pattern)
	assert.Equal(t, 2, page.Bracket.Columns)
	require.Len(t, page.Bracket.Rows, 4)
	team := page.Bracket.Rows[1][0]
	require.NotNil(t, team.Team)
	assert.Equal(t, "デルタ", team.Team.Name.Ja)
	game := page.Bracket.Rows[2][0]
	assert.Equal(t, "2-1", game.Text)
	assert.Equal(t, ptr(domain.MatchLive), game.Status)
	assert.Equal(t, "[A1-勝ち]", page.Bracket.Rows[3][0].Text)

	page, err = store.GetGroup(ctx, db, sc.Image.ID)
	require.NoError(t, err)
	assert.Equal(
		t,
		&store.ImageView{URL: "/ranking.jpg", Width: ptr(500), Height: ptr(695)},
		page.Image,
	)
	assert.Empty(t, page.Matches)
}

func TestGetMatch(t *testing.T) {
	db, _, sc := setup(t)
	page, err := store.GetMatch(context.Background(), db, sc.MatchAB.ID)
	require.NoError(t, err)
	info := page.Match
	assert.Equal(t, "A1", info.Code)
	assert.Equal(t, ptr("Qualifying pool A"), info.GroupName)
	assert.Equal(t, ptr("2026-10-17"), info.Date)
	assert.Equal(t, ptr("09:00"), info.Time)
	assert.Equal(t, ptr(9), info.Left.Score)
	assert.Equal(t, store.Breadcrumb{
		DivisionID: sc.Mixed.ID, DivisionName: domain.I18nText{Ja: "ビーチミックス", En: "Beach Mixed"}, GroupID: &sc.Pool.ID,
	}, info.Breadcrumb)

	var kinds []domain.EventKind
	var scores [][2]int
	for _, ev := range page.Events {
		kinds = append(kinds, ev.Kind)
		scores = append(scores, [2]int{ev.LeftScore, ev.RightScore})
	}
	assert.Equal(t, []domain.EventKind{"pull", "goal", "timeout", "goal", "goal", "end"}, kinds)
	assert.Equal(t, [][2]int{{0, 0}, {1, 0}, {1, 0}, {1, 1}, {2, 1}, {2, 1}}, scores)
	require.NotNil(t, page.Events[1].Assist)
	assert.Equal(t, sc.PlayerA10.ID, page.Events[1].Assist.ID)
	assert.Nil(t, page.Events[5].Side)

	require.Len(t, page.Lineups.Left, 2)
	assert.Equal(
		t,
		store.LineupLine{Player: page.Lineups.Left[0].Player, Goals: 2, Blocks: 2},
		page.Lineups.Left[0],
	)
	assert.Equal(t, sc.PlayerA7.ID, page.Lineups.Left[0].Player.ID)
	assert.Equal(t, 1, page.Lineups.Left[1].Assists)
	require.Len(t, page.Lineups.Right, 1)

	assert.Equal(t, domain.MatchStats{
		Left:  domain.SideStats{Goals: 2, FromOffense: 1, FromDefense: 1, FromTurnover: 1},
		Right: domain.SideStats{Goals: 1, FromOffense: 1, Timeouts: 1},
	}, page.Stats)
}

// 試合に持たせたグループ欄の文字、備考、Block 列と統計表を隠す印はそのまま返す。
func TestGetMatchDisplayValues(t *testing.T) {
	db, b, sc := setup(t)
	match := b.Match(sc.Mixed, &sc.TeamA, &sc.TeamB, storetest.Finished(1, 0), func(row *store.Match) {
		row.GroupID, row.GroupLabel = &sc.Pool.ID, ptr("Qualifying Pool A")
		row.Note, row.HideBlocks, row.HideStats = "架空の備考", true, true
	})
	page, err := store.GetMatch(context.Background(), db, match.ID)
	require.NoError(t, err)
	assert.Equal(t, ptr("Qualifying Pool A"), page.Match.GroupName, "グループの名前より試合の文字")
	assert.Equal(t, "架空の備考", page.Match.Note)
	assert.True(t, page.Match.HideBlocks)
	assert.True(t, page.Match.HideStats)
}

func TestGetMatchWithoutGroupOrEvents(t *testing.T) {
	db, _, sc := setup(t)
	page, err := store.GetMatch(context.Background(), db, sc.MatchWomen.ID)
	require.NoError(t, err)
	assert.Nil(t, page.Match.GroupName)
	assert.Nil(t, page.Match.Breadcrumb.GroupID)
	assert.Empty(t, page.Events)
	assert.Equal(
		t,
		store.Lineups{Left: []store.LineupLine{}, Right: []store.LineupLine{}},
		page.Lineups,
	)
}
