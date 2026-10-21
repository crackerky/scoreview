package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIDs(t *testing.T) {
	ids, err := parseIDs("525, 440,,451")
	require.NoError(t, err)
	assert.Equal(t, []int64{525, 440, 451}, ids)

	ids, err = parseIDs("")
	require.NoError(t, err)
	assert.Empty(t, ids)

	for _, bad := range []string{"abc", "0", "-1"} {
		_, err := parseIDs(bad)
		assert.Error(t, err, bad)
	}
}

func TestRepresentativeTournaments(t *testing.T) {
	ids := representativeTournaments()
	assert.Len(t, ids, 16, "docs/representative-tournaments.md の表の行数")
	assert.Equal(t, int64(525), ids[0])
}

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
	require.NoError(t, os.WriteFile(filepath.Join(other, ".git"), []byte("gitdir: /x/modules/y"), 0o600))
	assert.Equal(t, other, mainWorktree(other), "submodule など worktree でない .git ファイル")
}
