package importer

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/importer/parse"
	"scoreview/api/internal/store"
	"scoreview/api/internal/testdb"
)

// collect525 は parse/testdata のフィクスチャから大会 525 の Bundle を作る。グループページは 2 つ
// (6623 トーナメント、6627 順位決定)、試合ページは 45277 だけ。
func collect525(t *testing.T) *Bundle {
	t.Helper()
	im, err := New(WithFetcher(&fileFetcher{files: fixtures525()}))
	require.NoError(t, err)
	b, err := im.Collect(context.Background(), 525)
	require.NoError(t, err)
	return b
}

// withPlaceholderRow は未確定チームの行を確かめるため、スケジュールに試合 ID のない未試合
// "[A1-勝ち]" 対 "[A2-負け]" の行を足す(フィクスチャの 525 は全試合が終わっている)。
func withPlaceholderRow(b *Bundle) parse.MatchRow {
	s := b.Schedules[949]
	row := s.Rows[len(s.Rows)-1]
	row.Code, row.Time = "Z9", ptr("17:00")
	row.MatchID, row.Status, row.HasDetail = nil, "scheduled", true
	row.LeftScore, row.RightScore = nil, nil
	row.Left = parse.MatchRowSide{Placeholder: &parse.Placeholder{Code: "A1", Outcome: "win"}}
	row.Right = parse.MatchRowSide{Placeholder: &parse.Placeholder{Code: "A2", Outcome: "lose"}}
	s.Rows = append(s.Rows, row)
	return row
}

func TestDBSinkWritesReadableTournament(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	b := collect525(t)
	placeholderRow := withPlaceholderRow(b)

	var log bytes.Buffer
	require.NoError(t, DBSink{DB: db, Log: &log}.Write(ctx, b))
	assert.Contains(t, log.String(), "tournament 525 written")

	top, err := store.GetTournamentTop(ctx, db, 525)
	require.NoError(t, err)
	assert.Equal(t, domain.I18nText{
		Ja: "2026 Atami Beach Ultimate Beach Line Tournament",
		En: "2026 Atami Beach Ultimate Beach Line Tournament",
	}, top.Frame.Name)
	assert.Equal(t, "2026 Atami-B", top.Frame.ShortName, "試合ページのパンくずから")
	require.Len(t, top.Frame.Divisions, 1)
	assert.Equal(t, domain.I18nText{Ja: "ビーチミックス", En: "Beach Mixed"}, top.Frame.Divisions[0].Name)
	assert.Len(t, top.Frame.Divisions[0].Groups, 8, "ナビの 8 グループ(ページのないグループも作る)")
	require.Len(t, top.Frame.Galleries, 2)
	assert.Equal(t, int64(362), top.Frame.Galleries[0].ID)

	teamList, err := store.GetTeamList(ctx, db, 525)
	require.NoError(t, err)
	require.Len(t, teamList.Divisions, 1)
	assert.Equal(t, domain.DivisionMixed, teamList.Divisions[0].Kind, "日程の MC_MIX から")
	// フィクスチャのチーム一覧は 3 チームだけ。他のページに出るチームはその後ろに足される
	wantTeams := b.TeamList.Divisions[0].Entries
	require.Greater(t, len(teamList.Divisions[0].Teams), len(wantTeams))
	for i, want := range wantTeams {
		got := teamList.Divisions[0].Teams[i]
		assert.Equal(t, *want.ID, got.ID, "チーム一覧の並び")
		assert.Equal(t, domain.I18nText{Ja: want.Name.JA, En: want.Name.EN}, got.Name, "正式名の日英")
	}

	standings, err := store.GetDivisionStandings(ctx, db, 949)
	require.NoError(t, err)
	wantRanks := map[int64]*int{}
	for _, r := range b.Standings[949].Rows {
		wantRanks[r.TeamID] = r.Rank
	}
	// フィクスチャの順位表は 4 チームだけ。それ以外のチームは順位なし
	require.Len(t, standings.Standings, len(teamList.Divisions[0].Teams))
	for _, row := range standings.Standings {
		assert.Equal(t, wantRanks[row.Team.ID], row.Rank, "チーム %d の順位", row.Team.ID)
	}

	assertSchedule(t, db, b, placeholderRow)
	assertMatch45277(t, db, b.Matches[45277])
	assertBracket6623(t, db, b.Groups[6623])

	gallery, err := store.GetGallery(ctx, db, 362)
	require.NoError(t, err)
	assert.Len(t, gallery.Photos, len(b.Galleries[362].Photos))
}

func assertSchedule(t *testing.T, db *gorm.DB, b *Bundle, placeholderRow parse.MatchRow) {
	t.Helper()
	all := ""
	schedule, err := store.GetDivisionSchedule(
		context.Background(), db, 949, store.ScheduleFilter{YMD: &all})
	require.NoError(t, err)
	// トーナメント表にだけ出る試合(フィクスチャに試合ページのないもの)も同じディビジョンに入る
	want := b.Schedules[949].Rows
	require.GreaterOrEqual(t, len(schedule.Rows), len(want))
	byCode := map[string]domain.ScheduleRow{}
	for _, row := range schedule.Rows {
		byCode[row.Code] = row
	}
	for _, w := range want {
		got, ok := byCode[w.Code]
		require.True(t, ok, "試合 %s", w.Code)
		assert.Equal(t, w.LeftScore, got.Left.Score, "%s のスコア(左)", w.Code)
		assert.Equal(t, w.RightScore, got.Right.Score, "%s のスコア(右)", w.Code)
		assert.Equal(t, domain.MatchStatus(w.Status), got.Status, w.Code)
		if w.MatchID != nil {
			assert.Equal(t, *w.MatchID, got.ID, "参照サイトの試合 ID のまま")
		} else {
			assert.GreaterOrEqual(t, got.ID, int64(store.GeneratedIDFloor), "ID のない試合は自動採番")
		}
		if w.Left.TeamID != nil {
			require.NotNil(t, got.Left.Team, w.Code)
			assert.Equal(t, *w.Left.TeamID, got.Left.Team.ID)
			assert.Equal(t, w.Left.ShortName.JA, got.Left.Team.Name.Ja, "短縮名")
			assert.Equal(t, w.Left.ShortName.EN, got.Left.Team.Name.En, "短縮名(英)")
		}
	}
	got := byCode[placeholderRow.Code]
	assert.Nil(t, got.Left.Team)
	assert.Equal(t,
		&domain.Placeholder{Code: "A1", Outcome: domain.OutcomeWin}, got.Left.Placeholder)
	assert.Equal(t,
		&domain.Placeholder{Code: "A2", Outcome: domain.OutcomeLose}, got.Right.Placeholder)
	assert.Equal(t, []string{"2025-10-18"}, schedule.Dates, "年は日付選択肢から")
}

func assertMatch45277(t *testing.T, db *gorm.DB, page *parse.Match) {
	t.Helper()
	match, err := store.GetMatch(context.Background(), db, 45277)
	require.NoError(t, err)
	assert.Equal(t, page.Code, match.Match.Code)
	assert.Equal(t, &page.Date, match.Match.Date, "試合ページの YYYY/MM/DD")
	require.NotNil(t, match.Match.GroupName)
	assert.Equal(t, page.GroupName, *match.Match.GroupName)
	require.Len(t, match.Events, len(page.Events))
	last := match.Events[len(match.Events)-1]
	assert.Equal(t, *page.Left.Score, last.LeftScore, "イベントの累計が試合のスコアと一致する")
	assert.Equal(t, *page.Right.Score, last.RightScore)
	for i, ev := range page.Events {
		if ev.Score != nil && ev.Side != nil {
			score := match.Events[i].LeftScore
			if *ev.Side == "right" {
				score = match.Events[i].RightScore
			}
			assert.Equal(t, *ev.Score, score, "イベント %d 時点の累計", ev.Seq)
		}
	}
	assert.Len(t, match.Lineups.Left, len(page.Players.Left))
	assert.Equal(t, page.Stats.Left.FromDefence, match.Stats.Left.FromDefense)
}

func assertBracket6623(t *testing.T, db *gorm.DB, page *parse.Group) {
	t.Helper()
	group, err := store.GetGroup(context.Background(), db, 6623)
	require.NoError(t, err)
	require.NotNil(t, group.Bracket)
	want := page.Bracket
	assert.Equal(t, want.Pattern, group.Bracket.Pattern)
	assert.Equal(t, want.Columns, group.Bracket.Columns)
	require.Len(t, group.Bracket.Rows, len(want.Rows))
	for r, row := range want.Rows {
		for c, cell := range row {
			got := group.Bracket.Rows[r][c]
			assert.Equal(t, domain.BracketCellKind(cell.Kind), got.Kind, "セル %d,%d", r, c)
			assert.Equal(t, cell.MatchID, got.MatchID, "セル %d,%d の試合", r, c)
			assert.Equal(t, cell.Bottom, got.Borders.Bottom, "セル %d,%d の罫線", r, c)
			if cell.TeamID != nil {
				require.NotNil(t, got.Team, "セル %d,%d のチーム", r, c)
				assert.Equal(t, *cell.TeamID, got.Team.ID)
			}
		}
	}
}

func TestDBSinkReimportDoesNotDuplicate(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	counts := func() map[string]int64 {
		out := map[string]int64{}
		for name, model := range map[string]any{
			"tournaments": &store.Tournament{}, "divisions": &store.Division{},
			"groups": &store.Group{},
			"teams":  &store.Team{}, "players": &store.Player{}, "matches": &store.Match{},
			"match_events": &store.MatchEvent{}, "match_player_stats": &store.MatchPlayerStat{},
			"group_teams": &store.GroupTeam{}, "galleries": &store.Gallery{},
			"gallery_photos":     &store.GalleryPhoto{},
			"tournament_banners": &store.TournamentBanner{},
		} {
			var n int64
			require.NoError(t, db.Model(model).Count(&n).Error)
			out[name] = n
		}
		return out
	}

	b := collect525(t)
	withPlaceholderRow(b)
	require.NoError(t, DBSink{DB: db}.Write(ctx, b))
	first := counts()
	assert.Positive(t, first["matches"])
	assert.Positive(t, first["match_events"])

	require.NoError(t, DBSink{DB: db}.Write(ctx, b))
	assert.Equal(t, first, counts(), "同じ大会を取り込み直しても行が増えない")
}
