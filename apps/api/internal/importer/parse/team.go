package parse

import (
	"io"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// Team はチームページ scv_team/open/{チームID}。ページにチーム自身の ID はないので、呼び出し側が
// URL の ID と対応させる。
type Team struct {
	// Name は見出しの正式名(短縮名ではない。clone-spec D 1.1)。
	Name Text `json:"name"`
	// Comment はチーム紹介。ないときは空。
	Comment Text `json:"comment"`
	// Record は成績表(試合数〜順位)。ディスクゴルフ団体のチームページにはない(clone-spec D 1.2)。
	Record *TeamRecord `json:"record"`
	// Matches はチーム対戦一覧。試合 ID との照合に使う。
	Matches []MatchRow   `json:"matches"`
	Players []TeamPlayer `json:"players"`
	Staff   []TeamStaff  `json:"staff"`
}

// TeamRecord はチームの通算成績。順位はディビジョン内の順位で、未確定なら nil。
type TeamRecord struct {
	Games        int    `json:"games"`
	Wins         int    `json:"wins"`
	Draws        int    `json:"draws"`
	Losses       int    `json:"losses"`
	GoalsFor     int    `json:"goals_for"`
	GoalsAgainst int    `json:"goals_against"`
	GoalDiff     int    `json:"goal_diff"`
	WinPct       string `json:"win_pct"`
	Rank         *int   `json:"rank"`
}

// TeamPlayer は選手一覧の 1 行。並びは成績順で、登録順ではない(clone-spec D 1.1)。
// ディスクゴルフ団体の選手一覧は # と選手だけなので、成績は 0 のまま。
type TeamPlayer struct {
	ID int64 `json:"id"`
	// Number は背番号。"00" のような表記があるので文字列。
	Number  string  `json:"number"`
	Name    Text    `json:"name"`
	Gender  *string `json:"gender"`
	Games   int     `json:"games"`
	Assists int     `json:"assists"`
	Goals   int     `json:"goals"`
	Total   int     `json:"total"`
	Blocks  int     `json:"blocks"`
}

// TeamStaff はスタッフ一覧の 1 行。スタッフも選手と同じ ID 空間(clone-spec D 1.1)。
type TeamStaff struct {
	ID     int64   `json:"id"`
	Role   Text    `json:"role"`
	Name   Text    `json:"name"`
	Gender *string `json:"gender"`
}

// ParseTeam はチームページを読む。表の種類は見出し行の文言で見分ける。
func ParseTeam(r io.Reader, lang site.Lang) (*Team, error) {
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
	out := &Team{
		Name:    textOf(lang, cellText(sv.Find("span.SCV_MIDA_M").First())),
		Comment: textOf(lang, cellText(sv.Find("div.SCV_TEAM_COMM").First())),
		Matches: []MatchRow{},
		Players: []TeamPlayer{},
		Staff:   []TeamStaff{},
	}
	var tblErr error
	sv.Find("div.TBL > table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		h := headerLabels(table)
		if len(h) == 0 {
			return true
		}
		switch {
		case h[0] == "試合数" || h[0] == "Games":
			out.Record, tblErr = parseTeamRecord(table)
		case h[0] == "CD" && len(h) == listLayout().cells:
			// 旧大会の SOTG スコア一覧も CD 列から始まるが列数が違う(clone-spec D 1.1 の 6)ので読まない
			out.Matches, tblErr = parseMatchRows(table, listLayout(), lang)
		case h[0] == "#":
			out.Players, tblErr = parseTeamPlayers(table, len(h), lang)
		case h[0] == "役割" || h[0] == "Task":
			out.Staff, tblErr = parseTeamStaff(table, lang)
		}
		return tblErr == nil
	})
	if tblErr != nil {
		return nil, tblErr
	}
	return out, nil
}

func parseTeamRecord(table *goquery.Selection) (*TeamRecord, error) {
	tds := table.Find("tr").Eq(1).ChildrenFiltered("td")
	if tds.Length() != 9 {
		return nil, unexpected("team record has %d cells", tds.Length())
	}
	rec := &TeamRecord{WinPct: cellText(tds.Eq(7))}
	nums := []*int{&rec.Games, &rec.Wins, &rec.Draws, &rec.Losses, &rec.GoalsFor, &rec.GoalsAgainst, &rec.GoalDiff}
	var err error
	for k, dst := range nums {
		if *dst, err = atoi(cellText(tds.Eq(k))); err != nil {
			return nil, err
		}
	}
	if rec.Rank, err = atoiPtr(cellText(tds.Eq(8))); err != nil {
		return nil, err
	}
	return rec, nil
}

// parseTeamPlayers は選手一覧を読む。アルティメットは 8 列(# 選手 試合数 アシスト ゴール トータル
// ブロック --)、ディスクゴルフ団体は 2 列(# 選手)。
func parseTeamPlayers(table *goquery.Selection, cols int, lang site.Lang) ([]TeamPlayer, error) {
	if cols != 8 && cols != 2 {
		return nil, unexpected("player list has %d columns", cols)
	}
	out := []TeamPlayer{}
	var err error
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != cols {
			err = unexpected("player row has %d cells, want %d", tds.Length(), cols)
			return false
		}
		id := linkID(tds.Eq(1), site.PagePlayer)
		if id == nil {
			err = unexpected("player row without player link")
			return false
		}
		p := TeamPlayer{
			ID:     *id,
			Number: cellText(tds.Eq(0)),
			Name:   textOf(lang, cellText(tds.Eq(1))),
			Gender: genderOf(tds.Eq(1).Find("a").First()),
		}
		if cols == 8 {
			for k, dst := range []*int{&p.Games, &p.Assists, &p.Goals, &p.Total} {
				if *dst, err = atoi(cellText(tds.Eq(k + 2))); err != nil {
					return false
				}
			}
			// clone-spec D 1.1: ブロック 0 は空セル
			if p.Blocks, err = atoiBlank(cellText(tds.Eq(6))); err != nil {
				return false
			}
		}
		out = append(out, p)
		return true
	})
	return out, err
}

func parseTeamStaff(table *goquery.Selection, lang site.Lang) ([]TeamStaff, error) {
	out := []TeamStaff{}
	var err error
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != 2 {
			err = unexpected("staff row has %d cells", tds.Length())
			return false
		}
		id := linkID(tds.Eq(1), site.PagePlayer)
		if id == nil {
			err = unexpected("staff row without link")
			return false
		}
		out = append(out, TeamStaff{
			ID:     *id,
			Role:   textOf(lang, cellText(tds.Eq(0))),
			Name:   textOf(lang, cellText(tds.Eq(1))),
			Gender: genderOf(tds.Eq(1).Find("a").First()),
		})
		return true
	})
	return out, err
}

// MergeTeamEN は日本語版のチームページに英語の正式名、紹介、選手名、スタッフの役割と名前を入れる。
// 選手は ID で突き合わせる(並びは成績順なので両版で同じだが、ID のほうが確実)。
func MergeTeamEN(jp, en *Team) (*Team, error) {
	out := *jp
	out.Name = mergeText(jp.Name, en.Name)
	out.Comment = mergeText(jp.Comment, en.Comment)
	matches, err := mergeMatchRowsEN(jp.Matches, en.Matches)
	if err != nil {
		return nil, err
	}
	out.Matches = matches

	enPlayers := map[int64]TeamPlayer{}
	for _, p := range en.Players {
		enPlayers[p.ID] = p
	}
	if len(enPlayers) != len(jp.Players) {
		return nil, mismatch("players jp=%d en=%d", len(jp.Players), len(en.Players))
	}
	out.Players = make([]TeamPlayer, len(jp.Players))
	for i, p := range jp.Players {
		e, ok := enPlayers[p.ID]
		if !ok {
			return nil, mismatch("player %d is missing in en", p.ID)
		}
		p.Name = mergeText(p.Name, e.Name)
		out.Players[i] = p
	}

	enStaff := map[int64]TeamStaff{}
	for _, s := range en.Staff {
		enStaff[s.ID] = s
	}
	if len(enStaff) != len(jp.Staff) {
		return nil, mismatch("staff jp=%d en=%d", len(jp.Staff), len(en.Staff))
	}
	out.Staff = make([]TeamStaff, len(jp.Staff))
	for i, s := range jp.Staff {
		e, ok := enStaff[s.ID]
		if !ok {
			return nil, mismatch("staff %d is missing in en", s.ID)
		}
		s.Role = mergeText(s.Role, e.Role)
		s.Name = mergeText(s.Name, e.Name)
		out.Staff[i] = s
	}
	return &out, nil
}
