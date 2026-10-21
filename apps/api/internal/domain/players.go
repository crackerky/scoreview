package domain

import (
	"cmp"
	"slices"
	"strconv"
)

// PlayerSortKey は選手の並び順を決める値。
type PlayerSortKey struct {
	Goals   int
	Assists int
	Number  string
}

// ComparePlayers は選手の表示順で a が先なら負を返す。
// (得点+アシスト)降順 → 得点降順 → 背番号の数値昇順(clone-spec D 1.1 の 7、D 2.4 で機械検証済み)。
// 背番号が数値でなければ数値の後ろに文字列順で並べる(参照サイトに例が無く推測)。
// "0" と "00" のように数値が同じなら文字列順にして、順序を決定的にする。
func ComparePlayers(a, b PlayerSortKey) int {
	if c := cmp.Compare(b.Goals+b.Assists, a.Goals+a.Assists); c != 0 {
		return c
	}
	if c := cmp.Compare(b.Goals, a.Goals); c != 0 {
		return c
	}
	an, aErr := strconv.Atoi(a.Number)
	bn, bErr := strconv.Atoi(b.Number)
	aNumeric, bNumeric := aErr == nil, bErr == nil
	switch {
	case aNumeric && !bNumeric:
		return -1
	case !aNumeric && bNumeric:
		return 1
	case aNumeric && bNumeric && an != bn:
		return cmp.Compare(an, bn)
	}
	return cmp.Compare(a.Number, b.Number)
}

// SortPlayers は items を ComparePlayers の順に安定ソートする。
func SortPlayers[T any](items []T, key func(T) PlayerSortKey) {
	slices.SortStableFunc(items, func(a, b T) int { return ComparePlayers(key(a), key(b)) })
}
