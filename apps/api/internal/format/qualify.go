package format

import (
	"cmp"
	"fmt"
	"slices"

	"scoreview/api/internal/domain"
)

// MaxRemainingGames は勝ち上がり条件で全列挙する残り試合数の上限。3^8 = 6561 通りで、
// 4 チームのプールの全 6 試合(729 通り)を余裕をもって覆う。
const MaxRemainingGames = 8

// Chance はある順位になりうるか。
type Chance string

// Chance の値。
const (
	ChanceCertain    Chance = "certain"
	ChancePossible   Chance = "possible"
	ChanceImpossible Chance = "impossible"
)

// Condition は「次の試合にこの結果なら、この順位以上が確定」という条件。
type Condition struct {
	Match string
	// AllowDraw が真なら「引き分け以上で」、偽なら「勝てば」
	AllowDraw bool
	AtLeast   int
	Text      string
}

// Outlook は 1 チームの見通し。Chances[r-1] が r 位になりうるか。
type Outlook struct {
	TeamID     int64
	Chances    []Chance
	Best       int
	Worst      int
	Conditions []Condition
}

// Qualification は勝ち上がり条件(リーグ戦のグラフ)の計算結果。
type Qualification struct {
	Remaining []string
	Scenarios int
	Teams     []Outlook
}

// Qualify はプールの残り試合の勝ち・負け・引き分けを全列挙し、各チームが各順位になりうるかと、
// 次の試合の結果による簡単な条件を返す。得失点差と得点は場合分けせず、rule の先頭からスコアを
// 見ない基準(勝ち点、勝利数、直接対決の勝敗、棄権数)までで順位をつけ、そこで並んだチームは
// 並びの中のどの順位にもなりうるとみなす(簡略化)。残りが MaxRemainingGames を超えたらエラー。
func Qualify(teams []int64, games []Game, rule Rule) (*Qualification, error) {
	var remaining []int
	for i, g := range games {
		if state, _ := g.Result.decide(); state == resultPending || state == resultIncomplete {
			remaining = append(remaining, i)
		}
	}
	if len(remaining) > MaxRemainingGames {
		return nil, domain.Invalid("games", fmt.Sprintf("%d remaining games exceed %d",
			len(remaining), MaxRemainingGames))
	}
	prefix := Rule{Name: rule.Name, Mode: rule.Mode}
	for _, c := range rule.Criteria {
		if c.Kind.scoreDependent() {
			break
		}
		prefix.Criteria = append(prefix.Criteria, c)
	}

	out := &Qualification{}
	for _, i := range remaining {
		out.Remaining = append(out.Remaining, games[i].Code)
	}
	next := map[int64]int{} // チームの次の試合(games の添字)。試合番号ではなくラウンド(時刻)の順
	byRound := slices.Clone(remaining)
	slices.SortStableFunc(byRound, func(a, b int) int { return cmp.Compare(games[a].Round, games[b].Round) })
	for _, i := range byRound {
		for _, team := range []int64{games[i].Left, games[i].Right} {
			if _, ok := next[team]; !ok {
				next[team] = i
			}
		}
	}

	n := len(teams)
	possible := map[int64][]bool{}
	certain := map[int64]int{} // 全シナリオで同じ 1 つの順位なら その順位、食い違えば -1
	// worstAfter[team][outcome] は次の試合の結果ごとの最悪の順位(0 勝ち、1 引き分け、2 負け)
	worstAfter := map[int64]*[3]int{}
	for _, team := range teams {
		possible[team] = make([]bool, n)
		worstAfter[team] = &[3]int{}
	}

	scenario := make([]Game, len(games))
	total := 1
	for range remaining {
		total *= 3
	}
	for code := range total {
		copy(scenario, games)
		outcomes := make([]int, len(games)) // 左から見た 0 勝ち、1 引き分け、2 負け
		c := code
		for _, i := range remaining {
			outcomes[i] = c % 3
			c /= 3
			scenario[i].Result = syntheticResult(outcomes[i])
		}
		ranking := RankPool(RankInput{Teams: teams, Games: scenario, Rule: prefix})
		tierSize := map[int]int{}
		for _, row := range ranking.Rows {
			tierSize[row.Rank]++
		}
		for _, row := range ranking.Rows {
			best, worst := row.Rank, row.Rank+tierSize[row.Rank]-1
			for r := best; r <= worst; r++ {
				possible[row.TeamID][r-1] = true
			}
			switch prev, seen := certain[row.TeamID]; {
			case best != worst:
				certain[row.TeamID] = -1
			case !seen:
				certain[row.TeamID] = best
			case prev != best:
				certain[row.TeamID] = -1
			}
			if i, ok := next[row.TeamID]; ok {
				o := outcomes[i]
				if games[i].Right == row.TeamID {
					o = 2 - o
				}
				worstAfter[row.TeamID][o] = max(worstAfter[row.TeamID][o], worst)
			}
		}
	}
	out.Scenarios = total

	for _, team := range teams {
		look := Outlook{TeamID: team, Chances: make([]Chance, n)}
		for r := range n {
			switch {
			case certain[team] == r+1:
				look.Chances[r] = ChanceCertain
			case possible[team][r]:
				look.Chances[r] = ChancePossible
			default:
				look.Chances[r] = ChanceImpossible
			}
			if possible[team][r] {
				if look.Best == 0 {
					look.Best = r + 1
				}
				look.Worst = r + 1
			}
		}
		look.Conditions = conditions(team, look, next, games, worstAfter[team])
		out.Teams = append(out.Teams, look)
	}
	return out, nil
}

func syntheticResult(outcome int) *Result {
	left, right := 1, 0
	switch outcome {
	case 1:
		left, right = 0, 0
	case 2:
		left, right = 0, 1
	}
	return &Result{Status: domain.MatchFinished, Type: ResultNormal, LeftScore: &left, RightScore: &right}
}

func conditions(team int64, look Outlook, next map[int64]int, games []Game, worst *[3]int) []Condition {
	if look.Best == look.Worst {
		return []Condition{{AtLeast: look.Best, Text: fmt.Sprintf("%d位確定", look.Best)}}
	}
	i, ok := next[team]
	if !ok {
		return nil
	}
	code := games[i].Code
	win, draw := worst[0], worst[1]
	if win == 0 || win >= look.Worst {
		return nil // 勝っても確定する順位が変わらない
	}
	cond := Condition{Match: code, AtLeast: win}
	verb := "に勝てば"
	if draw != 0 && draw == win {
		cond.AllowDraw = true
		verb = "で引き分け以上なら"
	}
	if win == 1 {
		cond.Text = fmt.Sprintf("次の%s%s1位確定", code, verb)
	} else {
		cond.Text = fmt.Sprintf("次の%s%s%d位以上が確定", code, verb, win)
	}
	return []Condition{cond}
}
