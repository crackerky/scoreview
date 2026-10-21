package importer

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/importer/parse"
	"scoreview/api/internal/store"
)

// Plan は Bundle を DB の行に直した結果。Warnings は推測で決めた箇所と取りこぼし(人が確かめるもの)。
type Plan struct {
	Import   *store.TournamentImport
	Warnings []string
}

// planner は Bundle から行を組み立てる途中の状態。参照サイトの ID を持つ行は ID で引けるようにし、
// ID のない行(ダミーのディビジョン、個人戦のエントリー、未試合の試合)は ID 0 のまま持つ。
type planner struct {
	b *Bundle

	tournament store.Tournament
	divs       []*divPlan
	divByID    map[int64]*divPlan

	groups    []*groupPlan
	groupByID map[int64]*groupPlan

	teams    []*teamPlan
	teamByID map[int64]*teamPlan

	players     map[int64]*store.Player
	playerOrder []int64

	matches     []*matchPlan
	matchByID   map[int64]*matchPlan
	matchByKey  map[string]*matchPlan
	events      []store.MatchEvent
	stats       []store.MatchPlayerStat
	groupTeams  []store.GroupTeam
	warnings    []string
	tallies     map[string][]string
	tallyOrder  []string
	shortJA     map[int64]string
	shortEN     map[int64]string
	hiddenGroup int
}

type divPlan struct {
	index    int
	row      store.Division
	schedule *parse.Schedule
	teamList *parse.TeamListDivision
	nav      parse.TopDivision
}

type groupPlan struct {
	row  store.Group
	div  *divPlan
	page *parse.Group
}

type teamPlan struct {
	row store.Team
	div *divPlan
}

// matchPlan は 1 試合。参照サイトの一覧(スケジュール、グループの結果一覧、チーム対戦一覧)の行と
// 試合ページを突き合わせてまとめる。
type matchPlan struct {
	row   store.Match
	div   *divPlan
	page  *parse.Match
	date  *string // 決まった "YYYY-MM-DD"
	order int     // 作った順(並びを決定的にするため)
	// fromRow は一覧の行がある試合。ない試合(表と試合ページにだけ出る)は状態を推測する
	fromRow bool
}

// BuildPlan は Bundle を DB に書く行の集合にする。Bundle の内容が DB の制約を満たせないとき
// (トーナメント表の格子が形式に合わないなど)はエラーを返す。
func BuildPlan(b *Bundle) (*Plan, error) {
	if b.Top == nil {
		return nil, fmt.Errorf("tournament %d: bundle has no top page", b.TournamentID)
	}
	p := &planner{
		b:          b,
		divByID:    map[int64]*divPlan{},
		groupByID:  map[int64]*groupPlan{},
		teamByID:   map[int64]*teamPlan{},
		players:    map[int64]*store.Player{},
		matchByID:  map[int64]*matchPlan{},
		matchByKey: map[string]*matchPlan{},
		tallies:    map[string][]string{},
		shortJA:    map[int64]string{},
		shortEN:    map[int64]string{},
	}
	p.planTournament()
	p.planDivisions()
	p.planGroups()
	p.planTeams()
	p.planPlayers()
	if err := p.planMatches(); err != nil {
		return nil, err
	}
	if err := p.planBrackets(); err != nil {
		return nil, err
	}
	p.planWinners()
	p.planEventsAndStats()
	p.planGreatests()
	p.planPoolMembers()
	p.planTeamNames()
	p.finishTournament()
	imp := p.assemble()
	for _, cat := range p.tallyOrder {
		details := p.tallies[cat]
		const shown = 5
		sample := details
		if len(sample) > shown {
			sample = sample[:shown]
		}
		more := ""
		if len(details) > shown {
			more = ", ..."
		}
		p.warnings = append(p.warnings, fmt.Sprintf("%s: %d 件 (%s%s)", cat, len(details),
			strings.Join(sample, ", "), more))
	}
	return &Plan{Import: imp, Warnings: p.warnings}, nil
}

func (p *planner) warn(format string, args ...any) {
	p.warnings = append(p.warnings, fmt.Sprintf(format, args...))
}

// tally は同じ種類の警告を 1 行にまとめるために数える。
func (p *planner) tally(category, detail string) {
	if _, ok := p.tallies[category]; !ok {
		p.tallyOrder = append(p.tallyOrder, category)
	}
	p.tallies[category] = append(p.tallies[category], detail)
}

func (p *planner) planTournament() {
	b := p.b
	t := store.Tournament{
		ID:            b.TournamentID,
		NameJa:        b.Top.Name.JA,
		NameEn:        b.Top.Name.EN,
		Listed:        b.ListEntry != nil,
		LogoURL:       b.Top.LogoURL,
		TopHTMLJa:     b.Top.TopHTML.JA,
		TopHTMLEn:     b.Top.TopHTML.EN,
		TeamListLabel: b.Top.Nav.TeamListLabel,
		GroupLabel:    b.Top.Nav.GroupLabel,
		ShowSchedule:  b.Top.Nav.ShowSchedule,
	}
	if b.ListEntry != nil {
		t.Status = domain.TournamentStatus(b.ListEntry.Status)
		if t.NameJa == "" {
			t.NameJa = b.ListEntry.Name.JA
		}
	} else {
		// 推測: 一覧に載らない大会(clone-spec A 3)は区分が分からない。[最終順位] がリンクの
		// ディビジョンがあれば終了、なければエントリー締切りとする
		t.Status = domain.TournamentEntryClosed
		for _, d := range b.Top.Divisions {
			if d.StandingsLinked {
				t.Status = domain.TournamentFinished
			}
		}
		p.warn("一覧に載らない大会なので status=%s を推測した([最終順位] のリンクの有無から)", t.Status)
	}
	p.tournament = t
}

// planDivisions は大会トップの表の順にディビジョンを作る。チーム一覧の見出しにはディビジョン ID が
// ないので、並び順(数が合わなければ名前)で対応させる(clone-spec A 5)。
func (p *planner) planDivisions() {
	b := p.b
	var tl []parse.TeamListDivision
	if b.TeamList != nil {
		tl = b.TeamList.Divisions
	}
	for i, d := range b.Top.Divisions {
		dp := &divPlan{index: i, nav: d}
		dp.row = store.Division{
			NameJa: d.Name.JA, NameEn: d.Name.EN, SortOrder: i,
			ScheduleEnabled: d.ScheduleLinked, StandingsPublished: d.StandingsLinked,
			Kind: domain.DivisionNone, EntryKind: domain.EntryTeam, StatsKind: domain.StatsUltimate,
		}
		if d.ID != nil {
			dp.row.ID = *d.ID
			p.divByID[*d.ID] = dp
			dp.schedule = b.Schedules[*d.ID]
		}
		dp.teamList = matchTeamListDivision(tl, len(b.Top.Divisions), i, d.Name.JA)
		if dp.teamList == nil && len(tl) > 0 {
			p.warn("ディビジョン %q にチーム一覧の見出しを対応させられない", d.Name.JA)
		}
		if dp.teamList != nil {
			dp.row.EntryKind = domain.EntryKind(dp.teamList.EntryKind)
		}
		// 日程の MC_MEN / MC_WOM / MC_MIX から決める(clone-spec B 6)。決まらなければ none
		if kind, ok := DivisionKindFromSchedule(dp.schedule); ok {
			dp.row.Kind = domain.DivisionKind(kind)
		}
		p.divs = append(p.divs, dp)
	}
}

func matchTeamListDivision(
	tl []parse.TeamListDivision, topCount, index int, name string,
) *parse.TeamListDivision {
	if len(tl) == topCount {
		return &tl[index]
	}
	var found *parse.TeamListDivision
	for i := range tl {
		if tl[i].Name.JA == name {
			if found != nil {
				return nil
			}
			found = &tl[i]
		}
	}
	return found
}

// planGroups はナビの順にグループを作る。グループページがないもの(リンクなし、取得できない)は
// 種類が分からないので pool とする(推測)。
func (p *planner) planGroups() {
	for _, dp := range p.divs {
		for j, ng := range dp.nav.Groups {
			gp := &groupPlan{div: dp, row: store.Group{
				Name: ng.Name, NameEn: ng.NameEN, Kind: domain.GroupPool, SortOrder: j,
				Linked: ng.Linked, Published: true,
			}}
			if ng.ID != nil {
				gp.row.ID = *ng.ID
				gp.page = p.b.Groups[*ng.ID]
				if _, dup := p.groupByID[*ng.ID]; dup {
					p.warn("グループ %d がナビに 2 回出るので 2 回目を無視した", *ng.ID)
					continue
				}
				p.groupByID[*ng.ID] = gp
			}
			if gp.page == nil {
				if ng.Linked {
					p.warn("グループ %d (%s) のページがないので kind=pool とした(推測)", gp.row.ID, ng.Name)
				}
			} else {
				gp.row.Kind = domain.GroupKind(gp.page.Kind)
				if img := gp.page.Image; img != nil {
					gp.row.ImageURL = &img.URL
					gp.row.ImageWidth, gp.row.ImageHeight = img.Width, img.Height
				}
			}
			p.groups = append(p.groups, gp)
		}
	}
}

// ensureGroup は試合ページのパンくずにだけ出るグループ(ナビにない欠番のグループ)を、ナビに出さない
// グループとして作る。種類は分からないので pool とする(推測)。
func (p *planner) ensureGroup(id int64, dp *divPlan, name string) *groupPlan {
	if gp, ok := p.groupByID[id]; ok {
		return gp
	}
	gp := &groupPlan{div: dp, row: store.Group{
		ID: id, Name: name, Kind: domain.GroupPool, SortOrder: len(dp.nav.Groups) + p.hiddenGroup,
		Linked: false, Published: false,
	}}
	p.hiddenGroup++
	p.groupByID[id] = gp
	p.groups = append(p.groups, gp)
	p.warn("グループ %d (%s) はナビにないので published=false、kind=pool で作った(推測)", id, name)
	return gp
}

// planTeams はチーム一覧の順にチームを作る。チーム一覧に出ないが他のページに出るチームは、
// 出てきたページのディビジョンに後ろから足す。
func (p *planner) planTeams() {
	for _, dp := range p.divs {
		if dp.teamList == nil {
			continue
		}
		for k, e := range dp.teamList.Entries {
			tp := &teamPlan{div: dp, row: store.Team{
				NameFullJa: e.Name.JA, NameFullEn: e.Name.EN,
				BarStyle: domain.TeamBarStyle(e.BarStyle), SortOrder: k,
			}}
			if e.ID != nil {
				if _, dup := p.teamByID[*e.ID]; dup {
					p.warn("チーム %d がチーム一覧に 2 回出るので 2 回目を無視した", *e.ID)
					continue
				}
				tp.row.ID = *e.ID
				p.teamByID[*e.ID] = tp
			}
			p.teams = append(p.teams, tp)
		}
	}
	// チーム一覧に出ないチーム。ディビジョンが分かるページから順に探す
	for _, dp := range p.divs {
		if dp.schedule != nil {
			for _, r := range dp.schedule.Rows {
				p.ensureTeam(r.Left.TeamID, dp, "スケジュール")
				p.ensureTeam(r.Right.TeamID, dp, "スケジュール")
			}
		}
		if dp.row.ID != 0 {
			if st := p.b.Standings[dp.row.ID]; st != nil {
				for _, r := range st.Rows {
					p.ensureTeam(&r.TeamID, dp, "ディビジョン成績")
				}
			}
		}
	}
	for _, gp := range p.groups {
		if gp.page == nil {
			continue
		}
		if gp.page.Pool != nil {
			for _, t := range gp.page.Pool.Teams {
				p.ensureTeam(t.TeamID, gp.div, "グループ")
			}
		}
		if gp.page.Bracket != nil {
			for _, row := range gp.page.Bracket.Rows {
				for _, c := range row {
					p.ensureTeam(c.TeamID, gp.div, "トーナメント表")
				}
			}
		}
		for _, r := range gp.page.Matches {
			p.ensureTeam(r.Left.TeamID, gp.div, "グループの結果一覧")
			p.ensureTeam(r.Right.TeamID, gp.div, "グループの結果一覧")
		}
	}
	for _, id := range sortedKeys(p.b.Matches) {
		m := p.b.Matches[id]
		dp := p.breadcrumbDivision(m)
		if dp == nil {
			// ディビジョンの分からない試合ページは、相手チームのディビジョンに合わせる
			if tp := p.teamByID[m.Left.TeamID]; tp != nil {
				dp = tp.div
			} else if tp := p.teamByID[m.Right.TeamID]; tp != nil {
				dp = tp.div
			}
		}
		if dp == nil {
			continue
		}
		p.ensureTeam(&m.Left.TeamID, dp, "試合ページ")
		p.ensureTeam(&m.Right.TeamID, dp, "試合ページ")
	}
	for _, id := range sortedKeys(p.b.Teams) {
		tp := p.teamByID[id]
		if tp == nil {
			continue
		}
		for _, r := range p.b.Teams[id].Matches {
			p.ensureTeam(r.Left.TeamID, tp.div, "チームの対戦一覧")
			p.ensureTeam(r.Right.TeamID, tp.div, "チームの対戦一覧")
		}
	}
	for _, id := range sortedKeys(p.b.Teams) {
		if _, ok := p.teamByID[id]; !ok {
			p.tally("ディビジョンが分からないチームページを捨てた", fmt.Sprint(id))
		}
	}

	for _, tp := range p.teams {
		if tp.row.ID == 0 {
			continue
		}
		page := p.b.Teams[tp.row.ID]
		if page == nil {
			continue
		}
		if page.Name.JA != "" {
			tp.row.NameFullJa = page.Name.JA
		}
		if page.Name.EN != "" {
			tp.row.NameFullEn = page.Name.EN
		}
		tp.row.CommentJa, tp.row.CommentEn = page.Comment.JA, page.Comment.EN
	}
	p.planFinalRanks()
}

func (p *planner) breadcrumbDivision(m *parse.Match) *divPlan {
	if m.Breadcrumb == nil || m.Breadcrumb.DivisionID == nil {
		return nil
	}
	return p.divByID[*m.Breadcrumb.DivisionID]
}

func (p *planner) ensureTeam(id *int64, dp *divPlan, source string) {
	if id == nil {
		return
	}
	if _, ok := p.teamByID[*id]; ok {
		return
	}
	tp := &teamPlan{div: dp, row: store.Team{
		ID: *id, BarStyle: "default", SortOrder: extraTeamSortOrder + len(p.teams),
	}}
	p.teamByID[*id] = tp
	p.teams = append(p.teams, tp)
	p.tally("チーム一覧にないチームを "+source+" から足した", fmt.Sprint(*id))
}

// planFinalRanks はディビジョン成績の順位(順位決定戦の結果。scoreview-design 5)を final_rank に
// 入れる。ディビジョン成績がないディビジョンは、チームページの成績表の順位を使う。
func (p *planner) planFinalRanks() {
	for _, dp := range p.divs {
		if dp.row.ID == 0 {
			continue
		}
		st := p.b.Standings[dp.row.ID]
		if st == nil {
			continue
		}
		for _, r := range st.Rows {
			if tp := p.teamByID[r.TeamID]; tp != nil && r.Rank != nil && *r.Rank > 0 {
				tp.row.FinalRank = ptr(*r.Rank)
			}
		}
	}
	for _, tp := range p.teams {
		if tp.row.FinalRank != nil || tp.row.ID == 0 {
			continue
		}
		page := p.b.Teams[tp.row.ID]
		if page != nil && page.Record != nil && page.Record.Rank != nil && *page.Record.Rank > 0 {
			tp.row.FinalRank = ptr(*page.Record.Rank)
		}
	}
}

// planPlayers はチームページの選手一覧とスタッフ一覧から選手を作る。
func (p *planner) planPlayers() {
	for _, tp := range p.teams {
		if tp.row.ID == 0 {
			continue
		}
		page := p.b.Teams[tp.row.ID]
		if page == nil {
			continue
		}
		for _, pl := range page.Players {
			p.addPlayer(store.Player{
				ID: pl.ID, TeamID: tp.row.ID, Number: pl.Number,
				NameJa: pl.Name.JA, NameEn: pl.Name.EN, Gender: pl.Gender,
			})
		}
		for _, s := range page.Staff {
			p.addPlayer(store.Player{
				ID: s.ID, TeamID: tp.row.ID, NameJa: s.Name.JA, NameEn: s.Name.EN, Gender: s.Gender,
				RoleJa: ptr(s.Role.JA), RoleEn: ptr(s.Role.EN),
			})
		}
	}
	for _, dp := range p.divs {
		dp.row.StatsKind = p.divisionStatsKind(dp)
	}
}

func (p *planner) addPlayer(row store.Player) {
	if _, dup := p.players[row.ID]; dup {
		p.tally("同じ選手 ID が 2 回出たので 2 回目を無視した", fmt.Sprint(row.ID))
		return
	}
	p.players[row.ID] = &row
	p.playerOrder = append(p.playerOrder, row.ID)
}

// divisionStatsKind は選手ページの成績表の列構成を決める。選手ページを取っていればその値。
// 取っていなければ、個人戦か、チームページに成績表のない(ディスクゴルフ団体戦。clone-spec D 1.2)
// ディビジョンを discgolf とする(推測)。
func (p *planner) divisionStatsKind(dp *divPlan) domain.StatsKind {
	for _, id := range p.playerOrder {
		pl := p.b.Players[id]
		if pl == nil || pl.StatsKind == "" {
			continue
		}
		if tp := p.teamByID[p.players[id].TeamID]; tp != nil && tp.div == dp {
			return domain.StatsKind(pl.StatsKind)
		}
	}
	if dp.row.EntryKind == domain.EntryIndividual {
		return domain.StatsDiscgolf
	}
	pages, withoutRecord := 0, 0
	for _, tp := range p.teams {
		if tp.div != dp || tp.row.ID == 0 {
			continue
		}
		if page := p.b.Teams[tp.row.ID]; page != nil {
			pages++
			if page.Record == nil {
				withoutRecord++
			}
		}
	}
	if pages > 0 && pages == withoutRecord {
		return domain.StatsDiscgolf
	}
	return domain.StatsUltimate
}

// planTeamNames は短縮名を決める。短縮名は正式名から作れない(clone-spec D 1.3)ので、表に出る名前を
// スケジュール(日英)、チームページの対戦一覧(日英)、ディビジョン成績、グループ、試合ページ、
// トーナメント表の順に探す。
// どこにも出ないチーム(試合のないチーム)は正式名を使う(推測)。
func (p *planner) planTeamNames() {
	note := func(id *int64, name parse.Text) {
		if id == nil {
			return
		}
		if p.shortJA[*id] == "" && name.JA != "" {
			p.shortJA[*id] = name.JA
		}
		if p.shortEN[*id] == "" && name.EN != "" {
			p.shortEN[*id] = name.EN
		}
	}
	rows := func(rs []parse.MatchRow) {
		for _, r := range rs {
			note(r.Left.TeamID, r.Left.ShortName)
			note(r.Right.TeamID, r.Right.ShortName)
		}
	}
	for _, dp := range p.divs {
		if dp.schedule != nil {
			rows(dp.schedule.Rows)
		}
	}
	for _, id := range sortedKeys(p.b.Teams) {
		rows(p.b.Teams[id].Matches)
	}
	for _, id := range sortedKeys(p.b.Standings) {
		for _, r := range p.b.Standings[id].Rows {
			note(&r.TeamID, r.ShortName)
		}
	}
	for _, gp := range p.groups {
		if gp.page == nil {
			continue
		}
		rows(gp.page.Matches)
		if gp.page.Pool != nil {
			for _, t := range gp.page.Pool.Teams {
				note(t.TeamID, t.ShortName)
			}
		}
	}
	for _, id := range sortedKeys(p.b.Matches) {
		m := p.b.Matches[id]
		note(&m.Left.TeamID, m.Left.ShortName)
		note(&m.Right.TeamID, m.Right.ShortName)
	}
	for _, gp := range p.groups {
		if gp.page == nil || gp.page.Bracket == nil {
			continue
		}
		for _, row := range gp.page.Bracket.Rows {
			for _, c := range row {
				note(c.TeamID, c.Text)
			}
		}
	}
	for _, tp := range p.teams {
		id := tp.row.ID
		tp.row.NameShortJa, tp.row.NameShortEn = p.shortJA[id], p.shortEN[id]
		if id == 0 || tp.row.NameShortJa == "" {
			if id != 0 {
				p.tally("短縮名が分からないので正式名を使ったチーム(推測)", fmt.Sprint(id))
			}
			tp.row.NameShortJa = tp.row.NameFullJa
			if tp.row.NameShortEn == "" {
				tp.row.NameShortEn = tp.row.NameFullEn
			}
		}
		if tp.row.NameFullJa == "" {
			// チーム一覧にもチームページにもないチーム。表に出る短縮名を正式名の代わりにする
			tp.row.NameFullJa, tp.row.NameFullEn = tp.row.NameShortJa, tp.row.NameShortEn
		}
	}
}

// planPoolMembers はプールのリーグ結果の行を group_teams にする。参照サイトの順位が domain の計算と
// 食い違うときだけ rank_override を入れる(空欄は 0。scoreview-design 5)。
func (p *planner) planPoolMembers() {
	for _, gp := range p.groups {
		if gp.page == nil || gp.page.Pool == nil || gp.row.ID == 0 {
			continue
		}
		var entries []domain.PoolEntry
		var raw []*int
		for _, t := range gp.page.Pool.Teams {
			if t.TeamID == nil {
				p.tally("リーグ結果のチームにリンクがないので飛ばした", fmt.Sprint(gp.row.ID))
				continue
			}
			entries = append(entries, domain.PoolEntry{TeamID: *t.TeamID})
			raw = append(raw, t.Rank)
		}
		var results []domain.Result
		for _, mp := range p.matches {
			if mp.row.GroupID != nil && *mp.row.GroupID == gp.row.ID {
				results = append(results, resultOf(mp.row))
			}
		}
		computed := domain.RankPool(entries, results)
		for k, e := range entries {
			gt := store.GroupTeam{GroupID: gp.row.ID, TeamID: e.TeamID, SortOrder: k}
			if !equalIntPtr(raw[k], computed[k].Rank) {
				override := 0
				if raw[k] != nil {
					override = *raw[k]
				}
				gt.RankOverride = &override
				p.warn("グループ %d チーム %d: 参照サイトの順位 %s と計算 %s が違うので rank_override=%d",
					gp.row.ID, e.TeamID, intPtrString(raw[k]), intPtrString(computed[k].Rank),
					override)
			}
			p.groupTeams = append(p.groupTeams, gt)
		}
	}
}

func resultOf(m store.Match) domain.Result {
	r := domain.Result{
		LeftScore: m.LeftScore, RightScore: m.RightScore,
		Status: m.Status, WinnerSide: m.WinnerSide,
	}
	if m.LeftTeamID != nil {
		r.LeftTeamID = *m.LeftTeamID
	}
	if m.RightTeamID != nil {
		r.RightTeamID = *m.RightTeamID
	}
	return r
}

// finishTournament は試合から決まる大会の値(開始日、短縮名)を入れる。
func (p *planner) finishTournament() {
	for _, mp := range p.matches {
		if mp.row.MatchDate == nil {
			continue
		}
		if p.tournament.StartDate == nil || mp.row.MatchDate.Before(*p.tournament.StartDate) {
			p.tournament.StartDate = mp.row.MatchDate
		}
	}
	for _, id := range sortedKeys(p.b.Matches) {
		bc := p.b.Matches[id].Breadcrumb
		if bc != nil && bc.TournamentID == p.b.TournamentID && bc.TournamentShortName != "" {
			p.tournament.ShortName = bc.TournamentShortName
			break
		}
	}
}

// assemble は組み立てた行を store.TournamentImport に詰める。
func (p *planner) assemble() *store.TournamentImport {
	imp := &store.TournamentImport{Tournament: p.tournament}
	for i, bn := range p.b.Top.Banners {
		imp.Banners = append(imp.Banners, store.TournamentBanner{
			ImageURL: bn.ImageURL, LinkURL: bn.LinkURL, SortOrder: i,
		})
	}
	byDiv := map[*divPlan]*store.DivisionImport{}
	for _, dp := range p.divs {
		imp.Divisions = append(imp.Divisions, store.DivisionImport{Division: dp.row})
	}
	for i, dp := range p.divs {
		byDiv[dp] = &imp.Divisions[i]
	}
	for _, gp := range p.groups {
		d := byDiv[gp.div]
		d.Groups = append(d.Groups, gp.row)
	}
	for _, tp := range p.teams {
		d := byDiv[tp.div]
		d.Teams = append(d.Teams, tp.row)
	}
	for _, mp := range p.matches {
		d := byDiv[mp.div]
		d.Matches = append(d.Matches, mp.row)
	}
	imp.GroupTeams = p.groupTeams
	for _, id := range p.playerOrder {
		imp.Players = append(imp.Players, *p.players[id])
	}
	imp.Events = p.events
	imp.Stats = p.stats
	for i, ng := range p.b.Top.Nav.Galleries {
		g := store.GalleryImport{Gallery: store.Gallery{
			ID: ng.ID, Name: ng.Name, NameEn: ng.NameEN, SortOrder: i,
		}}
		if page := p.b.Galleries[ng.ID]; page != nil {
			if page.Name != "" {
				g.Gallery.Name = page.Name
			}
			for k, ph := range page.Photos {
				g.Photos = append(g.Photos, store.GalleryPhoto{
					ThumbURL: ph.ThumbURL, FullURL: ph.FullURL, SortOrder: k,
				})
			}
		}
		imp.Galleries = append(imp.Galleries, g)
	}
	return imp
}

// extraTeamSortOrder はチーム一覧に出ないチームの sort_order の起点。チーム一覧のチームより後ろに並べる。
const extraTeamSortOrder = 10_000

func ptr[T any](v T) *T { return &v }

func equalIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func intPtrString(v *int) string {
	if v == nil {
		return "空欄"
	}
	return fmt.Sprint(*v)
}

func sortedKeys[V any](m map[int64]V) []int64 {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func parseDate(s string) (*time.Time, error) {
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		return nil, fmt.Errorf("parse date %q: %w", s, err)
	}
	return &d, nil
}

// sortMatchPlans は試合を日付、時刻、作った順に並べる(グレイテストを入れる「最初の試合」を決めるため)。
func sortMatchPlans(ms []*matchPlan) {
	sort.SliceStable(ms, func(i, j int) bool {
		a, b := ms[i], ms[j]
		ad, bd := deref(a.date), deref(b.date)
		if ad != bd {
			return ad < bd
		}
		at, bt := deref(a.row.StartTime), deref(b.row.StartTime)
		if at != bt {
			return at < bt
		}
		return a.order < b.order
	})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
