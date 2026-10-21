package domain

import (
	"fmt"
	"sort"
	"time"
)

// 記録係の画面で、同じ試合を複数の端末で記録したときの食い違いを防ぐ規則
// (docs/operations/operator-experience.md「記録係:間違えにくく、直しやすい入力」)。
// 保存は store が行い、ここは判定だけを持つ。

// ScoreMismatchError は、記録係が確認したスコアと試合の現在のスコアが違うことを表す
// (確認シートを開いた後に別の端末で得点が入った)。errors.Is(err, ErrConflict) が真になる。
type ScoreMismatchError struct {
	Expected Score
	Actual   Score
}

func (e *ScoreMismatchError) Error() string {
	return fmt.Sprintf("the score is now %d-%d, not the confirmed %d-%d",
		e.Actual.Left, e.Actual.Right, e.Expected.Left, e.Expected.Right)
}

func (e *ScoreMismatchError) Unwrap() error { return ErrConflict }

// CheckExpectedScore は、記録係が確認したスコア expected が現在のスコア actual と同じかを確かめる。
// expected が nil(確認なしの操作。JavaScript なしの入力や本部の訂正)なら確かめない。
func CheckExpectedScore(expected *Score, actual Score) error {
	if expected == nil || *expected == actual {
		return nil
	}
	return &ScoreMismatchError{Expected: *expected, Actual: actual}
}

// ScorerClaimTTL は試合の記録担当の期限。記録画面は開いている間 ScorerClaimTTL より短い間隔で
// 延長する。端末を閉じた、圏外になったなどで延長が止まれば、この時間で他の人が引き継げる。
const ScorerClaimTTL = 30 * time.Second

// ConcurrentRecordWindow は、別の利用者が同じ種類の記録(得点、プル)をほぼ同時に入れたとみなす
// 間隔。2 台で同じ得点を記録したときの二重記録を知らせるため。1 点の攻防は 5 秒より長いので、
// この間隔で 2 件入るのは二重記録の疑いが強い。
const ConcurrentRecordWindow = 5 * time.Second

// RecordedEvent は二重記録の判定に使う得点経過(誰がいつ記録したか)。UserID が 0 なら記録者が
// 分からない(取込ツール、操作 ID の無い入力)。
type RecordedEvent struct {
	ClientID   string
	Kind       EventKind
	UserID     int64
	RecordedAt time.Time
}

// ConcurrentRecord は、別の利用者がほぼ同時に入れた同じ種類の 2 件の記録。First が先。
type ConcurrentRecord struct {
	Kind   EventKind
	First  RecordedEvent
	Second RecordedEvent
}

// FindConcurrentRecords は、得点とプルのうち、同じ種類の直前の記録と別の利用者が
// ConcurrentRecordWindow 以内に記録したものを、記録した時刻の順に返す。
func FindConcurrentRecords(events []RecordedEvent) []ConcurrentRecord {
	sorted := make([]RecordedEvent, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].RecordedAt.Before(sorted[j].RecordedAt)
	})
	last := map[EventKind]RecordedEvent{}
	found := []ConcurrentRecord{}
	for _, event := range sorted {
		if event.Kind != EventGoal && event.Kind != EventPull {
			continue
		}
		previous, ok := last[event.Kind]
		last[event.Kind] = event
		if !ok || previous.UserID == 0 || event.UserID == 0 || previous.UserID == event.UserID {
			continue
		}
		if event.RecordedAt.Sub(previous.RecordedAt) <= ConcurrentRecordWindow {
			found = append(found, ConcurrentRecord{Kind: event.Kind, First: previous, Second: event})
		}
	}
	return found
}
