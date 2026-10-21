package fetch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// repoMarker はリポジトリ直下にあるファイル。既定のキャッシュ置き場を決めるために親をたどって探す。
const repoMarker = "pnpm-workspace.yaml"

// DefaultCacheSubdir はリポジトリ直下からのキャッシュの場所(CLAUDE.md gotcha)。
const DefaultCacheSubdir = ".local/import-cache"

// DefaultCacheDir は既定のキャッシュの場所を返す。カレントから親をたどってリポジトリ直下を探し、
// そこが git worktree なら本体のリポジトリの .local を使う(worktree ごとに取り直さないため)。
func DefaultCacheDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, repoMarker)); err == nil {
			return filepath.Join(mainWorktree(dir), DefaultCacheSubdir), nil
		}
		if dir == filepath.Dir(dir) {
			return "", fmt.Errorf("%s not found above %s; pass the cache dir explicitly",
				repoMarker, wd)
		}
	}
}

// mainWorktree は root が git worktree なら本体のリポジトリの場所を返す。worktree の .git は
// "gitdir: {本体}/.git/worktrees/{名前}" と書かれたファイルである。読めなければ root をそのまま返す。
func mainWorktree(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".git")) // #nosec G304 -- リポジトリ直下の .git だけを読む
	if err != nil {
		return root
	}
	gitdir, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return root
	}
	worktrees := filepath.Dir(filepath.Clean(strings.TrimSpace(gitdir)))
	if filepath.Base(worktrees) != "worktrees" || filepath.Base(filepath.Dir(worktrees)) != ".git" {
		return root
	}
	return filepath.Dir(filepath.Dir(worktrees))
}
