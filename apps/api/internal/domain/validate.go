package domain

import (
	"net/url"
	"strings"
	"time"
)

// 管理画面の入力の検査。どれも違反なら ValidationError(HTTP 400)を返す。

// CheckRequired は前後の空白を除いて空でないことを確かめる。値そのものは加工しない
// (チームの正式名は後で大会をまたいで束ねるため加工せずに保存する。docs/line-integration.md)。
func CheckRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return Invalid(field, "is required")
	}
	return nil
}

// CheckURL は画像やリンクの URL が http(s) の絶対 URL か、"/" で始まるサイト内のパスであることを
// 確かめる。javascript: や data: の URL は、img の src や a の href に出したときにスクリプトを
// 実行させうるので通さない。"//host" のようなスキーム省略の URL も、行き先が分かりにくいので通さない。
func CheckURL(field, value string) error {
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return Invalid(field, "must be an http(s) URL or a path starting with /")
	}
	return nil
}

// CheckOptionalURL は nil を許す CheckURL。
func CheckOptionalURL(field string, value *string) error {
	if value == nil {
		return nil
	}
	return CheckURL(field, *value)
}

// ParseDay は "YYYY-MM-DD" を読む。
func ParseDay(field, value string) (time.Time, error) {
	day, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, Invalid(field, "must be YYYY-MM-DD")
	}
	return day, nil
}

// CheckClock は時刻が "HH:MM"(00:00〜23:59)であることを確かめる。
func CheckClock(field, value string) error {
	if len(value) != len("15:04") {
		return Invalid(field, "must be HH:MM")
	}
	if _, err := time.Parse("15:04", value); err != nil {
		return Invalid(field, "must be HH:MM")
	}
	return nil
}
