package parse

import (
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// Match は試合ページ scv_match/open/{試合ID}。ページに試合自身の ID はないので、呼び出し側が
// URL の ID と対応させる。
type Match struct {
	Code string `json:"code"`
	// GroupName は試合情報行のグループ名。2017 年の旧データでは空(clone-spec D 2.2)。
	GroupName string `json:"group_name"`
	// Date は "YYYY-MM-DD"(試合ページは年 4 桁で表示する)。
	Date string  `json:"date"`
	Time *string `json:"time"`
	// Field は表示どおりのコート、FieldNo は数字のときだけ。
	Field   string    `json:"field"`
	FieldNo *int      `json:"field_no"`
	Left    MatchSide `json:"left"`
	Right   MatchSide `json:"right"`
	// Breadcrumb はコメントアウトされたパンくず。大会の短縮名と、試合が属するディビジョンと
	// グループの ID が分かる唯一の箇所(clone-spec D 2.1)。
	Breadcrumb *Breadcrumb `json:"breadcrumb"`
	// Events は得点経過を古い順に並べ直したもの(ページは新しい順。clone-spec D 2.3)。
	Events []MatchEvent `json:"events"`
	// Players は出場選手表。
	Players MatchPlayers `json:"players"`
	// BlockColumn は出場選手表に Block 列があるか(試合ごとに違う)。
	BlockColumn bool `json:"block_column"`
	// Stats は統計表。統計表のない試合(2017 年の 151)では nil。
	Stats *MatchStats `json:"stats"`
	// Note はスコアボード下の行の中央の備考("スクラッチのため" など。多くは空)。
	Note string `json:"note"`
}

// MatchSide は対戦の片側。
type MatchSide struct {
	TeamID    int64 `json:"team_id"`
	ShortName Text  `json:"short_name"`
	Score     *int  `json:"score"`
}

// Breadcrumb はパンくずの内容。
type Breadcrumb struct {
	TournamentID int64 `json:"tournament_id"`
	// TournamentShortName は tournaments.short_name(例 "2026 Atami-B")。
	TournamentShortName string `json:"tournament_short_name"`
	DivisionID          *int64 `json:"division_id"`
	GroupID             *int64 `json:"group_id"`
}

// MatchEvent は得点経過の 1 行。
type MatchEvent struct {
	// Seq は古い順の通し番号(1 始まり)。同じ経過時刻の順序もこれで決まる。
	Seq        int    `json:"seq"`
	ElapsedSec int    `json:"elapsed_sec"`
	Kind       string `json:"kind"` // goal / timeout / pull / end
	// Side は goal なら得点した側、timeout は取った側、pull はプルした側(★ の列)。end は ★ が
	// あればその列の側、なければ nil。
	Side     *string `json:"side"`
	ScorerID *int64  `json:"scorer_id"`
	AssistID *int64  `json:"assist_id"`
	// Callahan はイベント列が「キャラハンゴール」の得点(アシストなし)。clone-spec に未記載で、
	// 525 の取込で見つかった(試合 45310)。
	Callahan bool `json:"callahan"`
	// Score は goal の行に出る得点側の累計(検証用)。
	Score *int `json:"score"`
}

// MatchPlayers は左右の出場選手。
type MatchPlayers struct {
	Left  []MatchPlayer `json:"left"`
	Right []MatchPlayer `json:"right"`
}

// MatchPlayer は出場選手表の 1 行。
type MatchPlayer struct {
	PlayerID int64   `json:"player_id"`
	Number   string  `json:"number"`
	Name     Text    `json:"name"`
	Gender   *string `json:"gender"`
	Assists  int     `json:"assists"`
	Goals    int     `json:"goals"`
	// Blocks は Block 列のない試合(44347、45130 など。clone-spec に未記載の変種)では nil。
	Blocks *int `json:"blocks"`
}

// MatchStats は統計表のうち、イベントから計算できない値の検証に使う数。
// ターンオーバーからの得点数は試合単位の数しか分からない(どの得点かは表に出ない)。
type MatchStats struct {
	Left  SideStats `json:"left"`
	Right SideStats `json:"right"`
}

// SideStats は片側の統計。
type SideStats struct {
	Goals         int `json:"goals"`
	FromOffence   int `json:"from_offence"`
	FromDefence   int `json:"from_defence"`
	FromTurnovers int `json:"from_turnovers"`
	Timeouts      int `json:"timeouts"`
}

var (
	matchInfoPattern  = regexp.MustCompile(`^(\d{4})/(\d{2})/(\d{2})\s*(\d{2}:\d{2})?\s*@Field:\s*(.*)$`)
	statsValuePattern = regexp.MustCompile(`^(\d+)`)
)

// eventKind は得点経過のイベント列の文言を種類に変える(clone-spec D 2.3、D 4)。
// キャラハンゴールは goal として扱い、呼び出し側が isCallahan で印を付ける。
func eventKind(s string) (string, bool) {
	if isCallahan(s) {
		return "goal", true
	}
	switch s {
	case "得点", "Goal":
		return "goal", true
	case "タイムアウト", "Timeout":
		return "timeout", true
	case "試合開始ｽﾛｰｵﾌ", "1st Pull":
		return "pull", true
	case "試合終了", "End":
		return "end", true
	}
	return "", false
}

// isCallahan はイベント列の文言がキャラハンゴールか。英語版の文言は未観測なので推測で
// "Callahan" を含むものとする。
func isCallahan(s string) bool {
	return s == "キャラハンゴール" || strings.Contains(s, "Callahan")
}

// ParseMatch は試合ページを読む。
func ParseMatch(r io.Reader, lang site.Lang) (*Match, error) {
	if err := checkLang(lang); err != nil {
		return nil, err
	}
	doc, err := load(r)
	if err != nil {
		return nil, err
	}
	sv, err := scoreView(doc)
	if err != nil {
		return nil, err
	}
	out := &Match{Events: []MatchEvent{}, Players: MatchPlayers{Left: []MatchPlayer{}, Right: []MatchPlayer{}}}
	board := sv.Find("div.TBL").First().ChildrenFiltered("table")
	if board.Length() < 2 {
		return nil, unexpected("score board has %d tables", board.Length())
	}
	if err := parseMatchInfo(board.Eq(0), out); err != nil {
		return nil, err
	}
	if err := parseMatchScore(board.Eq(1), out, lang); err != nil {
		return nil, err
	}
	if board.Length() >= 3 {
		// フッター行: -対戦結果- / 備考 / 表示時刻
		if tds := board.Eq(2).Find("td"); tds.Length() == 3 {
			out.Note = cellText(tds.Eq(1))
		}
	}
	out.BlockColumn = len(headerLabels(sv.Find("table#lplayers").First())) == 5
	if out.Breadcrumb, err = parseBreadcrumb(doc); err != nil {
		return nil, err
	}
	if out.Events, err = parseMatchEvents(sv.Find("table#events")); err != nil {
		return nil, err
	}
	if out.Players.Left, err = parseMatchPlayers(sv.Find("table#lplayers"), lang); err != nil {
		return nil, err
	}
	if out.Players.Right, err = parseMatchPlayers(sv.Find("table#rplayers"), lang); err != nil {
		return nil, err
	}
	if out.Stats, err = parseMatchStats(sv); err != nil {
		return nil, err
	}
	return out, nil
}

func parseMatchInfo(table *goquery.Selection, out *Match) error {
	tds := table.Find("td")
	if tds.Length() < 3 {
		return unexpected("match info has %d cells", tds.Length())
	}
	out.Code = cellText(tds.Eq(0))
	out.GroupName = cellText(tds.Eq(1))
	info := strings.ReplaceAll(cellText(tds.Eq(tds.Length()-1)), " ", " ")
	m := matchInfoPattern.FindStringSubmatch(info)
	if m == nil {
		return unexpected("match info %q", info)
	}
	out.Date = m[1] + "-" + m[2] + "-" + m[3]
	if m[4] != "" {
		out.Time = ptr(m[4])
	}
	out.Field = clean(m[5])
	if n, err := strconv.Atoi(out.Field); err == nil {
		out.FieldNo = &n
	}
	return nil
}

func parseMatchScore(table *goquery.Selection, out *Match, lang site.Lang) error {
	tds := table.Find("tr").First().ChildrenFiltered("td")
	if tds.Length() != 5 {
		return unexpected("score row has %d cells", tds.Length())
	}
	for _, side := range []struct {
		dst        *MatchSide
		team, pt   int
		sideLabels string
	}{{&out.Left, 0, 1, "left"}, {&out.Right, 4, 3, "right"}} {
		id := linkID(tds.Eq(side.team), site.PageTeam)
		if id == nil {
			return unexpected("%s team link not found", side.sideLabels)
		}
		score, err := atoiPtr(cellText(tds.Eq(side.pt)))
		if err != nil {
			return err
		}
		*side.dst = MatchSide{TeamID: *id, ShortName: textOf(lang, cellText(tds.Eq(side.team))), Score: score}
	}
	return nil
}

func parseBreadcrumb(doc *goquery.Document) (*Breadcrumb, error) {
	for _, c := range comments(doc.Find("#score_topmenu")) {
		if !strings.Contains(c, "breadcrumb") {
			continue
		}
		frag, err := goquery.NewDocumentFromReader(strings.NewReader(c))
		if err != nil {
			return nil, unexpected("breadcrumb: %v", err)
		}
		links := frag.Find("#breadcrumb a")
		top := linkID(links, site.PageTournamentTop)
		if top == nil {
			return nil, unexpected("breadcrumb without tournament link")
		}
		return &Breadcrumb{
			TournamentID:        *top,
			TournamentShortName: cellText(links.First()),
			DivisionID:          linkID(links, site.PageDivision),
			GroupID:             linkID(links, site.PageGroup),
		}, nil
	}
	return nil, nil
}

// parseMatchEvents は得点経過を読み、古い順に並べ直して seq を振る。
// 列: 経過 / アシスト / ゴール / 点 / イベント / 点 / ゴール / アシスト(clone-spec D 2.3)。
func parseMatchEvents(table *goquery.Selection) ([]MatchEvent, error) {
	if table.Length() == 0 {
		return nil, unexpected("events table not found")
	}
	var events []MatchEvent
	var err error
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != 8 {
			err = unexpected("event row has %d cells", tds.Length())
			return false
		}
		var ev MatchEvent
		ev, err = parseMatchEvent(tds)
		if err != nil {
			return false
		}
		events = append(events, ev)
		return true
	})
	if err != nil {
		return nil, err
	}
	out := make([]MatchEvent, 0, len(events))
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		ev.Seq = len(out) + 1
		out = append(out, ev)
	}
	return out, nil
}

func parseMatchEvent(tds *goquery.Selection) (MatchEvent, error) {
	text := func(i int) string { return cellText(tds.Eq(i)) }
	var ev MatchEvent
	var err error
	if ev.ElapsedSec, err = elapsedSec(text(0)); err != nil {
		return ev, err
	}
	kind, ok := eventKind(text(4))
	if !ok {
		return ev, unexpected("unknown event %q", text(4))
	}
	ev.Kind = kind
	ev.Callahan = isCallahan(text(4))
	leftFilled := text(1) != "" || text(2) != "" || text(3) != ""
	rightFilled := text(5) != "" || text(6) != "" || text(7) != ""
	if leftFilled && rightFilled {
		return ev, unexpected("event %q has both sides filled", text(4))
	}
	switch {
	case leftFilled:
		ev.Side = ptr("left")
	case rightFilled:
		ev.Side = ptr("right")
	default:
		// timeout / pull は ★ の列で側が決まる。★ のない行は側不明として nil のまま返す
		return ev, nil
	}
	if kind == "end" {
		// 試合終了の行にも ★ が付くことがある(44803、44867。どちらの側かの意味は未確認)
		return ev, nil
	}
	if kind != "goal" {
		return ev, nil
	}
	scorer, assist, point := 2, 1, 3
	if *ev.Side == "right" {
		scorer, assist, point = 6, 7, 5
	}
	ev.ScorerID = linkID(tds.Eq(scorer), site.PagePlayer)
	ev.AssistID = linkID(tds.Eq(assist), site.PagePlayer)
	if ev.Score, err = atoiPtr(text(point)); err != nil {
		return ev, err
	}
	return ev, nil
}

// parseMatchPlayers は出場選手表を読む。列: # / Player / Assist / Goal / Block(clone-spec D 2.4)。
// Block 列のない表もある。
func parseMatchPlayers(table *goquery.Selection, lang site.Lang) ([]MatchPlayer, error) {
	out := []MatchPlayer{}
	if table.Length() == 0 {
		return nil, unexpected("players table not found")
	}
	cols := len(headerLabels(table.First()))
	if cols != 5 && cols != 4 {
		return nil, unexpected("match player table has %d columns", cols)
	}
	var err error
	table.First().Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != cols {
			err = unexpected("match player row has %d cells, want %d", tds.Length(), cols)
			return false
		}
		id := linkID(tds.Eq(1), site.PagePlayer)
		if id == nil {
			err = unexpected("match player row without link")
			return false
		}
		p := MatchPlayer{
			PlayerID: *id,
			Number:   cellText(tds.Eq(0)),
			Name:     textOf(lang, cellText(tds.Eq(1))),
			Gender:   genderOf(tds.Eq(1).Find("a").First()),
		}
		if p.Assists, err = atoiBlank(cellText(tds.Eq(2))); err != nil {
			return false
		}
		if p.Goals, err = atoiBlank(cellText(tds.Eq(3))); err != nil {
			return false
		}
		if cols == 5 {
			var blocks int
			if blocks, err = atoiBlank(cellText(tds.Eq(4))); err != nil {
				return false
			}
			p.Blocks = &blocks
		}
		out = append(out, p)
		return true
	})
	return out, err
}

// parseMatchStats は統計表を読む。行は文言で見分ける(jp / en)。コメントアウトされた
// オフェンス時間などの行は DOM に出ないので読まない(clone-spec D 2.5)。
func parseMatchStats(sv *goquery.Selection) (*MatchStats, error) {
	var table *goquery.Selection
	sv.Find("div.TBL > table").EachWithBreak(func(_ int, t *goquery.Selection) bool {
		h := headerLabels(t)
		if len(h) > 0 && (h[0] == "統計情報" || h[0] == "Game Statistics") {
			table = t
			return false
		}
		return true
	})
	if table == nil {
		return nil, nil
	}
	out := &MatchStats{}
	var err error
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		tds := tr.ChildrenFiltered("td")
		if i == 0 || tds.Length() != 3 {
			return true
		}
		var lp, rp *int
		switch cellText(tds.Eq(0)) {
		case "得点数", "Goals":
			lp, rp = &out.Left.Goals, &out.Right.Goals
		case "オフェンスからの得点数", "Goals from starting on offence":
			lp, rp = &out.Left.FromOffence, &out.Right.FromOffence
		case "ディフェンスからの得点数", "Goals from starting on deffence":
			lp, rp = &out.Left.FromDefence, &out.Right.FromDefence
		case "ターンオーバーからの得点数", "Goals from turnovers":
			lp, rp = &out.Left.FromTurnovers, &out.Right.FromTurnovers
		case "タイムアウト数", "Time outs":
			lp, rp = &out.Left.Timeouts, &out.Right.Timeouts
		default:
			return true
		}
		if *lp, err = statsValue(cellText(tds.Eq(1))); err != nil {
			return false
		}
		*rp, err = statsValue(cellText(tds.Eq(2)))
		return err == nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// statsValue は "5   [71.43%]" の先頭の数を返す。
func statsValue(s string) (int, error) {
	m := statsValuePattern.FindStringSubmatch(s)
	if m == nil {
		return 0, unexpected("stats value %q", s)
	}
	return atoi(m[1])
}

// MergeMatchEN は日本語版の試合ページに英語のチーム短縮名と選手名を入れる。
func MergeMatchEN(jp, en *Match) (*Match, error) {
	if jp.Left.TeamID != en.Left.TeamID || jp.Right.TeamID != en.Right.TeamID || jp.Code != en.Code ||
		len(jp.Events) != len(en.Events) {
		return nil, mismatch("match %s differs between jp and en", jp.Code)
	}
	out := *jp
	out.Left.ShortName = mergeText(jp.Left.ShortName, en.Left.ShortName)
	out.Right.ShortName = mergeText(jp.Right.ShortName, en.Right.ShortName)
	var err error
	if out.Players.Left, err = mergeMatchPlayersEN(jp.Players.Left, en.Players.Left); err != nil {
		return nil, err
	}
	if out.Players.Right, err = mergeMatchPlayersEN(jp.Players.Right, en.Players.Right); err != nil {
		return nil, err
	}
	return &out, nil
}

func mergeMatchPlayersEN(jp, en []MatchPlayer) ([]MatchPlayer, error) {
	if len(jp) != len(en) {
		return nil, mismatch("match players jp=%d en=%d", len(jp), len(en))
	}
	out := make([]MatchPlayer, len(jp))
	for i, p := range jp {
		if p.PlayerID != en[i].PlayerID {
			return nil, mismatch("match player %d differs", i)
		}
		p.Name = mergeText(p.Name, en[i].Name)
		out[i] = p
	}
	return out, nil
}
