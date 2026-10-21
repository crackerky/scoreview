//go:build corpus

// 参照サイトの調査で取得した実 HTML(.local/survey-raw、git 管理外、実在の個人名を含む)の全ファイルに
// パーサを掛け、成功・失敗の件数と、取込で注意が要る変種の件数を集計する手元確認用のテスト。
// CI では実行しない。
//
//	cd apps/api && go test -tags=corpus -run TestCorpus -v ./internal/importer/parse/
//
// 場所は SCOREVIEW_CORPUS_DIR で変えられる。既定はカレントから親をたどって見つけた .local/survey-raw。
package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"scoreview/api/internal/importer/site"
)

func corpusDir(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("SCOREVIEW_CORPUS_DIR"); dir != "" {
		return dir
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for dir := wd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		cand := filepath.Join(dir, ".local", "survey-raw")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
	}
	t.Skip(".local/survey-raw not found")
	return ""
}

var (
	pscoreName = regexp.MustCompile(`pscore_(scv_[a-z_]+?)_open_`)
	catalogTop = regexp.MustCompile(`^t\d+\.html$`)
	bSchedule  = regexp.MustCompile(`^(tn|en|nolang|d|p|s)\d`)
)

// corpusKind はファイル名からページの種類を決める。調査時の保存名の規則による。
func corpusKind(rel string) string {
	base := filepath.Base(rel)
	dir := filepath.Dir(rel)
	if m := pscoreName.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	switch {
	case strings.HasPrefix(base, "sctnmt"), base == "top.html":
		return "sctnmt"
	case strings.HasPrefix(base, "scview_"), strings.HasPrefix(base, "scvtop_"),
		dir == "E" && catalogTop.MatchString(base), base == "t525.html", base == "top_999999.html":
		return "scv_top"
	case strings.HasPrefix(base, "tl_"):
		return "scv_team_list"
	case strings.HasPrefix(base, "tg_"):
		return "scv_tgallery"
	case strings.HasPrefix(base, "stn_"), dir == "B" && bSchedule.MatchString(base):
		return "scv_search"
	}
	return ""
}

var errSkip = errors.New("skip")

func parseCorpusFile(kind string, body []byte, lang site.Lang) (any, error) {
	r := bytes.NewReader(body)
	switch kind {
	case "sctnmt":
		return ParseTournamentList(r, lang)
	case "scv_top":
		return ParseTournamentTop(r, lang)
	case "scv_team_list":
		return ParseTeamList(r, lang)
	case "scv_search", "scv_search_div", "scv_search_tn":
		return ParseSchedule(r, lang)
	case "scv_division":
		return ParseDivisionStandings(r, lang)
	case "scv_group":
		return ParseGroup(r, lang)
	case "scv_team":
		return ParseTeam(r, lang)
	case "scv_match":
		return ParseMatch(r, lang)
	case "scv_tgallery":
		return ParseGallery(r, lang)
	case "scv_player":
		return ParsePlayer(r, lang)
	}
	return nil, errSkip
}

// observe は取込で注意が要る変種を数えるための印を返す。失敗ではない。
func observe(v any) []string {
	var notes []string
	note := func(cond bool, s string) {
		if cond {
			notes = append(notes, s)
		}
	}
	switch p := v.(type) {
	case *TournamentTop:
		for _, d := range p.Divisions {
			note(d.ID == nil, "division without id")
			for _, g := range d.Groups {
				note(!g.Linked, "group without link")
			}
		}
		note(len(p.Divisions) == 0, "no divisions")
		note(p.LogoURL == nil, "default logo")
		note(p.Nav.TeamListLabel == "entries", "label entries")
		note(!p.Nav.ShowSchedule, "no schedule")
	case *TeamList:
		for _, d := range p.Divisions {
			note(d.EntryKind == "individual", "individual division")
			for _, e := range d.Entries {
				note(e.ID == nil, "entry without id")
				note(e.BarStyle != "default" && e.BarStyle != "mixed" && e.BarStyle != "women",
					"bar style "+e.BarStyle)
			}
		}
	case *Schedule:
		for _, r := range p.Rows {
			note(r.MatchID == nil && r.Status == "finished" && r.HasDetail, "finished row without match id")
			note(!r.HasDetail, "finished without detail (終了)")
			note(r.Date == nil, "row with ambiguous year")
			note(r.FieldNo == nil, "non-numeric field")
			note(r.Left.Placeholder != nil || r.Right.Placeholder != nil, "placeholder")
			note(r.Status == "live", "live row")
			note(r.DivisionKind == "men", "MC_MEN")
		}
	case *DivisionStandings:
		note(len(p.Rows) == 0, "empty standings")
		for _, r := range p.Rows {
			note(r.Rank == nil, "standing without rank")
		}
	case *Group:
		note(true, "kind "+p.Kind)
		if p.Pool != nil {
			for _, tm := range p.Pool.Teams {
				note(tm.TeamID == nil, "pool team without id")
				note(tm.Rank == nil, "pool team without rank")
			}
			for _, c := range p.Pool.Cells {
				note(c.MatchID == nil, "pool cell without match link")
			}
		}
		if p.Bracket != nil {
			note(p.Bracket.Pattern == "", "bracket without pattern")
			for _, row := range p.Bracket.Rows {
				for _, c := range row {
					note(c.Kind == "game" && c.MatchID == nil, "bracket game without match link")
				}
			}
		}
	case *Team:
		note(p.Record == nil, "team without record")
		note(len(p.Staff) > 0, "team with staff")
		for _, pl := range p.Players {
			note(pl.Gender == nil, "player without gender")
		}
	case *Match:
		note(p.Breadcrumb == nil, "match without breadcrumb")
		note(p.GroupName == "", "match without group name")
		hasEnd := false
		last := map[string]int{}
		for _, e := range p.Events {
			hasEnd = hasEnd || e.Kind == "end"
			note(e.Kind != "end" && e.Side == nil, "event without side")
			note(e.Kind == "goal" && e.ScorerID == nil, "goal without scorer")
			if e.Kind == "goal" && e.Side != nil && e.Score != nil {
				note(*e.Score != last[*e.Side]+1, "goal score not cumulative")
				last[*e.Side] = *e.Score
			}
		}
		note(!hasEnd, "match without end event")
		note(p.Left.Score != nil && last["left"] != *p.Left.Score && len(p.Events) > 0,
			"left score differs from events")
		note(p.Right.Score != nil && last["right"] != *p.Right.Score && len(p.Events) > 0,
			"right score differs from events")
		note(len(p.Players.Left) > 0 && p.Players.Left[0].Blocks == nil, "players without block column")
	case *Player:
		note(p.IsStaff, "staff page")
		note(p.StatsKind == "discgolf", "discgolf stats")
	}
	return notes
}

// corpusPairs は日英が揃っているファイル。Merge 関数が構造を突き合わせられるかを確かめる。
var corpusPairs = []struct{ kind, jp, en string }{
	{"sctnmt", "A/sctnmt_nolang.html", "A/sctnmt_en.html"},
	{"scv_top", "A/scview_525.html", "A/scview_525_en.html"},
	{"scv_top", "A/scview_172.html", "A/scview_172_en.html"},
	{"scv_top", "A/scview_537.html", "A/scview_537_en.html"},
	{"scv_team_list", "A/tl_525.html", "A/tl_525_en.html"},
	{"scv_tgallery", "A/tg_362.html", "A/tg_362_en.html"},
	{"scv_search", "B/nolang525.html", "B/en525.html"},
	{"scv_search", "B/tn151.html", "B/en151.html"},
	{"scv_search", "B/tn479.html", "B/en479.html"},
	{"scv_search", "B/tn493.html", "B/en493.html"},
	{"scv_search", "B/tn226.html", "B/en226.html"},
	{"scv_group", "C/pscore_scv_group_open_5895_lang_jp.html", "C/pscore_scv_group_open_5895_lang_en.html"},
	{"scv_group", "C/pscore_scv_group_open_6623_lang_jp.html", "C/pscore_scv_group_open_6623_lang_en.html"},
	{"scv_group", "C/pscore_scv_group_open_6628_lang_jp.html", "C/pscore_scv_group_open_6628_lang_en.html"},
	{"scv_team", "D/pscore_scv_team_open_17322_lang_jp.html", "D/pscore_scv_team_open_17322_lang_en.html"},
	{"scv_team", "D/pscore_scv_team_open_17459_lang_jp.html", "D/pscore_scv_team_open_17459_lang_en.html"},
	{"scv_team", "D/pscore_scv_team_open_17461_lang_jp.html", "D/pscore_scv_team_open_17461_lang_en.html"},
	{"scv_team", "D/pscore_scv_team_open_4990_lang_jp.html", "D/pscore_scv_team_open_4990_lang_en.html"},
	{"scv_match", "D/pscore_scv_match_open_45277_lang_jp.html", "D/pscore_scv_match_open_45277_lang_en.html"},
	{"scv_match", "D/pscore_scv_match_open_44347_lang_jp.html", "D/pscore_scv_match_open_44347_lang_en.html"},
	{"scv_match", "D/pscore_scv_match_open_14693_lang_jp.html", "D/pscore_scv_match_open_14693_lang_en.html"},
	{"scv_player", "D/pscore_scv_player_open_604138_lang_jp.html", "D/pscore_scv_player_open_604138_lang_en.html"},
}

func mergeCorpus(jp, en any) error {
	var err error
	switch j := jp.(type) {
	case *TournamentList:
		_, err = MergeTournamentListEN(j, en.(*TournamentList))
	case *TournamentTop:
		_, err = MergeTournamentTopEN(j, en.(*TournamentTop))
	case *TeamList:
		_, err = MergeTeamListEN(j, en.(*TeamList))
	case *Gallery:
		_, err = MergeGalleryEN(j, en.(*Gallery))
	case *Schedule:
		_, err = MergeScheduleEN(j, en.(*Schedule))
	case *Group:
		_, err = MergeGroupEN(j, en.(*Group))
	case *Team:
		_, err = MergeTeamEN(j, en.(*Team))
	case *Match:
		_, err = MergeMatchEN(j, en.(*Match))
	case *Player:
		_, err = MergePlayerEN(j, en.(*Player))
	default:
		err = fmt.Errorf("no merge for %T", jp)
	}
	return err
}

type tally struct {
	ok, errorPage int
	failures      []string
	notes         map[string]int
}

func TestCorpus(t *testing.T) {
	root := corpusDir(t)
	results := map[string]*tally{}
	skipped := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		kind := corpusKind(rel)
		if kind == "" {
			skipped++
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lang, ok := site.LangOf(string(body))
		if !ok {
			lang = site.LangJP
		}
		tl := results[kind]
		if tl == nil {
			tl = &tally{notes: map[string]int{}}
			results[kind] = tl
		}
		if bytes.HasPrefix(body, []byte("HTTP Error")) {
			// 調査ツールが HTTP エラーの応答の代わりに保存した 1 行(本文ではない)
			tl.errorPage++
			return nil
		}
		v, perr := parseCorpusFile(kind, body, lang)
		switch {
		case perr == nil:
			tl.ok++
			for _, n := range observe(v) {
				tl.notes[n]++
			}
		case errors.Is(perr, ErrErrorPage):
			tl.errorPage++
		default:
			tl.failures = append(tl.failures, fmt.Sprintf("%s: %v", rel, perr))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	kinds := make([]string, 0, len(results))
	for k := range results {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	failed := 0
	for _, k := range kinds {
		tl := results[k]
		t.Logf("%-16s ok=%4d error_page=%2d failed=%d", k, tl.ok, tl.errorPage, len(tl.failures))
		for _, f := range tl.failures {
			t.Logf("    FAIL %s", f)
		}
		notes := make([]string, 0, len(tl.notes))
		for n := range tl.notes {
			notes = append(notes, n)
		}
		sort.Strings(notes)
		for _, n := range notes {
			t.Logf("    note %-36s %d", n, tl.notes[n])
		}
		failed += len(tl.failures)
	}
	t.Logf("skipped (not a page parser target) = %d", skipped)

	for _, p := range corpusPairs {
		load := func(rel string) (any, error) {
			body, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				return nil, err
			}
			lang, ok := site.LangOf(string(body))
			if !ok {
				lang = site.LangJP
			}
			return parseCorpusFile(p.kind, body, lang)
		}
		jp, err := load(p.jp)
		if err != nil {
			t.Logf("merge %s: jp: %v", p.jp, err)
			failed++
			continue
		}
		en, err := load(p.en)
		if err != nil {
			t.Logf("merge %s: en: %v", p.en, err)
			failed++
			continue
		}
		if err := mergeCorpus(jp, en); err != nil {
			t.Logf("MERGE FAIL %s + %s: %v", p.jp, p.en, err)
			failed++
			continue
		}
		t.Logf("merge ok %s", p.jp)
	}
	if failed > 0 {
		t.Errorf("%d files or pairs failed", failed)
	}
}

// TestCorpusDump は SCOREVIEW_DUMP に指定した 1 ファイル(corpus からの相対パス)のパース結果を
// JSON で表示する。パーサの出力を目で確かめるため。
func TestCorpusDump(t *testing.T) {
	rel := os.Getenv("SCOREVIEW_DUMP")
	if rel == "" {
		t.Skip("SCOREVIEW_DUMP is not set")
	}
	root := corpusDir(t)
	body, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	lang, ok := site.LangOf(string(body))
	if !ok {
		lang = site.LangJP
	}
	v, err := parseCorpusFile(corpusKind(rel), body, lang)
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(out))
}
