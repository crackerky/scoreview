package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrCacheMiss はキャッシュにページがないことを表す。
var ErrCacheMiss = errors.New("cache miss")

// cacheDirPerm と cacheFilePerm はキャッシュの権限。実在の個人名を含むので所有者だけが読める。
const (
	cacheDirPerm  = 0o700
	cacheFilePerm = 0o600
)

// notFoundSuffix は「存在しない」と分かったページの記録に付ける拡張子。中身は HTTP ステータス。
const notFoundSuffix = ".notfound"

// Cache は URL ごとに 1 ファイルで HTML を保存するディレクトリ。
type Cache struct {
	dir string
}

// NewCache は dir をキャッシュにする。dir がなければ作る。
func NewCache(dir string) (*Cache, error) {
	if dir == "" {
		return nil, fmt.Errorf("cache dir is empty")
	}
	if err := os.MkdirAll(dir, cacheDirPerm); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &Cache{dir: dir}, nil
}

// FileName は URL に対応するキャッシュのファイル名を返す。人が読めるように URL の英数字を残し、
// 記号の置き換えで別の URL と衝突しないよう URL の SHA-256 の先頭 8 桁を付ける。
func FileName(pageURL string) string {
	s := pageURL
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	var b strings.Builder
	for _, r := range s {
		switch {
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9', r == '-', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	sum := sha256.Sum256([]byte(pageURL))
	return b.String() + "-" + hex.EncodeToString(sum[:4]) + ".html"
}

// Get はキャッシュした HTML を返す。ない場合は ErrCacheMiss、存在しないと記録したページは
// *NotFoundError を返す。
func (c *Cache) Get(pageURL string) ([]byte, error) {
	path := filepath.Join(c.dir, FileName(pageURL))
	body, err := os.ReadFile(path) // #nosec G304 -- パスは URL から作ったファイル名だけで、dir の外を指さない
	if err == nil {
		return body, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read cache %s: %w", path, err)
	}
	status, err := os.ReadFile(path + notFoundSuffix) // #nosec G304 -- 同上
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("read cache %s: %w", path+notFoundSuffix, err)
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(status)))
	if err != nil {
		return nil, fmt.Errorf("broken not-found record %s: %w", path+notFoundSuffix, err)
	}
	return nil, &NotFoundError{URL: pageURL, Status: code}
}

// Put は HTML を保存する。書きかけのファイルを残さないよう一時ファイルから rename する。
func (c *Cache) Put(pageURL string, body []byte) error {
	return c.write(filepath.Join(c.dir, FileName(pageURL)), body)
}

// PutNotFound は pageURL が存在しない(HTTP status)ことを記録する。
func (c *Cache) PutNotFound(pageURL string, status int) error {
	path := filepath.Join(c.dir, FileName(pageURL)+notFoundSuffix)
	return c.write(path, []byte(strconv.Itoa(status)+"\n"))
}

func (c *Cache) write(path string, body []byte) error {
	tmp, err := os.CreateTemp(c.dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()        // 書き込みに失敗したファイルは捨てるので Close の結果は使わない
		_ = os.Remove(tmpName) // 同上。消せなくても次回の取得に影響しない一時ファイル
		return fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName) // 同上
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err := os.Chmod(tmpName, cacheFilePerm); err != nil {
		_ = os.Remove(tmpName) // 同上
		return fmt.Errorf("chmod %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName) // 同上
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}

// Offline はキャッシュだけを読み、ネットワークに出ない Fetcher。キャッシュにないページは
// ErrCacheMiss を包んだエラーにする。テストのフェイクや、取得済みデータの再パースに使う。
type Offline struct {
	Cache *Cache
}

// Fetch はキャッシュの HTML を返す。
func (o Offline) Fetch(_ context.Context, pageURL string) ([]byte, error) {
	body, err := o.Cache.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("offline fetch %s: %w", pageURL, err)
	}
	return body, nil
}
