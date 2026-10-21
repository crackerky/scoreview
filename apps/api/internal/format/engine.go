package format

import (
	"fmt"
	"slices"
	"strings"

	"scoreview/api/internal/domain"
)

// Template は形の指定から段、グループ、試合を組み立てる可変点。Build は Builder に段と
// グループを足すだけで、試合番号、グループ文字、格子は Builder と Engine が付ける。
type Template interface {
	Name() string
	// Build は params(テンプレートごとの型)を検査して b に段とグループを足す。
	// 型が違う、または値が不正なら domain.ValidationError を返す。
	Build(b *Builder, params any) error
}

// Spec は形の指定。StartLetter はこのディビジョンの最初のグループ文字で、大会内の通しにするため
// 前のディビジョンの続きを呼び出し側が渡す(R3 5-5。520 ではオープンが A–L、ウィメンが M–T)。
type Spec struct {
	Template    string
	StartLetter string
	Params      any
}

// Engine は登録されたテンプレートで Spec から Plan を作る。
type Engine struct {
	templates map[string]Template
}

// Option は Engine の設定。
type Option func(*Engine)

// WithTemplate はテンプレートを登録する。同じ名前の既定のテンプレートは置き換わる。
func WithTemplate(t Template) Option {
	return func(e *Engine) { e.templates[t.Name()] = t }
}

// NewEngine は既定の 3 テンプレート(pool_bands、blocks、round_robin)を登録した Engine を返す。
func NewEngine(opts ...Option) *Engine {
	e := &Engine{templates: map[string]Template{}}
	for _, t := range []Template{PoolBandsTemplate{}, BlocksTemplate{}, RoundRobinTemplate{}} {
		e.templates[t.Name()] = t
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Templates は登録されたテンプレート名を名前順で返す。
func (e *Engine) Templates() []string {
	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Generate は Spec から Plan を作る。出力は決定的。テンプレートが無い、パラメータが不正、
// 生成物の参照が壊れている場合は domain.ValidationError(errors.Is で domain.ErrBadRequest)を返す。
func (e *Engine) Generate(spec Spec) (*Plan, error) {
	t, ok := e.templates[spec.Template]
	if !ok {
		return nil, domain.Invalid("template", fmt.Sprintf("unknown template %q", spec.Template))
	}
	start := spec.StartLetter
	if start == "" {
		start = "A"
	}
	if _, err := letterIndex(start); err != nil {
		return nil, err
	}
	b := &Builder{start: start}
	if err := t.Build(b, spec.Params); err != nil {
		return nil, fmt.Errorf("template %s: %w", spec.Template, err)
	}
	plan := b.plan
	plan.Template = spec.Template
	if err := validatePlan(&plan); err != nil {
		return nil, err
	}
	for i := range plan.Groups {
		if plan.Groups[i].Kind == domain.GroupBracket {
			grid, err := buildGrid(&plan, i)
			if err != nil {
				return nil, err
			}
			plan.Groups[i].Layout = grid
		}
	}
	return &plan, nil
}

// Builder はテンプレートが段とグループを足すための道具。グループ文字と試合番号はここで振る。
type Builder struct {
	start string
	plan  Plan
}

// SetTeams は大会形式のチーム数(シードの数)を記録する。
func (b *Builder) SetTeams(n int) { b.plan.Teams = n }

// AddStage は段を足し、その序数を返す。
func (b *Builder) AddStage(name string) int {
	index := len(b.plan.Stages)
	b.plan.Stages = append(b.plan.Stages, Stage{Index: index, Name: name})
	return index
}

// Plan は組み立て途中の Plan を返す(テンプレートが前に足したグループを参照するため)。
func (b *Builder) Plan() *Plan { return &b.plan }

// GroupSpec は足すグループ。Name の "{L}" はグループ文字に置き換わる。Bracket が空ならプール。
type GroupSpec struct {
	Stage      int
	Name       string
	Bracket    BracketType
	Sources    []Source
	Final      bool
	RankOffset int
}

// AddGroup はグループと試合を足し、グループの序数を返す。プールは固定の試合順、トーナメントは
// 型の試合を作る。Sources の数が型の枠の数と合わなければエラー。
func (b *Builder) AddGroup(spec GroupSpec) (int, error) {
	index := len(b.plan.Groups)
	letter, err := groupLetter(b.start, index)
	if err != nil {
		return 0, err
	}
	group := Group{
		Index: index, Stage: spec.Stage, Letter: letter,
		Name:  strings.ReplaceAll(spec.Name, "{L}", letter),
		Final: spec.Final, RankOffset: spec.RankOffset,
	}
	for i, src := range spec.Sources {
		group.Slots = append(group.Slots, Slot{Position: i + 1, Source: src})
	}
	if spec.Bracket == "" {
		if len(spec.Sources) < 2 {
			return 0, domain.Invalid("group", fmt.Sprintf("pool %s needs at least 2 teams", letter))
		}
		group.Kind = domain.GroupPool
		b.plan.Groups = append(b.plan.Groups, group)
		b.addPoolMatches(group)
		return index, nil
	}
	def, err := bracketDefFor(spec.Bracket)
	if err != nil {
		return 0, err
	}
	if len(spec.Sources) != def.size {
		return 0, domain.Invalid("group", fmt.Sprintf("%s needs %d teams, got %d",
			spec.Bracket, def.size, len(spec.Sources)))
	}
	group.Kind = domain.GroupBracket
	group.Bracket = spec.Bracket
	group.Pattern = fmt.Sprintf("T%d%s", def.size, letter)
	b.plan.Groups = append(b.plan.Groups, group)
	b.addBracketMatches(group, def)
	return index, nil
}

func (b *Builder) addPoolMatches(group Group) {
	n := group.Size()
	var matches []Match
	for round, pairs := range poolRounds(n) {
		for _, pair := range pairs {
			number := pairNumber(n, pair)
			matches = append(matches, Match{
				Code:  fmt.Sprintf("%s%d", group.Letter, number),
				Group: group.Index, Number: number, Round: round + 1,
				Left: SideRef{Slot: pair[0]}, Right: SideRef{Slot: pair[1]},
			})
		}
	}
	slices.SortFunc(matches, func(a, c Match) int { return a.Number - c.Number })
	b.plan.Matches = append(b.plan.Matches, matches...)
}

func (b *Builder) addBracketMatches(group Group, def bracketDef) {
	code := func(number int) string { return fmt.Sprintf("%s%d", group.Letter, number) }
	side := func(s bracketSide) SideRef {
		if s.slot > 0 {
			return SideRef{Slot: s.slot}
		}
		return SideRef{From: &MatchRef{Code: code(s.from), Outcome: s.outcome}}
	}
	defs := slices.Clone(def.matches)
	slices.SortFunc(defs, func(a, c bracketMatchDef) int { return a.number - c.number })
	for _, m := range defs {
		b.plan.Matches = append(b.plan.Matches, Match{
			Code: code(m.number), Group: group.Index, Number: m.number, Round: m.round,
			Left: side(m.left), Right: side(m.right),
			WinnerPlace: m.winnerPlace, LoserPlace: m.loserPlace,
		})
	}
}

// validatePlan は生成物の参照(出どころ、試合の左右)が壊れていないことを確かめる。出どころは
// 前のグループ(band_rank は前の段)だけを指す。こうしておけば解決の順序に循環が起きない。
func validatePlan(plan *Plan) error {
	codes := map[string]int{}
	for _, m := range plan.Matches {
		if _, dup := codes[m.Code]; dup {
			return domain.Invalid("plan", "duplicate match code "+m.Code)
		}
		codes[m.Code] = m.Group
	}
	seeds := map[int]bool{}
	for _, g := range plan.Groups {
		for _, slot := range g.Slots {
			if err := validateSource(plan, g, slot.Source, codes, seeds); err != nil {
				return err
			}
		}
	}
	for _, m := range plan.Matches {
		for _, side := range []SideRef{m.Left, m.Right} {
			switch {
			case side.From != nil:
				if group, ok := codes[side.From.Code]; !ok || group != m.Group {
					return domain.Invalid("plan", fmt.Sprintf("%s refers to %s outside its group",
						m.Code, side.From.Code))
				}
			case side.Slot < 1 || side.Slot > plan.Groups[m.Group].Size():
				return domain.Invalid("plan", fmt.Sprintf("%s has no slot %d", m.Code, side.Slot))
			}
		}
	}
	return nil
}

func validateSource(plan *Plan, g Group, src Source, codes map[string]int, seeds map[int]bool) error {
	where := fmt.Sprintf("group %s", g.Letter)
	switch src.Kind {
	case SourceSeed:
		if src.Seed < 1 || src.Seed > plan.Teams || seeds[src.Seed] {
			return domain.Invalid("plan",
				fmt.Sprintf("%s: seed %d is out of range or reused", where, src.Seed))
		}
		seeds[src.Seed] = true
	case SourceGroupRank:
		if src.Group < 0 || src.Group >= g.Index || src.Rank < 1 || src.Rank > plan.Groups[src.Group].Size() {
			return domain.Invalid("plan",
				fmt.Sprintf("%s: bad group_rank(%d, %d)", where, src.Group, src.Rank))
		}
	case SourceMatch:
		if group, ok := codes[src.Match.Code]; !ok || group >= g.Index {
			return domain.Invalid("plan", fmt.Sprintf("%s: bad match source %s", where, src.Match.Code))
		}
	case SourceBandRank:
		if src.Stage < 0 || src.Stage >= g.Stage || src.Rank < 1 || src.Order < 1 {
			return domain.Invalid("plan", fmt.Sprintf("%s: bad band_rank(%d, %d, %d)",
				where, src.Stage, src.Rank, src.Order))
		}
	case SourceManual:
	default:
		return domain.Invalid("plan", fmt.Sprintf("%s: unknown source %q", where, src.Kind))
	}
	return nil
}
