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
func writeError(what string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf(
				"%w: %s: duplicate value (%s)",
				domain.ErrConflict,
				what,
				pgErr.ConstraintName,
			)
		case pgForeignKeyViolation:
			return fmt.Errorf(
				"%w: %s: broken reference (%s)",
				domain.ErrConflict,
				what,
				pgErr.ConstraintName,
			)
		case pgCheckViolation:
			return domain.Invalid(what, "violates "+pgErr.ConstraintName)
		}
	}
	return fmt.Errorf("%s: %w", what, err)
}
