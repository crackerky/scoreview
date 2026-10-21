package store

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 本部の進行画面(docs/operations/operator-experience.md「当日」)のデータ。運営者が次にやるべきことを
// 上から並べる。

// courtDelayThreshold は「遅れているコート」とみなす遅れ(分)。数分の遅れは試合の入れ替えで
// 自然に吸収されるので、運営者に操作を促すのは 5 分以上からにする(推測。運営会社に確認する)。
const courtDelayThreshold = 5

// venueZone は試合の日付と時刻(タイムゾーンなしの現地時刻)の基準。参照サイトの大会はすべて
// 日本国内なので日本時間にする。
func venueZone() *time.Location { return time.FixedZone("JST", 9*60*60) }

// TodoKind は「いまやること」の種類。
type TodoKind string

// TodoKind の値。
const (
	TodoTie          TodoKind = "tie"           // 未決の同順位
	TodoConfirmGroup TodoKind = "confirm_group" // 確定待ちのプール
	TodoConfirmStage TodoKind = "confirm_stage" // 確定待ちの段(全体順位を使う)
	TodoOverdue      TodoKind = "overdue"       // 予定時刻を過ぎて結果がない試合
	TodoCourtDelay   TodoKind = "court_delay"   // 遅れているコート
	TodoMismatch     TodoKind = "mismatch"      // 対戦チームが順位と食い違っている試合
	TodoOverlap      TodoKind = "overlap"       // 同じチームの試合の時刻が重なっている
	TodoWithdrawn    TodoKind = "withdrawn"     // 棄権したチームが入った試合前の試合
)

// TodoItem は「いまやること」の 1 件。対象の ID は種類に応じて入る。
type TodoItem struct {
	Kind    TodoKind
	Text    string
	GroupID *int64
	StageID *int64
	MatchID *int64
	TeamID  *int64
	Court   *string
	Minutes int
}

// FormatProgress はディビジョンの進行状況。
type FormatProgress struct {
	DivisionID  int64
	Title       string
	Version     int
	Tiebreak    format.TiebreakPreset
	RuleText    []string
	Advancement []string
	SlotMinutes int
	Courts      []string
	Todo        []TodoItem
	Stages      []StageProgress
	Groups      []GroupProgress
	Live        []ProgressMatch
	Next        []ProgressMatch
	Final       []FinalRankRow
	// Withdrawn は棄権したチーム(ID の昇順)、ClosedAt は大会を締めた時刻(締めていなければ nil)、
	// Suspension は大会の中断(中断していなければ nil)
	Withdrawn  []int64
	ClosedAt   *time.Time
	Suspension *Suspension
}

// StageProgress は段の状態。NeedsConfirm は段の全体順位を出どころにする枠があり、まだ確定していない。
type StageProgress struct {
	ID                int64
	Index             int
	Name              string
	Confirmed         bool
	ConfirmedUnplayed bool
	NeedsConfirm      bool
}

// GroupProgress はグループの状態。Rows はプールの順位(全枠が埋まっているとき)、Ties は未決の
// 同順位の組、Conditions は勝ち上がり条件。Label は運営の言葉の名前(format.OperatorName)。
type GroupProgress struct {
	ID                int64
	Letter            string
	Name              string
	Label             string
	Kind              domain.GroupKind
	Stage             int
	State             format.GroupState
	ConfirmedUnplayed bool
	Remaining         int
	Slots             []ProgressSide
	Rows              []ProgressRankRow
	Ties              [][]domain.TeamRef
	Conditions        []TeamCondition
}

// ProgressRankRow はプール順位の 1 行。Reason は同点を分けた根拠(「勝ち点同数、得失点差で上位」)。
type ProgressRankRow struct {
	Team       domain.TeamRef
	Rank       int
	Tally      domain.Tally
	Points     int
	Reason     string
	Unresolved bool
}

// TeamCondition はチームの勝ち上がり条件の文(「次の A3 に勝てば 1 位確定」)。
type TeamCondition struct {
	Team domain.TeamRef
	Text string
}

// ProgressSide は試合の片側や枠。Team が nil なら Label(「A1」「[E2-勝ち]」)を出す。
type ProgressSide struct {
	Label string
	Team  *domain.TeamRef
	Score *int
}

// ProgressMatch は進行画面の試合。GroupLabel はグループの運営の言葉の名前。
type ProgressMatch struct {
	ID         int64
	Code       string
	GroupName  string
	GroupLabel string
	Date       *string
	Time       *string
	Field      *string
	Status     domain.MatchStatus
	Left       ProgressSide
	Right      ProgressSide
}

// FinalRankRow は最終順位の 1 行。
type FinalRankRow struct {
	Rank int
	Team domain.TeamRef
	Tied bool
}

// GetFormatProgress はディビジョンの進行状況を返す。形の指定が無ければ ErrNotFound ではなく
// ErrConflict(ディビジョンはあるが形が無い)。now は「予定時刻を過ぎた」の判定に使う。
func GetFormatProgress(ctx context.Context, db *gorm.DB, divisionID int64, now time.Time) (FormatProgress, error) {
	if _, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision); err != nil {
		return FormatProgress{}, err
	}
	fs, ok, err := loadFormatState(ctx, db, divisionID)
	if err != nil {
		return FormatProgress{}, err
	}
	if !ok {
		return FormatProgress{}, fmt.Errorf("%w: この部門には大会の形が設定されていません", domain.ErrConflict)
	}
	teams, err := fs.loadTeams(ctx, db)
	if err != nil {
		return FormatProgress{}, err
	}
	withdrawn, err := withdrawnTeams(ctx, db, divisionID)
	if err != nil {
		return FormatProgress{}, err
	}
	v := progressBuilder{fs: fs, teams: teams, now: now.In(venueZone()), withdrawn: withdrawn}
	out := v.build()
	if out.Suspension, err = GetSuspension(ctx, db, fs.division.TournamentID); err != nil {
		return FormatProgress{}, err
	}
	return out, nil
}

func (fs *formatState) loadTeams(ctx context.Context, db *gorm.DB) (map[int64]Team, error) {
	var rows []Team
	if err := db.WithContext(ctx).Where(sqlByDivision, fs.division.ID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load teams of division %d: %w", fs.division.ID, err)
	}
	out := make(map[int64]Team, len(rows))
	for _, t := range rows {
		out[t.ID] = t
	}
	return out, nil
}

type progressBuilder struct {
	fs    *formatState
	teams map[int64]Team
	now   time.Time
	// withdrawn は棄権したチームと棄権した時刻(進行画面だけが読む)
	withdrawn map[int64]time.Time
}

func (v progressBuilder) ref(id int64) domain.TeamRef {
	if team, ok := v.teams[id]; ok {
		return teamRef(team)
	}
	return domain.TeamRef{ID: id}
}

func (v progressBuilder) refPtr(id *int64) *domain.TeamRef {
	if id == nil {
		return nil
	}
	ref := v.ref(*id)
	return &ref
}

func (v progressBuilder) slotMinutes() int {
	if v.fs.schedule != nil {
		return v.fs.schedule.SlotMinutes
	}
	return 0
}

func (v progressBuilder) build() FormatProgress {
	fs := v.fs
	out := FormatProgress{DivisionID: fs.division.ID, Title: format.Title(fs.plan), Version: fs.row.Version,
		Tiebreak: format.TiebreakPreset(fs.row.Tiebreak), RuleText: format.DescribeRule(fs.cfg.Tiebreak),
		Advancement: format.DescribeAdvancement(fs.plan), SlotMinutes: v.slotMinutes()}
	if fs.schedule != nil {
		out.Courts = fs.schedule.Courts
	}
	for _, g := range fs.plan.Groups {
		out.Groups = append(out.Groups, v.group(g))
	}
	out.Stages = v.stages()
	out.Todo = v.todo(out.Groups, out.Stages)
	out.Live, out.Next = v.liveAndNext()
	ranked := map[int64]bool{}
	for _, f := range fs.res.Final {
		out.Final = append(out.Final, FinalRankRow{Rank: f.Rank, Team: v.ref(f.TeamID), Tied: f.Tied})
		ranked[f.TeamID] = true
	}
	for team, rank := range fs.closedRanks {
		if !ranked[team] {
			out.Final = append(out.Final, FinalRankRow{Rank: rank, Team: v.ref(team), Tied: true})
		}
	}
	slices.SortStableFunc(out.Final, func(a, b FinalRankRow) int {
		if a.Rank != b.Rank {
			return a.Rank - b.Rank
		}
		return cmp.Compare(a.Team.ID, b.Team.ID)
	})
	for team := range v.withdrawn {
		out.Withdrawn = append(out.Withdrawn, team)
	}
	slices.Sort(out.Withdrawn)
	if fs.closing != nil {
		at := fs.closing.ClosedAt
		out.ClosedAt = &at
	}
	return out
}

func (v progressBuilder) group(g format.Group) GroupProgress {
	fs := v.fs
	row := fs.groups[g.Index]
	status := fs.res.Groups[g.Index]
	gp := GroupProgress{ID: row.ID, Letter: g.Letter, Name: g.Name, Label: format.OperatorName(g), Kind: g.Kind,
		Stage: g.Stage, State: status.State, ConfirmedUnplayed: row.ConfirmedUnplayed}
	for _, slot := range g.Slots {
		st := fs.res.Slots[format.SlotKey{Group: g.Index, Position: slot.Position}]
		side := ProgressSide{Label: slot.Source.OperatorLabel(fs.plan)}
		if st.Resolved() {
			side.Team = v.refPtr(&st.TeamID)
		}
		gp.Slots = append(gp.Slots, side)
	}
	for _, m := range fs.plan.GroupMatches(g.Index) {
		if stored, ok := fs.matches[m.Code]; !ok || stored.Status != domain.MatchFinished {
			gp.Remaining++
		}
	}
	if status.Ranking == nil {
		return gp
	}
	for _, r := range status.Ranking.Rows {
		row := ProgressRankRow{Team: v.ref(r.TeamID), Rank: r.Rank, Tally: tallyOf(r.Stats),
			Points: r.Stats.Points(), Reason: r.Reason.Text(), Unresolved: r.Reason.Unresolved}
		if !status.Done && row.Unresolved {
			// 試合の途中で並んでいるだけで、未決(運営者が決める同順位)ではない
			row.Reason, row.Unresolved = "", false
		}
		gp.Rows = append(gp.Rows, row)
	}
	if !status.Done {
		return v.withConditions(gp, g.Index, status)
	}
	for _, tie := range status.Ranking.Unresolved {
		refs := make([]domain.TeamRef, len(tie))
		for i, id := range tie {
			refs[i] = v.ref(id)
		}
		gp.Ties = append(gp.Ties, refs)
	}
	return v.withConditions(gp, g.Index, status)
}

func (v progressBuilder) withConditions(gp GroupProgress, index int, status format.GroupStatus) GroupProgress {
	if status.State == format.GroupPlaying {
		gp.Conditions = v.conditions(index)
	}
	return gp
}

func tallyOf(s format.Stats) domain.Tally {
	return domain.Tally{Games: s.Games, Wins: s.Wins, Draws: s.Draws, Losses: s.Losses,
		GoalsFor: s.GoalsFor, GoalsAgainst: s.GoalsAgainst}
}

// poolGames はプールのチーム(枠の順)と試合。枠が埋まっていなければ ok=false。
func (fs *formatState) poolGames(index int) ([]int64, []format.Game, bool) {
	g := fs.plan.Groups[index]
	teams := make([]int64, 0, g.Size())
	for _, slot := range g.Slots {
		st := fs.res.Slots[format.SlotKey{Group: index, Position: slot.Position}]
		if !st.Resolved() {
			return nil, nil, false
		}
		teams = append(teams, st.TeamID)
	}
	var games []format.Game
	for _, m := range fs.plan.GroupMatches(index) {
		mt := fs.res.Matches[m.Code]
		game := format.Game{Code: m.Code, Round: m.Round, Left: mt.Left.TeamID, Right: mt.Right.TeamID}
		if res, ok := fs.state.Results[m.Code]; ok && !mt.Mismatch {
			game.Result = &res
		}
		games = append(games, game)
	}
	return teams, games, true
}

// conditions は進行中のプールの勝ち上がり条件。残り試合が多すぎて全列挙できなければ空。
func (v progressBuilder) conditions(index int) []TeamCondition {
	teams, games, ok := v.fs.poolGames(index)
	if !ok {
		return nil
	}
	q, err := format.Qualify(teams, games, v.fs.cfg.Tiebreak)
	if err != nil {
		return nil // 残り試合が MaxRemainingGames を超えるプールでは出さない
	}
	var out []TeamCondition
	for _, look := range q.Teams {
		if text := outlookText(look); text != "" {
			out = append(out, TeamCondition{Team: v.ref(look.TeamID), Text: text})
		}
	}
	return out
}

// outlookText は勝ち上がり条件を運営者と観戦者の言葉にする。
func outlookText(look format.Outlook) string {
	if look.Best == look.Worst && look.Best > 0 {
		return fmt.Sprintf("%d 位が確定", look.Best)
	}
	for _, c := range look.Conditions {
		if c.Match == "" {
			continue
		}
		verb := "に勝てば"
		if c.AllowDraw {
			verb = "で引き分け以上なら"
		}
		if c.AtLeast == 1 {
			return fmt.Sprintf("次の %s %s 1 位確定", c.Match, verb)
		}
		if c.AtLeast == look.Best {
			// これより上の順位にはなれないので「以上」ではなくその順位に決まる
			return fmt.Sprintf("次の %s %s %d 位が確定", c.Match, verb, c.AtLeast)
		}
		return fmt.Sprintf("次の %s %s %d 位以上が確定", c.Match, verb, c.AtLeast)
	}
	return ""
}

func (v progressBuilder) stages() []StageProgress {
	fs := v.fs
	usesBand := map[int]bool{}
	for _, g := range fs.plan.Groups {
		for _, slot := range g.Slots {
			if slot.Source.Kind == format.SourceBandRank {
				usesBand[slot.Source.Stage] = true
			}
		}
	}
	var out []StageProgress
	for i, s := range fs.plan.Stages {
		row := fs.stages[i]
		out = append(out, StageProgress{ID: row.ID, Index: i, Name: s.Name, Confirmed: row.ConfirmedAt != nil,
			ConfirmedUnplayed: row.ConfirmedUnplayed, NeedsConfirm: usesBand[i] && row.ConfirmedAt == nil})
	}
	return out
}

// matchStart は試合の予定開始時刻(現地時刻)。日付か時刻が無ければ ok=false。
func matchStart(m Match) (time.Time, bool) {
	if m.MatchDate == nil || m.StartTime == nil {
		return time.Time{}, false
	}
	clock, err := time.Parse("15:04", (*m.StartTime)[:min(5, len(*m.StartTime))])
	if err != nil {
		return time.Time{}, false
	}
	day := *m.MatchDate
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0,
		venueZone()), true
}

// todo は「いまやること」を緊急度の順に並べる: 時刻を過ぎて結果のない試合 → 同じチームの時刻の
// 重なりと対戦チームの食い違い(このまま試合を始めると取り返しがつかない)→ 確定待ち → 未決の
// 同順位 → コートの遅れ。
func (v progressBuilder) todo(groups []GroupProgress, stages []StageProgress) []TodoItem {
	out := v.overdue()
	out = append(out, v.overlaps()...)
	out = append(out, v.mismatches()...)
	out = append(out, v.withdrawnTodo()...)
	var ties []TodoItem
	for _, g := range groups {
		id := g.ID
		for _, tie := range g.Ties {
			names := make([]string, len(tie))
			for i, t := range tie {
				names[i] = t.Name.Ja
			}
			ties = append(ties, TodoItem{Kind: TodoTie, GroupID: &id,
				Text: fmt.Sprintf("%sで %s が全基準で並んでいます。順位を指定してください", spaced(g.Label),
					joinNames(names))})
		}
		if g.State == format.GroupReady {
			out = append(out, TodoItem{Kind: TodoConfirmGroup, GroupID: &id,
				Text: fmt.Sprintf("%sの全試合が終わりました。順位を確かめて確定してください", spaced(g.Label))})
		}
	}
	for _, s := range stages {
		if !s.NeedsConfirm {
			continue
		}
		ready := true
		for _, g := range groups {
			if g.Stage == s.Index && g.Kind == domain.GroupPool && g.State != format.GroupConfirmed &&
				g.State != format.GroupReady {
				ready = false
			}
		}
		if ready {
			id := s.ID
			out = append(out, TodoItem{Kind: TodoConfirmStage, StageID: &id,
				Text: fmt.Sprintf("%s の全プールが終わりました。段を確定すると全体順位で次の組み合わせが決まります", s.Name)})
		}
	}
	out = append(out, ties...)
	return append(out, v.courtDelays()...)
}

// overlaps は、同じチームが時刻の重なる 2 試合に入っているもの(訂正で対戦チームが入れ替わった、
// 時刻を手で動かした)。両方とも終わっていれば過去のことなので出さない。
func (v progressBuilder) overlaps() []TodoItem {
	slot := time.Duration(max(v.slotMinutes(), 1)) * time.Minute
	type entry struct {
		match Match
		start time.Time
	}
	byTeam := map[int64][]entry{}
	var teams []int64
	for _, m := range v.sortedMatches() {
		start, ok := matchStart(m)
		if !ok {
			continue
		}
		for _, team := range []*int64{m.LeftTeamID, m.RightTeamID} {
			if team == nil {
				continue
			}
			if _, seen := byTeam[*team]; !seen {
				teams = append(teams, *team)
			}
			byTeam[*team] = append(byTeam[*team], entry{match: m, start: start})
		}
	}
	var out []TodoItem
	for _, team := range teams {
		list := byTeam[team]
		for i := 1; i < len(list); i++ {
			prev, cur := list[i-1], list[i]
			if !cur.start.Before(prev.start.Add(slot)) {
				continue
			}
			if prev.match.Status == domain.MatchFinished && cur.match.Status == domain.MatchFinished {
				continue
			}
			id := cur.match.ID
			out = append(out, TodoItem{Kind: TodoOverlap, MatchID: &id,
				Text: fmt.Sprintf("%s が %s の %s(%s)と %s の %s(%s)の両方に入っています。チームか時刻を直してください",
					v.ref(team).Name.Ja, clockOf(prev.match), prev.match.Code, courtLabel(prev.match.Field),
					clockOf(cur.match), cur.match.Code, courtLabel(cur.match.Field))})
		}
	}
	return out
}

// spaced は名前の後ろに助詞を続けるときの区切り。英数字で終わる名前("Pool A")だけ空白を挟む
// (日本語の名前「予選 A 組」には「予選 A 組の」と続ける)。
func spaced(name string) string {
	if r, _ := utf8.DecodeLastRuneInString(name); r < utf8.RuneSelf {
		return name + " "
	}
	return name
}

// clockOf は試合の開始時刻 "HH:MM"(無ければ空)。
func clockOf(m Match) string {
	return deref(m.StartTime)[:min(5, len(deref(m.StartTime)))]
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += "、"
		}
		out += n
	}
	return out
}

func (v progressBuilder) sortedMatches() []Match {
	out := make([]Match, 0, len(v.fs.matches))
	for _, m := range v.fs.matches {
		out = append(out, m)
	}
	slices.SortFunc(out, func(a, b Match) int {
		// cmp.Or は sqlsafety_test が GORM の Or と区別できないので使わない
		if c := compareMatchTime(a, b); c != 0 {
			return c
		}
		if c := domain.CompareFields(deref(a.Field), deref(b.Field)); c != 0 {
			return c
		}
		return cmp.Compare(a.Code, b.Code)
	})
	return out
}

func (v progressBuilder) overdue() []TodoItem {
	var out []TodoItem
	slot := time.Duration(v.slotMinutes()) * time.Minute
	for _, m := range v.sortedMatches() {
		start, ok := matchStart(m)
		if !ok || m.Status == domain.MatchFinished || !start.Add(slot).Before(v.now) {
			continue
		}
		if m.Status == domain.MatchScheduled && (m.LeftTeamID == nil || m.RightTeamID == nil) {
			continue // 対戦相手がまだ決まらない試合は結果を待つ段階ではない(前の試合の方を出す)
		}
		id := m.ID
		late := int(v.now.Sub(start.Add(slot)).Minutes())
		out = append(out, TodoItem{Kind: TodoOverdue, MatchID: &id, Minutes: late,
			Text: fmt.Sprintf("%s(%s %s)は終了予定を %d 分過ぎていますが、結果がありません", m.Code,
				deref(m.StartTime)[:min(5, len(deref(m.StartTime)))], courtLabel(m.Field), late)})
	}
	return out
}

func courtLabel(field *string) string {
	if field == nil {
		return "コート未定"
	}
	return "コート " + *field
}

// courtDelays はコートごとに、まだ始まっていない最初の試合の予定時刻がどれだけ過ぎたかを見る。
func (v progressBuilder) courtDelays() []TodoItem {
	first := map[string]Match{}
	var courts []string
	for _, m := range v.sortedMatches() {
		if m.Field == nil || m.Status != domain.MatchScheduled {
			continue
		}
		// 対戦相手が決まらずに待っている試合の遅れは、コートではなく確定待ちの問題(別の項目で出す)
		if m.LeftTeamID == nil || m.RightTeamID == nil {
			continue
		}
		if _, seen := first[*m.Field]; !seen {
			first[*m.Field] = m
			courts = append(courts, *m.Field)
		}
	}
	var out []TodoItem
	for _, court := range courts {
		m := first[court]
		start, ok := matchStart(m)
		if !ok {
			continue
		}
		late := int(v.now.Sub(start).Minutes())
		if late < courtDelayThreshold {
			continue
		}
		c := court
		out = append(out, TodoItem{Kind: TodoCourtDelay, Court: &c, Minutes: late,
			Text: fmt.Sprintf("コート %s は %s の %s がまだ始まっていません(%d 分遅れ)", court, m.Code,
				deref(m.StartTime)[:min(5, len(deref(m.StartTime)))], late)})
	}
	return out
}

// mismatches は、既に始まった試合で、記録した左右のチームが計算上のチーム(枠のチーム、前の試合の
// 勝者・敗者)と食い違うもの(確定後の訂正で順位が変わった)。結果は下流と最終順位に使っていない。
func (v progressBuilder) mismatches() []TodoItem {
	var out []TodoItem
	for _, m := range v.sortedMatches() {
		if !v.fs.res.Matches[m.Code].Mismatch {
			continue
		}
		id := m.ID
		out = append(out, TodoItem{Kind: TodoMismatch, MatchID: &id,
			Text: fmt.Sprintf("%s の対戦チームが、訂正後の順位と食い違っています。結果は次の試合と最終順位に"+
				"使っていません。チームを差し替えるか、結果を入れ直してください", m.Code)})
	}
	return out
}

func (v progressBuilder) match(m Match) ProgressMatch {
	pm := ProgressMatch{ID: m.ID, Code: m.Code, Date: formatDate(m), Time: formatClock(m), Field: m.Field,
		Status: m.Status}
	if pm.Field != nil {
		field := *pm.Field
		pm.Field = &field
	}
	if g, ok := v.fs.plan.MatchByCode(m.Code); ok {
		pm.GroupName = v.fs.plan.Groups[g.Group].Name
		pm.GroupLabel = format.OperatorName(v.fs.plan.Groups[g.Group])
		pm.Left = v.side(g, g.Left, m.LeftTeamID, m.LeftScore)
		pm.Right = v.side(g, g.Right, m.RightTeamID, m.RightScore)
	}
	return pm
}

func (v progressBuilder) side(m format.Match, side format.SideRef, team *int64, score *int) ProgressSide {
	out := ProgressSide{Team: v.refPtr(team), Score: score}
	if side.From != nil {
		out.Label = side.From.OperatorLabel()
	} else {
		out.Label = v.fs.plan.SlotSource(format.SlotKey{Group: m.Group, Position: side.Slot}).OperatorLabel(v.fs.plan)
	}
	return out
}

// liveAndNext は試合中の試合と、次の時間帯(まだ始まっていない試合のうち最も早い開始時刻)の試合。
func (v progressBuilder) liveAndNext() ([]ProgressMatch, []ProgressMatch) {
	var live, next []ProgressMatch
	var nextStart *time.Time
	for _, m := range v.sortedMatches() {
		switch m.Status {
		case domain.MatchLive:
			live = append(live, v.match(m))
		case domain.MatchScheduled:
			start, ok := matchStart(m)
			if !ok {
				continue
			}
			if nextStart == nil {
				nextStart = &start
			}
			if start.Equal(*nextStart) {
				next = append(next, v.match(m))
			}
		}
	}
	return live, next
}

// ListFormatMatches はディビジョンの試合を進行画面の形で返す(訂正の画面と時間割の画面で使う)。
func ListFormatMatches(ctx context.Context, db *gorm.DB, divisionID int64) ([]ProgressMatch, error) {
	fs, ok, err := loadFormatState(ctx, db, divisionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: この部門には大会の形が設定されていません", domain.ErrConflict)
	}
	teams, err := fs.loadTeams(ctx, db)
	if err != nil {
		return nil, err
	}
	v := progressBuilder{fs: fs, teams: teams}
	var out []ProgressMatch
	for _, m := range v.sortedMatches() {
		out = append(out, v.match(m))
	}
	return out, nil
}
