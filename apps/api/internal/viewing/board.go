package viewing

import (
	"slices"

	"scoreview/api/internal/domain"
)

// Board は大会の掲示板。スコアサイトと観戦アプリが、星取表、順位、日程、トーナメント表を描くのに
// 要るものを 1 つにまとめる。
type Board struct {
	Tournament Tournament
	Divisions  []BoardDivision
	// Matches は全試合。日付、時刻、コートの順(日程の既定の並び)
	Matches []BoardMatch
	Tables  TablesConfig
}

// BoardDivision は部門の組とトーナメント。組もトーナメントも無い部門も入れる(日程の部門の切り替えに使う)。
type BoardDivision struct {
	ID       int64
	Name     domain.I18nText
	Pools    []Pool
	Brackets []Bracket
}

// Pool は総当たりの組の、星取表と順位表に要るもの。
type Pool struct {
	GroupID int64
	// Name は組の呼び名。大会の形を決めた部門では日本語を組の試合の名前(「予選 A 組」)にする
	// (運営が組を呼ぶ言葉。組の名前は「A」のような記号のことがある)
	Name domain.I18nText
	// Done は組の試合がすべて終わったこと(試合が 1 つも無い組は偽)
	Done    bool
	Advance int
	// Standings は登録順(星取表の行の並び)。順位は Rank
	Standings []Standing
}

// Standing は組の 1 チームの成績。Rank が nil なら順位を出さない(まだ 1 試合も終わっていないなど)。
type Standing struct {
	Team   domain.TeamRef
	Rank   *int
	Tally  domain.Tally
	Points int
}

// BoardMatch は掲示板の試合。Group があれば、その組の詳しい画面へ案内できる。
type BoardMatch struct {
	domain.ScheduleRow
	Group *MatchGroup
}

// MatchGroup は試合が入っている組。
type MatchGroup struct {
	ID   int64
	Name domain.I18nText
	// Pool は総当たりの組
	Pool bool
}

// Assemble は読んだ記録から掲示板を組み立てる。記録を書き換えない。
func Assemble(snapshot Snapshot) Board {
	groups := make(map[int64]Group, len(snapshot.Groups))
	for _, group := range snapshot.Groups {
		groups[group.ID] = group
	}
	matches := make([]BoardMatch, 0, len(snapshot.Matches))
	for _, match := range snapshot.Matches {
		out := BoardMatch{ScheduleRow: match.ScheduleRow}
		if match.GroupID != nil {
			if group, ok := groups[*match.GroupID]; ok {
				out.Group = &MatchGroup{ID: group.ID, Name: group.Name, Pool: group.Kind == domain.GroupPool}
			}
		}
		matches = append(matches, out)
	}
	slices.SortFunc(matches, func(a, b BoardMatch) int { return domain.CompareMatchRows(a.MatchRow, b.MatchRow) })

	byGroup := map[int64][]BoardMatch{}
	for _, match := range matches {
		if match.Group != nil {
			byGroup[match.Group.ID] = append(byGroup[match.Group.ID], match)
		}
	}
	divisions := make([]BoardDivision, 0, len(snapshot.Divisions))
	index := make(map[int64]int, len(snapshot.Divisions))
	for i, division := range snapshot.Divisions {
		index[division.ID] = i
		divisions = append(divisions, BoardDivision{ID: division.ID, Name: division.Name,
			Pools: []Pool{}, Brackets: []Bracket{}})
	}
	for _, group := range snapshot.Groups {
		at, ok := index[group.DivisionID]
		if !ok || group.Kind != domain.GroupPool {
			continue
		}
		divisions[at].Pools = append(divisions[at].Pools, poolOf(group, byGroup[group.ID]))
	}
	for _, bracket := range snapshot.Brackets {
		if at, ok := index[bracket.DivisionID]; ok {
			divisions[at].Brackets = append(divisions[at].Brackets, bracket)
		}
	}
	return Board{Tournament: snapshot.Tournament, Divisions: divisions, Matches: matches, Tables: snapshot.Tables}
}

// poolOf は組の順位を計算する。matches はその組の試合で、日程の並び。
func poolOf(group Group, matches []BoardMatch) Pool {
	entries := make([]domain.PoolEntry, len(group.Entries))
	for i, entry := range group.Entries {
		entries[i] = domain.PoolEntry{TeamID: entry.Team.ID, RankOverride: entry.RankOverride}
	}
	results := make([]domain.Result, len(matches))
	done := len(matches) > 0
	name := group.Name
	for i, match := range matches {
		results[i] = resultOf(match.MatchRow)
		done = done && match.Status == domain.MatchFinished
		if i == 0 && match.Stage != "" {
			name.Ja = match.Stage
		}
	}
	standings := make([]Standing, len(entries))
	for i, standing := range domain.RankPool(entries, results) {
		standings[i] = Standing{Team: group.Entries[i].Team, Rank: standing.Rank, Tally: standing.Tally,
			Points: standing.Points}
	}
	return Pool{GroupID: group.ID, Name: name, Done: done, Advance: group.Advance, Standings: standings}
}

// resultOf は日程の行を成績の集計に使う結果にする。チームが決まっていない側は 0。
func resultOf(row domain.MatchRow) domain.Result {
	result := domain.Result{LeftScore: row.Left.Score, RightScore: row.Right.Score, Status: row.Status,
		WinnerSide: row.WinnerSide, ResultType: row.ResultType}
	if row.Left.Team != nil {
		result.LeftTeamID = row.Left.Team.ID
	}
	if row.Right.Team != nil {
		result.RightTeamID = row.Right.Team.ID
	}
	return result
}
