package storetest

import (
	"testing"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// Scenario は閲覧用の全ページを一通り描ける架空の大会。ID は空の DB に作れば毎回同じになる。
type Scenario struct {
	Tournament    store.Tournament // 終了、一覧に載る
	Unlisted      store.Tournament // 一覧に載らない(英語名なし)
	Upcoming      store.Tournament // 受付中
	Mixed         store.Division   // プール、ブラケット、画像のグループを持つ
	Women         store.Division
	Pool          store.Group
	Bracket       store.Group
	Image         store.Group
	Hidden        store.Group // published=false(ナビに出ない)
	TeamA         store.Team
	TeamB         store.Team
	TeamC         store.Team
	TeamD         store.Team
	TeamW1        store.Team
	TeamW2        store.Team
	PlayerA7      store.Player
	PlayerA10     store.Player
	StaffA        store.Player
	PlayerB1      store.Player
	MatchAB       store.Match // 終了 9-3、得点経過あり
	MatchCA       store.Match // 終了 5-5(引き分け)
	MatchBC       store.Match // 未試合
	MatchFinal    store.Match // 試合中、右が未確定 [A1-勝ち]
	MatchWomen    store.Match // 終了、詳細なし、コート "B"
	Gallery       store.Gallery
	BracketLayout domain.BracketLayout
}

func side(value domain.Side) *domain.Side { return &value }

// BuildScenario は Scenario を DB に作る。
func BuildScenario(tb testing.TB, builder *Builder) Scenario {
	tb.Helper()
	var sc Scenario
	sc.Tournament = builder.Tournament(func(row *store.Tournament) {
		row.NameJa, row.NameEn, row.ShortName = "2026 架空ビーチ大会", "2026 Fictional Beach", "2026 Beach"
		row.StartDate = Date(tb, "2026-10-17")
		row.LogoURL = Ptr("/logo.png")
		row.TopHTMLJa = `<p><span class="SCV_MIDA_M">【Result】</span><script>alert(1)</script></p>`
		row.TopHTMLEn = ""
	})
	sc.Unlisted = builder.Tournament(func(row *store.Tournament) {
		row.NameJa, row.NameEn, row.Listed = "非公開大会", "", false
	})
	sc.Upcoming = builder.Tournament(func(row *store.Tournament) {
		row.NameJa, row.NameEn, row.Status = "2027 架空カップ", "2027 Fictional Cup", domain.TournamentEntryOpen
		row.StartDate = Date(tb, "2027-05-01")
	})
	builder.Banner(
		sc.Tournament.ID,
		func(row *store.TournamentBanner) { row.ImageURL, row.SortOrder = "/b2.png", 2 },
	)
	builder.Banner(sc.Tournament.ID, func(row *store.TournamentBanner) {
		row.ImageURL, row.LinkURL, row.SortOrder = "/b1.png", Ptr("https://example.com/"), 1
	})

	sc.Mixed = builder.Division(sc.Tournament.ID, func(row *store.Division) {
		row.NameJa, row.NameEn, row.StandingsPublished = "ビーチミックス", "Beach Mixed", true
	})
	sc.Women = builder.Division(sc.Tournament.ID, func(row *store.Division) {
		row.NameJa, row.NameEn, row.Kind, row.SortOrder = "ウィメン", "Women's", domain.DivisionWomen, 1
	})

	buildTeams(tb, builder, &sc)
	buildGroups(tb, builder, &sc)
	buildMatches(tb, builder, &sc)

	sc.Gallery = builder.Gallery(sc.Tournament.ID, func(row *store.Gallery) {
		row.Name, row.NameEn = "Day2-AM", "---"
	})
	builder.Photo(
		sc.Gallery.ID,
		func(row *store.GalleryPhoto) { row.ThumbURL, row.FullURL = "/g/mini/1.jpg", "/g/1.jpg" },
	)
	builder.Photo(
		sc.Gallery.ID,
		func(row *store.GalleryPhoto) { row.ThumbURL, row.FullURL = "/g/mini/2.jpg", "/g/2.jpg" },
	)
	return sc
}

func buildTeams(tb testing.TB, builder *Builder, sc *Scenario) {
	tb.Helper()
	team := func(div store.Division, full, short, en string, order int, rank *int) store.Team {
		return builder.Team(div, func(row *store.Team) {
			row.NameFullJa, row.NameShortJa, row.NameFullEn, row.NameShortEn = full, short, en, en
			row.SortOrder, row.FinalRank = order, rank
			row.BarStyle = "mixed"
		})
	}
	sc.TeamA = team(sc.Mixed, "アルファ・スロー", "アルファ", "Alpha", 0, Ptr(2))
	sc.TeamB = team(sc.Mixed, "ブラボー", "ブラボー", "Bravo", 1, Ptr(1))
	sc.TeamC = team(sc.Mixed, "チャーリー", "チャーリー", "Charlie", 2, Ptr(3))
	sc.TeamD = team(sc.Mixed, "デルタ", "デルタ", "", 3, nil)
	sc.TeamW1 = team(sc.Women, "ウィスキー", "ウィスキー", "Whiskey", 0, nil)
	sc.TeamW2 = team(sc.Women, "ヤンキー", "ヤンキー", "Yankee", 1, nil)

	sc.PlayerA7 = builder.Player(sc.TeamA.ID, func(row *store.Player) {
		row.Number, row.NameJa, row.NameEn, row.Gender = "7", "架空 七郎", "Shichiro KAKU", Ptr("M")
	})
	sc.PlayerA10 = builder.Player(sc.TeamA.ID, func(row *store.Player) {
		row.Number, row.NameJa, row.NameEn, row.Gender = "10", "架空 十子", "", Ptr("F")
	})
	sc.StaffA = builder.Player(sc.TeamA.ID, func(row *store.Player) {
		row.Number, row.NameJa, row.NameEn = "", "架空 監督", "Coach KAKU"
		row.RoleJa, row.RoleEn = Ptr("マネージャ"), Ptr("Manager")
	})
	sc.PlayerB1 = builder.Player(sc.TeamB.ID, func(row *store.Player) {
		row.Number, row.NameJa, row.NameEn = "1", "架空 一太", "Ichita KAKU"
	})
}

func buildGroups(tb testing.TB, builder *Builder, sc *Scenario) {
	tb.Helper()
	sc.Pool = builder.Group(sc.Mixed.ID, domain.GroupPool)
	for i, team := range []store.Team{sc.TeamA, sc.TeamB, sc.TeamC} {
		builder.GroupTeam(sc.Pool.ID, team.ID, func(row *store.GroupTeam) { row.SortOrder = i })
	}
	sc.Image = builder.Group(sc.Mixed.ID, domain.GroupImage, func(row *store.Group) {
		row.Name, row.NameEn, row.SortOrder = "Temporary ranking", "Temporary ranking", 2
		row.ImageURL, row.ImageWidth, row.ImageHeight = Ptr("/ranking.jpg"), Ptr(500), Ptr(695)
	})
	sc.Hidden = builder.Group(sc.Mixed.ID, domain.GroupPool, func(row *store.Group) {
		row.Name, row.SortOrder, row.Published = "Hidden pool", 3, false
	})
	// ブラケットは試合 ID を参照するので、試合を作った後に格子を入れる(buildMatches)
	sc.Bracket = builder.Group(sc.Mixed.ID, domain.GroupBracket, func(row *store.Group) {
		row.Name, row.SortOrder, row.Linked = "Tounament A", 1, false
		row.BracketPattern, row.BracketPtnName = Ptr("T2A"), Ptr("Fictional =Tounament A=")
		row.BracketLayout = []byte(`{"columns":1,"rows":[]}`)
	})
	builder.GroupTeam(sc.Bracket.ID, sc.TeamD.ID)
}

func buildMatches(tb testing.TB, builder *Builder, sc *Scenario) {
	tb.Helper()
	inPool := func(row *store.Match) { row.GroupID = &sc.Pool.ID }
	sc.MatchAB = builder.Match(sc.Mixed, &sc.TeamA, &sc.TeamB, inPool, Finished(9, 3),
		At(tb, "2026-10-17", "09:00", "1"), func(row *store.Match) { row.Code = "A1" })
	sc.MatchCA = builder.Match(sc.Mixed, &sc.TeamC, &sc.TeamA, inPool, Finished(5, 5),
		At(tb, "2026-10-17", "10:00", "2"), func(row *store.Match) { row.Code = "A2" })
	sc.MatchBC = builder.Match(sc.Mixed, &sc.TeamB, &sc.TeamC, inPool,
		At(tb, "2026-10-18", "09:00", "1"), func(row *store.Match) { row.Code = "A3" })
	sc.MatchFinal = builder.Match(
		sc.Mixed,
		&sc.TeamD,
		nil,
		At(tb, "2026-10-18", "10:00", "2"),
		func(row *store.Match) {
			row.Code, row.GroupID, row.Status = "B1", &sc.Bracket.ID, domain.MatchLive
			row.LeftScore, row.RightScore = Ptr(2), Ptr(1)
			row.RightPlaceholderCode, row.RightPlaceholderOutcome = Ptr("A1"), Ptr(domain.OutcomeWin)
		},
	)
	sc.MatchWomen = builder.Match(
		sc.Women,
		&sc.TeamW1,
		&sc.TeamW2,
		Finished(7, 4),
		At(
			tb,
			"2026-10-18",
			"09:00",
			"B",
		),
		func(row *store.Match) { row.Code, row.HasDetail = "W1", false },
	)

	matchID := sc.MatchAB.ID
	builder.Event(matchID, domain.EventPull, side(domain.SideLeft), 0)
	builder.Goal(matchID, domain.SideLeft, 60, &sc.PlayerA7, &sc.PlayerA10)
	builder.Event(matchID, domain.EventTimeout, side(domain.SideRight), 90)
	builder.Goal(matchID, domain.SideRight, 120, &sc.PlayerB1, nil)
	builder.Goal(
		matchID,
		domain.SideLeft,
		180,
		&sc.PlayerA7,
		nil,
		func(ev *store.MatchEvent) { ev.Callahan = true },
	)
	builder.Event(matchID, domain.EventEnd, nil, 1200)
	builder.Stat(
		matchID,
		sc.PlayerA7.ID,
		domain.SideLeft,
		func(row *store.MatchPlayerStat) { row.Blocks = 2 },
	)
	builder.Stat(matchID, sc.PlayerA10.ID, domain.SideLeft)
	builder.Stat(
		matchID,
		sc.PlayerB1.ID,
		domain.SideRight,
		func(row *store.MatchPlayerStat) { row.Blocks = 1 },
	)

	sc.BracketLayout = domain.BracketLayout{Columns: 2, Rows: [][]domain.BracketCell{
		{{Kind: domain.BracketTxtYen, Text: "FINAL"}, {Kind: domain.BracketTxtSharp, Text: "Placement"}},
		{
			{Kind: domain.BracketTeam, TeamID: &sc.TeamD.ID, Borders: domain.BracketBorders{Bottom: true}},
			{},
		},
		{
			{
				Kind:    domain.BracketGame,
				MatchID: &sc.MatchFinal.ID,
				Borders: domain.BracketBorders{Top: true, Right: true},
			},
			{Kind: domain.BracketWin, Borders: domain.BracketBorders{Left: true, Bottom: true}},
		},
		{
			{
				Kind:    domain.BracketTeam,
				Text:    "[A1-勝ち]",
				Borders: domain.BracketBorders{Right: true, Bottom: true},
			},
			{},
		},
	}}
	data, err := domain.MarshalBracketLayout(sc.BracketLayout)
	if err != nil {
		tb.Fatalf("storetest: marshal bracket: %v", err)
	}
	err = builder.DB().
		Model(&store.Group{}).
		Where("id = ?", sc.Bracket.ID).
		Update("bracket_layout", data).
		Error
	if err != nil {
		tb.Fatalf("storetest: update bracket layout: %v", err)
	}
	sc.Bracket.BracketLayout = data
}
