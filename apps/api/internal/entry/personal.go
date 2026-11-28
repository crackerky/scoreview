package entry

import (
	"time"

	"scoreview/api/internal/domain"
)

// PersonalAsk は大会が申込でメンバー本人に聞く項目(Lイベントの顧客情報入力項目にあたる)。聞くと決めた
// 項目は、メンバー登録で答えが要る。
type PersonalAsk struct {
	Gender    bool
	Birthdate bool
}

// Personal はメンバー本人の性別と生年月日。nil はまだ答えていない(または聞いていない)。Gender は選手と
// 同じ M / F、Birthdate は "YYYY-MM-DD"。
type Personal struct {
	Gender    *domain.Gender
	Birthdate *string
}

// 性別の値(選手の性別と同じ)。
const (
	GenderMale   domain.Gender = "M"
	GenderFemale domain.Gender = "F"
)

// earliestBirthdate は生年月日として受ける最も古い日(打ち間違いを弾くため)。
const earliestBirthdate = "1900-01-01"

// CheckPersonal は聞く項目の答えを確かめる。聞く項目は答えが要り、聞かない項目の答えは捨てる(nil)。
// gender は M か F、birthdate は "YYYY-MM-DD" で 1900-01-01 から today までの日。
func CheckPersonal(ask PersonalAsk, gender, birthdate string, today time.Time) (Personal, error) {
	var out Personal
	if ask.Gender {
		value := domain.Gender(gender)
		if value != GenderMale && value != GenderFemale {
			return Personal{}, domain.Invalid("gender", "must be M or F")
		}
		out.Gender = &value
	}
	if ask.Birthdate {
		day, err := time.Parse(time.DateOnly, birthdate)
		if err != nil {
			return Personal{}, domain.Invalid("birthdate", "must be a date (YYYY-MM-DD)")
		}
		if birthdate < earliestBirthdate || day.After(today) {
			return Personal{}, domain.Invalid("birthdate", "must be between 1900-01-01 and today")
		}
		out.Birthdate = &birthdate
	}
	return out, nil
}

// Prefill は登録の画面に前もって入れておく性別と生年月日。このチームで答えた値があればそれを、無ければ
// 前回の入力(参加者の台帳)を項目ごとに使う。
func Prefill(mine *Member, profile *Profile) Personal {
	var out Personal
	if profile != nil {
		out = profile.Personal
	}
	if mine != nil {
		if mine.Personal.Gender != nil {
			out.Gender = mine.Personal.Gender
		}
		if mine.Personal.Birthdate != nil {
			out.Birthdate = mine.Personal.Birthdate
		}
	}
	return out
}
