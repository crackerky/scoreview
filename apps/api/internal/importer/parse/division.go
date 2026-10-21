package parse

import (
	"io"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// DivisionStandings はディビジョン成績 scv_division/open/{ディビジョンID}。
// 取込で使うのは順位表の順位(teams.final_rank)だけで、リーダー表は検証用。
type DivisionStandings struct {
	Rows []StandingRow `json:"rows"`
	// Leaders はスコアボードリーダー(得点)、BlockLeaders は(ブロック)。データがあるときだけ出る節。
	Leaders      []LeaderRow `json:"leaders"`
	BlockLeaders []LeaderRow `json:"block_leaders"`
}

// StandingRow は順位表の 1 行。並びは順位の昇順(clone-spec C 1.4)。
type StandingRow struct {
	TeamID       int64  `json:"team_id"`
	ShortName    Text   `json:"short_name"`
	Games        int    `json:"games"`
	Wins         int    `json:"wins"`
	Draws        int    `json:"draws"`
	Losses       int    `json:"losses"`
	GoalsFor     int    `json:"goals_for"`
	GoalsAgainst int    `json:"goals_against"`
	GoalDiff     int    `json:"goal_diff"`
	WinPct       string `json:"win_pct"`
	// Rank は順位決定戦の結果としての順位。未確定なら空欄で nil(clone-spec C 1.4)。
	Rank *int `json:"rank"`
}

// LeaderRow はスコアボードリーダーの 1 行。チーム名はリンクなしの短縮名。
type LeaderRow struct {
	TeamName string  `json:"team_name"`
	Number   string  `json:"number"`
	PlayerID *int64  `json:"player_id"`
	Player   Text    `json:"player"`
	Gender   *string `json:"gender"`
	Games    int     `json:"games"`
	// Assists と Goals は得点表だけ、Blocks はブロック表だけに入る。
	Assists int `json:"assists"`
	Goals   int `json:"goals"`
	Total   int `json:"total"`
	Blocks  int `json:"blocks"`
}

// ParseDivisionStandings はディビジョン成績を読む。表の種類は見出し行の文言で見分ける
// (順位表は「勝率」、得点表は「トータル」、ブロック表は「ブロック」の列を持つ)。
// 旧大会の SOTG ランキング表(clone-spec E 6)は読まずに飛ばす。
func ParseDivisionStandings(r io.Reader, lang site.Lang) (*DivisionStandings, error) {
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
	out := &DivisionStandings{Rows: []StandingRow{}, Leaders: []LeaderRow{}, BlockLeaders: []LeaderRow{}}
	var tblErr error
	sv.Find("div.TBL > table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		header := headerLabels(table)
		switch {
		case containsAny(header, "勝率", "Win %"):
			out.Rows, tblErr = parseStandingRows(table, lang)
		case containsAny(header, "トータル", "Tot."):
			out.Leaders, tblErr = parseLeaderRows(table, lang, false)
		case containsAny(header, "ブロック", "Block"):
			out.BlockLeaders, tblErr = parseLeaderRows(table, lang, true)
		}
		return tblErr == nil
	})
	if tblErr != nil {
		return nil, tblErr
	}
	return out, nil
}

// headerLabels は表の 1 行目のセルの文字列を返す。
func headerLabels(table *goquery.Selection) []string {
	var out []string
	table.Find("tr").First().ChildrenFiltered("td").Each(func(_ int, td *goquery.Selection) {
		out = append(out, cellText(td))
	})
	return out
}

func containsAny(list []string, wants ...string) bool {
	for _, s := range list {
		for _, w := range wants {
			if s == w {
				return true
			}
		}
	}
	return false
}

func parseStandingRows(table *goquery.Selection, lang site.Lang) ([]StandingRow, error) {
	out := []StandingRow{}
	var err error
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != 10 {
			err = unexpected("standings row has %d cells", tds.Length())
			return false
		}
		id := linkID(tds.Eq(0), site.PageTeam)
		if id == nil {
			err = unexpected("standings row without team link")
			return false
		}
		row := StandingRow{TeamID: *id, ShortName: textOf(lang, cellText(tds.Eq(0))), WinPct: cellText(tds.Eq(8))}
		nums := []*int{&row.Games, &row.Wins, &row.Draws, &row.Losses, &row.GoalsFor, &row.GoalsAgainst, &row.GoalDiff}
		for k, dst := range nums {
			if *dst, err = atoi(cellText(tds.Eq(k + 1))); err != nil {
				return false
			}
		}
		if row.Rank, err = atoiPtr(cellText(tds.Eq(9))); err != nil {
			return false
		}
		out = append(out, row)
		return true
	})
	return out, err
}

// parseLeaderRows はリーダー表を読む。得点表は 7 列(チーム # 選手 試合数 アシスト ゴール トータル)、
// ブロック表は 6 列(チーム # 選手 試合数 ブロック --)。clone-spec C 1.5。
func parseLeaderRows(table *goquery.Selection, lang site.Lang, blocks bool) ([]LeaderRow, error) {
	want := 7
	if blocks {
		want = 6
	}
	out := []LeaderRow{}
	var err error
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != want {
			err = unexpected("leader row has %d cells, want %d", tds.Length(), want)
			return false
		}
		a := tds.Eq(2).Find("a").First()
		row := LeaderRow{
			TeamName: cellText(tds.Eq(0)),
			Number:   cellText(tds.Eq(1)),
			PlayerID: linkID(tds.Eq(2), site.PagePlayer),
			Player:   textOf(lang, cellText(tds.Eq(2))),
			Gender:   genderOf(a),
		}
		if row.Games, err = atoi(cellText(tds.Eq(3))); err != nil {
			return false
		}
		if blocks {
			row.Blocks, err = atoiBlank(cellText(tds.Eq(4)))
		} else {
			if row.Assists, err = atoi(cellText(tds.Eq(4))); err != nil {
				return false
			}
			if row.Goals, err = atoi(cellText(tds.Eq(5))); err != nil {
				return false
			}
			row.Total, err = atoi(cellText(tds.Eq(6)))
		}
		if err != nil {
			return false
		}
		out = append(out, row)
		return true
	})
	return out, err
}

// MergeDivisionStandingsEN は日本語版の順位表とリーダー表に英語の短縮名と選手名を入れる。
func MergeDivisionStandingsEN(jp, en *DivisionStandings) (*DivisionStandings, error) {
	if len(jp.Rows) != len(en.Rows) || len(jp.Leaders) != len(en.Leaders) ||
		len(jp.BlockLeaders) != len(en.BlockLeaders) {
		return nil, mismatch("division standings row counts differ")
	}
	out := &DivisionStandings{
		Rows:         make([]StandingRow, len(jp.Rows)),
		Leaders:      make([]LeaderRow, len(jp.Leaders)),
		BlockLeaders: make([]LeaderRow, len(jp.BlockLeaders)),
	}
	for i, row := range jp.Rows {
		if row.TeamID != en.Rows[i].TeamID {
			return nil, mismatch("standings row %d: team jp=%d en=%d", i, row.TeamID, en.Rows[i].TeamID)
		}
		row.ShortName = mergeText(row.ShortName, en.Rows[i].ShortName)
		out.Rows[i] = row
	}
	for _, pair := range []struct{ jp, en, dst []LeaderRow }{
		{jp.Leaders, en.Leaders, out.Leaders}, {jp.BlockLeaders, en.BlockLeaders, out.BlockLeaders},
	} {
		for i, row := range pair.jp {
			if !equalIDPtr(row.PlayerID, pair.en[i].PlayerID) {
				return nil, mismatch("leader row %d: player differs", i)
			}
			row.Player = mergeText(row.Player, pair.en[i].Player)
			pair.dst[i] = row
		}
	}
	return out, nil
}
