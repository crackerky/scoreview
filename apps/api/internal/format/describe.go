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
	out := []string{"組の順位は、" + criterionPhrase(rule.Criteria[0]) + "を上位とします。"}
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
// (「各組の 1 位(A組1位、B組1位、C組1位、D組1位)は 1〜4 位を決めるトーナメントに進みます。
// 1 回戦は A組1位 対 B組1位、C組1位 対 D組1位」)。最初の段のグループは対象にしない。
func DescribeAdvancement(plan *Plan) []string {
	var out []string
	usesGroupRank := false
	for _, g := range plan.Groups {
		if g.Stage == 0 {
			continue
		}
		var labels []string
		for _, slot := range g.Slots {
			labels = append(labels, guideLabel(plan, slot.Source))
			if slot.Source.Kind == SourceGroupRank {
				usesGroupRank = true
			}
		}
		kind := "トーナメント"
		if g.Kind == domain.GroupPool {
			kind = "総当たり"
		}
		// 行き先は運営の言葉の名前で書く(エンジンの英語の名前は参加チームに見せない)
		var text string
		if first, last, ok := groupPlaces(g); ok && g.Final {
			places := fmt.Sprintf("%d〜%d 位", first, last)
			if first == last {
				places = fmt.Sprintf("%d 位", first)
			}
			text = fmt.Sprintf("%s(%s)は %sを決める%sに進みます。", entrantsPhrase(plan, g),
				strings.Join(labels, "、"), places, kind)
		} else {
			text = fmt.Sprintf("%s(%s)は%sに進みます。", entrantsPhrase(plan, g),
				strings.Join(labels, "、"), OperatorName(g))
		}
		if g.Kind == domain.GroupBracket {
			var pairs []string
			for _, m := range plan.GroupMatches(g.Index) {
				if m.Left.From == nil && m.Right.From == nil {
					pairs = append(pairs, fmt.Sprintf("%s 対 %s",
						guideLabel(plan, g.Slots[m.Left.Slot-1].Source), guideLabel(plan, g.Slots[m.Right.Slot-1].Source)))
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
		out = append(out, "組の順位は、本部が組の結果を確定してから次の組み合わせに反映します。")
	}
	return out
}

// guideLabel は案内文での出どころの書き方。試合の勝者・敗者は参照サイトの記号([E4-負け])ではなく
// 「E4 の負けチーム」と書く(運営者レビュー M1)。組の順位は Source.Label の「A組1位」。
func guideLabel(plan *Plan, src Source) string {
	if src.Kind == SourceMatch {
		if src.Match.Outcome == domain.OutcomeWin {
			return src.Match.Code + " の勝者"
		}
		return src.Match.Code + " の負けチーム"
	}
	return src.Label(plan)
}

// entrantsPhrase は「各組の 1 位」「各組の 1・2 位」のような言い方。出どころが揃わなければ
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
	// 同じ順位の一部だけが進む(ブロックの全体の並びを大きさで分けた Cuts)なら「各ブロックの n 位」とは
	// 言えないので、並べた出どころを見てもらう
	if g.Stage > 0 && kind == SourceGroupRank {
		perRank := map[int]int{}
		for _, slot := range g.Slots {
			perRank[slot.Source.Rank]++
		}
		for _, rank := range ranks {
			have := 0
			for _, prev := range plan.StageGroups(g.Stage - 1) {
				if plan.Groups[prev].Size() >= rank {
					have++
				}
			}
			if perRank[rank] != have {
				return "次のチーム"
			}
		}
	}
	parts := make([]string, len(ranks))
	for i, r := range ranks {
		parts[i] = fmt.Sprint(r)
	}
	unit := "組"
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
	// 最下位が同順位なら、その人数ぶん範囲を延ばす(16 チームの全順位型の最後は 15 位が 2 チーム
	// なので「1〜16 位」、1 回戦のみの 8 チームの最後は 7 位が 2 チームなので「1〜8 位」)
	tied := 0
	for _, m := range def.matches {
		for _, place := range []int{m.winnerPlace, m.loserPlace} {
			if place == high {
				tied++
			}
		}
	}
	return g.RankOffset + low, g.RankOffset + high + max(tied-1, 0), true
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
	case strings.HasSuffix(g.Name, "placement pool") && placed:
		return places + "を決める総当たり"
	}
	return g.Name
}
