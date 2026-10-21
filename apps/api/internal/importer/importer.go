// Package importer は参照サイト score.clubjr.com から 1 大会分のページを取得してパースし、
// 中間表現 Bundle にまとめる。DB への書き込みは Sink の実装(store を使うもの)が担う。
package importer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"scoreview/api/internal/importer/fetch"
	"scoreview/api/internal/importer/parse"
	"scoreview/api/internal/importer/site"
)

// Bundle は 1 大会分の中間表現。ページごとのパース結果を ID で引ける形で持ち、
// migrations/00001_init.sql の列を埋めるのに要る情報をすべて含む。日英は Merge 済み。
type Bundle struct {
	TournamentID int64 `json:"tournament_id"`
	// ListEntry は大会一覧での区分と並び。一覧に載らない大会(538、507 など。clone-spec A 3)は nil。
	ListEntry *parse.ListEntry     `json:"list_entry"`
	Top       *parse.TournamentTop `json:"top"`
	TeamList  *parse.TeamList      `json:"team_list"`
	// Schedules はディビジョン ID ごとの div 版スケジュール(全日程)。
	Schedules map[int64]*parse.Schedule `json:"schedules"`
	// Standings はディビジョン ID ごとの成績。[最終順位] がリンクのディビジョンだけ取る。
	Standings map[int64]*parse.DivisionStandings `json:"standings"`
	Groups    map[int64]*parse.Group             `json:"groups"`
	Teams     map[int64]*parse.Team              `json:"teams"`
	Matches   map[int64]*parse.Match             `json:"matches"`
	Galleries map[int64]*parse.Gallery           `json:"galleries"`
	// Players は WithPlayers(true) のときだけ取る選手ページ(キャラハンとグレイテスト用)。
	Players map[int64]*parse.Player `json:"players"`
	// Missing は参照サイトが存在しないと返したページの URL(英語版だけがない場合を含む)。
	Missing []string `json:"missing"`
}

// Sink は Bundle を受け取って書き込む先。DB への書き込みは store を使う実装を後で差し込む。
type Sink interface {
	Write(ctx context.Context, b *Bundle) error
}

// Importer は 1 大会分のページを取得してパースする。
type Importer struct {
	fetcher     fetch.Fetcher
	withPlayers bool
	progress    io.Writer
}

// Option は Importer の設定を変える。
type Option func(*Importer)

// WithFetcher は取得器を設定する。必須。テストではファイルから返すフェイクを渡す。
func WithFetcher(f fetch.Fetcher) Option { return func(im *Importer) { im.fetcher = f } }

// WithPlayers は選手ページも取得するかを設定する。既定は取得しない(1 大会で数百ページ増えるため)。
func WithPlayers(on bool) Option { return func(im *Importer) { im.withPlayers = on } }

// WithProgress は取得したページの URL を 1 行ずつ書く先を設定する。既定は書かない。
func WithProgress(w io.Writer) Option { return func(im *Importer) { im.progress = w } }

// New は Importer を作る。取得器がなければエラーにする。
func New(opts ...Option) (*Importer, error) {
	im := &Importer{progress: io.Discard}
	for _, opt := range opts {
		opt(im)
	}
	if im.fetcher == nil {
		return nil, fmt.Errorf("importer: fetcher is required (use WithFetcher)")
	}
	return im, nil
}

// Collect は大会 tournamentID のページを取得してパースし、Bundle を返す。大会トップが存在しなければ
// fetch.ErrNotFound を包んだエラーを返す。下位のページが存在しないときは Missing に記録して続ける。
// パースの失敗は握りつぶさずにエラーで返す。
func (im *Importer) Collect(ctx context.Context, tournamentID int64) (*Bundle, error) {
	b := &Bundle{
		TournamentID: tournamentID,
		Schedules:    map[int64]*parse.Schedule{},
		Standings:    map[int64]*parse.DivisionStandings{},
		Groups:       map[int64]*parse.Group{},
		Teams:        map[int64]*parse.Team{},
		Matches:      map[int64]*parse.Match{},
		Galleries:    map[int64]*parse.Gallery{},
		Players:      map[int64]*parse.Player{},
		Missing:      []string{},
	}
	top, err := getBoth(ctx, im, b, site.PageTournamentTop, tournamentID,
		parse.ParseTournamentTop, parse.MergeTournamentTopEN)
	if err != nil {
		return nil, err
	}
	if top == nil {
		return nil, fmt.Errorf("tournament %d: %w", tournamentID, fetch.ErrNotFound)
	}
	b.Top = top

	list, err := getBoth(ctx, im, b, site.PageTournamentList, 0,
		parse.ParseTournamentList, parse.MergeTournamentListEN)
	if err != nil {
		return nil, err
	}
	if list != nil {
		for i := range list.Entries {
			if list.Entries[i].ID == tournamentID {
				b.ListEntry = &list.Entries[i]
			}
		}
	}

	if b.TeamList, err = getBoth(ctx, im, b, site.PageTeamList, tournamentID,
		parse.ParseTeamList, parse.MergeTeamListEN); err != nil {
		return nil, err
	}
	if err := im.collectDivisions(ctx, b); err != nil {
		return nil, err
	}
	if err := im.collectGroups(ctx, b); err != nil {
		return nil, err
	}
	if err := im.collectTeams(ctx, b); err != nil {
		return nil, err
	}
	if err := im.collectMatches(ctx, b); err != nil {
		return nil, err
	}
	for _, g := range b.Top.Nav.Galleries {
		gallery, err := getJP(ctx, im, b, site.PageGallery, g.ID, parse.ParseGallery)
		if err != nil {
			return nil, err
		}
		if gallery != nil {
			b.Galleries[g.ID] = gallery
		}
	}
	if im.withPlayers {
		if err := im.collectPlayers(ctx, b); err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (im *Importer) collectDivisions(ctx context.Context, b *Bundle) error {
	for _, d := range b.Top.Divisions {
		if d.ID == nil {
			continue
		}
		if d.ScheduleLinked {
			s, err := getBoth(ctx, im, b, site.PageScheduleDiv, *d.ID, parse.ParseSchedule, parse.MergeScheduleEN)
			if err != nil {
				return err
			}
			if s != nil {
				b.Schedules[*d.ID] = s
			}
		}
		if d.StandingsLinked {
			st, err := getJP(ctx, im, b, site.PageDivision, *d.ID, parse.ParseDivisionStandings)
			if err != nil {
				return err
			}
			if st != nil {
				b.Standings[*d.ID] = st
			}
		}
	}
	return nil
}

func (im *Importer) collectGroups(ctx context.Context, b *Bundle) error {
	for _, d := range b.Top.Divisions {
		for _, g := range d.Groups {
			if g.ID == nil {
				continue
			}
			group, err := getJP(ctx, im, b, site.PageGroup, *g.ID, parse.ParseGroup)
			if err != nil {
				return err
			}
			if group != nil {
				b.Groups[*g.ID] = group
			}
		}
	}
	return nil
}

// collectTeams はチーム一覧、スケジュール、順位表、グループに出るチームのページを日英で取る。
func (im *Importer) collectTeams(ctx context.Context, b *Bundle) error {
	for _, id := range teamIDs(b) {
		team, err := getBoth(ctx, im, b, site.PageTeam, id, parse.ParseTeam, parse.MergeTeamEN)
		if err != nil {
			return err
		}
		if team != nil {
			b.Teams[id] = team
		}
	}
	return nil
}

// collectMatches は試合ページを日本語版だけ取る。英語版で変わるのは短縮名と選手名で、
// どちらもスケジュールとチームページの英語版から分かるため。
func (im *Importer) collectMatches(ctx context.Context, b *Bundle) error {
	for _, id := range matchIDs(b) {
		m, err := getJP(ctx, im, b, site.PageMatch, id, parse.ParseMatch)
		if err != nil {
			return err
		}
		if m != nil {
			b.Matches[id] = m
		}
	}
	return nil
}

func (im *Importer) collectPlayers(ctx context.Context, b *Bundle) error {
	ids := idSet{}
	for _, t := range b.Teams {
		for _, p := range t.Players {
			ids.add(p.ID)
		}
	}
	for _, id := range ids.sorted() {
		p, err := getJP(ctx, im, b, site.PagePlayer, id, parse.ParsePlayer)
		if err != nil {
			return err
		}
		if p != nil {
			b.Players[id] = p
		}
	}
	return nil
}

// fetchPage はページを取得する。存在しなければ Missing に記録して nil を返す。
func (im *Importer) fetchPage(ctx context.Context, b *Bundle, url string) ([]byte, error) {
	body, err := im.fetcher.Fetch(ctx, url)
	if errors.Is(err, fetch.ErrNotFound) {
		b.Missing = append(b.Missing, url)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintln(im.progress, url) // 進捗表示の書き込み失敗は取込の結果に影響しない
	return body, nil
}

func getJP[T any](
	ctx context.Context, im *Importer, b *Bundle, page site.Page, id int64,
	parseFn func(io.Reader, site.Lang) (*T, error),
) (*T, error) {
	url := site.URL(page, id, site.LangJP)
	body, err := im.fetchPage(ctx, b, url)
	if err != nil || body == nil {
		return nil, err
	}
	v, err := parseFn(bytes.NewReader(body), site.LangJP)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", url, err)
	}
	return v, nil
}

// getBoth は日英のページを取って Merge する。英語版だけがないときは日本語版だけを返す。
func getBoth[T any](
	ctx context.Context, im *Importer, b *Bundle, page site.Page, id int64,
	parseFn func(io.Reader, site.Lang) (*T, error),
	mergeFn func(jp, en *T) (*T, error),
) (*T, error) {
	jp, err := getJP(ctx, im, b, page, id, parseFn)
	if err != nil || jp == nil {
		return nil, err
	}
	url := site.URL(page, id, site.LangEN)
	body, err := im.fetchPage(ctx, b, url)
	if err != nil || body == nil {
		return jp, err
	}
	en, err := parseFn(bytes.NewReader(body), site.LangEN)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", url, err)
	}
	merged, err := mergeFn(jp, en)
	if err != nil {
		return nil, fmt.Errorf("merge %s: %w", url, err)
	}
	return merged, nil
}

type idSet map[int64]struct{}

func (s idSet) add(id int64) { s[id] = struct{}{} }

func (s idSet) addPtr(id *int64) {
	if id != nil {
		s.add(*id)
	}
}

func (s idSet) sorted() []int64 {
	out := make([]int64, 0, len(s))
	for id := range s {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func teamIDs(b *Bundle) []int64 {
	ids := idSet{}
	if b.TeamList != nil {
		for _, d := range b.TeamList.Divisions {
			for _, e := range d.Entries {
				ids.addPtr(e.ID)
			}
		}
	}
	for _, s := range b.Schedules {
		for _, r := range s.Rows {
			ids.addPtr(r.Left.TeamID)
			ids.addPtr(r.Right.TeamID)
		}
	}
	for _, st := range b.Standings {
		for _, r := range st.Rows {
			ids.add(r.TeamID)
		}
	}
	for _, g := range b.Groups {
		if g.Pool != nil {
			for _, t := range g.Pool.Teams {
				ids.addPtr(t.TeamID)
			}
		}
		if g.Bracket != nil {
			for _, row := range g.Bracket.Rows {
				for _, c := range row {
					ids.addPtr(c.TeamID)
				}
			}
		}
	}
	return ids.sorted()
}

func matchIDs(b *Bundle) []int64 {
	ids := idSet{}
	for _, s := range b.Schedules {
		for _, r := range s.Rows {
			ids.addPtr(r.MatchID)
		}
	}
	for _, g := range b.Groups {
		for _, r := range g.Matches {
			ids.addPtr(r.MatchID)
		}
		if g.Pool != nil {
			for _, c := range g.Pool.Cells {
				ids.addPtr(c.MatchID)
			}
		}
		if g.Bracket != nil {
			for _, row := range g.Bracket.Rows {
				for _, c := range row {
					ids.addPtr(c.MatchID)
				}
			}
		}
	}
	for _, t := range b.Teams {
		for _, r := range t.Matches {
			ids.addPtr(r.MatchID)
		}
	}
	return ids.sorted()
}
