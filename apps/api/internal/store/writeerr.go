package store

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"scoreview/api/internal/domain"
)

// PostgreSQL のエラーコード(https://www.postgresql.org/docs/17/errcodes-appendix.html)。
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

// writeError は管理画面からの書き込みの失敗を domain のエラーにする。一意制約と外部キーの違反は
// ErrConflict(同じ値が既にある、参照先が無い、まだ参照されている)、CHECK 違反は検証エラー。
// それ以外は文脈を付けてそのまま返す(httpapi で 500 になる)。
//
// 応答の文面には制約名を出さない(表の構造を利用者に見せない。セキュリティ点検 I-2)。
// 調べるときは what と、500 のログに出る元のエラーを使う。
func writeError(what string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf("%w: %s: the same value is already registered", domain.ErrConflict, what)
		case pgForeignKeyViolation:
			return fmt.Errorf(
				"%w: %s: it refers to something that does not exist, or is still referred to",
				domain.ErrConflict,
				what,
			)
		case pgCheckViolation:
			return domain.Invalid(what, "is out of the allowed values")
		}
	}
	return fmt.Errorf("%s: %w", what, err)
}
