package domain

import "time"

// ListSection は大会一覧の区分。管理者が決める大会の状態(TournamentStatus)に、試合の進み具合と
// 日付から決まる「開催中」を加えたもの。本サイト独自(参照サイトは状態の 3 区分だけ)。
type ListSection string

// ListSection の値。
const (
	SectionInProgress  ListSection = "in_progress"
	SectionEntryOpen   ListSection = "entry_open"
	SectionEntryClosed ListSection = "entry_closed"
	SectionFinished    ListSection = "finished"
)

// ListSectionOrder は大会一覧の区分の表示順。当日に見に来る人がいちばん多い開催中を先頭に置き、
// その後は参照サイトと同じ順(clone-spec A 3)。
func ListSectionOrder() []ListSection {
	return []ListSection{SectionInProgress, SectionEntryOpen, SectionEntryClosed, SectionFinished}
}

// PlayProgress は大会の試合の進み具合。FirstDay と LastDay は日付の入った試合の最初と最後の日で、
// 日付の入った試合が無ければ nil。
type PlayProgress struct {
	Matches  int
	Finished int
	Live     int
	FirstDay *time.Time
	LastDay  *time.Time
}

// ListSectionOf は大会を一覧のどの区分に出すかを決める。today は会場の暦の今日(時刻は見ない)。
//
// 管理者が終了にした大会は終了。それ以外は、全試合が終わったか最終日を過ぎたら終了、試合中の
// 試合があるか今日が試合のある期間に入っていれば開催中、どちらでもなければ管理者が決めた状態の
// まま。当日の切り替えを管理者の操作に頼ると、忙しい本部が押し忘れて一覧が実際とずれるため。
func ListSectionOf(status TournamentStatus, progress PlayProgress, today time.Time) ListSection {
	if status == TournamentFinished {
		return SectionFinished
	}
	day := calendarDay(today)
	if progress.Matches > 0 && progress.Finished == progress.Matches {
		return SectionFinished
	}
	if progress.LastDay != nil && day.After(calendarDay(*progress.LastDay)) {
		return SectionFinished
	}
	if progress.Live > 0 {
		return SectionInProgress
	}
	if progress.FirstDay != nil && !day.Before(calendarDay(*progress.FirstDay)) {
		return SectionInProgress
	}
	if status == TournamentEntryClosed {
		return SectionEntryClosed
	}
	return SectionEntryOpen
}

// calendarDay は時刻を捨てた暦の日(UTC の 0 時)。試合日は DATE 列なので時刻と時差を持たない。
func calendarDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
