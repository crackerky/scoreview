package parse

import (
	"io"

	"github.com/PuerkitoBio/goquery"

	"scoreview/api/internal/importer/site"
)

// TournamentStatus は大会一覧の区分。tournaments.status の値と同じ。
type TournamentStatus string

const (
	StatusEntryOpen   TournamentStatus = "entry_open"
	StatusEntryClosed TournamentStatus = "entry_closed"
	StatusFinished    TournamentStatus = "finished"
)

// listSection は一覧の区分見出しを区分に変える。clone-spec A 3: 区分は 3 種のみで、
// en でも日本語のまま。
func listSection(label string) (TournamentStatus, bool) {
	switch label {
	case "エントリー受付中":
		return StatusEntryOpen, true
	case "エントリー締切り":
		return StatusEntryClosed, true
	case "大会終了":
		return StatusFinished, true
	}
	return "", false
}

// TournamentList は大会一覧 /sctnmt。
type TournamentList struct {
	Entries []ListEntry `json:"entries"`
}

// ListEntry は一覧の 1 大会。Pos は一覧上の並び(1 始まり)。一覧は開催日の降順で、
// ID からは並びを再現できない(clone-spec A 3)。
type ListEntry struct {
	ID     int64            `json:"id"`
	Name   Text             `json:"name"`
	Status TournamentStatus `json:"status"`
	Pos    int              `json:"pos"`
}

// ParseTournamentList は大会一覧を読む。
func ParseTournamentList(r io.Reader, lang site.Lang) (*TournamentList, error) {
	if err := checkLang(lang); err != nil {
		return nil, err
	}
	doc, err := load(r)
	if err != nil {
		return nil, err
	}
	sv, err := scoreView(doc)
	if err != nil {
		return nil, err
	}
	out := &TournamentList{Entries: []ListEntry{}}
	var section TournamentStatus
	var walkErr error
	sv.Find(".TeamListMIDA, .teamimg_rnk").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		if sel.HasClass("TeamListMIDA") {
			label := cellText(sel)
			status, ok := listSection(label)
			if !ok {
				walkErr = unexpected("unknown list section %q", label)
				return false
			}
			section = status
			return true
		}
		if section == "" {
			walkErr = unexpected("tournament before any section")
			return false
		}
		id := linkID(sel, site.PageTournamentTop)
		if id == nil {
			walkErr = unexpected("tournament link without id: %q", cellText(sel))
			return false
		}
		out.Entries = append(out.Entries, ListEntry{
			ID:     *id,
			Name:   textOf(lang, cellText(sel)),
			Status: section,
			Pos:    len(out.Entries) + 1,
		})
		return true
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

// MergeTournamentListEN は日本語版の一覧に英語名を入れる。英語名が未登録の大会は空のまま。
func MergeTournamentListEN(jp, en *TournamentList) (*TournamentList, error) {
	if len(jp.Entries) != len(en.Entries) {
		return nil, mismatch("list entries jp=%d en=%d", len(jp.Entries), len(en.Entries))
	}
	out := &TournamentList{Entries: make([]ListEntry, len(jp.Entries))}
	for i, j := range jp.Entries {
		e := en.Entries[i]
		if j.ID != e.ID {
			return nil, mismatch("list entry %d: id jp=%d en=%d", i, j.ID, e.ID)
		}
		j.Name = mergeText(j.Name, e.Name)
		out.Entries[i] = j
	}
	return out, nil
}
