// Command importlevent は、Lイベントの申込の書き出し(CSV)を前の大会として取り込む
// (docs/line-integration.md「Lイベントからの移行」)。既定では読むだけで、取り込む中身(チーム名と
// 人数)と、確かめてほしいこと(重なった申込、申込の見つからないメンバー)を出す。-apply を付けたとき
// だけ DATABASE_URL の DB に書く。個人の名前と連絡先は画面に出さない。
//
//	go run ./cmd/importlevent -name 若葉杯2026 -date 2026-05-31 \
//	  -reps 代表者.csv -members 選手全員.csv -members 追加.csv \
//	  -alias '文教大学AlRS=文教大学AIRS' [-apply]
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"scoreview/api/internal/entry"
	"scoreview/api/internal/levent"
	"scoreview/api/internal/store"
)

// list は何度でも書けるフラグ。
type list []string

func (l *list) String() string { return strings.Join(*l, ", ") }

func (l *list) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "importlevent:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("importlevent", flag.ContinueOnError)
	name := flags.String("name", "", "大会の名前")
	date := flags.String("date", "", "大会の日(YYYY-MM-DD。分からなければ空)")
	reps := flags.String("reps", "", "代表者のみの書き出し")
	apply := flags.Bool("apply", false, "DB に書く(付けなければ読むだけ)")
	skip := flags.Bool("skip-unmatched", false, "申込の見つからないメンバーを除いて取り込む")
	var members, aliases list
	flags.Var(&members, "members", "選手全員の書き出し(何度でも)")
	flags.Var(&aliases, "alias", "チーム名の言い換え「メンバーの書き出しの名前=代表の書き出しの名前」(何度でも)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	opts := levent.Options{Name: *name, Aliases: map[string]string{}, Random: rand.Reader}
	if *date != "" {
		day, err := time.Parse(time.DateOnly, *date)
		if err != nil {
			return fmt.Errorf("-date: %w", err)
		}
		opts.StartDate = &day
	}
	for _, alias := range aliases {
		from, to, ok := strings.Cut(alias, "=")
		if !ok {
			return fmt.Errorf("-alias %q: write it as 「メンバーの書き出しの名前=代表の書き出しの名前」", alias)
		}
		opts.Aliases[from] = to
	}
	past, report, err := read(*reps, members, opts)
	if err != nil {
		return err
	}
	if err := entry.CheckPast(past); err != nil {
		return err
	}
	describe(out, past, report)
	if len(report.Unmatched) > 0 && !*skip {
		return errors.New("some members have no team entry; add -alias, or -skip-unmatched to leave them out")
	}
	if !*apply {
		_, _ = fmt.Fprintln(out, "読むだけです。取り込むときは -apply を付けてください。") // 書けなくても結果は同じ
		return nil
	}
	return write(out, past)
}

func read(repsPath string, memberPaths []string, opts levent.Options) (entry.PastTournament, levent.Report, error) {
	reps, err := os.Open(repsPath) // #nosec G304 -- 運営が手元で指定した書き出しを読む
	if err != nil {
		return entry.PastTournament{}, levent.Report{}, fmt.Errorf("open -reps: %w", err)
	}
	defer func() { _ = reps.Close() }() // 読み終えたファイルを閉じる失敗は、結果に影響しない
	files := make([]io.Reader, 0, len(memberPaths))
	for _, path := range memberPaths {
		file, err := os.Open(path) // #nosec G304 -- 運営が手元で指定した書き出しを読む
		if err != nil {
			return entry.PastTournament{}, levent.Report{}, fmt.Errorf("open -members: %w", err)
		}
		defer func() { _ = file.Close() }() // 同上
		files = append(files, file)
	}
	return levent.Read(reps, files, opts)
}

// describe は取り込む中身を、チーム名と人数で出す。
func describe(out io.Writer, past entry.PastTournament, report levent.Report) {
	lines := []string{fmt.Sprintf("大会: %s", past.Name)}
	people := map[string]bool{}
	for i, division := range past.Divisions {
		lines = append(lines, fmt.Sprintf("部門 %s(%s)", division.Name, division.Kind))
		for _, pastEntry := range past.Entries {
			if pastEntry.Division != i {
				continue
			}
			people[pastEntry.RepLineID] = true
			registered := 0
			for _, member := range pastEntry.Members {
				if member.LineID != "" {
					people[member.LineID] = true
					registered++
				}
			}
			lines = append(lines, fmt.Sprintf("  %s メンバー %d 人", pastEntry.FullName, registered))
		}
	}
	lines = append(lines, fmt.Sprintf("申込 %d 件、LINE の参加者 %d 人", len(past.Entries), len(people)))
	for _, team := range report.Merged {
		lines = append(lines, "同じ部門に 2 件目の申込があったので、最初の申込に寄せた: "+team)
	}
	for _, group := range report.Unmatched {
		lines = append(lines, fmt.Sprintf("申込の見つからないメンバー: %s(%s)%d 人", group.Team, group.Division,
			group.Count))
	}
	if report.Duplicates > 0 {
		lines = append(lines, fmt.Sprintf("同じチームに 2 回登録した人 %d 人(新しい登録を残した)", report.Duplicates))
	}
	if report.DroppedNumbers > 0 {
		lines = append(lines, fmt.Sprintf("長すぎて背番号にしなかった数 %d", report.DroppedNumbers))
	}
	_, _ = fmt.Fprintln(out, strings.Join(lines, "\n")) // 書けなくても取り込みの結果は変わらない
}

func write(out io.Writer, past entry.PastTournament) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	db, err := store.Open(dsn)
	if err != nil {
		return err
	}
	id, err := store.NewEntryStore(db).ImportPast(context.Background(), past)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "取り込みました。大会 ID %d(終わった大会、一覧に載せない)\n", id) // 同上
	return nil
}
