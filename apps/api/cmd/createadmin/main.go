// Command createadmin は最初の管理者を DATABASE_URL の DB に作る。
//
// パスワードはコマンドラインの引数に取らない(シェルの履歴とプロセス一覧に残るため)。
// 環境変数 SCOREVIEW_ADMIN_PASSWORD があればそれを、無ければ標準入力の 1 行目を使う。
//
//	cd apps/api && read -rs PW && printf '%s\n' "$PW" | \
//	  DATABASE_URL=postgres://... go run ./cmd/createadmin -email admin@example.com -name 管理者
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"gorm.io/gorm"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// passwordEnv はパスワードを渡す環境変数の名前。
const passwordEnv = "SCOREVIEW_ADMIN_PASSWORD"

func main() {
	email := flag.String("email", "", "管理者のメールアドレス(必須)")
	name := flag.String("name", "", "表示名")
	flag.Parse()
	if err := run(context.Background(), *email, *name, os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, "createadmin:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, email, name string, stdin io.Reader) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	if email == "" {
		return errors.New("-email is required")
	}
	password, err := readPassword(os.Getenv(passwordEnv), stdin)
	if err != nil {
		return err
	}
	db, err := store.Open(dsn)
	if err != nil {
		return err
	}
	user, err := createAdmin(ctx, db, email, name, password)
	if err != nil {
		return err
	}
	fmt.Printf("created admin #%d %s\n", user.ID, user.Email)
	return nil
}

// readPassword は環境変数の値があればそれを、無ければ r の 1 行目(末尾の改行を除く)を返す。
func readPassword(fromEnv string, r io.Reader) (string, error) {
	if fromEnv != "" {
		return fromEnv, nil
	}
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read password from stdin: %w", err)
	}
	password := strings.TrimRight(line, "\r\n")
	if password == "" {
		return "", fmt.Errorf("password is empty: set %s or pass it on stdin", passwordEnv)
	}
	return password, nil
}

// createAdmin はパスワードの長さを確かめ、argon2id でハッシュして管理者を作る。
func createAdmin(
	ctx context.Context,
	db *gorm.DB,
	email, name, password string,
) (store.UserRecord, error) {
	if err := auth.ValidatePassword(password); err != nil {
		return store.UserRecord{}, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return store.UserRecord{}, err
	}
	fields := store.UserFields{
		Email:       auth.NormalizeEmail(email),
		DisplayName: name,
		Role:        domain.RoleAdmin,
	}
	return store.CreateUser(ctx, db, fields, hash)
}
