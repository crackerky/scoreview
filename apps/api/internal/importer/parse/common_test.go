package parse

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

// testdata のフィクスチャは参照サイトの実 HTML から構造を残して縮め、人名を架空の名前に置き換えたもの。

func fixture(t *testing.T, name string) io.Reader {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = f.Close() // 読み取り専用のファイル。Close の失敗はテスト結果に影響しない
	})
	return f
}

// parseBoth は日英のフィクスチャを読んで Merge したものを返す。
func parseBoth[T any](
	t *testing.T, jpName, enName string,
	parse func(io.Reader, site.Lang) (*T, error),
	merge func(jp, en *T) (*T, error),
) *T {
	t.Helper()
	jp, err := parse(fixture(t, jpName), site.LangJP)
	require.NoError(t, err)
	en, err := parse(fixture(t, enName), site.LangEN)
	require.NoError(t, err)
	merged, err := merge(jp, en)
	require.NoError(t, err)
	return merged
}

func TestLoadDetectsErrorPage(t *testing.T) {
	// clone-spec A 7 の 500 画面の骨格
	page := `<!DOCTYPE html><html lang="en"><head><title>Error</title></head><body>
<div id="container"><h1>An Error Was Encountered</h1>
<p>TOURNAMENT_ID [999999]での操作に失敗しました。</p></div></body></html>`
	_, err := ParseTournamentTop(strings.NewReader(page), site.LangJP)
	require.ErrorIs(t, err, ErrErrorPage)
	assert.Contains(t, err.Error(), "TOURNAMENT_ID [999999]")

	notFound := `<html><head><title>404 Page Not Found</title></head><body><h1>404 Page Not Found</h1></body></html>`
	_, err = ParseTournamentList(strings.NewReader(notFound), site.LangJP)
	assert.ErrorIs(t, err, ErrErrorPage)
}

func TestParseRejectsUnknownLang(t *testing.T) {
	_, err := ParseTournamentList(strings.NewReader("<html></html>"), site.Lang("fr"))
	assert.Error(t, err)
}

func TestCleanKeepsIdeographicSpace(t *testing.T) {
	assert.Equal(t, "居酒屋　司", clean("\n\t居酒屋　司  "))
	assert.Equal(t, "グループ成績　", clean("グループ成績　"))
}

func TestElapsedSec(t *testing.T) {
	for in, want := range map[string]int{"00:00": 0, "17:00": 1020, "05:30": 330, "50:00": 3000} {
		got, err := elapsedSec(in)
		require.NoError(t, err)
		assert.Equal(t, want, got, in)
	}
	_, err := elapsedSec("1700")
	assert.ErrorIs(t, err, ErrUnexpected)
}
