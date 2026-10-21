package importer

import (
	"fmt"
	"regexp"
	"strings"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/importer/parse"
	"scoreview/api/internal/store"
)

// rowList は 1 つの一覧(スケジュール、グループの結果一覧、チーム対戦一覧)の行と、行ごとの試合。
// 年のない日付を前後の行から決めるために、一覧の並びのまま持つ。
type rowList struct {
	div   *divPlan
	rows  []parse.MatchRow
	plans []*matchPlan
}

// planMatches は一覧の行と試合ページから試合を作る。同じ試合は、試合 ID があれば ID で、なければ
// ディビジョン、CD、月日、時刻、コート、左右(チームか未確定の表記)で突き合わせる。
func (p *planner) planMatches() error {
	var lists []rowList
	for _, dp := range p.divs {
		if dp.schedule != nil {
			lists = append(lists, p.addRows(dp.schedule.Rows, dp, nil))
		}
	}
	for _, gp := range p.groups {
		if gp.page != nil {
			lists = append(lists, p.addRows(gp.page.Matches, gp.div, gp))
		}
	}
	for _, id := range sortedKeys(p.b.Teams) {
		if tp := p.teamByID[id]; tp != nil {
			lists = append(lists, p.addRows(p.b.Teams[id].Matches, tp.div, nil))
		}
	}
	for _, gp := range p.groups {
		if gp.page == nil {
			continue
		}
		if gp.page.Pool != nil {
			for _, c := range gp.page.Pool.Cells {
				p.ensureMatch(c.MatchID, gp.div, gp)
			}
		}
		if gp.page.Bracket != nil {
			// トーナメント表は別グループの試合も参照するので、グループは付けない
			for _, row := range gp.page.Bracket.Rows {
				for _, c := range row {
					p.ensureMatch(c.MatchID, gp.div, nil)
				}
			}
		}
	}
	for _, id := range sortedKeys(p.b.Matches) {
		p.applyMatchPage(id, p.b.Matches[id])
	}
	return p.resolveDates(lists)
}

// addRows は一覧の行を試合にする。試合 ID のない行は、同じ一覧で同じキーが繰り返し出ても別の試合として
// 扱えるよう、キーに一覧内の出現回数を付ける。
func (p *planner) addRows(rows []parse.MatchRow, dp *divPlan, gp *groupPlan) rowList {
	list := rowList{div: dp, rows: rows, plans: make([]*matchPlan, len(rows))}
	seen := map[string]int{}
	for i, r := range rows {
		p.ensureTeam(r.Left.TeamID, dp, "チームの対戦一覧")
		p.ensureTeam(r.Right.TeamID, dp, "チームの対戦一覧")
		var mp *matchPlan
		var key string
		if r.MatchID != nil {
			mp = p.matchByID[*r.MatchID]
		} else {
			base := rowKey(dp, r)
			key = fmt.Sprintf("%s#%d", base, seen[base])
			seen[base]++
			mp = p.matchByKey[key]
		}
		if mp == nil {
			mp = p.newMatch(dp)
			fillFromRow(&mp.row, r)
			mp.date = r.Date
			if r.MatchID != nil {
				mp.row.ID = *r.MatchID
				p.matchByID[mp.row.ID] = mp
			} else {
				p.matchByKey[key] = mp
			}
			for _, s := range []parse.MatchRowSide{r.Left, r.Right} {
				if s.TeamID == nil && s.Placeholder == nil && clean(s.ShortName.JA) != "" {
					p.tally("チームも未確定の表記もない対戦相手", s.ShortName.JA)
				}
			}
		}
		if gp != nil && gp.row.ID != 0 && mp.row.GroupID == nil {
			mp.row.GroupID = ptr(gp.row.ID)
		}
		noteLabel(&mp.row.LeftLabelEn, r.Left)
		noteLabel(&mp.row.RightLabelEn, r.Right)
		mp.fromRow = true
		list.plans[i] = mp
	}
	return list
}

func clean(s string) string { return strings.TrimSpace(s) }

// labelPattern は対戦相手の決まり方の英語表記。勝ち負け "[A1-Win]"、グループの順位 "[MA-3rd]"、
// 枠 "(A10)"(clone-spec B 8)。参照サイトの英語版は、英語名のないチームのリンクの文字にこれを出す。
var labelPattern = regexp.MustCompile(`^(\[[^\]]+-(Win|Lose|\d+(st|nd|rd|th))\]|\([0-9A-Za-z]+\))$`)

// isLabel は英語版の一覧のチームのリンクの文字が、チーム名ではなく決まり方の表記かを返す。
func isLabel(text string) bool { return labelPattern.MatchString(clean(text)) }

// noteLabel は一覧の行の英語の文字が決まり方の表記なら、試合のその側の表記として残す。
// 英語名のあるチームの行には名前が出るので、表記が分かるのは英語名のないチームの行だけ。
func noteLabel(dst *string, side parse.MatchRowSide) {
	if *dst == "" && side.TeamID != nil && isLabel(side.ShortName.EN) {
		*dst = clean(side.ShortName.EN)
	}
}

func (p *planner) newMatch(dp *divPlan) *matchPlan {
	mp := &matchPlan{
		div: dp, order: len(p.matches),
		row: store.Match{Status: domain.MatchScheduled, HasDetail: true},
	}
	p.matches = append(p.matches, mp)
	return mp
}

func rowKey(dp *divPlan, r parse.MatchRow) string {
	side := func(s parse.MatchRowSide) string {
		switch {
		case s.TeamID != nil:
			return fmt.Sprint(*s.TeamID)
		case s.Placeholder != nil:
			return "[" + s.Placeholder.Code + "-" + s.Placeholder.Outcome + "]"
		}
		return "?" + s.ShortName.JA
	}
	return strings.Join([]string{
		fmt.Sprint(dp.index), r.Code, r.MonthDay, deref(r.Time), r.Field,
		side(r.Left), side(r.Right),
	}, "|")
}

// fillFromRow は一覧の行の値を試合の行に入れる。スコアと状態は行の作成時の初期値である
// (取込は通知しない経路。scoreview-design 8)。
func fillFromRow(m *store.Match, r parse.MatchRow) {
	m.Code = r.Code
	m.StartTime = r.Time
	if r.Field != "" {
		m.Field = ptr(r.Field)
	}
	m.LeftTeamID, m.RightTeamID = r.Left.TeamID, r.Right.TeamID
	if ph := r.Left.Placeholder; ph != nil && r.Left.TeamID == nil {
		m.LeftPlaceholderCode = ptr(ph.Code)
		m.LeftPlaceholderOutcome = ptr(domain.Outcome(ph.Outcome))
	}
	if ph := r.Right.Placeholder; ph != nil && r.Right.TeamID == nil {
		m.RightPlaceholderCode = ptr(ph.Code)
		m.RightPlaceholderOutcome = ptr(domain.Outcome(ph.Outcome))
	}
	m.LeftScore, m.RightScore = r.LeftScore, r.RightScore
	m.Status = domain.MatchStatus(r.Status)
	m.HasDetail = r.HasDetail
}

// ensureMatch は一覧に出ず、対戦表やトーナメント表にだけ出る試合を作る。中身は試合ページから入れる。
func (p *planner) ensureMatch(id *int64, dp *divPlan, gp *groupPlan) {
	if id == nil {
		return
	}
	mp := p.matchByID[*id]
	if mp == nil {
		mp = p.newMatch(dp)
		mp.row.ID = *id
		p.matchByID[*id] = mp
		p.tally("一覧に出ず表にだけ出る試合", fmt.Sprint(*id))
	}
	if gp != nil && gp.row.ID != 0 && mp.row.GroupID == nil {
		mp.row.GroupID = ptr(gp.row.ID)
	}
}

// applyMatchPage は試合ページの値で試合を上書きする。日付(年 4 桁)、時刻、コート、チーム、スコアは
// 試合ページを正とし、所属はパンくずのディビジョンとグループを正とする(clone-spec D 2.1)。
func (p *planner) applyMatchPage(id int64, m *parse.Match) {
	mp := p.matchByID[id]
	if mp == nil {
		dp := p.breadcrumbDivision(m)
		if dp == nil {
			if tp := p.teamByID[m.Left.TeamID]; tp != nil {
				dp = tp.div
			}
		}
		if dp == nil {
			p.tally("ディビジョンが分からない試合ページを捨てた", fmt.Sprint(id))
			return
		}
		mp = p.newMatch(dp)
		mp.row.ID = id
		p.matchByID[id] = mp
		p.tally("一覧に出ず試合ページだけがある試合", fmt.Sprint(id))
	}
	mp.page = m
	if dp := p.breadcrumbDivision(m); dp != nil && dp != mp.div {
		p.tally("一覧とパンくずでディビジョンが違う試合(パンくずを使った)", fmt.Sprint(id))
		mp.div = dp
	}
	if bc := m.Breadcrumb; bc != nil && bc.GroupID != nil {
		gp := p.ensureGroup(*bc.GroupID, mp.div, m.GroupName)
		mp.row.GroupID = ptr(gp.row.ID)
	}
	// clone-spec D 2.2: 試合ページのグループ欄はグループの今の名前と違うことがある(大文字小文字、
	// 別のグループ名、2017 年は空)ので、違うときだけ試合に持たせる
	groupName := ""
	if mp.row.GroupID != nil {
		if gp := p.groupByID[*mp.row.GroupID]; gp != nil {
			groupName = gp.row.Name
		}
	}
	if clean(m.GroupName) != clean(groupName) {
		mp.row.GroupLabel = ptr(m.GroupName)
	}
	mp.row.Note = m.Note
	mp.row.HideBlocks = !m.BlockColumn
	mp.row.HideStats = m.Stats == nil
	if m.Code != "" {
		mp.row.Code = m.Code
	}
	mp.date = ptr(m.Date)
	if m.Time != nil {
		mp.row.StartTime = m.Time
	}
	if m.Field != "" {
		mp.row.Field = ptr(m.Field)
	}
	for _, s := range []struct {
		team      int64
		dst       **int64
		phCode    **string
		phOutcome **domain.Outcome
		score     *int
		scoreDst  **int
		label     string
	}{
		{
			m.Left.TeamID, &mp.row.LeftTeamID, &mp.row.LeftPlaceholderCode,
			&mp.row.LeftPlaceholderOutcome, m.Left.Score, &mp.row.LeftScore, "left",
		},
		{
			m.Right.TeamID, &mp.row.RightTeamID, &mp.row.RightPlaceholderCode,
			&mp.row.RightPlaceholderOutcome, m.Right.Score, &mp.row.RightScore, "right",
		},
	} {
		if _, ok := p.teamByID[s.team]; ok {
			*s.dst = ptr(s.team)
			*s.phCode, *s.phOutcome = nil, nil
		} else {
			p.tally("試合ページのチームが作れないので空にした", fmt.Sprintf("%d %s", id, s.label))
		}
		if s.score != nil {
			*s.scoreDst = ptr(*s.score)
		}
	}
	mp.row.HasDetail = true
	if !mp.fromRow {
		// 推測: 一覧の行がない試合は状態が分からない。スコアが両方あれば終了とする
		if mp.row.LeftScore != nil && mp.row.RightScore != nil {
			mp.row.Status = domain.MatchFinished
		}
	}
}

// resolveDates は試合の日付を決める。試合ページの YYYY/MM/DD を正とし、なければスケジュールの
// 日付選択肢(YYYYMMDD)から年を補う。年の候補が複数あれば一覧の前後の行の日付と連続する方を選ぶ。
func (p *planner) resolveDates(lists []rowList) error {
	for _, list := range lists {
		slots := make([]dateSlot, len(list.plans))
		for i, mp := range list.plans {
			r := list.rows[i]
			switch {
			case mp.date != nil:
				slots[i].Date = mp.date
			case r.Date != nil:
				slots[i].Date = r.Date
			case len(r.DateCandidates) > 0:
				slots[i].Candidates = r.DateCandidates
			default:
				slots[i].Candidates = p.dateCandidates(list.div, r.MonthDay)
			}
		}
		dates, ambiguous := resolveDateSlots(slots)
		for i, mp := range list.plans {
			if mp.date == nil {
				mp.date = dates[i]
			}
		}
		for _, i := range ambiguous {
			r := list.rows[i]
			p.warn("試合 %s (%s) の年を決められないので候補の先頭 %s を使った(候補 %v)",
				r.Code, r.MonthDay, deref(dates[i]), slots[i].Candidates)
		}
	}
	for _, mp := range p.matches {
		if mp.date == nil {
			p.tally("日付が分からない試合", fmt.Sprintf("%d %s", mp.row.ID, mp.row.Code))
			continue
		}
		d, err := parseDate(*mp.date)
		if err != nil {
			return fmt.Errorf("match %d: %w", mp.row.ID, err)
		}
		mp.row.MatchDate = d
	}
	return nil
}

// dateCandidates は "MM/DD" に合うスケジュールの日付選択肢を返す。そのディビジョンのスケジュールが
// なければ大会の全スケジュールから探す。
func (p *planner) dateCandidates(dp *divPlan, monthDay string) []string {
	if monthDay == "" {
		return nil
	}
	var schedules []*parse.Schedule
	if dp.schedule != nil {
		schedules = append(schedules, dp.schedule)
	} else {
		for _, id := range sortedKeys(p.b.Schedules) {
			schedules = append(schedules, p.b.Schedules[id])
		}
	}
	var out []string
	for _, s := range schedules {
		for _, d := range s.Dates {
			// d は "YYYY-MM-DD"、monthDay は "MM/DD"
			if len(d) == len(dateLayout) && d[5:7]+"/"+d[8:10] == monthDay && !contains(out, d) {
				out = append(out, d)
			}
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// planBrackets はトーナメント表の格子を groups.bracket_layout の形式にする。チーム名などの文字列は
// 日本語版の表記を保存する(チームのセルは表示時にチームの参照から名前を引く)。
func (p *planner) planBrackets() error {
	for _, gp := range p.groups {
		if gp.page == nil || gp.page.Bracket == nil {
			continue
		}
		br := gp.page.Bracket
		layout := domain.BracketLayout{
			Columns: br.Columns, Rows: make([][]domain.BracketCell, len(br.Rows)),
		}
		for r, row := range br.Rows {
			cells := make([]domain.BracketCell, len(row))
			for c, cell := range row {
				text := cell.Text.JA
				if text == "" {
					text = cell.Text.EN
				}
				cells[c] = domain.BracketCell{
					Borders: domain.BracketBorders{
						Top: cell.Top, Bottom: cell.Bottom, Left: cell.Left, Right: cell.Right,
					},
					Kind: domain.BracketCellKind(cell.Kind),
					Text: text, TeamID: cell.TeamID, MatchID: cell.MatchID,
				}
			}
			layout.Rows[r] = cells
		}
		data, err := domain.MarshalBracketLayout(layout)
		if err != nil {
			return fmt.Errorf("group %d: %w", gp.row.ID, err)
		}
		gp.row.BracketLayout = data
		if br.Pattern != "" {
			gp.row.BracketPattern = ptr(br.Pattern)
		}
		if br.PtnName != "" {
			gp.row.BracketPtnName = ptr(br.PtnName)
		}
	}
	return nil
}

// planWinners は同点で終わったトーナメントの試合の勝者を決める。トーナメント表の game セル(試合 ID)と
// 同じ行か隣の行の、次の列にある win セルのチームを勝者とする。スコアに差があれば勝者はスコアから
// 決まるので入れない。
func (p *planner) planWinners() {
	type pos struct {
		br   *parse.Bracket
		r, c int
	}
	games := map[int64][]pos{}
	for _, gp := range p.groups {
		if gp.page == nil || gp.page.Bracket == nil {
			continue
		}
		for r, row := range gp.page.Bracket.Rows {
			for c, cell := range row {
				if cell.Kind == "game" && cell.MatchID != nil {
					games[*cell.MatchID] = append(games[*cell.MatchID], pos{gp.page.Bracket, r, c})
				}
			}
		}
	}
	for _, mp := range p.matches {
		m := &mp.row
		if m.Status != domain.MatchFinished || m.LeftScore == nil || m.RightScore == nil ||
			*m.LeftScore != *m.RightScore || m.ID == 0 {
			continue
		}
		var winner *domain.Side
		for _, at := range games[m.ID] {
			for _, r := range []int{at.r, at.r - 1, at.r + 1} {
				if r < 0 || r >= len(at.br.Rows) || at.c+1 >= len(at.br.Rows[r]) {
					continue
				}
				cell := at.br.Rows[r][at.c+1]
				if cell.Kind != "win" || cell.TeamID == nil || winner != nil {
					continue
				}
				switch {
				case m.LeftTeamID != nil && *cell.TeamID == *m.LeftTeamID:
					winner = ptr(domain.SideLeft)
				case m.RightTeamID != nil && *cell.TeamID == *m.RightTeamID:
					winner = ptr(domain.SideRight)
				}
			}
		}
		if winner == nil {
			if len(games[m.ID]) > 0 {
				p.tally("同点のトーナメント試合で勝者を決められない", fmt.Sprint(m.ID))
			}
			continue
		}
		m.WinnerSide = winner
	}
}

// outcomeLabel は勝ち負けの表記 "[A1-Win]" の CD と勝敗。
var outcomeLabel = regexp.MustCompile(`^\[(.+)-(Win|Lose)\]$`)

// planWinnersFromLabels は、トーナメント表から勝者の分からない同点の試合の勝者を、後の試合の
// 決まり方の表記から決める。"[WK16-Win]" の側のチームは WK16 の勝者、"[WK16-Lose]" は敗者
// (151 の WK16 は 7-7 で試合詳細がなく、表記だけが勝者を示す。参照サイトの成績表もそのチームの
// 勝ちとして数える)。
func (p *planner) planWinnersFromLabels() {
	byCode := map[*divPlan]map[string][]*matchPlan{}
	for _, mp := range p.matches {
		if byCode[mp.div] == nil {
			byCode[mp.div] = map[string][]*matchPlan{}
		}
		byCode[mp.div][mp.row.Code] = append(byCode[mp.div][mp.row.Code], mp)
	}
	for _, mp := range p.matches {
		for _, side := range []struct {
			team  *int64
			label string
		}{{mp.row.LeftTeamID, mp.row.LeftLabelEn}, {mp.row.RightTeamID, mp.row.RightLabelEn}} {
			m := outcomeLabel.FindStringSubmatch(side.label)
			if m == nil || side.team == nil {
				continue
			}
			for _, target := range byCode[mp.div][m[1]] {
				setWinnerOfTie(&target.row, *side.team, m[2] == "Win")
			}
		}
	}
}

// setWinnerOfTie は勝者の決まっていない同点の終了試合に、team が勝った(won=false なら負けた)
// として勝者を入れる。team の出ていない試合は変えない。
func setWinnerOfTie(m *store.Match, team int64, won bool) {
	if m.Status != domain.MatchFinished || m.WinnerSide != nil || m.LeftScore == nil ||
		m.RightScore == nil || *m.LeftScore != *m.RightScore {
		return
	}
	var side domain.Side
	switch {
	case m.LeftTeamID != nil && *m.LeftTeamID == team:
		side = domain.SideLeft
	case m.RightTeamID != nil && *m.RightTeamID == team:
		side = domain.SideRight
	default:
		return
	}
	if !won {
		side = side.Opposite()
	}
	m.WinnerSide = &side
}

// planEventsAndStats は試合ページの得点経過と出場選手を match_events と match_player_stats にする。
// キャラハンはイベント(callahan=true の goal)から数えるので出場記録には入れない(migrations/00002)。
func (p *planner) planEventsAndStats() {
	withPage := make([]*matchPlan, 0, len(p.matches))
	for _, mp := range p.matches {
		if mp.page != nil && mp.row.ID != 0 {
			withPage = append(withPage, mp)
		}
	}
	// チームページの選手一覧にない選手(後で登録が消えた選手など)を、試合ページから先に足す
	for _, mp := range withPage {
		for _, side := range []struct {
			team    *int64
			players []parse.MatchPlayer
		}{{mp.row.LeftTeamID, mp.page.Players.Left}, {mp.row.RightTeamID, mp.page.Players.Right}} {
			if side.team == nil {
				continue
			}
			for _, pl := range side.players {
				if _, ok := p.players[pl.PlayerID]; ok {
					continue
				}
				p.addPlayer(store.Player{
					ID: pl.PlayerID, TeamID: *side.team, Number: pl.Number,
					NameJa: pl.Name.JA, NameEn: pl.Name.EN, Gender: pl.Gender,
				})
				p.tally("チームページにない選手を試合ページから足した", fmt.Sprint(pl.PlayerID))
			}
		}
	}
	for _, mp := range withPage {
		seen := map[int64]bool{}
		for _, side := range []struct {
			side    domain.Side
			players []parse.MatchPlayer
		}{{domain.SideLeft, mp.page.Players.Left}, {domain.SideRight, mp.page.Players.Right}} {
			for _, pl := range side.players {
				if seen[pl.PlayerID] || p.players[pl.PlayerID] == nil {
					p.tally("出場選手表で重複か不明の選手を飛ばした", fmt.Sprintf("%d/%d", mp.row.ID, pl.PlayerID))
					continue
				}
				seen[pl.PlayerID] = true
				stat := store.MatchPlayerStat{
					MatchID: mp.row.ID, PlayerID: pl.PlayerID, Side: side.side,
				}
				if pl.Blocks != nil {
					stat.Blocks = *pl.Blocks
				}
				p.stats = append(p.stats, stat)
			}
		}
		for _, ev := range mp.page.Events {
			if ev.Kind != string(domain.EventEnd) && ev.Side == nil {
				p.tally("側(★)の分からないイベントを飛ばした", fmt.Sprintf("%d seq %d", mp.row.ID, ev.Seq))
				continue
			}
			row := store.MatchEvent{
				MatchID: mp.row.ID, Seq: ev.Seq, ElapsedSec: ev.ElapsedSec,
				Kind:     domain.EventKind(ev.Kind),
				Callahan: ev.Callahan && ev.Kind == string(domain.EventGoal),
			}
			if ev.Side != nil {
				// 試合終了の行の ★ も側として残す(clone-spec D 2.3)
				row.Side = ptr(domain.Side(*ev.Side))
			}
			if ev.Kind == string(domain.EventGoal) {
				row.ScorerID = p.knownPlayer(ev.ScorerID, mp.row.ID)
				row.AssistID = p.knownPlayer(ev.AssistID, mp.row.ID)
			}
			p.events = append(p.events, row)
		}
	}
}

func (p *planner) knownPlayer(id *int64, matchID int64) *int64 {
	if id == nil {
		return nil
	}
	if _, ok := p.players[*id]; !ok {
		p.tally("得点経過の選手が見つからないので空にした", fmt.Sprintf("%d/%d", matchID, *id))
		return nil
	}
	return ptr(*id)
}

// planGreatests は -players で取った選手ページのグレイテスト(通算値)を、その選手が出場した最初の
// 試合の match_player_stats.greatests に入れる。参照サイトには試合ごとの値がないので、最初の試合に
// まとめるのは推測である(選手ページの通算値は一致する)。
func (p *planner) planGreatests() {
	if len(p.b.Players) == 0 {
		return
	}
	ordered := append([]*matchPlan(nil), p.matches...)
	sortMatchPlans(ordered)
	rank := map[int64]int{}
	for i, mp := range ordered {
		if mp.row.ID != 0 {
			rank[mp.row.ID] = i
		}
	}
	for _, id := range sortedKeys(p.b.Players) {
		pl := p.b.Players[id]
		if pl.Stats == nil || pl.Stats.Greatests == 0 {
			continue
		}
		best := -1
		for i, st := range p.stats {
			if st.PlayerID == id && (best < 0 || rank[st.MatchID] < rank[p.stats[best].MatchID]) {
				best = i
			}
		}
		if best < 0 {
			p.tally("出場記録がないのでグレイテストを入れられない選手", fmt.Sprint(id))
			continue
		}
		p.stats[best].Greatests = pl.Stats.Greatests
	}
}
