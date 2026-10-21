package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/bracket"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

var (
	admin = domain.Actor{UserID: 1, Role: domain.RoleAdmin}
	staff = domain.Actor{UserID: 2, Role: domain.RoleStaff}
	t0    = time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
)

func sidePtr(value domain.Side) *domain.Side { return &value }

func goal(side domain.Side, scorer, assist *store.Player) store.EventInput {
	input := store.EventInput{
		NewEvent: domain.NewEvent{Kind: domain.EventGoal, Side: sidePtr(side)},
	}
	if scorer != nil {
		input.ScorerID = &scorer.ID
	}
	if assist != nil {
		input.AssistID = &assist.ID
	}
	return input
}

func endEvent(winner *domain.Side) store.EventInput {
	return store.EventInput{NewEvent: domain.NewEvent{Kind: domain.EventEnd}, WinnerSide: winner}
}

type matchFixture struct {
	db       *gorm.DB
	division store.Division
	left     store.Team
	right    store.Team
	match    store.Match
	l7, l10  store.Player
	r1       store.Player
}

func newMatchFixture(t *testing.T) matchFixture {
	t.Helper()
	db := testdb.New(t)
	b := storetest.New(t, db)
	tour := b.Tournament()
	division := b.Division(tour.ID)
	left := b.Team(division, func(row *store.Team) { row.NameShortJa = "左" })
	right := b.Team(division, func(row *store.Team) { row.NameShortJa = "右" })
	return matchFixture{
		db: db, division: division, left: left, right: right,
		match: b.Match(division, &left, &right, func(row *store.Match) { row.Code = "A1" }),
		l7:    b.Player(left.ID, func(row *store.Player) { row.Number = "7" }),
		l10:   b.Player(left.ID, func(row *store.Player) { row.Number = "10" }),
		r1:    b.Player(right.ID),
	}
}

func TestAddMatchEventRecomputesScoreAndStatus(t *testing.T) {
	f := newMatchFixture(t)
	ctx := context.Background()
	pull := store.EventInput{
		NewEvent: domain.NewEvent{Kind: domain.EventPull, Side: sidePtr(domain.SideRight)},
	}

	detail, err := store.AddMatchEvent(ctx, f.db, staff, f.match.ID, pull, t0)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchLive, detail.Match.Status, "最初のイベントで試合中になる")
	assert.Equal(t, 0, *detail.Match.Left.Score)
	assert.Equal(t, 0, detail.Events[0].ElapsedSec)

	// 経過秒を省くと、最初のイベントの記録時刻からの経過になる
	detail, err = store.AddMatchEvent(
		ctx,
		f.db,
		staff,
		f.match.ID,
		goal(domain.SideLeft, &f.l7, &f.l10),
		t0.Add(95*time.Second),
	)
	require.NoError(t, err)
	assert.Equal(t, 95, detail.Events[1].ElapsedSec)
	explicit := goal(domain.SideRight, &f.r1, nil)
	explicit.ElapsedSec = new(int)
	*explicit.ElapsedSec = 300
	detail, err = store.AddMatchEvent(ctx, f.db, staff, f.match.ID, explicit, t0.Add(time.Hour))
	require.NoError(t, err)
	detail, err = store.AddMatchEvent(
		ctx,
		f.db,
		staff,
		f.match.ID,
		goal(domain.SideLeft, &f.l10, nil),
		t0.Add(time.Hour),
	)
	require.NoError(t, err)
	assert.Equal(t, 2, *detail.Match.Left.Score)
	assert.Equal(t, 1, *detail.Match.Right.Score)
	last := detail.Events[len(detail.Events)-1]
	assert.Equal(t, [2]int{2, 1}, [2]int{last.LeftScore, last.RightScore})
	assert.Equal(t, "10", last.Scorer.Number)

	// 得点者は得点した側のチームの選手であること
	_, err = store.AddMatchEvent(
		ctx,
		f.db,
		staff,
		f.match.ID,
		goal(domain.SideLeft, &f.r1, nil),
		t0,
	)
	assert.ErrorIs(t, err, domain.ErrBadRequest)

	// 得点を取り消すとスコアが戻る
	detail, err = store.RemoveMatchEvent(ctx, f.db, staff, f.match.ID, last.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, *detail.Match.Left.Score)

	detail, err = store.AddMatchEvent(
		ctx,
		f.db,
		staff,
		f.match.ID,
		endEvent(nil),
		t0.Add(time.Hour),
	)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchFinished, detail.Match.Status)
	assert.Nil(t, detail.Match.WinnerSide, "1-1 のプールの試合は引き分け")

	// 終了した試合には足せない。終了の取り消しは admin だけ
	_, err = store.AddMatchEvent(ctx, f.db, staff, f.match.ID, goal(domain.SideLeft, nil, nil), t0)
	assert.ErrorIs(t, err, domain.ErrConflict)
	endID := detail.Events[len(detail.Events)-1].ID
	_, err = store.RemoveMatchEvent(ctx, f.db, staff, f.match.ID, endID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
	detail, err = store.RemoveMatchEvent(ctx, f.db, admin, f.match.ID, endID)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchLive, detail.Match.Status)

	_, err = store.RemoveMatchEvent(ctx, f.db, admin, f.match.ID, 999999)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestChangeMatchStatus(t *testing.T) {
	f := newMatchFixture(t)
	ctx := context.Background()

	// 得点経過の無い試合は、スコアを直接入れて終了できる(詳細なしの試合)
	_, err := store.ChangeMatchStatus(ctx, f.db, staff, f.match.ID,
		store.StatusInput{Status: domain.MatchFinished})
	assert.ErrorIs(t, err, domain.ErrBadRequest, "スコアが無いと終了できない")
	detail, err := store.ChangeMatchStatus(ctx, f.db, staff, f.match.ID, store.StatusInput{
		Status: domain.MatchFinished, Score: &domain.Score{Left: 3, Right: 9},
	})
	require.NoError(t, err)
	assert.Equal(t, 9, *detail.Match.Right.Score)

	_, err = store.ChangeMatchStatus(
		ctx,
		f.db,
		staff,
		f.match.ID,
		store.StatusInput{Status: domain.MatchLive},
	)
	assert.ErrorIs(t, err, domain.ErrForbidden)
	detail, err = store.ChangeMatchStatus(
		ctx,
		f.db,
		admin,
		f.match.ID,
		store.StatusInput{Status: domain.MatchLive},
	)
	require.NoError(t, err)
	assert.Equal(t, domain.MatchLive, detail.Match.Status)
	_, err = store.ChangeMatchStatus(
		ctx,
		f.db,
		admin,
		f.match.ID,
		store.StatusInput{Status: domain.MatchScheduled},
	)
	assert.ErrorIs(t, err, domain.ErrConflict)

	// 得点経過がある試合のスコアは経過から決まる
	_, err = store.AddMatchEvent(ctx, f.db, staff, f.match.ID, goal(domain.SideLeft, nil, nil), t0)
	require.NoError(t, err)
	_, err = store.ChangeMatchStatus(ctx, f.db, staff, f.match.ID, store.StatusInput{
		Status: domain.MatchFinished, Score: &domain.Score{Left: 5, Right: 0},
	})
	assert.ErrorIs(t, err, domain.ErrBadRequest)
	detail, err = store.ChangeMatchStatus(
		ctx,
		f.db,
		staff,
		f.match.ID,
		store.StatusInput{Status: domain.MatchFinished},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, *detail.Match.Left.Score)
	assert.Equal(t, 0, *detail.Match.Right.Score)
}

func TestReplaceMatchPlayerStats(t *testing.T) {
	f := newMatchFixture(t)
	ctx := context.Background()
	detail, err := store.ReplaceMatchPlayerStats(ctx, f.db, f.match.ID, []store.PlayerStatFields{
		{PlayerID: f.l7.ID, Side: domain.SideLeft, Blocks: 2},
		{PlayerID: f.r1.ID, Side: domain.SideRight, Blocks: 1, Greatests: 1},
	})
	require.NoError(t, err)
	assert.Len(t, detail.PlayerStats, 2)
	_, err = store.ReplaceMatchPlayerStats(ctx, f.db, f.match.ID, []store.PlayerStatFields{
		{PlayerID: f.r1.ID, Side: domain.SideLeft, Blocks: 1},
	})
	assert.ErrorIs(t, err, domain.ErrBadRequest, "左のチームではない選手")
	_, err = store.ReplaceMatchPlayerStats(ctx, f.db, f.match.ID, []store.PlayerStatFields{
		{PlayerID: f.l7.ID, Side: domain.SideLeft, Blocks: -1},
	})
	assert.ErrorIs(t, err, domain.ErrBadRequest)
}

type bracketFixture struct {
	db      *gorm.DB
	group   store.Group
	teams   []store.Team
	matches map[string]store.MatchRecord
}

func newBracketFixture(t *testing.T) bracketFixture {
	t.Helper()
	db := testdb.New(t)
	b := storetest.New(t, db)
	division := b.Division(b.Tournament().ID)
	var teams []store.Team
	seeds := make([]store.MatchSideFields, 8)
	for i := range 8 {
		team := b.Team(
			division,
			func(row *store.Team) { row.NameShortJa = fmt.Sprintf("T%d", i+1) },
		)
		teams = append(teams, team)
		seeds[i] = store.MatchSideFields{TeamID: &teams[i].ID}
	}
	created, err := store.CreateGroup(context.Background(), db, division.ID, store.GroupFields{
		Name: "Tounament A", Kind: domain.GroupBracket, Linked: true, Published: true,
	})
	require.NoError(t, err)
	record, err := store.ApplyBracketPlan(
		context.Background(),
		db,
		created.ID,
		bracket.EightTeams().Plan(),
		seeds,
		"B",
	)
	require.NoError(t, err)
	assert.Equal(t, "T8", record.Bracket.Pattern)
	assert.Equal(t, "T8 =Tounament A=", record.Bracket.PtnName)
	assert.Len(t, record.Teams, 8)
	group := store.Group{ID: created.ID, DivisionID: division.ID}
	return bracketFixture{
		db:      db,
		group:   group,
		teams:   teams,
		matches: matchesByCode(t, db, division.ID),
	}
}

func matchesByCode(t *testing.T, db *gorm.DB, divisionID int64) map[string]store.MatchRecord {
	t.Helper()
	list, err := store.ListMatches(context.Background(), db, divisionID)
	require.NoError(t, err)
	out := map[string]store.MatchRecord{}
	for _, match := range list {
		out[match.Code] = match
	}
	return out
}

func (f bracketFixture) cellTeam(t *testing.T, row, col int) *int64 {
	t.Helper()
	page, err := store.GetGroup(context.Background(), f.db, f.group.ID)
	require.NoError(t, err)
	require.NotNil(t, page.Bracket)
	if team := page.Bracket.Rows[row][col].Team; team != nil {
		return &team.ID
	}
	return nil
}

func TestApplyBracketPlanCreatesMatchesAndPlaceholders(t *testing.T) {
	f := newBracketFixture(t)
	// 参照サイトの T8 全順位型(決勝が 1、1 回戦が 4〜7、3 位決定戦が 8)
	require.Len(t, f.matches, 12)
	b4 := f.matches["B4"]
	assert.Equal(t, f.teams[0].ID, b4.Left.Team.ID)
	assert.Equal(t, f.teams[1].ID, b4.Right.Team.ID)
	assert.Equal(
		t,
		&domain.Placeholder{Code: "B4", Outcome: domain.OutcomeWin},
		f.matches["B2"].Left.Placeholder,
	)
	assert.Equal(
		t,
		&domain.Placeholder{Code: "B3", Outcome: domain.OutcomeWin},
		f.matches["B1"].Right.Placeholder,
	)
	assert.Equal(
		t,
		&domain.Placeholder{Code: "B2", Outcome: domain.OutcomeLose},
		f.matches["B8"].Left.Placeholder,
	)
	assert.Nil(t, f.matches["B2"].Left.Team)

	// 格子: 1 回戦のチーム名と、未試合の game セルの試合 CD
	assert.Equal(t, f.teams[0].ID, *f.cellTeam(t, 2, 0))
	page, err := store.GetGroup(context.Background(), f.db, f.group.ID)
	require.NoError(t, err)
	assert.Equal(t, "B4", page.Bracket.Rows[3][0].Text)
	assert.Equal(t, b4.ID, *page.Bracket.Rows[3][0].MatchID)
	assert.Len(t, page.Matches, 12)

	// 試合があるグループには作り直せない
	seeds := make([]store.MatchSideFields, 8)
	for i := range seeds {
		seeds[i] = store.MatchSideFields{TeamID: &f.teams[i].ID}
	}
	_, err = store.ApplyBracketPlan(
		context.Background(),
		f.db,
		f.group.ID,
		bracket.EightTeams().Plan(),
		seeds,
		"C",
	)
	assert.ErrorIs(t, err, domain.ErrConflict)
}

func TestApplyBracketPlanAcceptsPlaceholderSeeds(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	division := b.Division(b.Tournament().ID)
	teamA, teamB := b.Team(division), b.Team(division)
	group, err := store.CreateGroup(context.Background(), db, division.ID, store.GroupFields{
		Name: "1st placement", Kind: domain.GroupBracket,
	})
	require.NoError(t, err)
	seeds := []store.MatchSideFields{
		{TeamID: &teamA.ID},
		{Placeholder: &domain.Placeholder{Code: "A9", Outcome: domain.OutcomeWin}},
		{TeamID: &teamB.ID},
		{Placeholder: &domain.Placeholder{Code: "A10", Outcome: domain.OutcomeWin}},
	}
	record, err := store.ApplyBracketPlan(
		context.Background(),
		db,
		group.ID,
		bracket.FourTeams().Plan(),
		seeds,
		"S",
	)
	require.NoError(t, err)
	assert.Len(t, record.Teams, 2, "確定したチームだけがグループのチームになる")
	matches := matchesByCode(t, db, division.ID)
	assert.Equal(
		t,
		&domain.Placeholder{Code: "A9", Outcome: domain.OutcomeWin},
		matches["S2"].Right.Placeholder,
	)

	_, err = store.ApplyBracketPlan(
		context.Background(),
		db,
		group.ID,
		bracket.FourTeams().Plan(),
		seeds[:3],
		"T",
	)
	assert.ErrorIs(t, err, domain.ErrBadRequest)
}

func finish(
	t *testing.T,
	db *gorm.DB,
	matchID int64,
	left, right int,
	winner *domain.Side,
) store.MatchDetail {
	t.Helper()
	ctx := context.Background()
	for range left {
		_, err := store.AddMatchEvent(ctx, db, staff, matchID, goal(domain.SideLeft, nil, nil), t0)
		require.NoError(t, err)
	}
	for range right {
		_, err := store.AddMatchEvent(ctx, db, staff, matchID, goal(domain.SideRight, nil, nil), t0)
		require.NoError(t, err)
	}
	detail, err := store.AddMatchEvent(ctx, db, staff, matchID, endEvent(winner), t0)
	require.NoError(t, err)
	return detail
}

func TestFinishingAMatchAdvancesWinnerAndLoser(t *testing.T) {
	f := newBracketFixture(t)
	ctx := context.Background()
	finish(t, f.db, f.matches["B4"].ID, 3, 5, nil)
	finish(t, f.db, f.matches["B5"].ID, 6, 2, nil)

	got := matchesByCode(t, f.db, f.group.DivisionID)
	assert.Equal(t, f.teams[1].ID, got["B2"].Left.Team.ID, "B4 の勝者(右)")
	assert.Equal(t, f.teams[2].ID, got["B2"].Right.Team.ID, "B5 の勝者(左)")
	// 格子: B4 の勝者のセル(準決勝の上のチーム)に勝者が入る
	assert.Equal(t, f.teams[1].ID, *f.cellTeam(t, 3, 1))

	// 準決勝が終わると決勝と 3 位決定戦が埋まる
	finish(t, f.db, f.matches["B6"].ID, 1, 0, nil)
	finish(t, f.db, f.matches["B7"].ID, 0, 1, nil)
	// 同点のトーナメントの試合は勝者の指定が必要(clone-spec C 2.3 の 4-4)
	b2 := f.matches["B2"].ID
	for range 4 {
		_, err := store.AddMatchEvent(ctx, f.db, staff, b2, goal(domain.SideLeft, nil, nil), t0)
		require.NoError(t, err)
		_, err = store.AddMatchEvent(ctx, f.db, staff, b2, goal(domain.SideRight, nil, nil), t0)
		require.NoError(t, err)
	}
	_, err := store.AddMatchEvent(ctx, f.db, staff, b2, endEvent(nil), t0)
	assert.ErrorIs(t, err, domain.ErrBadRequest)
	detail, err := store.AddMatchEvent(
		ctx,
		f.db,
		staff,
		b2,
		endEvent(sidePtr(domain.SideRight)),
		t0,
	)
	require.NoError(t, err)
	assert.Equal(t, domain.SideRight, *detail.Match.WinnerSide)

	got = matchesByCode(t, f.db, f.group.DivisionID)
	assert.Equal(t, f.teams[2].ID, got["B1"].Left.Team.ID, "B2 の勝者(4-4 で右)")
	assert.Equal(t, f.teams[1].ID, got["B8"].Left.Team.ID, "B2 の敗者は 3 位決定戦へ")
	assert.Equal(t, f.teams[0].ID, got["B10"].Left.Team.ID, "B4 の敗者は 5〜8 位の敗者戦へ")

	// 終了を取り消す(admin)と、まだ始まっていない後続試合のチームは空に戻る
	b4 := matchesByCode(t, f.db, f.group.DivisionID)["B4"]
	b4Detail, err := store.GetMatchDetail(ctx, f.db, b4.ID)
	require.NoError(t, err)
	endID := b4Detail.Events[len(b4Detail.Events)-1].ID
	_, err = store.RemoveMatchEvent(ctx, f.db, admin, b4.ID, endID)
	require.NoError(t, err)
	got = matchesByCode(t, f.db, f.group.DivisionID)
	assert.Equal(t, f.teams[1].ID, got["B2"].Left.Team.ID, "B2 は終了済みなので変えない")
	assert.Nil(t, got["B10"].Left.Team, "未試合の B10 は B4 の敗者が未確定に戻る")

	b3 := got["B3"]
	require.NotNil(t, b3.Left.Team)
	b6Detail, err := store.GetMatchDetail(ctx, f.db, got["B6"].ID)
	require.NoError(t, err)
	_, err = store.RemoveMatchEvent(
		ctx,
		f.db,
		admin,
		got["B6"].ID,
		b6Detail.Events[len(b6Detail.Events)-1].ID,
	)
	require.NoError(t, err)
	got = matchesByCode(t, f.db, f.group.DivisionID)
	assert.Nil(t, got["B3"].Left.Team, "未試合の B3 は B6 の勝者が未確定に戻る")
	assert.Nil(t, f.cellTeam(t, 11, 1))
}

func TestUpdateMatchKeepsScoreAndStatus(t *testing.T) {
	f := newMatchFixture(t)
	ctx := context.Background()
	_, err := store.AddMatchEvent(ctx, f.db, staff, f.match.ID, goal(domain.SideLeft, nil, nil), t0)
	require.NoError(t, err)
	date, clock, field := "2026-10-18", "13:30", "A"
	record, err := store.UpdateMatch(ctx, f.db, f.match.ID, store.MatchFields{
		Code:      "A1",
		Date:      &date,
		Time:      &clock,
		Field:     &field,
		HasDetail: true,
		Left: store.MatchSideFields{
			TeamID: &f.left.ID,
		},
		Right: store.MatchSideFields{TeamID: &f.right.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, domain.MatchLive, record.Status)
	assert.Equal(t, 1, *record.Left.Score)
	assert.Equal(t, "13:30", *record.Time)

	_, err = store.UpdateMatch(ctx, f.db, f.match.ID, store.MatchFields{
		Left: store.MatchSideFields{
			TeamID: &f.left.ID,
		},
		Right: store.MatchSideFields{TeamID: &f.left.ID},
	})
	assert.ErrorIs(t, err, domain.ErrBadRequest)
}
