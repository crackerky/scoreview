package parse

import (
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// TeamList はチーム一覧 scv_team_list/open/{大会ID}。
type TeamList struct {
	// Divisions は大会トップの表と同じ順(clone-spec A 5)。見出しにディビジョン ID はないので、
	// 大会トップと並び順で対応させる。
	Divisions []TeamListDivision `json:"divisions"`
}

// TeamListDivision は見出し 1 つ分。
type TeamListDivision struct {
	Name Text `json:"name"`
	// Count は見出しの "-- N Teams --" の N。
	Count int `json:"count"`
	// EntryKind は "team"(Teams、チーム名がリンク)か "individual"(Players、ディスクゴルフ個人戦で
	// 名前がリンクなし)。clone-spec A 5、E 7。
	EntryKind string          `json:"entry_kind"`
	Entries   []TeamListEntry `json:"entries"`
}

// TeamListEntry は 1 チーム(個人戦では 1 人)。並びはディビジョン内の登録順(ID 順ではない)。
type TeamListEntry struct {
	// ID は個人戦ではリンクがないので nil。
	ID *int64 `json:"id"`
	// Name は正式名(短縮名ではない)。
	Name Text `json:"name"`
	// BarStyle は色帯の class(FTitle → default など)。teams.bar_style の値。
	BarStyle string `json:"bar_style"`
	// Members は黒いセルの "x11" の人数。個人戦は &nbsp; で nil。
	Members *int `json:"members"`
}

var teamListHeading = regexp.MustCompile(`^(.*?)\s+--\s+(\d+) (Teams|Players)\s+--$`)

var membersPattern = regexp.MustCompile(`^x(\d+)$`)

// barStyleOf は色帯の class を teams.bar_style の値に変える(clone-spec A 2.2、A 5)。
func barStyleOf(sel *goquery.Selection) (string, bool) {
	switch {
	case sel.HasClass("FTitle"):
		return "default", true
	case sel.HasClass("FTitle_Mix"):
		return "mixed", true
	case sel.HasClass("FTitle_Women"):
		return "women", true
	case sel.HasClass("FTitle_Univ"):
		return "univ", true
	case sel.HasClass("FTitle_Waku1"):
		return "waku1", true
	case sel.HasClass("FTitle_Waku2"):
		return "waku2", true
	}
	return "", false
}

// ParseTeamList はチーム一覧を読む。
func ParseTeamList(r io.Reader, lang site.Lang) (*TeamList, error) {
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
	out := &TeamList{Divisions: []TeamListDivision{}}
	var cur *TeamListDivision
	var pending *TeamListEntry
	var walkErr error
	// 見出し、チーム名の色帯、人数の黒いセルが文書順に並ぶ
	sv.Find(".TeamListMIDA, .TeamListBLOCK > div").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		switch {
		case sel.HasClass("TeamListMIDA"):
			text := strings.ReplaceAll(sel.Text(), " ", " ")
			m := teamListHeading.FindStringSubmatch(clean(text))
			if m == nil {
				walkErr = unexpected("team list heading %q", clean(text))
				return false
			}
			n, err := strconv.Atoi(m[2])
			if err != nil {
				walkErr = unexpected("team count %q", m[2])
				return false
			}
			kind := "team"
			if m[3] == "Players" {
				kind = "individual"
			}
			out.Divisions = append(out.Divisions, TeamListDivision{
				Name: textOf(lang, clean(m[1])), Count: n, EntryKind: kind, Entries: []TeamListEntry{},
			})
			cur = &out.Divisions[len(out.Divisions)-1]
		case sel.HasClass("AreaSepa"):
			if cur == nil || pending == nil {
				walkErr = unexpected("member cell without team")
				return false
			}
			if t := cellText(sel); t != "" {
				m := membersPattern.FindStringSubmatch(t)
				if m == nil {
					walkErr = unexpected("member cell %q", t)
					return false
				}
				n, err := strconv.Atoi(m[1])
				if err != nil {
					walkErr = unexpected("member count %q", t)
					return false
				}
				pending.Members = &n
			}
			cur.Entries = append(cur.Entries, *pending)
			pending = nil
		default:
			style, ok := barStyleOf(sel)
			if !ok || cur == nil {
				walkErr = unexpected("team bar without style or heading: %q", cellText(sel))
				return false
			}
			name := sel.Find(".teamimg_rnk")
			pending = &TeamListEntry{
				ID:       linkID(name, site.PageTeam),
				Name:     textOf(lang, cellText(name)),
				BarStyle: style,
			}
		}
		return true
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if pending != nil {
		return nil, unexpected("team %q without member cell", pending.Name)
	}
	for _, d := range out.Divisions {
		if len(d.Entries) != d.Count {
			return nil, unexpected("division %q: heading says %d, found %d", d.Name, d.Count, len(d.Entries))
		}
	}
	return out, nil
}

// MergeTeamListEN は日本語版のチーム一覧に英語のディビジョン名とチーム名を入れる。
func MergeTeamListEN(jp, en *TeamList) (*TeamList, error) {
	if len(jp.Divisions) != len(en.Divisions) {
		return nil, mismatch("team list divisions jp=%d en=%d", len(jp.Divisions), len(en.Divisions))
	}
	out := &TeamList{Divisions: make([]TeamListDivision, len(jp.Divisions))}
	for i, d := range jp.Divisions {
		e := en.Divisions[i]
		if len(d.Entries) != len(e.Entries) {
			return nil, mismatch("team list division %d: entries jp=%d en=%d", i, len(d.Entries), len(e.Entries))
		}
		d.Name = mergeText(d.Name, e.Name)
		entries := make([]TeamListEntry, len(d.Entries))
		for k, je := range d.Entries {
			ee := e.Entries[k]
			if !equalIDPtr(je.ID, ee.ID) {
				return nil, mismatch("team list division %d entry %d: id differs", i, k)
			}
			je.Name = mergeText(je.Name, ee.Name)
			entries[k] = je
		}
		d.Entries = entries
		out.Divisions[i] = d
	}
	return out, nil
}
