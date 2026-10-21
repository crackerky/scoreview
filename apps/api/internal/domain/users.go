package domain

// Role は管理画面の利用者の役割(docs/requirements.md 権限モデル)。
type Role string

// Role の値。
const (
	// RoleAdmin は全大会の登録、編集、削除とユーザー管理ができる。
	RoleAdmin Role = "admin"
	// RoleStaff は tournament_staff に登録された大会の試合の状態、イベント、選手記録だけを変えられる。
	RoleStaff Role = "staff"
)

// 管理画面だけで使う資源の種別名。参照サイトに対応する画面が無いので、閲覧用の種別名
// (errors.go)と同じ「{種別}_ID」の形にそろえた独自の名前。
const (
	ResourceUser  Resource = "USER_ID"
	ResourceEvent Resource = "MATCH_EVENT_ID"
)

// Valid は既知の役割かどうか。
func (r Role) Valid() bool { return r == RoleAdmin || r == RoleStaff }

// User は管理画面の利用者。パスワードのハッシュは含めない(外へ出さないため)。
type User struct {
	ID          int64
	Email       string
	DisplayName string
	Role        Role
}

// Actor は試合を更新する操作の実行者。状態遷移の権限(finished → live は admin だけ)の判定に使う。
type Actor struct {
	UserID int64
	Role   Role
}
