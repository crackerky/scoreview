// Package levent は、Lイベント(運営会社が今使っている公式アカウントの拡張機能)の申込の書き出し
// (CSV)を読み、前の大会の申込(entry.PastTournament)にする(docs/line-integration.md「Lイベント
// からの移行」)。
//
// Lイベントの書き出しは、大会ごとに「代表者のみ」(チームの申込 1 件が 1 行)と「選手全員」(メンバー
// 1 人が 1 行)に分かれ、どちらの行にも申し込んだ人の公式アカウントから見たユーザー ID(UID)が付く。
// 列は大会の質問によって違うので、見出しの名前で読む。生年月日、性別、LINE の表示名は読まない
// (このサイトが扱う情報に入れていない。/privacy)。
package levent

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/unicode/norm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

// submittedLayout は「申し込み日時」の書き方(例 2026-06-04 11:42:16 UTC)。
const submittedLayout = "2006-01-02 15:04:05 MST"

// Options は読み方。Aliases はチーム名の言い換え(メンバーの書き出しのチーム名 → 代表の書き出しの
// チーム名)で、表記の揺れの範囲(空白、大文字小文字、括弧、語の順)を超える違いだけを書く。Random は
// 招待コードの乱数の元。
type Options struct {
	Name      string
	StartDate *time.Time
	Aliases   map[string]string
	Random    io.Reader
}

// Report は読んだ結果のうち、確かめてほしいもの(チーム名と人数だけで、個人の情報は入れない)。
type Report struct {
	// Merged は同じ部門に同じチームの申込が 2 件以上あったチーム(最初の申込に寄せた)
	Merged []string
	// Unmatched は申込が見つからないメンバーのチーム名と部門と人数
	Unmatched []Unmatched
	// Duplicates は同じ人が同じチームに 2 回登録していた数(新しい登録を残した)
	Duplicates int
	// DroppedNumbers は長すぎて背番号にしなかった数
	DroppedNumbers int
}

// Unmatched は申込が見つからないメンバーのまとまり。
type Unmatched struct {
	Team     string
	Division string
	Count    int
}

// row は書き出しの 1 行のうち、使う列。
type row struct {
	name      string
	uid       string
	submitted time.Time
	team      string
	divisions []string
	email     string
	number    string
}

// Read は代表の書き出しとメンバーの書き出し(いくつでも。追加の募集の書き出しなど)を読む。
// メンバーのチームが代表の申込に見つからなければ、Report.Unmatched に入れてそのメンバーを除く。
func Read(reps io.Reader, members []io.Reader, opts Options) (entry.PastTournament, Report, error) {
	var report Report
	repRows, err := readRows(reps)
	if err != nil {
		return entry.PastTournament{}, report, fmt.Errorf("read the representatives: %w", err)
	}
	past := entry.PastTournament{Name: strings.TrimSpace(opts.Name), StartDate: opts.StartDate}
	slices.SortStableFunc(repRows, func(a, b row) int { return a.submitted.Compare(b.submitted) })
	index := map[teamAt]int{}
	for _, rep := range repRows {
		division := divisionOf(&past, firstOr(rep.divisions, ""))
		at := teamAt{division: division, key: teamKey(rep.team)}
		if _, ok := index[at]; ok {
			report.Merged = append(report.Merged, strings.TrimSpace(rep.team))
			continue
		}
		code, err := entry.NewInviteCode(opts.Random)
		if err != nil {
			return entry.PastTournament{}, report, err
		}
		index[at] = len(past.Entries)
		past.Entries = append(past.Entries, entry.PastEntry{Division: division,
			FullName: strings.TrimSpace(rep.team), RepName: rep.name, RepEmail: rep.email, RepLineID: rep.uid,
			SubmittedAt: rep.submitted, InviteCode: code})
	}
	aliases := map[string]string{}
	for from, to := range opts.Aliases {
		aliases[teamKey(from)] = teamKey(to)
	}
	var memberRows []row
	for i, file := range members {
		rows, err := readRows(file)
		if err != nil {
			return entry.PastTournament{}, report, fmt.Errorf("read the members (file %d): %w", i+1, err)
		}
		memberRows = append(memberRows, rows...)
	}
	slices.SortStableFunc(memberRows, func(a, b row) int { return a.submitted.Compare(b.submitted) })
	unmatched := map[Unmatched]int{}
	for _, member := range memberRows {
		key := teamKey(member.team)
		if alias, ok := aliases[key]; ok {
			key = alias
		}
		at, ok := findEntry(past, index, key, memberDivisions(member))
		if !ok {
			unmatched[Unmatched{Team: strings.TrimSpace(member.team), Division: strings.Join(member.divisions, ",")}]++
			continue
		}
		addMember(&past.Entries[at], member, &report)
	}
	for group, count := range unmatched {
		group.Count = count
		report.Unmatched = append(report.Unmatched, group)
	}
	slices.SortFunc(report.Unmatched, func(a, b Unmatched) int { return strings.Compare(a.Team, b.Team) })
	return past, report, nil
}

// teamAt は部門の中のチーム(部門の添字と、表記を揃えたチーム名)。
type teamAt struct {
	division int
	key      string
}

func firstOr(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

// divisionOf は部門の名前の添字を返す(無ければ足す)。名前が空なら「オープン」にする。
func divisionOf(past *entry.PastTournament, name string) int {
	if name == "" {
		name = "オープン"
	}
	for i, division := range past.Divisions {
		if division.Name == name {
			return i
		}
	}
	past.Divisions = append(past.Divisions, entry.PastDivision{Name: name, Kind: divisionKind(name)})
	return len(past.Divisions) - 1
}

// divisionKind は部門の名前から種類を決める。オープン(男女を問わない部門)は色なし(本番の大会の
// 作り方に合わせる)。
func divisionKind(name string) domain.DivisionKind {
	switch {
	case strings.Contains(name, "ウィメン"), strings.Contains(name, "女子"):
		return domain.DivisionWomen
	case strings.Contains(name, "ミックス"):
		return domain.DivisionMixed
	case name == "メン", strings.Contains(name, "男子"):
		return domain.DivisionMen
	default:
		return domain.DivisionNone
	}
}

// memberDivisions はメンバーの行の部門の候補。チーム名の後ろの「(オープン)」が先、無ければ部門の列。
func memberDivisions(member row) []string {
	if _, division := splitDivision(member.team); division != "" {
		return []string{division}
	}
	return member.divisions
}

// findEntry はメンバーの申込を探す。部門の候補の順に探し、部門が分からなければ(書き出しに部門の
// 列もチーム名の部門も無い)、チーム名が合う申込がちょうど 1 つのときだけ選ぶ。
func findEntry(past entry.PastTournament, index map[teamAt]int, key string, divisions []string) (int, bool) {
	for _, name := range divisions {
		for i, division := range past.Divisions {
			if division.Name != name {
				continue
			}
			if at, ok := index[teamAt{division: i, key: key}]; ok {
				return at, true
			}
		}
	}
	if len(divisions) > 0 {
		return 0, false
	}
	found, at := 0, 0
	for candidate, i := range index {
		if candidate.key == key {
			found, at = found+1, i
		}
	}
	return at, found == 1
}

// addMember は申込にメンバーを足す。同じ人がもう登録していれば、新しい登録で書き換える(行は時刻の
// 順に来る)。
func addMember(target *entry.PastEntry, member row, report *Report) {
	number := member.number
	if utf8.RuneCountInString(number) > entry.MaxNumberText {
		number = ""
		report.DroppedNumbers++
	}
	added := entry.PastMember{Name: member.name, Number: number, LineID: member.uid, SubmittedAt: member.submitted}
	for i, current := range target.Members {
		if current.LineID == member.uid {
			target.Members[i] = added
			report.Duplicates++
			return
		}
	}
	target.Members = append(target.Members, added)
}

// readRows は書き出しを読む。文字コードは UTF-8(BOM 付きを含む)か Shift_JIS(Windows-31J)。
func readRows(file io.Reader) ([]row, error) {
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) // UTF-8 の BOM
	if !utf8.Valid(raw) {
		decoded, err := japanese.ShiftJIS.NewDecoder().Bytes(raw)
		if err != nil {
			return nil, fmt.Errorf("decode Shift_JIS csv: %w", err)
		}
		raw = decoded
	}
	records, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("the csv has no header")
	}
	columns, err := findColumns(records[0])
	if err != nil {
		return nil, err
	}
	rows := make([]row, 0, len(records)-1)
	for line, record := range records[1:] {
		parsed, err := columns.read(record)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line+2, err)
		}
		rows = append(rows, parsed)
	}
	return rows, nil
}

// columns は使う列の位置(無い列は -1)。
type columns struct {
	name, uid, submitted, team, division, email, number int
}

func findColumns(header []string) (columns, error) {
	found := columns{name: -1, uid: -1, submitted: -1, team: -1, division: -1, email: -1, number: -1}
	for i, title := range header {
		title = strings.TrimSpace(title)
		switch {
		case title == "名前":
			found.name = i
		case title == "UID":
			found.uid = i
		case title == "申し込み日時":
			found.submitted = i
		case strings.HasPrefix(title, "チーム名"):
			found.team = i
		case title == "部門":
			found.division = i
		case strings.Contains(title, "メールアドレス"):
			found.email = i
		case strings.HasPrefix(title, "背番号"):
			found.number = i
		}
	}
	for title, at := range map[string]int{"名前": found.name, "UID": found.uid, "申し込み日時": found.submitted,
		"チーム名": found.team} {
		if at < 0 {
			return found, fmt.Errorf("the csv has no %s column", title)
		}
	}
	return found, nil
}

func (c columns) read(record []string) (row, error) {
	cell := func(at int) string {
		if at < 0 || at >= len(record) {
			return ""
		}
		return oneLine(record[at])
	}
	submitted, err := time.Parse(submittedLayout, cell(c.submitted))
	if err != nil {
		return row{}, fmt.Errorf("申し込み日時 %q: %w", cell(c.submitted), err)
	}
	parsed := row{name: cell(c.name), uid: cell(c.uid), submitted: submitted.UTC(), team: cell(c.team),
		email: cell(c.email), number: strings.TrimSpace(norm.NFKC.String(cell(c.number)))}
	for division := range strings.SplitSeq(cell(c.division), ",") {
		if name := strings.TrimSuffix(strings.TrimSpace(norm.NFKC.String(division)), "部門"); name != "" {
			parsed.divisions = append(parsed.divisions, name)
		}
	}
	return parsed, nil
}

// oneLine は前後の空白を除き、改行などの制御文字を空白にする(名前とチーム名は 1 行で持つ)。
func oneLine(text string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text))
}

// knownDivisions はチーム名の後ろに付く部門の名前(メンバーの書き出しは「チーム名(オープン)」と書く)。
func knownDivisions() []string {
	return []string{"オープン", "ウィメン", "ミックス", "メン"}
}

// splitDivision はチーム名の後ろの「(部門)」を分ける。無ければ部門は空。
func splitDivision(team string) (string, string) {
	text := strings.TrimSpace(norm.NFKC.String(team))
	if !strings.HasSuffix(text, ")") {
		return text, ""
	}
	open := strings.LastIndex(text, "(")
	if open < 0 {
		return text, ""
	}
	inner := strings.TrimSpace(text[open+1 : len(text)-1])
	if !slices.Contains(knownDivisions(), inner) {
		return text, ""
	}
	return strings.TrimSpace(text[:open]), inner
}

// teamKey は表記の揺れを除いたチーム名。部門の括弧を除き、NFKC で全角と半角を揃え、英字を小文字に
// し、文字と数字の並びだけを語にする(空白、括弧、記号、引用符の違いを無くす)。語は英数字とそれ以外の
// 境目でも分け、並べ替えてつなぐ(「MISTRAL (千葉大学)」と「千葉大学MISTRAL」を同じにする)。
func teamKey(team string) string {
	name, _ := splitDivision(team)
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = current[:0]
		}
	}
	for _, r := range strings.ToLower(name) {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			flush()
			continue
		}
		if len(current) > 0 && isASCII(current[len(current)-1]) != isASCII(r) {
			flush()
		}
		current = append(current, r)
	}
	flush()
	slices.Sort(words)
	return strings.Join(words, " ")
}

func isASCII(r rune) bool {
	return r < utf8.RuneSelf
}
