package entry

// 参加者の台帳(docs/line-integration.md「参加者の台帳」)。公式アカウントに話しかけた人は line_users に
// 自動で載り、申込とメンバー登録のたびに、そのとき入れた名前と連絡先を前回の入力として残す。次の申込では
// 前回の入力を最初から入れ、前に申し込んだチームを候補に出す(同じ代表が毎回同じ中身を書き直さないため)。

// MaxPastTeams は申込の画面に出す、前に申し込んだチームの数(新しい順)。
const MaxPastTeams = 10

// Profile は参加者の前回の入力。
type Profile struct {
	Name  string
	Phone string
	Email string
}

// Contact は代表の連絡先。
type Contact struct {
	Phone string
	Email string
}

// ProfileUpdate は前回の入力の上書き。Contact が nil なら名前だけを上書きする(メンバー登録では連絡先を
// 聞かないので、代表として入れた連絡先を消さない)。
type ProfileUpdate struct {
	Name    string
	Contact *Contact
}

// PastTeam は参加者が前に代表として申し込んだチーム(今の大会の申込は除く)。Members はその申込の今の
// メンバーで、本人が登録で直した名前と背番号を含む。
type PastTeam struct {
	FullName  string
	ShortName string
	Members   []MemberDraft
}

// RepProfile は代表の申込(検査済み)から、前回の入力の上書きを作る。
func RepProfile(draft Draft) ProfileUpdate {
	return ProfileUpdate{Name: draft.RepName, Contact: &Contact{Phone: draft.RepPhone, Email: draft.RepEmail}}
}

// MemberProfile はメンバー登録(検査済み)から、前回の入力の上書きを作る。名前だけを上書きする。
func MemberProfile(join JoinDraft) ProfileUpdate {
	return ProfileUpdate{Name: join.Name}
}
