// Package bracket は管理画面で選ぶトーナメント表のテンプレートを持つ(docs/architecture.md の可変点)。
// テンプレートは 1 ファイルに 1 つ置き、Registry へ Functional Options で登録する。
// テンプレートは格子(参照サイトと同じく罫線だけで山を描く表)と試合の設計図(domain.BracketPlan)を作り、
// 保存は store が行う。
package bracket

import "scoreview/api/internal/domain"

// Template はトーナメント表のテンプレート。
type Template struct {
	// ID は管理 API で指定する識別子(例 "single8")
	ID string
	// Name は管理画面の選択肢に出す名前
	Name domain.I18nText
	// Teams は参加チーム数
	Teams int
	// plan は設計図を作る。呼ぶたびに新しい値を返す(呼び出し側が書き換えてよい)
	plan func() domain.BracketPlan
}

// Plan は設計図を作る。
func (t Template) Plan() domain.BracketPlan { return t.plan() }

// Registry は使えるテンプレートの一覧。登録順を保つ。
type Registry struct {
	templates []Template
}

// Option は Registry に登録するテンプレートを足す Functional Option。
type Option func(*Registry)

// WithTemplate はテンプレートを登録する。同じ ID を 2 度登録したら後のものが勝つ。
func WithTemplate(t Template) Option {
	return func(r *Registry) {
		for i, existing := range r.templates {
			if existing.ID == t.ID {
				r.templates[i] = t
				return
			}
		}
		r.templates = append(r.templates, t)
	}
}

// NewRegistry は opts のテンプレートを登録した Registry を返す。
func NewRegistry(opts ...Option) *Registry {
	r := &Registry{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Default は標準のテンプレート(4、8、16 チーム)を登録した Registry を返す。
func Default() *Registry {
	return NewRegistry(
		WithTemplate(FourTeams()),
		WithTemplate(EightTeams()),
		WithTemplate(SixteenTeams()),
	)
}

// Get は ID のテンプレートを返す。
func (r *Registry) Get(id string) (Template, bool) {
	for _, t := range r.templates {
		if t.ID == id {
			return t, true
		}
	}
	return Template{}, false
}

// List は登録順のテンプレートを返す。
func (r *Registry) List() []Template {
	return append([]Template(nil), r.templates...)
}
