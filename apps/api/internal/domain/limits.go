package domain

import "fmt"

// 記録の数値の上限。DB の列は integer(int4、約 21 億まで)で、超える値は保存で失敗して 500 になる。
// 現実の値を十分に超える範囲で、保存の前に検証エラー(400)にする(セキュリティ点検 L-2)。
// web の zod(apps/web/src/api/admin.types.ts の同名の定数)と揃える。
const (
	// MaxElapsedSec は得点経過の経過秒の上限。1 日(大会は 1 日単位で運営され、試合は数時間で終わる)
	MaxElapsedSec = 24 * 60 * 60
	// MaxScore は試合のスコアと不戦勝の規定スコアの上限。アルティメットは 15 点前後で終わる
	MaxScore = 999
	// MaxPlayerStat は 1 試合の選手のブロック数とグレイテスト数の上限
	MaxPlayerStat = 999
)

// CheckRange は value が min 以上 max 以下であることを確かめる。
func CheckRange(field string, value, minValue, maxValue int) error {
	if value < minValue || value > maxValue {
		return Invalid(field, fmt.Sprintf("must be %d to %d", minValue, maxValue))
	}
	return nil
}
