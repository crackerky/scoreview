package format

import (
	"fmt"
	"slices"
	"strings"

	"scoreview/api/internal/domain"
)

// TiebreakPreset はタイブレークの設定の名前(division_formats.tiebreak)。
type TiebreakPreset string

// TiebreakPreset の値(docs/format-engine.md「プール順位」)。
const (
	// TiebreakCurrent は現行方式(直接対決あり)。初期値
	TiebreakCurrent TiebreakPreset = "current"
	// TiebreakCurrentNoH2H は現行方式から直接対決を外したもの(直接対決を見ているかは未確認のため)
	TiebreakCurrentNoH2H TiebreakPreset = "current_no_h2h"
	// TiebreakJFDA は JFDA 競技規則
	TiebreakJFDA TiebreakPreset = "jfda"
)

// RuleFor はプリセットのタイブレークを返す。未知の名前は domain.ValidationError。
func RuleFor(preset TiebreakPreset) (Rule, error) {
	switch preset {
	case TiebreakCurrent:
		return CurrentRule(true), nil
	case TiebreakCurrentNoH2H:
		return CurrentRule(false), nil
	case TiebreakJFDA:
		return JFDARule(), nil
	}
	return Rule{}, domain.Invalid("tiebreak", fmt.Sprintf("unknown preset %q", preset))
}

// criterionPhrase は参加チームへの案内に使う基準の言い方。
func criterionPhrase(c Criterion) string {
	var text string
	switch c.Kind {
	case CritPoints:
		text = "勝ち点(勝ち 3、引き分け 1、負け 0)の多い方"
	case CritWins:
		text = "勝利数の多い方"
	case CritH2HPoints:
		text = "並んだチームどうしの直接対決の勝者"
	case CritH2HWins:
		text = "並んだチームどうしの試合の勝利数の多い方"
	case CritFewerForfeits:
		text = "棄権した試合の少ない方"
	case CritGoalDiff:
		text = "得失点差の大きい方"
	case CritGoalsFor:
		text = "得点の多い方"
	case CritH2HGoalDiff:
		text = "並んだチームどうしの試合の得失点差の大きい方"
	case CritH2HGoalsFor:
		text = "並んだチームどうしの試合の得点の多い方"
	case CritCommonGoalDiff:
		text = "共通の対戦相手との試合の得失点差の大きい方"
	case CritCommonGoalsFor:
		text = "共通の対戦相手との試合の得点の多い方"
	case CritPointsPerGame, CritGoalDiffPerGame, CritGoalsForPerGame:
		text = c.Kind.Label() + "の大きい方"
	default:
		text = c.Kind.Label()
	}
	if c.MaxTied > 0 {
		text += fmt.Sprintf("(%d チームが並んだときだけ)", c.MaxTied)
	}
	return text
}

// DescribeRule はタイブレークを、運営者が参加チームへの案内にそのまま使える日本語の文にする。
func DescribeRule(rule Rule) []string {
	if len(rule.Criteria) == 0 {
		return []string{"順位は運営が決めます。"}
	}
	out := []string{"プールの順位は、" + criterionPhrase(rule.Criteria[0]) + "を上位とします。"}
	if len(rule.Criteria) > 1 {
		var rest []string
		for i, c := range rule.Criteria[1:] {
			rest = append(rest, fmt.Sprintf("(%d) %s", i+1, criterionPhrase(c)))
		}
		out = append(out, "並んだときは、次の順に比べて上位を決めます。"+strings.Join(rest, "、")+"。")
	}
	if rule.Mode == TieRestart {
		out = append(out, "3 チーム以上が並び、一部のチームだけが分かれたときは、残ったチームの間で最初の基準から比べ直します。")
	}
	out = append(out, "すべての基準で並んだときは、運営が立ち会ってブリックマーク投げ(またはディスクフリップ)で決めます。")
	return out
}

// DescribeAdvancement は勝ち上がりの規則を、参加チームへの案内に使える日本語の文にする
// (「各プールの 1 位(A1、B1、C1、D1)は 1st placement tournament に進み、1〜4 位を決めます。
// 1 回戦は A1 対 B1、C1 対 D1」)。最初の段のグループは対象にしない。
func DescribeAdvancement(plan *Plan) []string {
	var out []string
	usesGroupRank := false
	for _, g := range plan.Groups {
		if g.Stage == 0 {
			continue
		}
		var labels []string
		for _, slot := range g.Slots {
			labels = append(labels, slot.Source.Label(plan))
			if slot.Source.Kind == SourceGroupRank {
				usesGroupRank = true
			}
		}
		text := fmt.Sprintf("%s(%s)は %s に進み", entrantsPhrase(plan, g), strings.Join(labels, "、"), g.Name)
		if first, last, ok := groupPlaces(g); ok && g.Final {
			if first == last {
				text += fmt.Sprintf("、%d 位を決めます。", first)
			} else {
				text += fmt.Sprintf("、%d〜%d 位を決めます。", first, last)
			}
		} else {
			text += "ます。"
		}
		if g.Kind == domain.GroupBracket {
			var pairs []string
			for _, m := range plan.GroupMatches(g.Index) {
				if m.Left.From == nil && m.Right.From == nil {
					pairs = append(pairs, fmt.Sprintf("%s 対 %s",
						g.Slots[m.Left.Slot-1].Source.Label(plan), g.Slots[m.Right.Slot-1].Source.Label(plan)))
				}
			}
			if len(pairs) > 0 {
				text += "1 回戦は " + strings.Join(pairs, "、") + "。"
			}
		} else {
			text += "総当たりで順位を決めます。"
		}
		out = append(out, text)
	}
	if usesGroupRank {
		out = append(out, "「A1」はプール A の 1 位を表します。プールの順位は、本部がプールの結果を確定してから次の組み合わせに反映します。")
	}
	return out
}

// entrantsPhrase は「各プールの 1 位」「各プールの 1・2 位」のような言い方。出どころが揃わなければ
// 「次のチーム」。
func entrantsPhrase(plan *Plan, g Group) string {
	var ranks []int
	kind := SourceKind("")
	for _, slot := range g.Slots {
		src := slot.Source
		if kind == "" {
			kind = src.Kind
		}
		if src.Kind != kind || (src.Kind != SourceGroupRank && src.Kind != SourceBandRank) {
			return "次のチーム"
		}
		if !slices.Contains(ranks, src.Rank) {
			ranks = append(ranks, src.Rank)
		}
	}
	slices.Sort(ranks)
	parts := make([]string, len(ranks))
	for i, r := range ranks {
		parts[i] = fmt.Sprint(r)
	}
	unit := "プール"
	if g.Stage > 0 && len(plan.StageGroups(g.Stage-1)) > 0 {
		prev := plan.Groups[plan.StageGroups(g.Stage - 1)[0]]
		if prev.Kind == domain.GroupBracket {
			unit = "ブロック"
		}
	}
	return fmt.Sprintf("各%sの %s 位", unit, strings.Join(parts, "・"))
}

// groupPlaces はグループで決まる最終順位の範囲。
func groupPlaces(g Group) (int, int, bool) {
	if g.Kind == domain.GroupPool {
		return g.RankOffset + 1, g.RankOffset + g.Size(), true
	}
	def, err := bracketDefFor(g.Bracket)
	if err != nil {
		return 0, 0, false
	}
	low, high := 0, 0
	for _, m := range def.matches {
		for _, place := range []int{m.winnerPlace, m.loserPlace} {
			if place == 0 {
				continue
			}
			if low == 0 || place < low {
				low = place
			}
			high = max(high, place)
		}
	}
	if low == 0 {
		return 0, 0, false
	}
	return g.RankOffset + low, g.RankOffset + high, true
}

// OperatorName は管理画面(本部)に出すグループの名前。エンジンが作った英語の名前
// ("Qualifying pool A"、"1st placement tournament")を運営の言葉(「予選 A 組」「1〜4 位を決める
// トーナメント」)にする。運営者が付けた名前はそのまま返す。閲覧ページはグループ名のデータを
// 出すので、ここは管理画面だけに使う(グループ名のデータは変えない)。
func OperatorName(g Group) string {
	first, last, placed := groupPlaces(g)
	places := fmt.Sprintf("%d〜%d 位", first, last)
	if first == last {
		places = fmt.Sprintf("%d 位", first)
	}
	switch {
	case g.Name == "Qualifying pool "+g.Letter:
		if g.Final && placed {
			return "総当たり(" + places + "を決める)"
		}
		return "予選 " + g.Letter + " 組"
	case g.Name == "Tournament "+g.Letter:
		return g.Letter + " ブロック"
	case g.Name == "Championship pool" && placed:
		return "決勝リーグ(" + places + "を決める)"
	case strings.HasSuffix(g.Name, "placement tournament") && placed:
		return places + "を決めるトーナメント"
	}
	return g.Name
}

// OperatorLabel は枠の出どころの管理画面の表記(「A 組 1 位」「E4 の負けチーム」「シード 3」)。
func (s Source) OperatorLabel(plan *Plan) string {
	switch s.Kind {
	case SourceSeed:
		return fmt.Sprintf("シード %d", s.Seed)
	case SourceGroupRank:
		return fmt.Sprintf("%s 組 %d 位", plan.Groups[s.Group].Letter, s.Rank)
	case SourceMatch:
		return s.Match.OperatorLabel()
	}
	return s.Label(plan)
}

// OperatorLabel は試合の勝者・敗者の管理画面の表記(「E4 の勝ちチーム」)。閲覧ページは参照サイトの
// "[E4-勝ち]"(Label)のまま。
func (r MatchRef) OperatorLabel() string {
	if r.Outcome == domain.OutcomeWin {
		return r.Code + " の勝ちチーム"
	}
	return r.Code + " の負けチーム"
}
