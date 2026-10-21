package domain

// I18nText は日英の両方の文字列。英語が未登録なら En は空文字のまま(参照サイトが空表示するため、補完しない)。
type I18nText struct {
	Ja string
	En string
}

// TournamentStatus は大会一覧の区分。
type TournamentStatus string

// TournamentStatus の値。大会一覧ではこの順に区分を並べる(clone-spec A 3)。
const (
	TournamentEntryOpen   TournamentStatus = "entry_open"
	TournamentEntryClosed TournamentStatus = "entry_closed"
	TournamentFinished    TournamentStatus = "finished"
)

// TournamentStatusOrder は大会一覧の区分の表示順。
func TournamentStatusOrder() []TournamentStatus {
	return []TournamentStatus{TournamentEntryOpen, TournamentEntryClosed, TournamentFinished}
}

// DivisionKind は日程のディビジョン列の色を決める区分(clone-spec B 6)。
type DivisionKind string

// DivisionKind の値。
const (
	DivisionMen   DivisionKind = "men"
	DivisionWomen DivisionKind = "women"
	DivisionMixed DivisionKind = "mixed"
	DivisionNone  DivisionKind = "none"
)

// TeamBarStyle はチーム一覧の色帯(clone-spec A 5)。
type TeamBarStyle string

// EntryKind はディビジョンの参加単位。individual はディスクゴルフ個人戦。
type EntryKind string

// EntryKind の値。
const (
	EntryTeam       EntryKind = "team"
	EntryIndividual EntryKind = "individual"
)

// StatsKind は選手ページの成績表の列構成。
type StatsKind string

// StatsKind の値。
const (
	StatsUltimate StatsKind = "ultimate"
	StatsDiscgolf StatsKind = "discgolf"
)

// MatchStatus は試合の状態。
type MatchStatus string

// MatchStatus の値。
const (
	MatchScheduled MatchStatus = "scheduled"
	MatchLive      MatchStatus = "live"
	MatchFinished  MatchStatus = "finished"
)

// GroupKind はグループ成績ページの型。
type GroupKind string

// GroupKind の値。
const (
	GroupPool    GroupKind = "pool"
	GroupBracket GroupKind = "bracket"
	GroupImage   GroupKind = "image"
)

// Side は試合記録上の左右。
type Side string

// Side の値。
const (
	SideLeft  Side = "left"
	SideRight Side = "right"
)

// Opposite は反対側を返す。
func (s Side) Opposite() Side {
	if s == SideLeft {
		return SideRight
	}
	return SideLeft
}

// Gender は選手の性別。参照サイトでは選手リンクの色(playerLinkM / playerLinkF)に使う。
type Gender string

// Outcome は未確定の対戦相手が前の試合の勝者か敗者か。
type Outcome string

// Outcome の値。
const (
	OutcomeWin  Outcome = "win"
	OutcomeLose Outcome = "lose"
)

// EventKind は得点経過のイベントの種類。
type EventKind string

// EventKind の値。
const (
	EventGoal    EventKind = "goal"
	EventTimeout EventKind = "timeout"
	EventPull    EventKind = "pull"
	EventEnd     EventKind = "end"
)

// TeamRef は表の中で使うチームの参照。Name は短縮名(clone-spec D 1.3)。
type TeamRef struct {
	ID   int64
	Name I18nText
}

// PlayerRef は選手(またはスタッフ)の参照。Number は "00" のような表記を保つため文字列。
type PlayerRef struct {
	ID     int64
	Number string
	Name   I18nText
	Gender *Gender
}

// Placeholder は未確定の対戦相手。表示は "[A1-勝ち]"(clone-spec B 8)。
type Placeholder struct {
	Code    string
	Outcome Outcome
}

// MatchSide は日程や結果一覧の 1 行の片側。Team が nil のときだけ Placeholder が入りうる。
type MatchSide struct {
	Team        *TeamRef
	Placeholder *Placeholder
	Score       *int
}

// MatchRow は日程や結果一覧の 1 行。Date は "YYYY-MM-DD"、Time は "HH:MM"。
type MatchRow struct {
	ID        int64
	HasDetail bool
	Code      string
	Date      *string
	Time      *string
	Field     *int
	Status    MatchStatus
	Left      MatchSide
	Right     MatchSide
}
