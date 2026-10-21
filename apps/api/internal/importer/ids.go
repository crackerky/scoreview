package importer

import (
	"fmt"
	"strconv"
	"strings"
)

// RepresentativeTournaments は docs/representative-tournaments.md の 16 大会の ID(表の順)。
// 取込と画面比較の対象で、cmd/importer と cmd/importcheck の -representative が使う。
func RepresentativeTournaments() []int64 {
	return []int64{525, 440, 451, 520, 435, 342, 224, 162, 151, 527, 539, 538, 536, 537, 523, 237}
}

// ParseTournamentIDs は "525,440" のようなカンマ区切りの大会 ID を読む。空の要素は飛ばす。
// 正の整数でない要素があればエラーにする。
func ParseTournamentIDs(list string) ([]int64, error) {
	var ids []int64
	for _, s := range strings.Split(list, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("%q is not a tournament id", s)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
