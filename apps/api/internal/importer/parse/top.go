package parse

import (
	"io"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// defaultLogoPath はロゴ未登録の大会に出る既定ロゴ(clone-spec A 4)。取込では「ロゴなし」として扱う。
const defaultLogoPath = "/img/ClubJr/logo_scv_default.png"

// TournamentTop は大会トップ /scview/{id}。ナビ(ul#score_navi)とサイドバーの内容も持つ。
type TournamentTop struct {
	ID int64 `json:"id"`
	// Name.JA は本文の div.text_title(lang によらず日本語名)、Name.EN は英語版の <title>
	// (英語名が未登録なら空。clone-spec A 1.7)。
	Name      Text          `json:"name"`
	LogoURL   *string       `json:"logo_url"`
	Divisions []TopDivision `json:"divisions"`
	// TopHTML は #SCV_TopPage の中身。言語別の 2 フィールド(clone-spec A 4)。無害化はしていない。
	TopHTML Text     `json:"top_html"`
	Nav     Nav      `json:"nav"`
	Banners []Banner `json:"banners"`
}

// TopDivision は大会トップのディビジョン表の 1 行。並びは参照サイトの DB 順(ID 順ではない)。
type TopDivision struct {
	// ID は [スケジュール]/[最終順位] のリンク、なければナビから取る。両方にない
	// (ディスクゴルフなど、ナビにスケジュールがなくリンクがダミー)ときは nil。
	ID              *int64 `json:"id"`
	Name            Text   `json:"name"`
	ScheduleLinked  bool   `json:"schedule_linked"`
	StandingsLinked bool   `json:"standings_linked"`
	// Groups はナビの「グループ成績」でこのディビジョンの下に並ぶグループ(ナビの順)。
	Groups []NavGroup `json:"groups"`
}

// NavGroup はナビのグループ。リンクのないグループ(名前だけの <p>)は ID が nil で Linked=false。
type NavGroup struct {
	ID     *int64 `json:"id"`
	Name   string `json:"name"`
	NameEN string `json:"name_en"` // 英語版のナビの名前(Merge で入る)。空のことがある
	Linked bool   `json:"linked"`
}

// Nav は大会ナビから分かる大会の設定。clone-spec E 3 の 7 パターンを表す。
type Nav struct {
	// TeamListLabel は "teams"(チーム一覧)か "entries"(エントリー一覧)。
	TeamListLabel string `json:"team_list_label"`
	// GroupLabel は "groups"(グループ成績)か "results"(成績)。
	GroupLabel string `json:"group_label"`
	// ShowSchedule はスケジュールとディビジョン成績の項目があるか(常に対で出る。clone-spec E 3)。
	ShowSchedule bool         `json:"show_schedule"`
	Galleries    []NavGallery `json:"galleries"`
}

// NavGallery はナビのギャラリー。ギャラリー ID は大会 ID と別体系(clone-spec A 6)。
type NavGallery struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	NameEN string `json:"name_en"` // 英語版のナビの名前(Merge で入る)。"---" が多い
}

// Banner はサイドバーのバナー画像。並びは表示順。
type Banner struct {
	ImageURL string  `json:"image_url"`
	LinkURL  *string `json:"link_url"`
}

// ParseTournamentTop は大会トップを読む。
func ParseTournamentTop(r io.Reader, lang site.Lang) (*TournamentTop, error) {
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
	out := &TournamentTop{Divisions: []TopDivision{}}

	nav, err := parseNav(doc)
	if err != nil {
		return nil, err
	}
	out.ID = nav.tournamentID
	out.Nav = nav.Nav

	if lang == site.LangJP {
		out.Name = Text{JA: cellText(sv.Find(".text_title").First())}
	} else {
		out.Name = Text{EN: titleName(doc)}
	}
	if out.LogoURL, err = parseLogo(sv); err != nil {
		return nil, err
	}
	if out.Divisions, err = parseTopDivisions(sv, lang); err != nil {
		return nil, err
	}
	if err := attachNav(out.Divisions, nav); err != nil {
		return nil, err
	}
	topHTML, err := sv.Find("#SCV_TopPage").Html()
	if err != nil {
		return nil, unexpected("render #SCV_TopPage: %v", err)
	}
	out.TopHTML = textOf(lang, strings.TrimSpace(topHTML))
	if out.Banners, err = parseBanners(doc); err != nil {
		return nil, err
	}
	return out, nil
}

func parseLogo(sv *goquery.Selection) (*string, error) {
	img := sv.Find(`img[alt="ロゴ"]`).First()
	src, ok := img.Attr("src")
	if !ok {
		return nil, unexpected("logo img not found")
	}
	if strings.HasSuffix(strings.TrimSpace(src), defaultLogoPath) {
		return nil, nil
	}
	abs, err := site.AbsURL(src)
	if err != nil {
		return nil, unexpected("logo url: %v", err)
	}
	return &abs, nil
}

// parseTopDivisions はディビジョン表を読む。clone-spec A 4: リンクは a.Select_Link、
// ないときは span.Dummy_Link。
func parseTopDivisions(sv *goquery.Selection, lang site.Lang) ([]TopDivision, error) {
	table := sv.Find("div.TBL table").First()
	if table.Length() == 0 {
		return nil, unexpected("division table not found")
	}
	out := []TopDivision{}
	var rowErr error
	table.Find("tr").Each(func(i int, tr *goquery.Selection) {
		if i == 0 || rowErr != nil {
			return // 見出し行
		}
		tds := tr.ChildrenFiltered("td")
		if tds.Length() != 2 {
			rowErr = unexpected("division row has %d cells", tds.Length())
			return
		}
		info := tds.Eq(1)
		d := TopDivision{Name: textOf(lang, cellText(tds.Eq(0))), Groups: []NavGroup{}}
		if id := linkID(info, site.PageScheduleDiv); id != nil {
			d.ID, d.ScheduleLinked = id, true
		}
		if id := linkID(info, site.PageDivision); id != nil {
			if d.ID != nil && *d.ID != *id {
				rowErr = unexpected("division row links to %d and %d", *d.ID, *id)
				return
			}
			d.ID, d.StandingsLinked = id, true
		}
		out = append(out, d)
	})
	return out, rowErr
}

type parsedNav struct {
	Nav
	tournamentID int64
	// navDivisions はナビのスケジュール/ディビジョン成績に並ぶディビジョン(ID と名前)。
	navDivisions   []navDivision
	groupDivisions []navGroupDivision
}

type navDivision struct {
	id   int64
	name string
}

type navGroupDivision struct {
	name   string
	groups []NavGroup
}

// parseNav は ul#score_navi を読む。項目の種類はラベルではなくリンク先で見分け、
// ラベルの違い(チーム一覧/エントリー一覧、グループ成績/成績)だけを文言で判定する。
func parseNav(doc *goquery.Document) (parsedNav, error) {
	out := parsedNav{Nav: Nav{TeamListLabel: "teams", GroupLabel: "groups", Galleries: []NavGallery{}}}
	ul := doc.Find("ul#score_navi")
	if ul.Length() != 1 {
		return out, unexpected("ul#score_navi count = %d", ul.Length())
	}
	foundTop, foundGroups := false, false
	var navErr error
	ul.ChildrenFiltered("li").EachWithBreak(func(_ int, li *goquery.Selection) bool {
		label := strings.TrimSpace(li.ChildrenFiltered("a, p").First().Text())
		sub := li.ChildrenFiltered("ul")
		switch {
		case linkID(li.ChildrenFiltered("a"), site.PageTournamentTop) != nil:
			out.tournamentID = *linkID(li.ChildrenFiltered("a"), site.PageTournamentTop)
			foundTop = true
		case linkID(li.ChildrenFiltered("a"), site.PageTeamList) != nil:
			if label == "エントリー一覧" || label == "Entry List" {
				out.TeamListLabel = "entries"
			}
		case sub.Length() == 0:
			// 幅調整の空 li(clone-spec A 1.3)
		case sub.Find("a").Length() > 0 && linkID(sub, site.PageScheduleAll) != nil:
			out.ShowSchedule = true
			sub.Find("a").Each(func(_ int, a *goquery.Selection) {
				if id := linkID(a, site.PageScheduleDiv); id != nil {
					out.navDivisions = append(out.navDivisions, navDivision{*id, cellText(a)})
				}
			})
		case linkID(sub, site.PageDivision) != nil:
			sub.Find("a").Each(func(_ int, a *goquery.Selection) {
				if id := linkID(a, site.PageDivision); id != nil {
					out.navDivisions = append(out.navDivisions, navDivision{*id, cellText(a)})
				}
			})
		case linkID(sub, site.PageGallery) != nil:
			sub.Find("a").Each(func(_ int, a *goquery.Selection) {
				if id := linkID(a, site.PageGallery); id != nil {
					out.Galleries = append(out.Galleries, NavGallery{ID: *id, Name: navName(a)})
				}
			})
		default:
			// グループ成績(または成績): li > p(ディビジョン名) + ul > li > (a | p)
			foundGroups = true
			if label == "成績" || label == "Result" {
				out.GroupLabel = "results"
			}
			out.groupDivisions, navErr = parseNavGroups(sub)
			return navErr == nil
		}
		return true
	})
	if navErr != nil {
		return out, navErr
	}
	if !foundTop {
		return out, unexpected("nav has no tournament top link")
	}
	if !foundGroups {
		return out, unexpected("nav has no group menu")
	}
	return out, nil
}

// navName はナビのグループとギャラリーの名前。名前の前後の半角空白は名前の一部として残す
// (参照サイトのグループ "17th placement tournament " は、ページの title にも空白つきで出る)。
// 名前が空のリンクは、参照サイト(CakePHP の link())がリンク先の URL を文字として出すので空にする
// (英語名のないグループの英語版で観測)。
func navName(sel *goquery.Selection) string {
	name := strings.Trim(sel.Text(), "\r\n\t")
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		return ""
	}
	return name
}

func parseNavGroups(sub *goquery.Selection) ([]navGroupDivision, error) {
	out := []navGroupDivision{}
	var err error
	sub.ChildrenFiltered("li").Each(func(_ int, li *goquery.Selection) {
		p := li.ChildrenFiltered("p")
		if p.Length() != 1 {
			err = unexpected("group menu division without name")
			return
		}
		d := navGroupDivision{name: cellText(p), groups: []NavGroup{}}
		li.ChildrenFiltered("ul").ChildrenFiltered("li").Each(func(_ int, g *goquery.Selection) {
			item := g.ChildrenFiltered("a, p").First()
			ng := NavGroup{Name: navName(item)}
			if id := linkID(item, site.PageGroup); id != nil {
				ng.ID, ng.Linked = id, true
			}
			d.groups = append(d.groups, ng)
		})
		out = append(out, d)
	})
	return out, err
}

// attachNav はナビのグループとディビジョン ID を、大会トップの表のディビジョンに並び順で対応させる。
// clone-spec A 1.3: ナビのディビジョンの並びは大会トップの表と同じ DB 順。
func attachNav(divs []TopDivision, nav parsedNav) error {
	if len(nav.groupDivisions) != len(divs) {
		return unexpected("nav group divisions = %d, top divisions = %d",
			len(nav.groupDivisions), len(divs))
	}
	for i := range divs {
		divs[i].Groups = nav.groupDivisions[i].groups
	}
	// スケジュールのないディビジョンはナビのスケジュールに出ない(237 で確認)ので、並び順ではなく
	// 名前で対応させる。同じ名前のディビジョンが複数ある大会では ID を決めない。
	for i := range divs {
		var found []int64
		for _, nd := range nav.navDivisions {
			if nd.name == divs[i].Name.JA || nd.name == divs[i].Name.EN {
				found = appendUnique(found, nd.id)
			}
		}
		sameAsTop := divs[i].ID != nil && len(found) == 1 && found[0] == *divs[i].ID
		switch {
		case divs[i].ID != nil && len(found) > 0 && !sameAsTop:
			return unexpected("division %q: top id %d, nav ids %v", divs[i].Name, *divs[i].ID, found)
		case divs[i].ID == nil && len(found) == 1:
			divs[i].ID = ptr(found[0])
		}
	}
	return nil
}

// parseBanners はサイドバー #sub のバナー画像を読む。枠の画像(09_menu_outline_*)は除く。
func parseBanners(doc *goquery.Document) ([]Banner, error) {
	out := []Banner{}
	var err error
	doc.Find("#sub div img").Each(func(_ int, img *goquery.Selection) {
		if err != nil {
			return
		}
		src, ok := img.Attr("src")
		if !ok || strings.TrimSpace(src) == "" {
			return
		}
		abs, aerr := site.AbsURL(src)
		if aerr != nil {
			err = unexpected("banner url: %v", aerr)
			return
		}
		b := Banner{ImageURL: abs}
		if href, ok := img.ParentsFiltered("a").First().Attr("href"); ok && strings.TrimSpace(href) != "" {
			link, lerr := site.AbsURL(href)
			if lerr != nil {
				err = unexpected("banner link: %v", lerr)
				return
			}
			b.LinkURL = &link
		}
		out = append(out, b)
	})
	return out, err
}

// MergeTournamentTopEN は日本語版の大会トップに英語版の大会名、ディビジョン名、自由記述を入れる。
func MergeTournamentTopEN(jp, en *TournamentTop) (*TournamentTop, error) {
	if jp.ID != en.ID {
		return nil, mismatch("tournament id jp=%d en=%d", jp.ID, en.ID)
	}
	if len(jp.Divisions) != len(en.Divisions) {
		return nil, mismatch("divisions jp=%d en=%d", len(jp.Divisions), len(en.Divisions))
	}
	if len(jp.Nav.Galleries) != len(en.Nav.Galleries) {
		return nil, mismatch("galleries jp=%d en=%d", len(jp.Nav.Galleries), len(en.Nav.Galleries))
	}
	out := *jp
	out.Name = mergeText(jp.Name, en.Name)
	out.TopHTML = mergeText(jp.TopHTML, en.TopHTML)
	out.Divisions = make([]TopDivision, len(jp.Divisions))
	for i, d := range jp.Divisions {
		e := en.Divisions[i]
		if !equalIDPtr(d.ID, e.ID) || len(d.Groups) != len(e.Groups) {
			return nil, mismatch("division %d differs between jp and en", i)
		}
		d.Name = mergeText(d.Name, e.Name)
		d.Groups = slices.Clone(d.Groups)
		for k := range d.Groups {
			if !equalIDPtr(d.Groups[k].ID, e.Groups[k].ID) {
				return nil, mismatch("division %d group %d differs between jp and en", i, k)
			}
			d.Groups[k].NameEN = e.Groups[k].Name
		}
		out.Divisions[i] = d
	}
	out.Nav.Galleries = slices.Clone(jp.Nav.Galleries)
	for k := range out.Nav.Galleries {
		if out.Nav.Galleries[k].ID != en.Nav.Galleries[k].ID {
			return nil, mismatch("gallery %d differs between jp and en", k)
		}
		out.Nav.Galleries[k].NameEN = en.Nav.Galleries[k].Name
	}
	return &out, nil
}

func appendUnique(ids []int64, id int64) []int64 {
	for _, v := range ids {
		if v == id {
			return ids
		}
	}
	return append(ids, id)
}

func equalIDPtr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
