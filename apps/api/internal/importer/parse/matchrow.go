package parse

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// MatchRow は試合の一覧表(スケジュール、グループの結果一覧、チーム対戦一覧)の 1 行。
type MatchRow struct {
	// MatchID は状況列の [対戦結果] リンクの試合 ID。未試合と「終了」の行にはリンクがなく nil
	// (clone-spec B 6、D 2)。
	MatchID *int64 `json:"match_id"`
	// HasDetail は状況列が「終了」(en "END")のとき false。試合詳細ページがない終了試合である。
	HasDetail bool `json:"has_detail"`
	// Status は行の状態クラス MC_MAE / MC_NOW / MC_END を scheduled / live / finished にしたもの。
	Status string `json:"status"`
	// DivisionKind と DivisionLabel はスケジュールのディビジョン列だけにある。DivisionLabel は
	// 表示幅 10 で切り詰めた名前(clone-spec B 7)で、正式名ではない。
	DivisionKind  string `json:"division_kind,omitempty"`
	DivisionLabel *Text  `json:"division_label,omitempty"`
	Code          string `json:"code"`
	// MonthDay は表示どおりの "MM/DD"。Date は年を補った "YYYY-MM-DD" で、年が分かるページ
	// (スケジュールの日付選択肢)のときだけ入る。
	MonthDay string  `json:"month_day"`
	Date     *string `json:"date"`
	// DateCandidates は日付選択肢に同じ月日が複数の年で出る(172 の 2017/08/10 と 2018/08/10)ため
	// 年を決められないときの候補。そのとき Date は nil。
	DateCandidates []string `json:"date_candidates,omitempty"`
	Time           *string  `json:"time"`
	// Field はコート列の表示どおりの値。数字でないコート("A" など、467)がある。FieldNo は数字のときだけ入る。
	Field      string       `json:"field"`
	FieldNo    *int         `json:"field_no"`
	Left       MatchRowSide `json:"left"`
	Right      MatchRowSide `json:"right"`
	LeftScore  *int         `json:"left_score"`
	RightScore *int         `json:"right_score"`
}

// MatchRowSide は対戦の片側。確定していれば TeamID と短縮名、未確定なら Placeholder。
type MatchRowSide struct {
	TeamID *int64 `json:"team_id"`
	// ShortName は短縮名(正式名ではない。clone-spec B 7、D 1.3)。
	ShortName   Text         `json:"short_name"`
	Placeholder *Placeholder `json:"placeholder"`
}

// Placeholder は未確定の対戦相手 "[A1-勝ち]"(en "[A1-Win]")。clone-spec B 8。
type Placeholder struct {
	Code    string `json:"code"`
	Outcome string `json:"outcome"` // win / lose
}

var placeholderPattern = regexp.MustCompile(`^\[(.+)-(勝ち|負け|Win|Lose)\]$`)

func parsePlaceholder(s string) *Placeholder {
	m := placeholderPattern.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	outcome := "win"
	if m[2] == "負け" || m[2] == "Lose" {
		outcome = "lose"
	}
	return &Placeholder{Code: m[1], Outcome: outcome}
}

// matchRowLayout は表ごとの列の位置。
type matchRowLayout struct {
	cells    int
	division int // ディビジョン列の位置。ない表は -1
	code     int
	date     int
	time     int
	field    int
	left     int
	lscore   int
	rscore   int
	right    int
	status   int
}

// scheduleLayout は scv_search_tn / scv_search_div の 11 列(clone-spec B 5)。
func scheduleLayout() matchRowLayout {
	return matchRowLayout{
		cells: 11, division: 0, date: 1, time: 2, field: 3, code: 4,
		left: 5, lscore: 6, rscore: 8, right: 9, status: 10,
	}
}

// listLayout はグループの結果一覧とチーム対戦一覧の 10 列(clone-spec C 2.1(c)、D 1.1)。
func listLayout() matchRowLayout {
	return matchRowLayout{
		cells: 10, division: -1, code: 0, date: 1, time: 2, field: 3,
		left: 4, lscore: 5, rscore: 7, right: 8, status: 9,
	}
}

func rowStatus(td *goquery.Selection) (string, bool) {
	switch {
	case td.HasClass("MC_END"):
		return "finished", true
	case td.HasClass("MC_MAE"):
		return "scheduled", true
	case td.HasClass("MC_NOW"):
		// 推測: 試合中の行は調査時に観測できなかった(clone-spec B 6)。CSS の定義だけから対応させる
		return "live", true
	}
	return "", false
}

func divisionKindOf(td *goquery.Selection) string {
	switch {
	case td.HasClass("MC_MEN"):
		return "men"
	case td.HasClass("MC_WOM"):
		return "women"
	case td.HasClass("MC_MIX"):
		return "mixed"
	}
	return "none"
}

// parseMatchRows は table の中の試合行(先頭セルが MC_* の行)を読む。見出し行は飛ばす。
func parseMatchRows(table *goquery.Selection, layout matchRowLayout, lang site.Lang) ([]MatchRow, error) {
	out := []MatchRow{}
	var rowErr error
	table.Find("tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		tds := tr.ChildrenFiltered("td")
		status, ok := rowStatus(tds.First())
		if !ok {
			return true
		}
		if tds.Length() != layout.cells {
			rowErr = unexpected("match row has %d cells, want %d", tds.Length(), layout.cells)
			return false
		}
		row, err := parseMatchRow(tds, layout, lang)
		if err != nil {
			rowErr = err
			return false
		}
		row.Status = status
		out = append(out, row)
		return true
	})
	return out, rowErr
}

func parseMatchRow(tds *goquery.Selection, layout matchRowLayout, lang site.Lang) (MatchRow, error) {
	cell := func(i int) *goquery.Selection { return tds.Eq(i) }
	row := MatchRow{
		Code:     cellText(cell(layout.code)),
		MonthDay: cellText(cell(layout.date)),
		Left:     parseMatchRowSide(cell(layout.left), lang),
		Right:    parseMatchRowSide(cell(layout.right), lang),
	}
	if layout.division >= 0 {
		row.DivisionKind = divisionKindOf(cell(layout.division))
		row.DivisionLabel = ptr(textOf(lang, cellText(cell(layout.division))))
	}
	if t := cellText(cell(layout.time)); t != "" {
		if !validTime(t) {
			return row, unexpected("time %q is not HH:MM", t)
		}
		row.Time = &t
	}
	row.Field = cellText(cell(layout.field))
	if n, err := strconv.Atoi(row.Field); err == nil {
		row.FieldNo = &n
	}
	var err error
	if row.LeftScore, err = atoiPtr(cellText(cell(layout.lscore))); err != nil {
		return row, err
	}
	if row.RightScore, err = atoiPtr(cellText(cell(layout.rscore))); err != nil {
		return row, err
	}
	statusCell := cell(layout.status)
	row.MatchID = linkID(statusCell, site.PageMatch)
	// clone-spec B 6: 「終了」/"END" はスコアがあるのに試合詳細がない終了試合
	text := cellText(statusCell)
	row.HasDetail = row.MatchID != nil || (text != "終了" && text != "END")
	return row, nil
}

func parseMatchRowSide(td *goquery.Selection, lang site.Lang) MatchRowSide {
	text := cellText(td)
	side := MatchRowSide{TeamID: linkID(td, site.PageTeam)}
	if side.TeamID == nil {
		if ph := parsePlaceholder(text); ph != nil {
			side.Placeholder = ph
			return side
		}
	}
	side.ShortName = textOf(lang, text)
	return side
}

// mergeMatchRowsEN は jp の行に en の短縮名とディビジョン名を入れる。行は同じ順に並ぶ前提で、
// 試合 ID、CD、チーム ID が一致しなければ ErrMismatch。
func mergeMatchRowsEN(jp, en []MatchRow) ([]MatchRow, error) {
	if len(jp) != len(en) {
		return nil, mismatch("match rows jp=%d en=%d", len(jp), len(en))
	}
	out := make([]MatchRow, len(jp))
	for i, j := range jp {
		e := en[i]
		if !equalIDPtr(j.MatchID, e.MatchID) || j.Code != e.Code ||
			!equalIDPtr(j.Left.TeamID, e.Left.TeamID) || !equalIDPtr(j.Right.TeamID, e.Right.TeamID) {
			return nil, mismatch("match row %d (%s) differs between jp and en", i, j.Code)
		}
		j.Left.ShortName = mergeText(j.Left.ShortName, e.Left.ShortName)
		j.Right.ShortName = mergeText(j.Right.ShortName, e.Right.ShortName)
		if j.DivisionLabel != nil && e.DivisionLabel != nil {
			j.DivisionLabel = ptr(mergeText(*j.DivisionLabel, *e.DivisionLabel))
		}
		out[i] = j
	}
	return out, nil
}

// validTime は s が "HH:MM" か。表の時刻は "09:00 " のように末尾に空白が付く(clone-spec B 5)ので、
// cellText で除いてから渡す。
func validTime(s string) bool {
	return len(s) == 5 && s[2] == ':' && strings.Trim(s[:2]+s[3:], "0123456789") == ""
}
