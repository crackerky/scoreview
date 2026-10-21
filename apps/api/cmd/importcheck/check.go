package main

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"scoreview/api/internal/importer"
	"scoreview/api/internal/importer/parse"
)

// result は 1 つの確認項目の結果。
type result struct {
	name   string
	ok     bool
	detail string
}

// checker は 1 大会分の確認。b はキャッシュの HTML をパースしたもの(参照サイトの値)。
type checker struct {
	api      *apiClient
	b        *importer.Bundle
	listed   *listDTO
	players  bool
	requests int
	non200   []string
	results  []result
	// siteInconsistent は参照サイトの HTML でも得点経過の累計とスコアが食い違う試合(情報として出す)。
	siteInconsistent []string
}

func (c *checker) add(name string, mismatches []string, okDetail string) {
	if len(mismatches) == 0 {
		c.results = append(c.results, result{name: name, ok: true, detail: okDetail})
		return
	}
	const shown = 8
	detail := strings.Join(mismatches[:min(len(mismatches), shown)], "; ")
	if len(mismatches) > shown {
		detail += fmt.Sprintf("; ... (%d 件)", len(mismatches))
	}
	c.results = append(c.results, result{name: name, ok: false, detail: detail})
}

// fetch は path を叩いて out に読む。200 でなければ記録して false を返す。
func (c *checker) fetch(ctx context.Context, path string, out any) bool {
	c.requests++
	status, err := c.api.get(ctx, path, out)
	if err != nil {
		c.non200 = append(c.non200, fmt.Sprintf("%s: %v", path, err))
		return false
	}
	if status != http.StatusOK {
		c.non200 = append(c.non200, fmt.Sprintf("%s: HTTP %d", path, status))
		return false
	}
	return true
}

func (c *checker) run(ctx context.Context) {
	id := c.b.TournamentID
	var top topDTO
	if !c.fetch(ctx, fmt.Sprintf("/api/v1/tournaments/%d", id), &top) {
		return
	}
	c.checkTop(top)
	var teams teamsDTO
	c.fetch(ctx, fmt.Sprintf("/api/v1/tournaments/%d/teams", id), &teams)
	c.checkTeams(teams)
	c.fetch(ctx, fmt.Sprintf("/api/v1/tournaments/%d/schedule", id), &ignoredDTO{})
	c.fetch(ctx, fmt.Sprintf("/api/v1/tournaments/%d/schedule?ymd=", id), &ignoredDTO{})

	matchIDs := map[int64]bool{}
	schedules := map[int64]scheduleDTO{}
	for _, d := range top.Frame.Divisions {
		var s scheduleDTO
		if c.fetch(ctx, fmt.Sprintf("/api/v1/divisions/%d/schedule?ymd=", d.ID), &s) {
			schedules[d.ID] = s
			for _, r := range s.Rows {
				matchIDs[r.MatchID] = true
			}
		}
	}
	c.checkSchedules(schedules)
	standings := map[int64]standingsDTO{}
	for _, d := range top.Frame.Divisions {
		var s standingsDTO
		if c.fetch(ctx, fmt.Sprintf("/api/v1/divisions/%d/standings", d.ID), &s) {
			standings[d.ID] = s
		}
	}
	c.checkStandings(standings)
	groups := map[int64]groupDTO{}
	for _, d := range top.Frame.Divisions {
		for _, g := range d.Groups {
			var gr groupDTO
			if c.fetch(ctx, fmt.Sprintf("/api/v1/groups/%d", g.ID), &gr) {
				groups[g.ID] = gr
				collectGroupMatches(gr, matchIDs)
			}
		}
	}
	c.checkPools(groups)

	var playerIDs []int64
	for _, d := range teams.Divisions {
		for _, t := range d.Teams {
			var tp teamDTO
			if c.fetch(ctx, fmt.Sprintf("/api/v1/teams/%d", t.ID), &tp) {
				for _, p := range tp.Players {
					playerIDs = append(playerIDs, p.Player.ID)
				}
				for _, s := range tp.Staff {
					playerIDs = append(playerIDs, s.ID)
				}
			}
		}
	}
	matches := map[int64]matchDTO{}
	for _, mid := range sortedIDs(matchIDs) {
		var m matchDTO
		if c.fetch(ctx, fmt.Sprintf("/api/v1/matches/%d", mid), &m) {
			matches[mid] = m
		}
	}
	c.checkMatches(matches)
	if c.players {
		for _, pid := range playerIDs {
			c.fetch(ctx, fmt.Sprintf("/api/v1/players/%d", pid), &ignoredDTO{})
		}
	}
	for _, g := range top.Frame.Galleries {
		c.fetch(ctx, fmt.Sprintf("/api/v1/galleries/%d", g.ID), &ignoredDTO{})
	}
}

func collectGroupMatches(g groupDTO, ids map[int64]bool) {
	for _, r := range g.Matches {
		ids[r.MatchID] = true
	}
	if g.Bracket != nil {
		for _, row := range g.Bracket.Rows {
			for _, cell := range row {
				if cell.MatchID != nil {
					ids[*cell.MatchID] = true
				}
			}
		}
	}
}

func (c *checker) checkTop(top topDTO) {
	b := c.b
	var bad []string
	if top.Frame.Name.Ja != b.Top.Name.JA || top.Frame.Name.En != b.Top.Name.EN {
		bad = append(bad, fmt.Sprintf("api %q/%q html %q/%q",
			top.Frame.Name.Ja, top.Frame.Name.En, b.Top.Name.JA, b.Top.Name.EN))
	}
	c.add("大会名(日英)", bad, top.Frame.Name.Ja)

	section := ""
	for _, s := range c.listed.Sections {
		for _, t := range s.Tournaments {
			if t.ID == b.TournamentID {
				section = s.Status
			}
		}
	}
	want := ""
	if b.ListEntry != nil {
		want = string(b.ListEntry.Status)
	}
	bad = nil
	if section != want {
		bad = append(bad, fmt.Sprintf("api %q html %q", section, want))
	}
	c.add("大会一覧の区分", bad, orNone(want, "一覧に載らない"))

	bad = nil
	if len(top.Frame.Divisions) != len(b.Top.Divisions) {
		bad = append(bad, fmt.Sprintf("ディビジョン数 api %d html %d",
			len(top.Frame.Divisions), len(b.Top.Divisions)))
	} else {
		for i, d := range b.Top.Divisions {
			if got := len(top.Frame.Divisions[i].Groups); got != len(d.Groups) {
				bad = append(bad, fmt.Sprintf("%s: api %d html %d", d.Name.JA, got, len(d.Groups)))
			}
		}
	}
	total := 0
	for _, d := range b.Top.Divisions {
		total += len(d.Groups)
	}
	c.add("グループ数", bad, fmt.Sprintf("%d ディビジョン、%d グループ", len(b.Top.Divisions), total))
}

func (c *checker) checkTeams(teams teamsDTO) {
	b := c.b
	if b.TeamList == nil {
		c.add("チーム数", []string{"チーム一覧の HTML がない"}, "")
		return
	}
	var bad []string
	total := 0
	if len(teams.Divisions) != len(b.TeamList.Divisions) {
		bad = append(bad, fmt.Sprintf("ディビジョン数 api %d html %d",
			len(teams.Divisions), len(b.TeamList.Divisions)))
	} else {
		for i, d := range b.TeamList.Divisions {
			total += len(d.Entries)
			if got := len(teams.Divisions[i].Teams); got != len(d.Entries) {
				bad = append(bad, fmt.Sprintf("%s: api %d html %d", d.Name.JA, got, len(d.Entries)))
			}
		}
	}
	c.add("チーム数", bad, fmt.Sprint(total))
}

func (c *checker) checkSchedules(schedules map[int64]scheduleDTO) {
	var bad []string
	total, checked := 0, 0
	for _, did := range sortedKeys(c.b.Schedules) {
		want := c.b.Schedules[did]
		got, ok := schedules[did]
		if !ok {
			bad = append(bad, fmt.Sprintf("ディビジョン %d のスケジュールがない", did))
			continue
		}
		total += len(want.Rows)
		if len(got.Rows) != len(want.Rows) {
			bad = append(bad, fmt.Sprintf("ディビジョン %d: api %d html %d",
				did, len(got.Rows), len(want.Rows)))
		}
		byID := map[int64]matchRowDTO{}
		for _, r := range got.Rows {
			byID[r.MatchID] = r
		}
		for _, w := range want.Rows {
			if w.MatchID == nil {
				continue
			}
			checked++
			r, ok := byID[*w.MatchID]
			switch {
			case !ok:
				bad = append(bad, fmt.Sprintf("試合 %d がない", *w.MatchID))
			case !equalInt(r.Left.Score, w.LeftScore) || !equalInt(r.Right.Score, w.RightScore):
				bad = append(bad, fmt.Sprintf("試合 %d %s のスコア api %s-%s html %s-%s",
					*w.MatchID, w.Code, intString(r.Left.Score), intString(r.Right.Score),
					intString(w.LeftScore), intString(w.RightScore)))
			case r.Status != w.Status:
				bad = append(bad, fmt.Sprintf("試合 %d の状態 api %s html %s",
					*w.MatchID, r.Status, w.Status))
			}
		}
	}
	if len(c.b.Schedules) == 0 {
		c.add("試合数(スケジュール)", nil, "スケジュールなし")
		return
	}
	c.add("試合数(スケジュール)", bad, fmt.Sprintf("%d 試合(ID のある %d 試合のスコアと状態も一致)", total, checked))
}

func (c *checker) checkStandings(standings map[int64]standingsDTO) {
	var bad []string
	var shown []string
	for _, did := range sortedKeys(c.b.Standings) {
		want := map[int64]int{}
		for _, r := range c.b.Standings[did].Rows {
			if r.Rank != nil && *r.Rank <= 3 {
				want[r.TeamID] = *r.Rank
			}
		}
		got := map[int64]int{}
		for _, r := range standings[did].Standings {
			if r.Rank != nil && *r.Rank <= 3 {
				got[r.Team.ID] = *r.Rank
			}
		}
		if !mapsEqual(want, got) {
			bad = append(bad, fmt.Sprintf("ディビジョン %d: api %v html %v", did, got, want))
		}
		shown = append(shown, fmt.Sprintf("%d: %d チーム", did, len(want)))
	}
	if len(c.b.Standings) == 0 {
		c.add("順位表の 1〜3 位", nil, "ディビジョン成績なし")
		return
	}
	c.add("順位表の 1〜3 位", bad, strings.Join(shown, ", "))
}

func (c *checker) checkPools(groups map[int64]groupDTO) {
	var bad []string
	pools, teams := 0, 0
	for _, gid := range sortedKeys(c.b.Groups) {
		g := c.b.Groups[gid]
		got, ok := groups[gid]
		if !ok {
			continue
		}
		if got.Group.Kind != g.Kind {
			bad = append(bad, fmt.Sprintf("グループ %d の種類 api %s html %s",
				gid, got.Group.Kind, g.Kind))
			continue
		}
		if g.Pool == nil || got.Pool == nil {
			continue
		}
		pools++
		apiRank := map[int64]*int{}
		for _, r := range got.Pool.Results {
			apiRank[r.Team.ID] = r.Rank
		}
		for _, t := range g.Pool.Teams {
			if t.TeamID == nil {
				continue
			}
			teams++
			if r, ok := apiRank[*t.TeamID]; !ok || !equalInt(r, t.Rank) {
				bad = append(bad, fmt.Sprintf("グループ %d チーム %d: api %s html %q",
					gid, *t.TeamID, intString(r), t.RankRaw))
			}
		}
	}
	c.add("プールの順位(グループ種類を含む)", bad, fmt.Sprintf("%d プール、%d チーム", pools, teams))
}

func (c *checker) checkMatches(matches map[int64]matchDTO) {
	var bad []string
	for _, mid := range sortedKeys(c.b.Matches) {
		w := c.b.Matches[mid]
		got, ok := matches[mid]
		if !ok {
			bad = append(bad, fmt.Sprintf("試合 %d を api で開けない", mid))
			continue
		}
		if !equalInt(got.Match.Left.Score, w.Left.Score) ||
			!equalInt(got.Match.Right.Score, w.Right.Score) {
			bad = append(bad, fmt.Sprintf("試合 %d のスコア api %s-%s html %s-%s", mid,
				intString(got.Match.Left.Score), intString(got.Match.Right.Score),
				intString(w.Left.Score), intString(w.Right.Score)))
		}
		if len(got.Events) != len(w.Events) {
			bad = append(bad, fmt.Sprintf("試合 %d の得点経過 api %d 行 html %d 行",
				mid, len(got.Events), len(w.Events)))
			continue
		}
		// api の得点経過の最後の累計を、HTML の得点経過の得点の数と比べる。参照サイトには得点経過と
		// スコアが食い違う試合がある(得点の行が欠けている)ので、スコアとの食い違いは別に数える
		wantL, wantR := goalCounts(w)
		if n := len(got.Events); n > 0 {
			last := got.Events[n-1]
			if last.LeftScore != wantL || last.RightScore != wantR {
				bad = append(bad, fmt.Sprintf("試合 %d の得点経過の累計 api %d-%d html %d-%d", mid,
					last.LeftScore, last.RightScore, wantL, wantR))
			}
		}
		if w.Left.Score != nil && w.Right.Score != nil && len(w.Events) > 0 &&
			(wantL != *w.Left.Score || wantR != *w.Right.Score) {
			c.siteInconsistent = append(c.siteInconsistent, fmt.Sprintf("%d (得点経過 %d-%d、スコア %d-%d)",
				mid, wantL, wantR, *w.Left.Score, *w.Right.Score))
		}
	}
	c.add("試合ページのスコアと得点経過", bad, fmt.Sprintf("%d 試合", len(c.b.Matches)))
}

// goalCounts は試合ページの得点経過の得点の数を左右で数える。
func goalCounts(m *parse.Match) (left, right int) {
	for _, ev := range m.Events {
		if ev.Kind != "goal" || ev.Side == nil {
			continue
		}
		if *ev.Side == "left" {
			left++
		} else {
			right++
		}
	}
	return left, right
}

func equalInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func intString(v *int) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprint(*v)
}

func orNone(s, none string) string {
	if s == "" {
		return none
	}
	return s
}

func mapsEqual(a, b map[int64]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

func sortedIDs(set map[int64]bool) []int64 {
	out := make([]int64, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func sortedKeys[V any](m map[int64]V) []int64 {
	out := make([]int64, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}
