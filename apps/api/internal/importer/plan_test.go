package importer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/importer/parse"
	"scoreview/api/internal/store"
)

// miniBundle は 1 ディビジョン(ID 10)に、グループ 20(プール)と 21(トーナメント)を持つ大会。
// チーム 1、2、3 がプールで総当たりし、トーナメントでは 1 と 2 が 4-4 で引き分け、2 が勝ち上がる。
func miniBundle() *Bundle {
	id := func(v int64) *int64 { return &v }
	score := func(v int) *int { return &v }
	team := func(tid int64, name string) parse.MatchRowSide {
		return parse.MatchRowSide{TeamID: id(tid), ShortName: parse.Text{JA: name}}
	}
	row := func(mid int64, code string, l, r int64, ls, rs int) parse.MatchRow {
		return parse.MatchRow{
			MatchID: id(mid), HasDetail: true, Status: "finished", Code: code, MonthDay: "05/01",
			Left: team(l, "T"+code+"L"), Right: team(r, "T"+code+"R"),
			LeftScore: score(ls), RightScore: score(rs),
		}
	}
	poolRows := []parse.MatchRow{
		row(101, "A1", 1, 2, 5, 0), row(102, "A2", 1, 3, 5, 0), row(103, "A3", 2, 3, 5, 0),
	}
	final := row(104, "F1", 1, 2, 4, 4)
	schedule := &parse.Schedule{
		Scope: "division", ScopeID: 10, Dates: []string{"2026-05-01"},
		Rows: append(append([]parse.MatchRow{}, poolRows...), final),
	}
	for i := range schedule.Rows {
		schedule.Rows[i].Date = ptr("2026-05-01")
		schedule.Rows[i].DivisionKind = "women"
	}
	// 計算では 1, 2, 3 位。参照サイトの表示は 1 位、空欄、2 位
	poolTeams := []parse.PoolTeam{
		{TeamID: id(1), RankRaw: "1", Rank: score(1)},
		{TeamID: id(2), RankRaw: ""},
		{TeamID: id(3), RankRaw: "2", Rank: score(2)},
	}
	pool := &parse.Group{
		Name: "Pool A", Kind: "pool", Matches: poolRows, Pool: &parse.Pool{Teams: poolTeams},
	}
	cell := func(kind string, tid, mid *int64) parse.BracketCell {
		return parse.BracketCell{Kind: kind, TeamID: tid, MatchID: mid}
	}
	bracket := &parse.Group{Name: "Final", Kind: "bracket", Bracket: &parse.Bracket{
		Pattern: "T2A", Columns: 3, Rows: [][]parse.BracketCell{
			{cell("team", id(1), nil), cell("", nil, nil), cell("", nil, nil)},
			{cell("", nil, nil), cell("game", nil, id(104)), cell("win", id(2), nil)},
			{cell("team", id(2), nil), cell("", nil, nil), cell("rank", nil, nil)},
		},
	}}
	return &Bundle{
		TournamentID: 9,
		Top: &parse.TournamentTop{
			ID: 9, Name: parse.Text{JA: "テスト大会", EN: "Test Cup"},
			Nav: parse.Nav{TeamListLabel: "teams", GroupLabel: "groups", ShowSchedule: true},
			Divisions: []parse.TopDivision{{
				ID: id(10), Name: parse.Text{JA: "ウィメン", EN: "Women"},
				ScheduleLinked: true, StandingsLinked: true,
				Groups: []parse.NavGroup{
					{ID: id(20), Name: "Pool A", Linked: true},
					{ID: id(21), Name: "Final", Linked: true},
				},
			}},
		},
		TeamList: &parse.TeamList{Divisions: []parse.TeamListDivision{{
			Name: parse.Text{JA: "ウィメン"}, Count: 3, EntryKind: "team",
			Entries: []parse.TeamListEntry{
				{ID: id(1), Name: parse.Text{JA: "チーム一", EN: "Team One"}, BarStyle: "women"},
				{ID: id(2), Name: parse.Text{JA: "チーム二"}, BarStyle: "women"},
				{ID: id(3), Name: parse.Text{JA: "チーム三"}, BarStyle: "women"},
			},
		}}},
		Schedules: map[int64]*parse.Schedule{10: schedule},
		Standings: map[int64]*parse.DivisionStandings{},
		Groups:    map[int64]*parse.Group{20: pool, 21: bracket},
		Teams:     map[int64]*parse.Team{},
		Matches:   map[int64]*parse.Match{},
		Galleries: map[int64]*parse.Gallery{},
		Players:   map[int64]*parse.Player{},
	}
}

func findMatch(t *testing.T, imp *store.TournamentImport, id int64) store.Match {
	t.Helper()
	for _, d := range imp.Divisions {
		for _, m := range d.Matches {
			if m.ID == id {
				return m
			}
		}
	}
	t.Fatalf("match %d not found", id)
	return store.Match{}
}

func TestBuildPlanRankOverrideOnlyWhenDifferent(t *testing.T) {
	plan, err := BuildPlan(miniBundle())
	require.NoError(t, err)
	got := map[int64]*int{}
	for _, gt := range plan.Import.GroupTeams {
		assert.Equal(t, int64(20), gt.GroupID)
		got[gt.TeamID] = gt.RankOverride
	}
	require.Len(t, got, 3)
	assert.Nil(t, got[1], "計算と同じなら入れない")
	assert.Equal(t, ptr(0), got[2], "空欄は 0")
	assert.Equal(t, ptr(2), got[3], "計算(3 位)と違う表示")
}

func TestBuildPlanWinnerOfDrawnBracketMatch(t *testing.T) {
	plan, err := BuildPlan(miniBundle())
	require.NoError(t, err)
	final := findMatch(t, plan.Import, 104)
	require.NotNil(t, final.WinnerSide, "game セルの次の列の win セル")
	assert.Equal(t, domain.SideRight, *final.WinnerSide)
	assert.Nil(t, findMatch(t, plan.Import, 101).WinnerSide, "スコアに差があれば入れない")
	assert.Equal(t, int64(20), *findMatch(t, plan.Import, 101).GroupID, "グループの結果一覧の試合")
}

func TestBuildPlanTournamentAndDivision(t *testing.T) {
	plan, err := BuildPlan(miniBundle())
	require.NoError(t, err)
	tour := plan.Import.Tournament
	assert.Equal(t, domain.TournamentFinished, tour.Status, "一覧にない大会は [最終順位] のリンクから推測")
	assert.False(t, tour.Listed)
	require.NotNil(t, tour.StartDate)
	assert.Equal(t, "2026-05-01", tour.StartDate.Format("2006-01-02"))
	assert.True(t, slicesContainPrefix(plan.Warnings, "一覧に載らない大会"), plan.Warnings)

	div := plan.Import.Divisions[0]
	assert.Equal(t, domain.DivisionWomen, div.Division.Kind, "日程の MC_WOM から")
	assert.Equal(t, domain.StatsUltimate, div.Division.StatsKind)
	require.Len(t, div.Groups, 2)
	assert.Equal(t, domain.GroupBracket, div.Groups[1].Kind)
	layout, err := domain.ParseBracketLayout(div.Groups[1].BracketLayout)
	require.NoError(t, err)
	assert.Equal(t, domain.BracketRank, layout.Rows[2][2].Kind)
	require.Len(t, div.Teams, 3)
	assert.Equal(t, "TA1L", div.Teams[0].NameShortJa, "短縮名は日程の表記")
	assert.Equal(t, "Team One", div.Teams[0].NameFullEn)
}

func TestBuildPlanStatusWithoutStandingsLink(t *testing.T) {
	b := miniBundle()
	b.Top.Divisions[0].StandingsLinked = false
	plan, err := BuildPlan(b)
	require.NoError(t, err)
	assert.Equal(t, domain.TournamentEntryClosed, plan.Import.Tournament.Status)
}

func TestBuildPlanUsesMatchPageDateAndEvents(t *testing.T) {
	b := miniBundle()
	scorer, assist := int64(501), int64(502)
	left, right := "left", "right"
	b.Matches[101] = &parse.Match{
		Code: "A1", Date: "2025-05-01", Time: ptr("09:30"), Field: "B",
		Left:  parse.MatchSide{TeamID: 1, Score: ptr(1)},
		Right: parse.MatchSide{TeamID: 2, Score: ptr(0)},
		Events: []parse.MatchEvent{
			{Seq: 1, Kind: "pull", Side: &right},
			{
				Seq: 2, ElapsedSec: 60, Kind: "goal", Side: &left,
				ScorerID: &scorer, AssistID: &assist,
			},
			{Seq: 3, ElapsedSec: 70, Kind: "timeout"}, // ★ のないタイムアウトは飛ばす
			{Seq: 4, ElapsedSec: 90, Kind: "end"},
		},
		Players: parse.MatchPlayers{
			Left: []parse.MatchPlayer{
				{PlayerID: scorer, Number: "7", Blocks: ptr(2)}, {PlayerID: assist, Number: "8"},
			},
			Right: []parse.MatchPlayer{},
		},
	}
	plan, err := BuildPlan(b)
	require.NoError(t, err)
	m := findMatch(t, plan.Import, 101)
	assert.Equal(t, "2025-05-01", m.MatchDate.Format("2006-01-02"), "試合ページの年を正とする")
	assert.Equal(t, "09:30", *m.StartTime)
	assert.Equal(t, "B", *m.Field)
	require.Len(t, plan.Import.Events, 3)
	assert.Nil(t, plan.Import.Events[2].Side, "end は側なし")
	assert.Equal(t, &scorer, plan.Import.Events[1].ScorerID)
	require.Len(t, plan.Import.Players, 2, "チームページがなければ試合ページの選手を足す")
	require.Len(t, plan.Import.Stats, 2)
	assert.Equal(t, 2, plan.Import.Stats[0].Blocks)
	assert.True(t, slicesContainPrefix(plan.Warnings, "側(★)の分からないイベント"), plan.Warnings)
}

func TestBuildPlanGreatestsGoToFirstMatch(t *testing.T) {
	b := miniBundle()
	pid := int64(501)
	for _, mid := range []int64{102, 101} {
		b.Matches[mid] = &parse.Match{
			Code: "X", Date: "2026-05-01",
			Left: parse.MatchSide{TeamID: 1}, Right: parse.MatchSide{TeamID: 3},
			Players: parse.MatchPlayers{Left: []parse.MatchPlayer{{PlayerID: pid}}},
		}
	}
	b.Matches[101].Right.TeamID = 2
	b.Matches[101].Time, b.Matches[102].Time = ptr("09:00"), ptr("10:00")
	b.Players[pid] = &parse.Player{
		TeamID: 1, StatsKind: "ultimate", Stats: &parse.PlayerStats{Greatests: 3},
	}
	plan, err := BuildPlan(b)
	require.NoError(t, err)
	byMatch := map[int64]int{}
	for _, st := range plan.Import.Stats {
		byMatch[st.MatchID] = st.Greatests
	}
	assert.Equal(t, map[int64]int{101: 3, 102: 0}, byMatch, "通算値を最初の試合にまとめる(推測)")
}

func slicesContainPrefix(list []string, prefix string) bool {
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
