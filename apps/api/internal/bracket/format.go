package bracket

import (
	"fmt"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// FormatTemplate は大会形式エンジン(internal/format)のトーナメントの型をテンプレートにする。
// 参照サイトの型(試合番号は決勝が 1、3 位決定・5 位決定・敗者戦の行き先つき。R3 4 章)を
// 単独のトーナメント表としても選べるようにするため。型が無ければエラー。
func FormatTemplate(id string, kind format.BracketType, name domain.I18nText) (Template, error) {
	first, err := format.BracketPlanFor(kind)
	if err != nil {
		return Template{}, fmt.Errorf("format template %s: %w", id, err)
	}
	return Template{
		ID:    id,
		Name:  name,
		Teams: first.Teams,
		plan: func() domain.BracketPlan {
			// 同じ型は上で一度作れているので、ここでエラーにはならない
			plan, _ := format.BracketPlanFor(kind)
			return plan
		},
	}, nil
}

// WithFormatTypes は大会形式エンジンの型のうち、単独で使える表(T7、T8 全順位型、T16)を登録する。
// 既定の Registry(Default)には入れていない。single4/8/16 を置き換えるかは管理画面の担当と決める。
func WithFormatTypes() Option {
	return func(r *Registry) {
		for _, spec := range []struct {
			id   string
			kind format.BracketType
			name domain.I18nText
		}{
			{"t7", format.BracketT7, domain.I18nText{
				Ja: "7 チーム(1〜7 位、最上位は 1 回戦免除)", En: "7 teams (1st-7th, top seed bye)"}},
			{"t8_all_places", format.BracketT8, domain.I18nText{
				Ja: "8 チーム全順位(1〜8 位)", En: "8 teams (1st-8th)"}},
			{"t16", format.BracketT16, domain.I18nText{
				Ja: "16 チーム(1〜8 位)", En: "16 teams (1st-8th)"}},
		} {
			t, err := FormatTemplate(spec.id, spec.kind, spec.name)
			if err != nil {
				// 型はこの一覧の定数で、format が必ず持つ。作れないならプログラムの誤り
				panic(err)
			}
			WithTemplate(t)(r)
		}
	}
}
