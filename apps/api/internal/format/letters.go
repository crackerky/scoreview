package format

import (
	"encoding/json"
	"fmt"
)

// NextLetter は使用済みのグループ文字(試合 CD の先頭の英大文字)のうち最も後ろの次の文字を返す。
// 何も使っていなければ "A"。大会内でグループ文字を通しにするため(R3 5-5)、別のディビジョンの
// 文字を渡して新しいディビジョンの StartLetter を決める。不正な文字は無視する。
func NextLetter(used []string) string {
	next := 0
	for _, letters := range used {
		index, err := letterIndex(letters)
		if err != nil {
			continue
		}
		next = max(next, index+1)
	}
	return letterName(next)
}

// LetterRange は start から count 個のグループ文字を返す。start が不正なら domain.ValidationError。
func LetterRange(start string, count int) ([]string, error) {
	base, err := letterIndex(start)
	if err != nil {
		return nil, err
	}
	out := make([]string, count)
	for i := range out {
		out[i] = letterName(base + i)
	}
	return out, nil
}

// sourceJSON は枠の出どころの保存形(group_slots.source)。group は形の中のグループの序数。
type sourceJSON struct {
	Kind    SourceKind `json:"kind"`
	Seed    int        `json:"seed,omitempty"`
	Group   *int       `json:"group,omitempty"`
	Rank    int        `json:"rank,omitempty"`
	Match   string     `json:"match,omitempty"`
	Outcome string     `json:"outcome,omitempty"`
	Stage   *int       `json:"stage,omitempty"`
	Order   int        `json:"order,omitempty"`
}

// SourceEquals は保存した出どころ(group_slots.source の JSON)が s と同じかどうか。JSON の
// キーの順や空白は問わない(PostgreSQL の jsonb は並べ替えて返す)。読めない JSON は違うとみなす。
func SourceEquals(s Source, stored []byte) bool {
	want, err := EncodeSource(s)
	if err != nil {
		return false
	}
	var a, b sourceJSON
	if json.Unmarshal(want, &a) != nil || json.Unmarshal(stored, &b) != nil {
		return false
	}
	same := func(x, y *int) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return a.Kind == b.Kind && a.Seed == b.Seed && same(a.Group, b.Group) && a.Rank == b.Rank &&
		a.Match == b.Match && a.Outcome == b.Outcome && same(a.Stage, b.Stage) && a.Order == b.Order
}

// EncodeSource は出どころを group_slots.source の JSON にする
// ({"kind":"group_rank","group":0,"rank":1} など)。
func EncodeSource(s Source) ([]byte, error) {
	out := sourceJSON{Kind: s.Kind}
	switch s.Kind {
	case SourceSeed:
		out.Seed = s.Seed
	case SourceGroupRank:
		group := s.Group
		out.Group, out.Rank = &group, s.Rank
	case SourceMatch:
		out.Match, out.Outcome = s.Match.Code, string(s.Match.Outcome)
	case SourceBandRank:
		stage := s.Stage
		out.Stage, out.Rank, out.Order = &stage, s.Rank, s.Order
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("encode source: %w", err)
	}
	return data, nil
}
