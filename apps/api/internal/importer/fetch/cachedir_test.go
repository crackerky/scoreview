package fetch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainWorktree(t *testing.T) {
	repo := t.TempDir()
	worktree := filepath.Join(repo, ".claude", "worktrees", "agent-x")
	require.NoError(t, os.MkdirAll(worktree, 0o700))
	pointer := "gitdir: " + filepath.Join(repo, ".git", "worktrees", "agent-x") + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(worktree, ".git"), []byte(pointer), 0o600))
	assert.Equal(t, repo, mainWorktree(worktree), "worktree なら本体のリポジトリ")

	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o700))
	assert.Equal(t, repo, mainWorktree(repo), "本体は .git がディレクトリなのでそのまま")

	other := t.TempDir()
	submodule := []byte("gitdir: /x/modules/y")
	require.NoError(t, os.WriteFile(filepath.Join(other, ".git"), submodule, 0o600))
	assert.Equal(t, other, mainWorktree(other), "submodule など worktree でない .git ファイル")
}

func TestDefaultCacheDirEndsWithLocalCache(t *testing.T) {
	dir, err := DefaultCacheDir()
	require.NoError(t, err, "テストはリポジトリの中で走る")
	tail := filepath.Join(filepath.Base(filepath.Dir(dir)), filepath.Base(dir))
	assert.Equal(t, DefaultCacheSubdir, tail)
}
