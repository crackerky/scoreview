package domain

// 記録係の割り当て(docs/operations/operator-experience.md「記録係を試合やコートごとに割り当てる」)。
// 記録係は、自分の割り当てが 1 つでも覆う試合だけを入力できる。

// Assignment は記録係の割り当て 1 件。MatchID があればその試合だけを指す。無ければ Field(コート)、
// Date("YYYY-MM-DD")、From〜To("HH:MM"。From 以上 To 未満)の範囲を指し、nil の条件は問わない
// (全部 nil なら大会の全試合)。
type Assignment struct {
	MatchID *int64
	Field   *string
	Date    *string
	From    *string
	To      *string
}

// MatchSlot は割り当ての判定に使う試合の ID、コート、日付("YYYY-MM-DD")、開始時刻("HH:MM")。
type MatchSlot struct {
	ID    int64
	Field *string
	Date  *string
	Time  *string
}

// Validate は割り当ての値を確かめる。試合を指すなら範囲の条件は持たない。時刻は "HH:MM" で From < To。
func (a Assignment) Validate() error {
	if a.MatchID != nil {
		if a.Field != nil || a.Date != nil || a.From != nil || a.To != nil {
			return Invalid("matchId", "cannot be combined with field, date, from or to")
		}
		return nil
	}
	if a.Field != nil {
		if err := CheckRequired("field", *a.Field); err != nil {
			return err
		}
	}
	if a.Date != nil {
		if _, err := ParseDay("date", *a.Date); err != nil {
			return err
		}
	}
	if a.From != nil {
		if err := CheckClock("from", *a.From); err != nil {
			return err
		}
	}
	if a.To != nil {
		if err := CheckClock("to", *a.To); err != nil {
			return err
		}
	}
	if a.From != nil && a.To != nil && *a.From >= *a.To {
		return Invalid("to", "must be later than from")
	}
	return nil
}

// Covers は割り当てが試合を覆うかどうか。範囲の条件があるのに試合の側の値が未定(コートや時刻が
// 決まっていない)なら覆わない(担当が曖昧な試合を入力させない)。
func (a Assignment) Covers(match MatchSlot) bool {
	if a.MatchID != nil {
		return *a.MatchID == match.ID
	}
	equal := func(cond, value *string) bool {
		return cond == nil || (value != nil && *cond == *value)
	}
	if !equal(a.Field, match.Field) || !equal(a.Date, match.Date) {
		return false
	}
	// "HH:MM" は辞書順と時刻の順が一致する
	if a.From != nil && (match.Time == nil || *match.Time < *a.From) {
		return false
	}
	if a.To != nil && (match.Time == nil || *match.Time >= *a.To) {
		return false
	}
	return true
}

// AnyCovers は割り当てのどれかが試合を覆うかどうか。
func AnyCovers(assignments []Assignment, match MatchSlot) bool {
	for _, assignment := range assignments {
		if assignment.Covers(match) {
			return true
		}
	}
	return false
}
