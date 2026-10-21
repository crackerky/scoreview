package parity

import "fmt"

// maxDiffsPerPage は 1 ページあたりに記録する差分の上限。同じ原因の差分が表の全行に出るので、
// 先頭の数件で原因を特定できる。
const maxDiffsPerPage = 8

// Result は 1 ページの比較結果。Cells* は本文の表のセル単位の一致数。
type Result struct {
	Path         string
	Kind         string
	CellsTotal   int
	CellsText    int
	CellsClasses int
	CellsLink    int
	Diffs        []string
	// StructureOK は表の数、各表の行数と列数、見出し、ナビ、メニューがすべて一致したか
	StructureOK bool
}

// FullMatch は構造とセルの文字、クラス、リンクがすべて一致したかを返す。
func (r Result) FullMatch() bool {
	return r.StructureOK && r.CellsText == r.CellsTotal && r.CellsClasses == r.CellsTotal &&
		r.CellsLink == r.CellsTotal
}

// Compare は参照サイトのページ ref と本サイトのページ ours を比べる。
func Compare(path, kind string, ref, ours Page) Result {
	res := Result{Path: path, Kind: kind, StructureOK: true}
	note := func(format string, args ...any) {
		if len(res.Diffs) < maxDiffsPerPage {
			res.Diffs = append(res.Diffs, fmt.Sprintf(format, args...))
		}
	}

	if ref.Title != ours.Title {
		res.StructureOK = false
		note("title: ref=%q ours=%q", ref.Title, ours.Title)
	}
	compareLinks("navi", ref.Navi, ours.Navi, &res, note)
	compareLinks("menu", ref.Menu, ours.Menu, &res, note)
	if !equalStrings(ref.Headings, ours.Headings) {
		res.StructureOK = false
		note("headings: ref=%q ours=%q", ref.Headings, ours.Headings)
	}

	if len(ref.Tables) != len(ours.Tables) {
		res.StructureOK = false
		note("tables: ref=%d ours=%d", len(ref.Tables), len(ours.Tables))
	}
	for ti := range min(len(ref.Tables), len(ours.Tables)) {
		refRows, ourRows := ref.Tables[ti].Rows, ours.Tables[ti].Rows
		if len(refRows) != len(ourRows) {
			res.StructureOK = false
			note("table %d rows: ref=%d ours=%d", ti, len(refRows), len(ourRows))
		}
		for ri := range refRows {
			refRow := refRows[ri]
			res.CellsTotal += len(refRow)
			if ri >= len(ourRows) {
				continue
			}
			ourRow := ourRows[ri]
			if len(refRow) != len(ourRow) {
				res.StructureOK = false
				note("table %d row %d cols: ref=%d ours=%d", ti, ri, len(refRow), len(ourRow))
			}
			for ci := range min(len(refRow), len(ourRow)) {
				refCell, ourCell := refRow[ci], ourRow[ci]
				if refCell.Text == ourCell.Text {
					res.CellsText++
				} else {
					note("table %d [%d,%d] text: ref=%q ours=%q", ti, ri, ci, refCell.Text, ourCell.Text)
				}
				if refCell.Classes == ourCell.Classes {
					res.CellsClasses++
				} else {
					note("table %d [%d,%d] class: ref=%q ours=%q", ti, ri, ci, refCell.Classes, ourCell.Classes)
				}
				if refCell.Href == ourCell.Href {
					res.CellsLink++
				} else {
					note("table %d [%d,%d] link: ref=%q ours=%q", ti, ri, ci, refCell.Href, ourCell.Href)
				}
			}
		}
	}
	return res
}

func compareLinks(name string, ref, ours []Link, res *Result, note func(string, ...any)) {
	if len(ref) != len(ours) {
		res.StructureOK = false
		note("%s: ref=%d items ours=%d items", name, len(ref), len(ours))
	}
	for i := range min(len(ref), len(ours)) {
		if ref[i] != ours[i] {
			res.StructureOK = false
			note("%s[%d]: ref=%+v ours=%+v", name, i, ref[i], ours[i])
			return
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
