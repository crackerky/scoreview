// Package format は大会形式エンジン。形の指定(テンプレートとパラメータ)から段、グループ、枠、
// 試合を生成し、結果から順位と勝ち上がりを解決し、時間割を組む。GORM と HTTP を知らない純粋な
// パッケージで、生成物は ID の代わりにローカルな参照(グループの序数、試合 CD)でつなぐ
// (保存は store が行う。docs/format-engine.md)。
package format

import (
	"fmt"

	"scoreview/api/internal/domain"
)

// Plan はテンプレートから生成した大会形式の全体。同じ Spec からは常に同じ Plan ができる。
// Groups の添字がグループの序数で、Source.Group と Match.Group はこの序数を指す。
type Plan struct {
	Template string
	Teams    int
	Stages   []Stage
	Groups   []Group
	// Matches はグループ順、グループ内は試合番号順に並ぶ
	Matches []Match
}

// Stage は同時期に行うグループのまとまり(予選プール、順位決定トーナメントなど)。
type Stage struct {
	Index int
	Name  string
}

// BracketType はトーナメントの型(R3 4 章)。
type BracketType string

// BracketType の値。T2 は参照サイトの 2 プールの同順位 1 戦(490)を表すために足した最小の型。
const (
	BracketT2      BracketType = "T2"
	BracketT4      BracketType = "T4"
	BracketT7      BracketType = "T7"
	BracketT8      BracketType = "T8"
	BracketT8Block BracketType = "T8Block"
	BracketT8R1    BracketType = "T8R1"
	BracketT12     BracketType = "T12"
	BracketT16     BracketType = "T16"
)

// Group は総当たり(pool)かトーナメント(bracket)の 1 グループ。groups 表の 1 行に当たる。
type Group struct {
	Index  int
	Stage  int
	Letter string
	Name   string
	Kind   domain.GroupKind
	// Bracket はトーナメントの型。プールでは空
	Bracket BracketType
	// Pattern は参照サイトの GeneratePtn("T8E" など)。プールでは空
	Pattern string
	Slots   []Slot
	// RankOffset は最終順位への変換に足す数。グループ内の r 位がディビジョンの RankOffset+r 位
	RankOffset int
	// Final はグループの順位がディビジョンの最終順位になるか。予選プールとブロックは偽
	Final bool
	// Layout はトーナメント表の格子。プールでは nil
	Layout *Grid
}

// Size はグループのチーム数。
func (g Group) Size() int { return len(g.Slots) }

// Slot はグループの参加位置。Position は 1 から。プールではシード順の位置、トーナメントでは
// 表の上からの位置。
type Slot struct {
	Position int
	Source   Source
}

// SlotKey は枠を大会形式の中で一意に指す。
type SlotKey struct {
	Group    int
	Position int
}

// SourceKind は枠に入るチームの決まり方。
type SourceKind string

// SourceKind の値(docs/format-engine.md「基本の考え方」)。
const (
	SourceSeed      SourceKind = "seed"
	SourceGroupRank SourceKind = "group_rank"
	SourceMatch     SourceKind = "match"
	SourceBandRank  SourceKind = "band_rank"
	SourceManual    SourceKind = "manual"
)

// Source は枠の出どころ。Kind に応じて使うフィールドが決まる。
//   - seed(n): Seed
//   - group_rank(g, r): Group, Rank
//   - match(m, win|lose): Match
//   - band_rank(stage, r, k): Stage, Rank, Order
//   - manual: なし
type Source struct {
	Kind  SourceKind
	Seed  int
	Group int
	Rank  int
	Match MatchRef
	Stage int
	Order int
}

// SeedSource は seed(n)。
func SeedSource(n int) Source { return Source{Kind: SourceSeed, Seed: n} }

// GroupRankSource は group_rank(g, r)。
func GroupRankSource(group, rank int) Source {
	return Source{Kind: SourceGroupRank, Group: group, Rank: rank}
}

// MatchSource は match(m, win|lose)。
func MatchSource(code string, outcome domain.Outcome) Source {
	return Source{Kind: SourceMatch, Match: MatchRef{Code: code, Outcome: outcome}}
}

// BandRankSource は band_rank(stage, r, k)。段の全プールの r 位を並べた k 番目。
func BandRankSource(stage, rank, order int) Source {
	return Source{Kind: SourceBandRank, Stage: stage, Rank: rank, Order: order}
}

// Label は出どころを運営者向けの短い表記にする。group_rank は "A1"、試合の勝者と敗者は
// 参照サイトの未確定表記 "[A1-勝ち]"(clone-spec B 8)。
func (s Source) Label(plan *Plan) string {
	switch s.Kind {
	case SourceSeed:
		return fmt.Sprintf("シード%d", s.Seed)
	case SourceGroupRank:
		return fmt.Sprintf("%s%d", plan.Groups[s.Group].Letter, s.Rank)
	case SourceMatch:
		return s.Match.Label()
	case SourceBandRank:
		return fmt.Sprintf("%s %d位の%d番目", plan.Stages[s.Stage].Name, s.Rank, s.Order)
	default:
		return "未定"
	}
}

// MatchRef は試合 Code の勝者か敗者。
type MatchRef struct {
	Code    string
	Outcome domain.Outcome
}

// Label は参照サイトの未確定表記(clone-spec B 8)。
func (r MatchRef) Label() string {
	if r.Outcome == domain.OutcomeWin {
		return "[" + r.Code + "-勝ち]"
	}
	return "[" + r.Code + "-負け]"
}

// SideRef は試合の片側。グループの枠(Slot、1 から)か、同じグループの前の試合の勝者・敗者(From)の
// どちらか一方が入る。
type SideRef struct {
	Slot int
	From *MatchRef
}

// Match は 1 試合。Code は試合番号 CD("{グループ文字}{連番}")。
type Match struct {
	Code   string
	Group  int
	Number int
	Left   SideRef
	Right  SideRef
	// Round はプールのラウンド(1 から)、トーナメントの回戦(1 回戦が 1)。時間割の論理順序に使う
	Round int
	// WinnerPlace と LoserPlace はこの試合で決まるグループ内の順位(1 から)。0 は決まらない
	// (次の試合へ進む)。同じ順位を複数の試合が決めるときは同順位
	WinnerPlace int
	LoserPlace  int
}

// SlotSource は枠の出どころを返す。
func (p *Plan) SlotSource(key SlotKey) Source {
	return p.Groups[key.Group].Slots[key.Position-1].Source
}

// MatchByCode は試合を CD で引く。
func (p *Plan) MatchByCode(code string) (Match, bool) {
	for _, m := range p.Matches {
		if m.Code == code {
			return m, true
		}
	}
	return Match{}, false
}

// GroupMatches はグループの試合を番号順で返す。
func (p *Plan) GroupMatches(group int) []Match {
	var out []Match
	for _, m := range p.Matches {
		if m.Group == group {
			out = append(out, m)
		}
	}
	return out
}

// StageGroups は段に属するグループの序数を返す。
func (p *Plan) StageGroups(stage int) []int {
	var out []int
	for _, g := range p.Groups {
		if g.Stage == stage {
			out = append(out, g.Index)
		}
	}
	return out
}
