package parse

import (
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// Group はグループ成績 scv_group/open/{グループID}。Kind に応じて Pool / Bracket / Image の
// どれか 1 つが入る。
type Group struct {
	// DivisionName と Name は見出し "< {ディビジョン名} - {グループ名} >" から取る。
	// グループ名は言語によらず同じ(clone-spec A 1.3)なので Name は文字列。
	DivisionName Text        `json:"division_name"`
	Name         string      `json:"name"`
	Kind         string      `json:"kind"` // pool / bracket / image
	Pool         *Pool       `json:"pool"`
	Bracket      *Bracket    `json:"bracket"`
	Image        *GroupImage `json:"image"`
	// Matches は「結果一覧」の試合(プールとトーナメントにある)。日付は年なしの MM/DD。
	Matches []MatchRow `json:"matches"`
}

// Pool は総当たり。clone-spec C 2.1。
type Pool struct {
	// Teams はリーグ結果の行。並びはマトリクスと同じ登録順で、順位順ではない。
	Teams []PoolTeam `json:"teams"`
	// Cells はマトリクスの上三角で試合のあるセル。スコアは試合記録の左右順のまま(行チーム視点ではない)。
	Cells []PoolCell `json:"cells"`
}

// PoolTeam はリーグ結果の 1 行。
type PoolTeam struct {
	TeamID       *int64 `json:"team_id"`
	ShortName    Text   `json:"short_name"`
	Games        int    `json:"games"`
	Wins         int    `json:"wins"`
	Draws        int    `json:"draws"`
	Losses       int    `json:"losses"`
	GoalsFor     int    `json:"goals_for"`
	GoalsAgainst int    `json:"goals_against"`
	GoalDiff     int    `json:"goal_diff"`
	// RankRaw は順位セルの生の値。計算した順位と比べて group_teams.rank_override を決めるために
	// 残す(空欄になる例外がある。clone-spec C 2.2)。Rank は数値のときだけ入る。
	RankRaw string `json:"rank_raw"`
	Rank    *int   `json:"rank"`
}

// PoolCell はマトリクスの 1 セル。Row と Col はリーグ結果の Teams の添字(0 始まり)。
type PoolCell struct {
	Row     int    `json:"row"`
	Col     int    `json:"col"`
	MatchID *int64 `json:"match_id"`
	Score   string `json:"score"`
	Live    bool   `json:"live"`
}

// Bracket はトーナメント表。1 枚の table のセル罫線で描かれている(clone-spec C 2.3)ので、
// 格子をそのまま持つ。
type Bracket struct {
	Pattern string          `json:"pattern"`  // GeneratePtn。例 "T8A"
	PtnName string          `json:"ptn_name"` // PtnName。例 "2026 Atami Line =Tournament A="
	Columns int             `json:"columns"`
	Rows    [][]BracketCell `json:"rows"`
}

// BracketCell は格子の 1 セル。
type BracketCell struct {
	Top    bool `json:"top"`
	Right  bool `json:"right"`
	Bottom bool `json:"bottom"`
	Left   bool `json:"left"`
	// Kind は TN_TMPL のセルの class(team, win, lose, game, txt_yen, txt_sharp, txt_and,
	// txt_left, txt_right, rank)。空セルは ""。
	Kind string `json:"kind"`
	// Text はセルの文字列。チーム名(短縮名)は言語で変わり、見出し("Placement" など)は
	// 日英とも英語固定(clone-spec C 2.3)。
	Text    Text   `json:"text"`
	MatchID *int64 `json:"match_id"`
	TeamID  *int64 `json:"team_id"`
}

// GroupImage は画像だけのグループ(Temporary ranking など。clone-spec E 10)。
type GroupImage struct {
	URL    string `json:"url"`
	Width  *int   `json:"width"`
	Height *int   `json:"height"`
}

var (
	generatePtnPattern = regexp.MustCompile(`^\s*GeneratePtn:\s*(\S+)\s*$`)
	ptnNamePattern     = regexp.MustCompile(`^\s*PtnName\s*:(.*?)\s*$`)
	wdivPattern        = regexp.MustCompile(`^WDiv(\d+)$`)
	styleSizePattern   = regexp.MustCompile(`(?i)\b(width|height)\s*:\s*(\d+)px`)
)

// isBracketKind は class が TN_TMPL のセル種別か(clone-spec C 2.3 の CSS)。
func isBracketKind(class string) bool {
	switch class {
	case "team", "win", "lose", "game", "txt_yen", "txt_sharp", "txt_and", "txt_left", "txt_right", "rank":
		return true
	}
	return false
}

// ParseGroup はグループ成績を読む。種別は clone-spec C 2 の判別法による:
// div.TN_TMPL があればトーナメント、"- MATRIX -" があれば総当たり、画像だけなら image。
func ParseGroup(r io.Reader, lang site.Lang) (*Group, error) {
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
	out := &Group{Matches: []MatchRow{}}
	heading := headingText(sv.Find("span.SCV_MIDA_M").First())
	div, name, ok := strings.Cut(heading, " - ")
	if !ok {
		return nil, unexpected("group heading %q has no ' - '", heading)
	}
	out.DivisionName = textOf(lang, clean(div))
	out.Name = clean(name)

	matrix := sv.Find("td").FilterFunction(func(_ int, td *goquery.Selection) bool {
		return cellText(td) == "- MATRIX -"
	})
	switch {
	case sv.Find("div.TN_TMPL").Length() > 0:
		out.Kind = "bracket"
		if out.Bracket, err = parseBracket(sv, lang); err != nil {
			return nil, err
		}
	case matrix.Length() > 0:
		out.Kind = "pool"
		if out.Pool, err = parsePool(sv, matrix.First().Closest("table"), lang); err != nil {
			return nil, err
		}
	case sv.Find("div.TBL img").Length() > 0:
		out.Kind = "image"
		if out.Image, err = parseGroupImage(sv.Find("div.TBL img").First()); err != nil {
			return nil, err
		}
	default:
		return nil, unexpected("unknown group kind")
	}

	// 結果一覧は CD 列から始まる表
	var rowsErr error
	sv.Find("div.TBL > table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		h := headerLabels(table)
		if len(h) == 0 || h[0] != "CD" {
			return true
		}
		out.Matches, rowsErr = parseMatchRows(table, listLayout(), lang)
		return false
	})
	if rowsErr != nil {
		return nil, rowsErr
	}
	return out, nil
}

func parsePool(sv, matrix *goquery.Selection, lang site.Lang) (*Pool, error) {
	pool := &Pool{Teams: []PoolTeam{}, Cells: []PoolCell{}}
	var err error
	sv.Find("div.TBL > table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		h := headerLabels(table)
		if len(h) == 0 || h[0] != "- RESULT -" {
			return true
		}
		pool.Teams, err = parsePoolResult(table, lang)
		return false
	})
	if err != nil {
		return nil, err
	}
	rowIndex := 0
	matrix.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true // 見出し行(列のチーム)
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != len(pool.Teams)+1 {
			err = unexpected("matrix row has %d cells for %d teams", tds.Length(), len(pool.Teams))
			return false
		}
		tds.Each(func(col int, td *goquery.Selection) {
			if col == 0 || (!td.HasClass("RM_ARI") && !td.HasClass("RM_NOW")) {
				return
			}
			pool.Cells = append(pool.Cells, PoolCell{
				Row: rowIndex, Col: col - 1,
				MatchID: linkID(td, site.PageMatch),
				Score:   cellText(td),
				// 推測: 試合中のセルは調査時に観測できなかった(clone-spec C 2.1)
				Live: td.HasClass("RM_NOW"),
			})
		})
		rowIndex++
		return true
	})
	if err != nil {
		return nil, err
	}
	if rowIndex != len(pool.Teams) {
		return nil, unexpected("matrix has %d rows, result has %d teams", rowIndex, len(pool.Teams))
	}
	return pool, nil
}

// parsePoolResult はリーグ結果の表を読む。勝ち点列は HTML コメントなので数えない(clone-spec C 2.1(b))。
func parsePoolResult(table *goquery.Selection, lang site.Lang) ([]PoolTeam, error) {
	out := []PoolTeam{}
	var err error
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != 9 {
			err = unexpected("pool result row has %d cells", tds.Length())
			return false
		}
		team := PoolTeam{
			TeamID:    linkID(tds.Eq(0), site.PageTeam),
			ShortName: textOf(lang, cellText(tds.Eq(0))),
			RankRaw:   cellText(tds.Eq(8)),
		}
		nums := []*int{&team.Games, &team.Wins, &team.Draws, &team.Losses,
			&team.GoalsFor, &team.GoalsAgainst, &team.GoalDiff}
		for k, dst := range nums {
			if *dst, err = atoi(cellText(tds.Eq(k + 1))); err != nil {
				return false
			}
		}
		if team.Rank, err = atoiPtr(team.RankRaw); err != nil {
			return false
		}
		out = append(out, team)
		return true
	})
	return out, err
}

func parseBracket(sv *goquery.Selection, lang site.Lang) (*Bracket, error) {
	b := &Bracket{Rows: [][]BracketCell{}}
	for _, c := range comments(sv) {
		if m := generatePtnPattern.FindStringSubmatch(c); m != nil {
			b.Pattern = m[1]
		}
		if m := ptnNamePattern.FindStringSubmatch(c); m != nil {
			b.PtnName = m[1]
		}
	}
	table := sv.Find("div.TN_TMPL > table").First()
	if table.Length() == 0 {
		return nil, unexpected("bracket table not found")
	}
	var err error
	table.Find("tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		row := []BracketCell{}
		tr.ChildrenFiltered("td").EachWithBreak(func(_ int, td *goquery.Selection) bool {
			var cell BracketCell
			cell, err = parseBracketCell(td, b, lang)
			row = append(row, cell)
			return err == nil
		})
		if err != nil {
			return false
		}
		if len(row) != b.Columns {
			err = unexpected("bracket row has %d cells, want %d", len(row), b.Columns)
			return false
		}
		b.Rows = append(b.Rows, row)
		return true
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

func parseBracketCell(td *goquery.Selection, b *Bracket, lang site.Lang) (BracketCell, error) {
	var cell BracketCell
	for _, class := range strings.Fields(td.AttrOr("class", "")) {
		switch {
		case wdivPattern.MatchString(class):
			n, err := atoi(wdivPattern.FindStringSubmatch(class)[1])
			if err != nil {
				return cell, err
			}
			if b.Columns == 0 {
				b.Columns = n
			} else if b.Columns != n {
				return cell, unexpected("bracket mixes WDiv%d and WDiv%d", b.Columns, n)
			}
		case class == "BdTop":
			cell.Top = true
		case class == "BdRight":
			cell.Right = true
		case class == "BdBtm":
			cell.Bottom = true
		case class == "BdLeft":
			cell.Left = true
		case isBracketKind(class):
			if cell.Kind != "" {
				return cell, unexpected("bracket cell has two kinds %s and %s", cell.Kind, class)
			}
			cell.Kind = class
		default:
			return cell, unexpected("unknown bracket cell class %q", class)
		}
	}
	cell.Text = textOf(lang, cellText(td))
	cell.MatchID = linkID(td, site.PageMatch)
	cell.TeamID = linkID(td, site.PageTeam)
	return cell, nil
}

func parseGroupImage(img *goquery.Selection) (*GroupImage, error) {
	src, ok := img.Attr("src")
	if !ok {
		return nil, unexpected("group image without src")
	}
	abs, err := site.AbsURL(src)
	if err != nil {
		return nil, unexpected("group image url: %v", err)
	}
	out := &GroupImage{URL: abs}
	for _, m := range styleSizePattern.FindAllStringSubmatch(img.AttrOr("style", ""), -1) {
		n, err := atoi(m[2])
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(m[1], "width") {
			out.Width = &n
		} else {
			out.Height = &n
		}
	}
	return out, nil
}

// MergeGroupEN は日本語版のグループに英語のディビジョン名、チームの短縮名を入れる。
func MergeGroupEN(jp, en *Group) (*Group, error) {
	if jp.Kind != en.Kind || jp.Name != en.Name {
		return nil, mismatch("group jp=%s/%s en=%s/%s", jp.Kind, jp.Name, en.Kind, en.Name)
	}
	out := *jp
	out.DivisionName = mergeText(jp.DivisionName, en.DivisionName)
	matches, err := mergeMatchRowsEN(jp.Matches, en.Matches)
	if err != nil {
		return nil, err
	}
	out.Matches = matches
	switch jp.Kind {
	case "pool":
		if out.Pool, err = mergePoolEN(jp.Pool, en.Pool); err != nil {
			return nil, err
		}
	case "bracket":
		if out.Bracket, err = mergeBracketEN(jp.Bracket, en.Bracket); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func mergePoolEN(jp, en *Pool) (*Pool, error) {
	if len(jp.Teams) != len(en.Teams) {
		return nil, mismatch("pool teams jp=%d en=%d", len(jp.Teams), len(en.Teams))
	}
	out := &Pool{Teams: make([]PoolTeam, len(jp.Teams)), Cells: jp.Cells}
	for i, t := range jp.Teams {
		if !equalIDPtr(t.TeamID, en.Teams[i].TeamID) {
			return nil, mismatch("pool team %d differs", i)
		}
		t.ShortName = mergeText(t.ShortName, en.Teams[i].ShortName)
		out.Teams[i] = t
	}
	return out, nil
}

func mergeBracketEN(jp, en *Bracket) (*Bracket, error) {
	if len(jp.Rows) != len(en.Rows) || jp.Columns != en.Columns {
		return nil, mismatch("bracket size differs")
	}
	out := *jp
	out.Rows = make([][]BracketCell, len(jp.Rows))
	for i, row := range jp.Rows {
		merged := make([]BracketCell, len(row))
		for k, c := range row {
			e := en.Rows[i][k]
			if c.Kind != e.Kind || !equalIDPtr(c.TeamID, e.TeamID) || !equalIDPtr(c.MatchID, e.MatchID) {
				return nil, mismatch("bracket cell %d,%d differs", i, k)
			}
			c.Text = mergeText(c.Text, e.Text)
			merged[k] = c
		}
		out.Rows[i] = merged
	}
	return &out, nil
}
