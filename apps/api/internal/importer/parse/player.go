package parse

import (
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// Player は選手ページ scv_player/open/{選手ID}。取込に要るのはキャラハンとグレイテストの数だけ
// (チームページと試合ページにない値。clone-spec D 3)。ページに選手自身の ID はない。
type Player struct {
	TeamID        int64 `json:"team_id"`
	TeamShortName Text  `json:"team_short_name"`
	// PrevID と NextID は同チーム内の ID 順で前後の選手(末尾と先頭は循環。clone-spec D 3.1)。
	PrevID *int64 `json:"prev_id"`
	NextID *int64 `json:"next_id"`
	// IsStaff のとき Role が入り、Number は空。スタッフページには成績もイベントもない。
	IsStaff bool   `json:"is_staff"`
	Number  string `json:"number"`
	Role    Text   `json:"role"`
	Name    Text   `json:"name"`
	// StatsKind は成績表の列構成 "ultimate" / "discgolf"(divisions.stats_kind)。スタッフは空。
	StatsKind string       `json:"stats_kind"`
	Stats     *PlayerStats `json:"stats"`
	// Events は本人がゴールかアシストをした得点(古い試合から)。
	Events []PlayerEvent `json:"events"`
}

// PlayerStats はアルティメットの通算成績。キャラハンとグレイテストは試合ごとの内訳が分からない。
type PlayerStats struct {
	Games     int `json:"games"`
	Assists   int `json:"assists"`
	Goals     int `json:"goals"`
	Total     int `json:"total"`
	Callahans int `json:"callahans"`
	Greatests int `json:"greatests"`
	Blocks    int `json:"blocks"`
}

// PlayerEvent はイベント表の 1 行。
type PlayerEvent struct {
	Code       string `json:"code"`
	MonthDay   string `json:"month_day"`
	OppTeamID  *int64 `json:"opp_team_id"`
	ElapsedSec int    `json:"elapsed_sec"`
	Kind       string `json:"kind"`
	Callahan   bool   `json:"callahan"`
	ScorerID   *int64 `json:"scorer_id"`
	AssistID   *int64 `json:"assist_id"`
}

var (
	playerHeadingPattern = regexp.MustCompile(`^#(\S*) : (.*)$`)
	staffHeadingPattern  = regexp.MustCompile(`^(.*?) : (.*)$`)
	leadingIntPattern    = regexp.MustCompile(`^(-?\d+)`)
)

// ParsePlayer は選手ページを読む。
func ParsePlayer(r io.Reader, lang site.Lang) (*Player, error) {
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
	out := &Player{Events: []PlayerEvent{}}
	nav := sv.Find("table.WMAX").First().Find("td")
	if nav.Length() != 3 {
		return nil, unexpected("player nav has %d cells", nav.Length())
	}
	out.PrevID = linkID(nav.Eq(0), site.PagePlayer)
	out.NextID = linkID(nav.Eq(2), site.PagePlayer)
	teamID := linkID(nav.Eq(1), site.PageTeam)
	if teamID == nil {
		return nil, unexpected("player page without team link")
	}
	out.TeamID = *teamID
	short := strings.TrimSuffix(strings.TrimPrefix(cellText(nav.Eq(1)), "["), "]")
	out.TeamShortName = textOf(lang, short)

	// 見出し "{短縮名} - #{背番号} : {名前}"、スタッフは "{短縮名} - {役割} : {名前}"(clone-spec D 3.1)
	heading := cellText(sv.Find("span.SCV_MIDA_M").First())
	rest, ok := strings.CutPrefix(heading, short+" - ")
	if !ok {
		return nil, unexpected("player heading %q does not start with team %q", heading, short)
	}
	if m := playerHeadingPattern.FindStringSubmatch(rest); m != nil {
		out.Number, out.Name = m[1], textOf(lang, clean(m[2]))
	} else if m := staffHeadingPattern.FindStringSubmatch(rest); m != nil {
		out.IsStaff = true
		out.Role, out.Name = textOf(lang, clean(m[1])), textOf(lang, clean(m[2]))
	} else {
		return nil, unexpected("player heading %q", heading)
	}

	var tblErr error
	sv.Find("div.TBL > table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		h := headerLabels(table)
		switch {
		case len(h) == 0:
		case h[0] == "CD":
			out.Events, tblErr = parsePlayerEvents(table)
		case containsAny(h, "キャラハン", "Callahan"):
			out.StatsKind = "ultimate"
			out.Stats, tblErr = parsePlayerStats(table)
		case h[0] == "試合数" || h[0] == "Games":
			// ディスクゴルフの列(得点 / キャッチ / スローミス…)。値は全大会で 0 だった(clone-spec D 3.1 の 6)
			out.StatsKind = "discgolf"
		}
		return tblErr == nil
	})
	if tblErr != nil {
		return nil, tblErr
	}
	return out, nil
}

func parsePlayerStats(table *goquery.Selection) (*PlayerStats, error) {
	tds := table.Find("tr").Eq(1).ChildrenFiltered("td")
	if tds.Length() != 7 {
		return nil, unexpected("player stats has %d cells", tds.Length())
	}
	st := &PlayerStats{}
	dsts := []*int{&st.Games, &st.Assists, &st.Goals, &st.Total, &st.Callahans, &st.Greatests, &st.Blocks}
	for k, dst := range dsts {
		// "5 &nbsp;&nbsp;(avg:1.25)" の先頭の数。ブロック 0 は空セル(clone-spec D 3.1)
		text := cellText(tds.Eq(k))
		if text == "" {
			continue
		}
		m := leadingIntPattern.FindStringSubmatch(text)
		if m == nil {
			return nil, unexpected("player stats value %q", text)
		}
		n, err := atoi(m[1])
		if err != nil {
			return nil, err
		}
		*dst = n
	}
	return st, nil
}

// parsePlayerEvents はイベント表を読む。対戦チームのセルは </td> が閉じられていないが、
// HTML パーサが次の <td> で閉じる(clone-spec D 3.1)。
func parsePlayerEvents(table *goquery.Selection) ([]PlayerEvent, error) {
	out := []PlayerEvent{}
	var err error
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != 7 {
			err = unexpected("player event row has %d cells", tds.Length())
			return false
		}
		ev := PlayerEvent{
			Code:      cellText(tds.Eq(0)),
			MonthDay:  cellText(tds.Eq(1)),
			OppTeamID: linkID(tds.Eq(2), site.PageTeam),
			ScorerID:  linkID(tds.Eq(5), site.PagePlayer),
			AssistID:  linkID(tds.Eq(6), site.PagePlayer),
		}
		if ev.ElapsedSec, err = elapsedSec(cellText(tds.Eq(3))); err != nil {
			return false
		}
		kind, ok := eventKind(cellText(tds.Eq(4)))
		if !ok {
			err = unexpected("unknown player event %q", cellText(tds.Eq(4)))
			return false
		}
		ev.Kind = kind
		ev.Callahan = isCallahan(cellText(tds.Eq(4)))
		out = append(out, ev)
		return true
	})
	return out, err
}

// MergePlayerEN は日本語版の選手ページに英語の名前、役割、チーム短縮名を入れる。
func MergePlayerEN(jp, en *Player) (*Player, error) {
	if jp.TeamID != en.TeamID || jp.IsStaff != en.IsStaff || jp.Number != en.Number ||
		len(jp.Events) != len(en.Events) {
		return nil, mismatch("player page differs between jp and en")
	}
	out := *jp
	out.TeamShortName = mergeText(jp.TeamShortName, en.TeamShortName)
	out.Role = mergeText(jp.Role, en.Role)
	out.Name = mergeText(jp.Name, en.Name)
	return &out, nil
}
