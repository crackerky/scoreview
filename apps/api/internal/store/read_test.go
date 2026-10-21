package store_test

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"testing"

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

func TestListTournaments(t *testing.T) {
	db, b, sc := setup(t)
	undated := b.Tournament(func(row *store.Tournament) { row.NameJa = "日付未定の大会" })

	sections, err := store.ListTournaments(context.Background(), db)
	require.NoError(t, err)
	require.Len(t, sections, 2, "entry_closed has no tournament and is omitted")
	assert.Equal(t, domain.TournamentEntryOpen, sections[0].Status)
	assert.Equal(
		t,
		[]store.TournamentSummary{
			{ID: sc.Upcoming.ID, Name: domain.I18nText{Ja: "2027 架空カップ", En: "2027 Fictional Cup"}},
		},
		sections[0].Tournaments,
	)
	assert.Equal(t, domain.TournamentFinished, sections[1].Status)
	// 開催日の新しい順で、未定は先頭。一覧に載せない大会は出ない
	var ids []int64
	for _, tour := range sections[1].Tournaments {
		ids = append(ids, tour.ID)
	}
	assert.Equal(t, []int64{undated.ID, sc.Tournament.ID}, ids)
}

func TestGetTournamentTopAndFrame(t *testing.T) {
	db, _, sc := setup(t)
	page, err := store.GetTournamentTop(context.Background(), db, sc.Tournament.ID)
	require.NoError(t, err)

	frame := page.Frame
	assert.Equal(t, "2026 Beach", frame.ShortName)
	require.Len(t, frame.Divisions, 2)
	assert.Equal(t, sc.Mixed.ID, frame.Divisions[0].ID)
	// 非公開(published=false)のグループは出ず、sort_order 順
	assert.Equal(t, []store.FrameGroup{
		{ID: sc.Pool.ID, Name: "Qualifying pool A", Linked: true},
		{ID: sc.Bracket.ID, Name: "Tounament A", Linked: false},
		{ID: sc.Image.ID, Name: "Temporary ranking", Linked: true},
	}, frame.Divisions[0].Groups)
	assert.Equal(t, []store.FrameGroup{}, frame.Divisions[1].Groups)
	assert.Equal(t, []store.FrameGallery{{ID: sc.Gallery.ID, Name: "Day2-AM"}}, frame.Galleries)
	assert.Equal(t, []store.FrameBanner{
		{ImageURL: "/b1.png", LinkURL: ptr("https://example.com/")}, {ImageURL: "/b2.png"},
	}, frame.Banners)

	assert.Equal(t, ptr("/logo.png"), page.LogoURL)
	assert.Contains(t, page.TopHTML.Ja, "<script>", "store returns raw HTML; httpapi sanitizes")
	assert.Equal(t, []store.TopDivision{
		{
			ID:              sc.Mixed.ID,
			Name:            domain.I18nText{Ja: "ビーチミックス", En: "Beach Mixed"},
			ScheduleLinked:  true,
			StandingsLinked: true,
		},
		{ID: sc.Women.ID, Name: domain.I18nText{Ja: "ウィメン", En: "Women's"}, ScheduleLinked: true},
	}, page.Divisions)
}

func TestGetTournamentTopReturnsUnlisted(t *testing.T) {
	db, _, sc := setup(t)
	page, err := store.GetTournamentTop(context.Background(), db, sc.Unlisted.ID)
	require.NoError(t, err)
	assert.Equal(t, "", page.Frame.Name.En)
	assert.Empty(t, page.Divisions)
}

func TestNotFound(t *testing.T) {
	db, _, _ := setup(t)
	ctx := context.Background()
	const missing = 999999
	cases := []struct {
		resource domain.Resource
		call     func() error
	}{
		{
			domain.ResourceTournament,
			func() error { _, err := store.GetTournamentTop(ctx, db, missing); return err },
		},
		{
			domain.ResourceTournament,
			func() error { _, err := store.GetTeamList(ctx, db, missing); return err },
		},
		{domain.ResourceTournament, func() error {
			_, err := store.GetTournamentSchedule(ctx, db, missing, store.ScheduleFilter{})
			return err
		}},
		{domain.ResourceDivision, func() error {
			_, err := store.GetDivisionSchedule(ctx, db, missing, store.ScheduleFilter{})
			return err
		}},
		{
			domain.ResourceDivision,
			func() error { _, err := store.GetDivisionStandings(ctx, db, missing); return err },
		},
		{domain.ResourceGroup, func() error { _, err := store.GetGroup(ctx, db, missing); return err }},
		{domain.ResourceTeam, func() error { _, err := store.GetTeam(ctx, db, missing); return err }},
		{domain.ResourceMatch, func() error { _, err := store.GetMatch(ctx, db, missing); return err }},
		{domain.ResourcePlayer, func() error { _, err := store.GetPlayer(ctx, db, missing); return err }},
		{
			domain.ResourceGallery,
			func() error { _, err := store.GetGallery(ctx, db, missing); return err },
		},
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

func TestGetTeamList(t *testing.T) {
	db, b, sc := setup(t)
	individual := b.Division(sc.Tournament.ID, func(row *store.Division) {
		row.NameJa, row.EntryKind, row.SortOrder = "ＤＧシングルス", domain.EntryIndividual, 9
	})
	solo := b.Team(individual, func(row *store.Team) { row.NameFullJa = "架空 個人" })
	b.Player(solo.ID)

	page, err := store.GetTeamList(context.Background(), db, sc.Tournament.ID)
	require.NoError(t, err)
	require.Len(t, page.Divisions, 3)
	mixed := page.Divisions[0]
	assert.Equal(t, domain.DivisionMixed, mixed.Kind)
	require.Len(t, mixed.Teams, 4)
	assert.Equal(t, sc.TeamA.ID, mixed.Teams[0].ID)
	assert.Equal(t, "アルファ・スロー", mixed.Teams[0].Name.Ja, "team list shows the full name")
	assert.Equal(t, ptr(2), mixed.Teams[0].MemberCount, "staff is not counted")
	assert.Equal(t, ptr(0), mixed.Teams[3].MemberCount)
	assert.Nil(t, page.Divisions[2].Teams[0].MemberCount, "individual entries have no member count")
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

func TestGetDivisionSchedule(t *testing.T) {
	db, _, sc := setup(t)
	ctx := context.Background()
	page, err := store.GetDivisionSchedule(ctx, db, sc.Mixed.ID, store.ScheduleFilter{})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]int64{sc.MatchAB.ID, sc.MatchCA.ID, sc.MatchBC.ID, sc.MatchFinal.ID},
		scheduleIDs(page),
		"absent ymd means all days for a division",
	)
	assert.Equal(
		t,
		&store.DivisionRef{ID: sc.Mixed.ID, Name: domain.I18nText{Ja: "ビーチミックス", En: "Beach Mixed"}},
		page.Division,
	)
	assert.Equal(t, "", page.Selected.YMD)
	require.Len(t, page.Teams, 4)
	assert.Equal(t, store.ScheduleTeamOption{
		Value: strconv.FormatInt(sc.TeamA.ID, 10), Name: domain.I18nText{Ja: "アルファ・スロー", En: "Alpha"},
	}, page.Teams[0])

	teamB := strconv.FormatInt(sc.TeamB.ID, 10)
	page, err = store.GetDivisionSchedule(ctx, db, sc.Mixed.ID, store.ScheduleFilter{Team: teamB})
	require.NoError(t, err)
	assert.Equal(t, []int64{sc.MatchAB.ID, sc.MatchBC.ID}, scheduleIDs(page))
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
	assert.LessOrEqual(t, before, int64(7))
}

func TestGetDivisionStandings(t *testing.T) {
	db, _, sc := setup(t)
	page, err := store.GetDivisionStandings(context.Background(), db, sc.Mixed.ID)
	require.NoError(t, err)

	var order []int64
	for _, row := range page.Standings {
		order = append(order, row.Team.ID)
	}
	assert.Equal(
		t,
		[]int64{sc.TeamB.ID, sc.TeamA.ID, sc.TeamC.ID, sc.TeamD.ID},
		order,
		"final_rank, nulls last",
	)
	assert.Equal(
		t,
		domain.Tally{Games: 2, Wins: 1, Draws: 1, GoalsFor: 14, GoalsAgainst: 8},
		page.Standings[1].Tally,
	)
	assert.Nil(t, page.Standings[3].Rank)
	assert.Equal(t, domain.Tally{}, page.Standings[3].Tally, "live match is not counted")

	require.Len(t, page.Leaders, 3)
	assert.Equal(t, sc.PlayerA7.ID, page.Leaders[0].Player.ID)
	assert.Equal(t, store.LeaderRow{
		Team: domain.TeamRef{ID: sc.TeamB.ID, Name: domain.I18nText{Ja: "ブラボー", En: "Bravo"}},
		Player: domain.PlayerRef{
			ID: sc.PlayerB1.ID, Number: "1", Name: domain.I18nText{Ja: "架空 一太", En: "Ichita KAKU"},
		},
		Games: 1, Goals: 1,
	}, page.Leaders[1])
	assert.Equal(t, sc.PlayerA10.ID, page.Leaders[2].Player.ID)
	assert.Equal(t, 1, page.Leaders[2].Assists)

	require.Len(t, page.BlockLeaders, 2)
	assert.Equal(t, sc.PlayerA7.ID, page.BlockLeaders[0].Player.ID)
	assert.Equal(t, 2, page.BlockLeaders[0].Blocks)
	assert.Equal(t, sc.PlayerB1.ID, page.BlockLeaders[1].Player.ID)
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
	assert.Equal(t, store.MatrixCell{
		Kind: store.MatrixMatch, MatchID: sc.MatchBC.ID, Status: domain.MatchScheduled, Text: "",
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

func TestGetTeam(t *testing.T) {
	db, _, sc := setup(t)
	page, err := store.GetTeam(context.Background(), db, sc.TeamA.ID)
	require.NoError(t, err)
	assert.Equal(t, store.TeamLayoutStandard, page.Team.Layout)
	assert.Equal(t, "アルファ・スロー", page.Team.FullName.Ja)
	assert.Equal(t, "アルファ", page.Team.ShortName.Ja)
	assert.Equal(
		t,
		domain.Tally{Games: 2, Wins: 1, Draws: 1, GoalsFor: 14, GoalsAgainst: 8},
		page.Record,
	)
	assert.Equal(t, ptr(2), page.Rank)
	require.Len(t, page.Matches, 2)
	assert.Equal(t, sc.MatchAB.ID, page.Matches[0].ID)
	require.Len(t, page.Players, 2)
	assert.Equal(t, store.PlayerLine{
		Player: domain.PlayerRef{
			ID: sc.PlayerA7.ID, Number: "7", Name: domain.I18nText{Ja: "架空 七郎", En: "Shichiro KAKU"},
			Gender: ptr(domain.Gender("M")),
		},
		Games: 1, Goals: 2, Blocks: 2,
	}, page.Players[0])
	assert.Equal(t, []store.StaffLine{{
		ID: sc.StaffA.ID, Role: domain.I18nText{Ja: "マネージャ", En: "Manager"},
		Name: domain.I18nText{Ja: "架空 監督", En: "Coach KAKU"},
	}}, page.Staff)
}

func TestGetTeamDiscgolfIsRosterOnly(t *testing.T) {
	db, b, sc := setup(t)
	division := b.Division(
		sc.Tournament.ID,
		func(row *store.Division) { row.StatsKind = domain.StatsDiscgolf },
	)
	team := b.Team(division)
	page, err := store.GetTeam(context.Background(), db, team.ID)
	require.NoError(t, err)
	assert.Equal(t, store.TeamLayoutRosterOnly, page.Team.Layout)
	assert.Empty(t, page.Matches)
	assert.Nil(t, page.Rank)
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

func TestGetPlayer(t *testing.T) {
	db, _, sc := setup(t)
	ctx := context.Background()
	page, err := store.GetPlayer(ctx, db, sc.PlayerA7.ID)
	require.NoError(t, err)
	assert.Nil(t, page.Player.Role)
	assert.Equal(
		t,
		sc.StaffA.ID,
		page.Prev.Player.ID,
		"the first player's prev wraps to the last (staff)",
	)
	assert.Equal(t, ptr(domain.I18nText{Ja: "マネージャ", En: "Manager"}), page.Prev.Role)
	assert.Equal(t, sc.PlayerA10.ID, page.Next.Player.ID)
	assert.Equal(t, "アルファ", page.Team.Name.Ja)
	assert.Equal(t, domain.StatsUltimate, page.StatsKind)
	assert.Equal(t, store.PlayerStats{Games: 1, Goals: 2, Callahans: 1, Blocks: 2}, page.Stats,
		"callahans count the player's callahan goals")
	require.Len(t, page.Events, 2)
	assert.Equal(t, 60, page.Events[0].ElapsedSec)
	assert.Equal(t, "A1", page.Events[0].Code)
	assert.Equal(t, sc.TeamB.ID, page.Events[0].Opponent.ID)
	assert.Equal(t, sc.PlayerA10.ID, page.Events[0].Assist.ID)
	assert.Nil(t, page.Events[1].Assist)

	staff, err := store.GetPlayer(ctx, db, sc.StaffA.ID)
	require.NoError(t, err)
	assert.Equal(t, sc.PlayerA7.ID, staff.Next.Player.ID, "the last wraps to the first")
	assert.Empty(t, staff.Events)
}

func TestGetGallery(t *testing.T) {
	db, _, sc := setup(t)
	page, err := store.GetGallery(context.Background(), db, sc.Gallery.ID)
	require.NoError(t, err)
	assert.Equal(t, "Day2-AM", page.Name)
	assert.Equal(
		t,
		[]store.Photo{
			{ThumbURL: "/g/mini/1.jpg", FullURL: "/g/1.jpg"},
			{ThumbURL: "/g/mini/2.jpg", FullURL: "/g/2.jpg"},
		},
		page.Photos,
	)
}

func TestReserveGeneratedIDRange(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	ctx := context.Background()
	b.Tournament(func(row *store.Tournament) { row.ID = 525 })
	huge := b.Tournament(func(row *store.Tournament) { row.ID = store.GeneratedIDFloor + 41 })
	require.NoError(t, store.ReserveGeneratedIDRange(ctx, db))

	// 参照サイトの ID より大きい既存の ID があればその次から
	assert.Equal(t, huge.ID+1, b.Tournament().ID)
	// 空の表は下限から
	gallery := b.Gallery(huge.ID)
	assert.Equal(t, int64(store.GeneratedIDFloor), gallery.ID)
	// 何度呼んでもシーケンスを戻さない
	b.Gallery(huge.ID)
	require.NoError(t, store.ReserveGeneratedIDRange(ctx, db))
	assert.Equal(t, int64(store.GeneratedIDFloor+2), b.Gallery(huge.ID).ID)
}
