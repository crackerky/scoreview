// Package domain は GORM と HTTP を知らない純粋なドメイン型と計算を持つ。
package domain

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors。errors.Is で判定し、HTTP ステータスへの変換は httpapi の 1 か所で行う。
var (
	ErrNotFound     = errors.New("not_found")
	ErrBadRequest   = errors.New("bad_request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	// ErrTooManyRequests はログイン試行の制限を超えたことを表す
	ErrTooManyRequests = errors.New("too_many_requests")
	// ErrUnavailable は混雑で今は処理できないことを表す(パスワード照合の同時実行数の上限など)
	ErrUnavailable = errors.New("unavailable")
)

// RetryAfterError は、待てば受け付ける拒否を表す。After は次に試してよいまでの時間。
// errors.Is は Err(ErrTooManyRequests か ErrUnavailable)で判定する。
type RetryAfterError struct {
	Err   error
	After time.Duration
}

func (e *RetryAfterError) Error() string {
	return fmt.Sprintf("%s (retry after %s)", e.Err.Error(), e.After)
}

func (e *RetryAfterError) Unwrap() error { return e.Err }

// RetryAfter は RetryAfterError を作る。
func RetryAfter(err error, after time.Duration) error {
	return &RetryAfterError{Err: err, After: after}
}

// Resource は存在しない ID を報告するときの種別名。参照サイトのエラー画面の文面
// ("TOURNAMENT_ID [999999]での操作に失敗しました。") に合わせた識別子を使う。
type Resource string

const (
	ResourceTournament Resource = "TOURNAMENT_ID"
	ResourceDivision   Resource = "EVENT_DIVISION_ID"
	ResourceGroup      Resource = "GROUP_ID"
	ResourceTeam       Resource = "ENTRY_TEAM_ID"
	ResourcePlayer     Resource = "ENTRY_PLAYER_ID"
	ResourceMatch      Resource = "MATCH_ID"
	ResourceGallery    Resource = "TGALLERY_ID"
)

// NotFoundError は ID が存在しないことを表す。errors.Is(err, ErrNotFound) が真になる。
type NotFoundError struct {
	Resource Resource
	ID       int64
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s [%d]", e.Resource, e.ID)
}

func (e *NotFoundError) Unwrap() error { return ErrNotFound }

// NotFound は NotFoundError を作る。
func NotFound(resource Resource, id int64) error {
	return &NotFoundError{Resource: resource, ID: id}
}

// ValidationError は入力の検証失敗を表す。errors.Is(err, ErrBadRequest) が真になる。
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

func (e *ValidationError) Unwrap() error { return ErrBadRequest }

// Invalid は ValidationError を作る。
func Invalid(field, reason string) error {
	return &ValidationError{Field: field, Reason: reason}
}
