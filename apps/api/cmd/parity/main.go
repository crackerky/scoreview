// Command parity は、取込ツールのキャッシュにある参照サイトのページと、本サイトの同じパスのページを
// 画面に出る内容の単位で突き合わせ、ページ種別ごとの一致率を出す。参照サイトにはアクセスしない。
//
//	go run ./cmd/parity -site http://localhost:3000 -tournament 525,440
//
// 詳細(差分の実例。実在の個人名を含む)は -out の report.md にだけ書き、標準出力には件数だけを出す。
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"scoreview/api/internal/importer/fetch"
	"scoreview/api/internal/parity"
)

// requestTimeout は本サイトの 1 ページの取得の上限。取込済みの最大の大会(282 試合)でも数百 ms で返る。
const requestTimeout = 30 * time.Second

func main() {
	site := flag.String("site", "http://localhost:3000", "本サイトの origin")
	cacheDir := flag.String("cache-dir", "../../.local/import-cache", "取込ツールのキャッシュ")
	outDir := flag.String("out", "../../.local/parity", "詳細レポートの出力先(git 管理外)")
	ids := flag.String("tournament", "", "比べる大会 ID(カンマ区切り)")
	flag.Parse()
	if err := run(*site, *cacheDir, *outDir, *ids); err != nil {
		fmt.Fprintln(os.Stderr, "parity:", err)
		os.Exit(1)
	}
}

type kindSummary struct {
	pages, fullMatch, structureOK, cells, text, classes, links int
}

func run(site, cacheDir, outDir, ids string) error {
	tournamentIDs, err := parseIDs(ids)
	if err != nil {
		return err
	}
	cache, err := fetch.NewCache(cacheDir)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: requestTimeout}
	ctx := context.Background()

	var results []parity.Result
	for _, id := range tournamentIDs {
		paths, err := parity.Discover(cache, id)
		if err != nil {
			return err
		}
		for _, path := range paths {
			ref, err := parity.LoadReference(cache, path)
			if err != nil {
				return err
			}
			ours, status, err := parity.LoadOurs(ctx, client, site, path)
			if err != nil {
				return err
			}
			res := parity.Compare(path, parity.Kind(path), ref, ours)
			if status != http.StatusOK {
				res.StructureOK = false
				res.Diffs = append([]string{fmt.Sprintf("HTTP %d", status)}, res.Diffs...)
			}
			results = append(results, res)
		}
	}
	return report(results, outDir)
}

func report(results []parity.Result, outDir string) error {
	summaries := map[string]*kindSummary{}
	for _, res := range results {
		sum := summaries[res.Kind]
		if sum == nil {
			sum = &kindSummary{}
			summaries[res.Kind] = sum
		}
		sum.pages++
		if res.FullMatch() {
			sum.fullMatch++
		}
		if res.StructureOK {
			sum.structureOK++
		}
		sum.cells += res.CellsTotal
		sum.text += res.CellsText
		sum.classes += res.CellsClasses
		sum.links += res.CellsLink
	}
	kinds := make([]string, 0, len(summaries))
	for kind := range summaries {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)

	var table strings.Builder
	table.WriteString("| 種別 | ページ | 完全一致 | 構造一致 | セル | 文字一致 | class 一致 | リンク一致 |\n")
	table.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, kind := range kinds {
		sum := summaries[kind]
		fmt.Fprintf(&table, "| %s | %d | %d | %d | %d | %s | %s | %s |\n", kind, sum.pages, sum.fullMatch,
			sum.structureOK, sum.cells, pct(sum.text, sum.cells), pct(sum.classes, sum.cells),
			pct(sum.links, sum.cells))
	}
	fmt.Print(table.String())

	var detail strings.Builder
	detail.WriteString("# 参照サイトとの突き合わせ\n\n")
	detail.WriteString(table.String())
	detail.WriteString("\n## 一致しなかったページ\n\n")
	for _, res := range results {
		if res.FullMatch() {
			continue
		}
		fmt.Fprintf(&detail, "### %s\n\n", res.Path)
		for _, diff := range res.Diffs {
			fmt.Fprintf(&detail, "- %s\n", diff)
		}
		detail.WriteString("\n")
	}
	// 0o700 / 0o600: 差分の実例に実在の個人名が含まれるので所有者だけが読めるようにする
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", outDir, err)
	}
	path := filepath.Join(outDir, "report.md")
	if err := os.WriteFile(path, []byte(detail.String()), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Println("detail:", path)
	return nil
}

func pct(n, total int) string {
	if total == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", float64(n)*100/float64(total))
}

func parseIDs(raw string) ([]int64, error) {
	if raw == "" {
		return nil, fmt.Errorf("-tournament is required")
	}
	var ids []int64
	for _, part := range strings.Split(raw, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid tournament id %q: %w", part, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
