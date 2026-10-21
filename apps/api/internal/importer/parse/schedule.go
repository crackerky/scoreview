package parse

import (
	"io"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// Schedule はスケジュール scv_search_div/open/{ディビジョンID} または scv_search_tn/open/{大会ID}。
//
// 取込では div 版をディビジョンごとに GET する。tn 版は GET では最終日しか出さず、全日程は
// POST が要るため(clone-spec B 3、B 4)。
type Schedule struct {
	// Scope は "division"(hidden ediv_id)か "tournament"(hidden tid)。ScopeID はその値。
	Scope   string `json:"scope"`
	ScopeID int64  `json:"scope_id"`
	// Dates は日付選択肢(option value の YYYYMMDD)を "YYYY-MM-DD" にしたもの。
	Dates []string `json:"dates"`
	// TeamOptions はチーム選択肢。div 版は value がチーム ID、label が正式名(clone-spec B 4)。
	TeamOptions []ScheduleTeamOption `json:"team_options"`
	Rows        []MatchRow           `json:"rows"`
}

// ScheduleTeamOption はチーム選択肢の 1 つ。
type ScheduleTeamOption struct {
	Value string `json:"value"`
	Label Text   `json:"label"`
}

var ymdPattern = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)

// ParseSchedule はスケジュールを読む。表の日付 "MM/DD" の年は、日付選択肢の YYYYMMDD から補う。
// 選択肢の年は参照サイトの登録値のまま使う(525 は大会名が 2026 でも 2025/10/18。clone-spec A 3)。
func ParseSchedule(r io.Reader, lang site.Lang) (*Schedule, error) {
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
	out := &Schedule{Dates: []string{}, TeamOptions: []ScheduleTeamOption{}}
	if err := parseScheduleScope(sv, out); err != nil {
		return nil, err
	}
	byMonthDay := map[string][]string{}
	var optErr error
	sv.Find(`select[name="ymd"] option`).Each(func(_ int, opt *goquery.Selection) {
		v := clean(opt.AttrOr("value", ""))
		if v == "" {
			return
		}
		m := ymdPattern.FindStringSubmatch(v)
		if m == nil {
			optErr = unexpected("date option %q", v)
			return
		}
		date := m[1] + "-" + m[2] + "-" + m[3]
		out.Dates = append(out.Dates, date)
		byMonthDay[m[2]+"/"+m[3]] = append(byMonthDay[m[2]+"/"+m[3]], date)
	})
	if optErr != nil {
		return nil, optErr
	}
	sv.Find(`select[name="team"] option`).Each(func(_ int, opt *goquery.Selection) {
		v := clean(opt.AttrOr("value", ""))
		if v == "" {
			return
		}
		out.TeamOptions = append(out.TeamOptions, ScheduleTeamOption{Value: v, Label: textOf(lang, cellText(opt))})
	})

	table := sv.Find("div.TBL table").First()
	if table.Length() == 0 {
		return nil, unexpected("schedule table not found")
	}
	if out.Rows, err = parseMatchRows(table, scheduleLayout(), lang); err != nil {
		return nil, err
	}
	for i := range out.Rows {
		dates := byMonthDay[out.Rows[i].MonthDay]
		switch len(dates) {
		case 0:
			return nil, unexpected("row %s: date %q is not in the date options",
				out.Rows[i].Code, out.Rows[i].MonthDay)
		case 1:
			out.Rows[i].Date = ptr(dates[0])
		default:
			out.Rows[i].DateCandidates = dates
		}
	}
	return out, nil
}

func parseScheduleScope(sv *goquery.Selection, out *Schedule) error {
	for _, h := range []struct{ name, scope string }{{"ediv_id", "division"}, {"tid", "tournament"}} {
		input := sv.Find(`input[name="` + h.name + `"]`)
		if input.Length() == 0 {
			continue
		}
		id, err := strconv.ParseInt(clean(input.AttrOr("value", "")), 10, 64)
		if err != nil {
			return unexpected("hidden %s: %v", h.name, err)
		}
		out.Scope, out.ScopeID = h.scope, id
		return nil
	}
	return unexpected("schedule has neither ediv_id nor tid")
}

// MergeScheduleEN は日本語版のスケジュールに英語の短縮名とディビジョン名を入れる。
// チーム選択肢の英語名は div 版でだけ入れる。tn 版は value がチーム名そのもので、英語名が未登録の
// チームは空の選択肢になり件数が揃わない(clone-spec B 4)。
func MergeScheduleEN(jp, en *Schedule) (*Schedule, error) {
	if jp.Scope != en.Scope || jp.ScopeID != en.ScopeID {
		return nil, mismatch("schedule scope jp=%s/%d en=%s/%d", jp.Scope, jp.ScopeID, en.Scope, en.ScopeID)
	}
	out := *jp
	if jp.Scope == "division" {
		if len(jp.TeamOptions) != len(en.TeamOptions) {
			return nil, mismatch("team options jp=%d en=%d", len(jp.TeamOptions), len(en.TeamOptions))
		}
		out.TeamOptions = make([]ScheduleTeamOption, len(jp.TeamOptions))
		for i, o := range jp.TeamOptions {
			if o.Value != en.TeamOptions[i].Value {
				return nil, mismatch("team option %d: value jp=%s en=%s", i, o.Value, en.TeamOptions[i].Value)
			}
			o.Label = mergeText(o.Label, en.TeamOptions[i].Label)
			out.TeamOptions[i] = o
		}
	}
	rows, err := mergeMatchRowsEN(jp.Rows, en.Rows)
	if err != nil {
		return nil, err
	}
	out.Rows = rows
	return &out, nil
}
