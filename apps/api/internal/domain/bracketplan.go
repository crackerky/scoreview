package domain

// BracketSlotRole は、トーナメント表のセルが試合のどのチームを出すか。
type BracketSlotRole string

// BracketSlotRole の値。
const (
	SlotLeft   BracketSlotRole = "left"   // 試合の左(表の上)のチーム
	SlotRight  BracketSlotRole = "right"  // 試合の右(表の下)のチーム
	SlotWinner BracketSlotRole = "winner" // 試合の勝者
	SlotLoser  BracketSlotRole = "loser"  // 試合の敗者
)

// BracketPlan は管理画面のテンプレートが作るトーナメント表の設計図。試合とチームはまだ DB に無いので、
// 格子の MatchID と TeamID は空のまま、試合は Matches の添字で、チームはシード(表の上からの順)で参照する。
// store がこれを試合の行、格子、bracket_slots にして保存する。
type BracketPlan struct {
	// Pattern は groups.bracket_pattern に入れる識別子(参照サイトの GeneratePtn。例 "T8")
	Pattern string
	// Name は groups.bracket_ptn_name の前半(参照サイトの PtnName は "{テンプレ名} =グループ名=")
	Name string
	// Teams は参加チーム数(シードの数)
	Teams   int
	Layout  BracketLayout
	Matches []PlannedMatch
	// Games は game セル(スコアを出すセル)と、それが表す試合の添字
	Games []PlannedCell
	// Slots はチーム名のセルと、それが表す試合の添字と役割
	Slots []PlannedSlot
}

// PlannedMatch はテンプレートが作る試合。Number は表の中の通し番号(1 始まり)で、試合 CD は
// 「接頭辞 + Number」になる(例 接頭辞 "B" の 5 番目は "B5")。
type PlannedMatch struct {
	Number int
	Left   PlannedEntrant
	Right  PlannedEntrant
}

// PlannedEntrant は試合の片側。Seed(0 始まりのシード)か From(前の試合の勝者か敗者)のどちらか一方。
type PlannedEntrant struct {
	Seed *int
	From *PlannedRef
}

// PlannedRef は前の試合の勝者か敗者。Match は BracketPlan.Matches の添字。
// 保存すると "[B1-勝ち]" のような未確定チームの参照(matches.*_placeholder_*)になる。
type PlannedRef struct {
	Match   int
	Outcome Outcome
}

// PlannedCell は格子のセルの位置と、それが表す試合の添字。
type PlannedCell struct {
	Row   int
	Col   int
	Match int
}

// PlannedSlot はチーム名のセルの位置と、そこに出す試合の添字と役割。
type PlannedSlot struct {
	PlannedCell
	Role BracketSlotRole
}
