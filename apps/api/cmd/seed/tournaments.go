package main

import (
	"fmt"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// このファイルは投入する大会ごとの中身。参照サイトで確認したバリエーション(clone-spec E 11)を
// 一通り含める。名前はすべて架空。

const beachTopHTML = `<p><span class="SCV_MIDA_M" style="background-color: rgb(255, 102, 0); ` +
	`font-family: &quot;ヒラギノ角ゴ Pro W3&quot;, Meiryo, sans-serif; color: rgb(255, 255, 255); ` +
	`padding: 4px 10px; border-radius: 4px;">【Web Site】</span></p>` +
	`<p><a href="https://example.com/fictional-beach" target="_blank">https://example.com/fictional-beach</a></p>` +
	`<p><span class="SCV_MIDA_M">【Result】</span><br><strong>■1st place：ブラボー</strong><br>` +
	`<img src="https://placehold.co/600x400?text=1st" style="width: 600px; height: 400px;"></p>`

func nato(names ...string) []teamName {
	out := make([]teamName, 0, len(names))
	for _, name := range names {
		out = append(out, teamName{full: name, short: name, en: name})
	}
	return out
}

// beachLeague は終了した主大会。ディビジョン 4 種(mixed/men/women/none)、プール、
// トーナメント表、画像だけのグループ、未確定チーム、試合中、詳細なし、引き分けを含む。
func (s *seeder) beachLeague() error {
	tour, err := s.tournament(store.Tournament{
		NameJa: "2026 架空ビーチリーグ", NameEn: "2026 Fictional Beach League", ShortName: "2026 F-Beach",
		Status: domain.TournamentFinished, Listed: true, StartDate: date("2026-10-17"),
		LogoURL: ptr(
			image(600, 213, "Fictional+Beach"),
		), TopHTMLJa: beachTopHTML, TopHTMLEn: beachTopHTML,
		ShowSchedule: true,
	})
	if err != nil {
		return err
	}
	for idx := 1; idx <= 3; idx++ {
		banner := store.TournamentBanner{
			TournamentID: tour.ID, ImageURL: image(160, 64, fmt.Sprintf("Banner+%d", idx)), SortOrder: idx,
		}
		if idx == 1 {
			banner.LinkURL = ptr("https://example.com/")
		}
		if err := s.create(&banner); err != nil {
			return err
		}
	}
	for idx, session := range []string{"Day1-AM", "Day2-PM"} {
		gallery := store.Gallery{TournamentID: tour.ID, Name: session, SortOrder: idx}
		if err := s.create(&gallery); err != nil {
			return err
		}
		for num := 1; num <= 6; num++ {
			text := fmt.Sprintf("%s+%d", session, num)
			photo := store.GalleryPhoto{
				GalleryID: gallery.ID, ThumbURL: image(120, 80, text), FullURL: image(945, 630, text), SortOrder: num,
			}
			if err := s.create(&photo); err != nil {
				return err
			}
		}
	}
	if err := s.beachMixed(tour); err != nil {
		return err
	}
	return s.beachOthers(tour)
}

func (s *seeder) beachMixed(tour store.Tournament) error {
	mixed, err := s.division(store.Division{
		TournamentID: tour.ID, NameJa: "ビーチミックス", NameEn: "Beach Mixed", Kind: domain.DivisionMixed,
		ScheduleEnabled: true, StandingsPublished: true,
	})
	if err != nil {
		return err
	}
	names := []teamName{
		{full: "アルファ・スローズ", short: "アルファ", en: "Alpha Throws"},
		{full: "ブラボー", short: "ブラボー", en: "Bravo"},
		{full: "チャーリー", short: "チャーリー", en: "Charlie"},
		{full: "架空によるデルタ向上委員会", short: "デルタ向上委員会", en: "Delta"},
		{full: "エコー", short: "エコー", en: "Echo"},
		{full: "フォックストロット", short: "フォックス", en: "Foxtrot"},
		{full: "ゴルフ", short: "ゴルフ", en: "Golf"},
		{full: "ホテル", short: "ホテル", en: ""},
	}
	teams, players, err := s.teams(mixed, names, "mixed", 7)
	if err != nil {
		return err
	}
	// 最終順位は順位決定戦の結果で、同順位(5, 5, 7, 7)を含む(clone-spec C 1.4)
	for idx, rank := range []int{2, 1, 3, 4, 5, 5, 7, 7} {
		if err := s.tx.Model(&teams[idx]).Update("final_rank", rank).Error; err != nil {
			return fmt.Errorf("set final rank: %w", err)
		}
	}

	poolA, err := s.group(store.Group{
		DivisionID: mixed.ID, Name: "Qualifying pool A", Kind: domain.GroupPool, Linked: true, Published: true,
	}, teams[:4])
	if err != nil {
		return err
	}
	// 0-1 と 2-3 以外は直接対決の例: 0 と 1 は勝ち点で並び、0 が直接対決に勝つ(clone-spec C 2.2)
	if err := s.roundRobin(mixed, poolA, teams[:4], "A", "2026-10-17", []matchSpec{
		finished(9, 3), finished(5, 5), finished(4, 6), finished(7, 8), finished(10, 2), finished(6, 6),
	}, players); err != nil {
		return err
	}
	poolB, err := s.group(store.Group{
		DivisionID: mixed.ID, Name: "Qualifying pool B", Kind: domain.GroupPool, Linked: true, Published: true,
		SortOrder: 1,
	}, teams[4:])
	if err != nil {
		return err
	}
	live := matchSpec{status: domain.MatchLive, scores: [2]int{3, 2}}
	if err := s.roundRobin(mixed, poolB, teams[4:], "C", "2026-10-17", []matchSpec{
		finished(8, 4), finished(6, 7), finished(9, 9), finished(5, 3), live, {status: domain.MatchScheduled},
	}, players); err != nil {
		return err
	}
	if err := s.bracket(mixed, teams, players); err != nil {
		return err
	}
	_, err = s.group(store.Group{
		DivisionID: mixed.ID, Name: "Temporary ranking", Kind: domain.GroupImage, Linked: true, Published: true,
		SortOrder: 3, ImageURL: ptr(image(500, 695, "Temporary+ranking")), ImageWidth: ptr(500), ImageHeight: ptr(695),
	}, nil)
	if err != nil {
		return err
	}
	// 非公開のグループ(ナビに出ない。参照サイトの欠番)
	_, err = s.group(store.Group{
		DivisionID: mixed.ID, Name: "Hidden pool", Kind: domain.GroupPool, Linked: true, Published: false, SortOrder: 4,
	}, nil)
	return err
}

// bracket は 8 チームのトーナメント表(準々決勝 4、準決勝 2、決勝 1)を作る。準決勝の 1 つは試合中、
// 決勝は片側が未確定 [B6-勝ち]。
func (s *seeder) bracket(
	mixed store.Division,
	teams []store.Team,
	players map[int64][]store.Player,
) error {
	group, err := s.group(store.Group{
		DivisionID: mixed.ID, Name: "Tounament A", Kind: domain.GroupBracket, Linked: true, Published: true,
		SortOrder: 2, BracketPattern: ptr("T8A"), BracketPtnName: ptr("Fictional Line =Tounament A="),
		BracketLayout: []byte(`{"columns":1,"rows":[]}`),
	}, teams)
	if err != nil {
		return err
	}
	at := func(code, clock, field string) matchSpec {
		return matchSpec{code: code, group: &group, day: "2026-10-18", clock: clock, field: field}
	}
	// 準々決勝: 0-7, 3-4, 1-6, 2-5(シード順)
	seeds := []int{0, 7, 3, 4, 1, 6, 2, 5}
	quarterScores := [][2]int{{11, 4}, {6, 7}, {9, 8}, {4, 4}}
	var quarter [4]store.Match
	var winners [4]*store.Team
	for idx := range quarter {
		spec := at(fmt.Sprintf("B%d", idx+1), fmt.Sprintf("%02d:00", 9+idx/2), fmt.Sprintf("%d", 1+idx%2))
		spec.left, spec.right = &teams[seeds[idx*2]], &teams[seeds[idx*2+1]]
		spec.status, spec.scores = domain.MatchFinished, quarterScores[idx]
		winner := spec.left
		if spec.scores[0] < spec.scores[1] {
			winner = spec.right
		}
		if spec.scores[0] == spec.scores[1] {
			// 同点でも勝者が決まる試合(clone-spec C 2.3 の 4-4)
			spec.winner = ptr(domain.SideRight)
			winner = spec.right
		}
		if quarter[idx], err = s.match(mixed, spec, players); err != nil {
			return err
		}
		winners[idx] = winner
	}
	semi1 := at("B5", "11:00", "1")
	semi1.left, semi1.right, semi1.status, semi1.scores = winners[0], winners[1], domain.MatchFinished, [2]int{
		10,
		7,
	}
	semi2 := at("B6", "11:00", "2")
	semi2.left, semi2.right, semi2.status, semi2.scores = winners[2], winners[3], domain.MatchLive, [2]int{
		4,
		5,
	}
	final := at("B7", "13:00", "1")
	final.left, final.status = winners[0], domain.MatchScheduled
	final.rightHolder = &domain.Placeholder{Code: "B6", Outcome: domain.OutcomeWin}
	var semis [2]store.Match
	for idx, spec := range []matchSpec{semi1, semi2} {
		if semis[idx], err = s.match(mixed, spec, players); err != nil {
			return err
		}
	}
	finalMatch, err := s.match(mixed, final, players)
	if err != nil {
		return err
	}

	layout := eightTeamLayout(teams, seeds, quarter, winners, semis, winners[0], finalMatch)
	data, err := domain.MarshalBracketLayout(layout)
	if err != nil {
		return fmt.Errorf("bracket layout: %w", err)
	}
	if err := s.tx.Model(&group).Update("bracket_layout", data).Error; err != nil {
		return fmt.Errorf("save bracket layout: %w", err)
	}
	return nil
}

// eightTeamLayout は 8 チームの勝ち抜き表の格子を作る。列は 1 回戦、準決勝、決勝、順位の 4 列で、
// 罫線だけで山を描く(clone-spec C 2.3)。上下 2 行の対戦は、上の行に下罫線、上の行の次から
// 下の行までに右罫線を引き、中央の行に game セル、次の列の中央の行に勝者(win)を置く。
func eightTeamLayout(
	teams []store.Team,
	seeds []int,
	quarter [4]store.Match,
	quarterWinners [4]*store.Team,
	semis [2]store.Match,
	semiWinner *store.Team,
	final store.Match,
) domain.BracketLayout {
	const columns, rows = 4, 16
	grid := make([][]domain.BracketCell, rows)
	for r := range grid {
		grid[r] = make([]domain.BracketCell, columns)
	}
	grid[0][0] = domain.BracketCell{Kind: domain.BracketTxtYen, Text: "1st Round"}
	grid[0][1] = domain.BracketCell{Kind: domain.BracketTxtYen, Text: "Semifinals"}
	grid[0][2] = domain.BracketCell{Kind: domain.BracketTxtYen, Text: "FINAL"}
	grid[0][3] = domain.BracketCell{Kind: domain.BracketTxtSharp, Text: "Placement"}

	// bout は column 列の top 行と bottom 行の対戦を描き、中央の行を返す
	bout := func(column, top, bottom int, matchID int64) int {
		grid[top][column].Borders.Bottom = true
		for r := top + 1; r <= bottom; r++ {
			grid[r][column].Borders.Right = true
		}
		grid[bottom][column].Borders.Bottom = true
		mid := (top + bottom) / 2
		grid[mid][column].Kind = domain.BracketGame
		grid[mid][column].MatchID = &matchID
		grid[mid][column+1].Borders.Bottom = true
		grid[mid][column+1].Kind = domain.BracketWin
		return mid
	}
	setTeam := func(cell *domain.BracketCell, team *store.Team) {
		if team != nil {
			cell.TeamID = &team.ID
		}
	}
	for idx, seed := range seeds {
		cell := &grid[1+idx*2][0]
		cell.Kind = domain.BracketTeam
		setTeam(cell, &teams[seed])
	}
	var quarterRows [4]int
	for idx := range quarter {
		quarterRows[idx] = bout(0, 1+idx*4, 3+idx*4, quarter[idx].ID)
		setTeam(&grid[quarterRows[idx]][1], quarterWinners[idx])
	}
	semiRow1 := bout(1, quarterRows[0], quarterRows[1], semis[0].ID)
	setTeam(&grid[semiRow1][2], semiWinner)
	semiRow2 := bout(1, quarterRows[2], quarterRows[3], semis[1].ID)
	finalRow := bout(2, semiRow1, semiRow2, final.ID)
	grid[finalRow-1][3] = domain.BracketCell{Kind: domain.BracketTxtSharp, Text: "1st"}
	return domain.BracketLayout{Columns: columns, Rows: grid}
}

func (s *seeder) beachOthers(tour store.Tournament) error {
	specs := []struct {
		division store.Division
		names    []teamName
		bar      domain.TeamBarStyle
	}{
		{
			store.Division{NameJa: "メン", NameEn: "Men's", Kind: domain.DivisionMen, SortOrder: 1},
			nato("インディア", "ジュリエット", "キロ", "リマ"),
			"default",
		},
		{
			store.Division{NameJa: "ウィメン", NameEn: "Women's", Kind: domain.DivisionWomen, SortOrder: 2},
			nato("マイク", "ノベンバー", "オスカー"),
			"women",
		},
		{
			store.Division{NameJa: "オープン", NameEn: "Open", Kind: domain.DivisionNone, SortOrder: 3},
			nato("パパ", "ケベック"),
			"univ",
		},
	}
	for idx, spec := range specs {
		row := spec.division
		row.TournamentID, row.ScheduleEnabled = tour.ID, true
		division, err := s.division(row)
		if err != nil {
			return err
		}
		teams, players, err := s.teams(division, spec.names, spec.bar, 5)
		if err != nil {
			return err
		}
		pool, err := s.group(store.Group{
			DivisionID: division.ID, Name: "Round robin", Kind: domain.GroupPool, Linked: true, Published: true,
		}, teams)
		if err != nil {
			return err
		}
		results := []matchSpec{
			finished(7, 5),
			finished(3, 8),
			finished(6, 6),
			finished(9, 1),
			finished(4, 7),
			finished(5, 2),
		}
		// コートが数字でない会場(大会 224 の "A" "B")。ここでは詳細のない終了試合も混ぜる
		results[0].noDetail = true
		prefix := []string{"M", "W", "O"}[idx]
		if err := s.roundRobin(division, pool, teams, prefix, "2026-10-18", results, players); err != nil {
			return err
		}
		if idx == 2 {
			// オープンの試合は 1 つだけ、コートが "A"
			if err := s.tx.Model(&store.Match{}).Where("division_id = ?", division.ID).Update("field", "A").Error; err != nil {
				return fmt.Errorf("set letter field: %w", err)
			}
		}
	}
	return nil
}

// upcomingCup は受付中の大会。ディビジョンとチームはあるが試合は無い。
func (s *seeder) upcomingCup() error {
	tour, err := s.tournament(store.Tournament{
		NameJa: "2027 架空カップ", NameEn: "2027 Fictional Cup", ShortName: "2027 F-Cup",
		Status: domain.TournamentEntryOpen, Listed: true, StartDate: date("2027-05-03"), ShowSchedule: true,
	})
	if err != nil {
		return err
	}
	division, err := s.division(store.Division{
		TournamentID: tour.ID, NameJa: "ミックス", NameEn: "Mixed", Kind: domain.DivisionMixed, ScheduleEnabled: true,
	})
	if err != nil {
		return err
	}
	_, _, err = s.teams(division, nato("ロメオ", "シエラ", "タンゴ"), "mixed", 0)
	return err
}

// nightSeries は締切り済みの 4 人制の大会。未試合だけで、未確定の対戦相手 [A1-勝ち] を含む。
func (s *seeder) nightSeries() error {
	tour, err := s.tournament(store.Tournament{
		NameJa: "2026 架空ナイト 4on4 Game 9", NameEn: "2026 Fictional Night 4on4 Game 9", ShortName: "F-Night 9",
		Status: domain.TournamentEntryClosed, Listed: true, StartDate: date("2026-11-03"), ShowSchedule: true,
	})
	if err != nil {
		return err
	}
	division, err := s.division(store.Division{
		TournamentID: tour.ID, NameJa: "４人制部門", NameEn: "4 on 4", Kind: domain.DivisionNone, ScheduleEnabled: true,
	})
	if err != nil {
		return err
	}
	teams, players, err := s.teams(division, nato("ユニフォーム", "ビクター", "ウィスキー", "エックスレイ"), "default", 4)
	if err != nil {
		return err
	}
	scheduled := func(code, clock string, left, right *store.Team) matchSpec {
		return matchSpec{
			code:   code,
			day:    "2026-11-03",
			clock:  clock,
			field:  "1",
			left:   left,
			right:  right,
			status: domain.MatchScheduled,
		}
	}
	specs := []matchSpec{
		scheduled("A1", "19:00", &teams[0], &teams[1]),
		scheduled("A2", "19:00", &teams[2], &teams[3]),
		{code: "A3", day: "2026-11-03", clock: "20:00", field: "2", status: domain.MatchScheduled,
			leftHolder:  &domain.Placeholder{Code: "A1", Outcome: domain.OutcomeWin},
			rightHolder: &domain.Placeholder{Code: "A2", Outcome: domain.OutcomeWin}},
		{code: "A4", day: "2026-11-03", clock: "20:00", field: "1", status: domain.MatchScheduled,
			leftHolder: &domain.Placeholder{Code: "A1", Outcome: domain.OutcomeLose}, right: &teams[3]},
	}
	for _, spec := range specs {
		if _, err := s.match(division, spec, players); err != nil {
			return err
		}
	}
	return nil
}

// unlisted は一覧に載せない大会(URL を直接開けば表示する。参照サイトの 538)。
func (s *seeder) unlisted() error {
	tour, err := s.tournament(store.Tournament{
		NameJa: "非公開の架空大会", NameEn: "Unlisted Fictional Tournament", Status: domain.TournamentFinished,
		Listed: false, StartDate: date("2026-09-01"), ShowSchedule: true,
	})
	if err != nil {
		return err
	}
	_, err = s.division(
		store.Division{TournamentID: tour.ID, NameJa: "オープン", NameEn: "Open", ScheduleEnabled: true},
	)
	return err
}

// discgolf はディスクゴルフ個人戦。ナビのラベルが「エントリー一覧」「成績」になり、スケジュールを出さない
// (clone-spec E 7)。参加者は 1 人チームとして登録する。
func (s *seeder) discgolf() error {
	tour, err := s.tournament(store.Tournament{
		NameJa: "2026 架空ディスクゴルフ大会", NameEn: "", Status: domain.TournamentEntryOpen, Listed: true,
		StartDate: date(
			"2026-11-21",
		), TeamListLabel: "entries", GroupLabel: "results", ShowSchedule: false,
	})
	if err != nil {
		return err
	}
	division, err := s.division(store.Division{
		TournamentID: tour.ID, NameJa: "ＤＧシングルス（土曜日）", NameEn: "DG Singles (Sat)",
		ScheduleEnabled: false, EntryKind: domain.EntryIndividual, StatsKind: domain.StatsDiscgolf,
	})
	if err != nil {
		return err
	}
	var entries []teamName
	for num := 1; num <= 6; num++ {
		name := fmt.Sprintf("架空 参加者%d", num)
		entries = append(entries, teamName{full: name, short: name, en: fmt.Sprintf("Entrant %d", num)})
	}
	teams, _, err := s.teams(division, entries, "default", 0)
	if err != nil {
		return err
	}
	_, err = s.group(store.Group{
		DivisionID: division.ID, Name: "Result", Kind: domain.GroupPool, Linked: false, Published: true,
	}, teams)
	return err
}

// noEnglishName は英語名が未登録の大会(参照サイトの 47 大会。一覧の英語表示で名前が空になる)。
func (s *seeder) noEnglishName() error {
	tour, err := s.tournament(store.Tournament{
		NameJa: "2017 架空オープン", NameEn: "", Status: domain.TournamentFinished, Listed: true,
		StartDate: date("2017-06-10"), ShowSchedule: true,
	})
	if err != nil {
		return err
	}
	division, err := s.division(store.Division{
		TournamentID: tour.ID, NameJa: "メン", Kind: domain.DivisionMen, ScheduleEnabled: true, StandingsPublished: true,
	})
	if err != nil {
		return err
	}
	teams, players, err := s.teams(
		division,
		[]teamName{{full: "ズールー", short: "ズールー"}, {full: "アルファ", short: "アルファ"}},
		"default",
		3,
	)
	if err != nil {
		return err
	}
	for idx, rank := range []int{1, 2} {
		if err := s.tx.Model(&teams[idx]).Update("final_rank", rank).Error; err != nil {
			return fmt.Errorf("set final rank: %w", err)
		}
	}
	_, err = s.match(division, matchSpec{
		code: "F1", day: "2017-06-10", clock: "15:00", field: "1", left: &teams[0], right: &teams[1],
		status: domain.MatchFinished, scores: [2]int{13, 11},
	}, players)
	return err
}
