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
func ComparePlayers(a, b PlayerSortKey) int {
	if c := cmp.Compare(b.Goals+b.Assists, a.Goals+a.Assists); c != 0 {
		return c
	}
	if c := cmp.Compare(b.Goals, a.Goals); c != 0 {
		return c
	}
	return compareNatural(a.Number, b.Number)
}

// compareNatural は数字の文字列を数値順で先に、数字でない文字列をその後に文字列順で並べる。
// 数値が同じ("0" と "00")なら文字列順にして、順序を決定的にする。背番号とコートで共用する。
func compareNatural(a, b string) int {
	an, aErr := strconv.Atoi(a)
	bn, bErr := strconv.Atoi(b)
	aNumeric, bNumeric := aErr == nil, bErr == nil
	switch {
	case aNumeric && !bNumeric:
		return -1
	case !aNumeric && bNumeric:
		return 1
	case aNumeric && bNumeric && an != bn:
		return cmp.Compare(an, bn)
	}
	return cmp.Compare(a, b)
}

// staffRoleRank はスタッフ一覧での役割(日本語名)の順位。参照サイトのチームページ 101 件で、
// 役割は 監督 → コーチ → トレーナー → マネージャ の順に並び(五十音順ではない)、同じ役割の中は
// 選手 ID の昇順だった。知らない役割はその後ろ(参照サイトに例がなく推測)。
func staffRoleRank(role string) int {
	switch role {
	case "監督":
		return 0
	case "コーチ":
		return 1
	case "トレーナー":
		return 2
	case "マネージャ":
		return 3
	}
	return 4
}

// CompareStaffRoles はスタッフ一覧で役割 a が b より先なら負を返す。順位が同じ(知らない役割
// どうし)なら文字列順。
func CompareStaffRoles(a, b string) int {
	if c := cmp.Compare(staffRoleRank(a), staffRoleRank(b)); c != 0 {
		return c
	}
	return cmp.Compare(a, b)
}

// SortPlayers は items を ComparePlayers の順に安定ソートする。
func SortPlayers[T any](items []T, key func(T) PlayerSortKey) {
	slices.SortStableFunc(items, func(a, b T) int { return ComparePlayers(key(a), key(b)) })
}
