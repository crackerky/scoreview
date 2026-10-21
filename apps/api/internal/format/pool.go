package format

import "slices"

// poolRounds は n チームの総当たりのラウンドごとの組み合わせ(枠の位置、1 から)を返す。
//   - 3 チーム: (1-3) → (1-2) → (2-3)(R3 5-4、19/22 プールで一致)
//   - 4 チーム: (1-4, 2-3) → (1-3, 2-4) → (1-2, 3-4)(R3 5-4、31/38 プールで一致)
//   - 5 チーム以上: サークル法。奇数なら休みを各チーム 1 回ずつにし、上位シードどうし(1-2)を
//     含むラウンドを最後に回す(UPA マニュアル 2.3 の原則。R2 1.1)
func poolRounds(n int) [][][2]int {
	switch n {
	case 2:
		return [][][2]int{{{1, 2}}}
	case 3:
		return [][][2]int{{{1, 3}}, {{1, 2}}, {{2, 3}}}
	case 4:
		return [][][2]int{{{1, 4}, {2, 3}}, {{1, 3}, {2, 4}}, {{1, 2}, {3, 4}}}
	}
	return circleRounds(n)
}

func circleRounds(n int) [][][2]int {
	size := n
	if size%2 == 1 {
		size++ // 最後の番号は休み
	}
	others := make([]int, 0, size-1)
	for team := 2; team <= size; team++ {
		others = append(others, team)
	}
	rounds := make([][][2]int, 0, size-1)
	for round := range size - 1 {
		line := append([]int{1}, rotate(others, round)...)
		var pairs [][2]int
		for i := range size / 2 {
			a, b := line[i], line[size-1-i]
			if a > n || b > n {
				continue
			}
			pairs = append(pairs, [2]int{min(a, b), max(a, b)})
		}
		slices.SortFunc(pairs, comparePair)
		rounds = append(rounds, pairs)
	}
	// 1-2 を含むラウンドを最後へ
	top := slices.IndexFunc(rounds, func(pairs [][2]int) bool {
		return slices.Contains(pairs, [2]int{1, 2})
	})
	last := rounds[top]
	rounds = append(rounds[:top], rounds[top+1:]...)
	return append(rounds, last)
}

func rotate(values []int, by int) []int {
	out := make([]int, len(values))
	for i := range values {
		out[i] = values[(i+by)%len(values)]
	}
	return out
}

func comparePair(a, b [2]int) int {
	if a[0] != b[0] {
		return a[0] - b[0]
	}
	return a[1] - b[1]
}

// pairNumber はプール内の試合の連番。組み合わせの行・列の順 (1-2)=1、(1-3)=2、…(R3 5-5、
// 37/38 の 4 チームプールで一致)で、時刻順ではない。
func pairNumber(n int, pair [2]int) int {
	number := 0
	for a := 1; a < pair[0]; a++ {
		number += n - a
	}
	return number + pair[1] - pair[0]
}

// snakeSeeds はシード 1..teams を groups 個へ蛇行(1,2,3,4,4,3,2,1,…)で配る(R2 3.1、USAU と JFDA の
// 標準)。戻り値の [g] はグループ g のシード番号を良い順に並べたもの。大きさの差は 1 以内になる。
func snakeSeeds(teams, groups int) [][]int {
	out := make([][]int, groups)
	for seed := 1; seed <= teams; seed++ {
		row, col := (seed-1)/groups, (seed-1)%groups
		if row%2 == 1 {
			col = groups - 1 - col
		}
		out[col] = append(out[col], seed)
	}
	return out
}
