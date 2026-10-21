package store

import "scoreview/api/internal/domain"

// このファイルはページ単位の読み取り結果(ビュー型)。docs/api.md の閲覧用エンドポイントと 1 対 1 に
// 対応し、httpapi はこれを DTO に写すだけにする。スライスは nil にせず、空なら長さ 0 で返す。

// Frame は大会ナビとサイドバーを描くための骨格。閲覧用の全ページが持つ。
type Frame struct {
	ID            int64
	Name          domain.I18nText
	ShortName     string
	TeamListLabel string // teams | entries
	GroupsLabel   string // groups | results
	ShowSchedule  bool
	Divisions     []FrameDivision
	Galleries     []FrameGallery
	Banners       []FrameBanner
}

// FrameDivision はナビのディビジョン。Groups は published のグループだけ(sort_order, id 順)。
type FrameDivision struct {
	ID     int64
	Name   domain.I18nText
	Groups []FrameGroup
}

// FrameGroup はナビのグループ。Linked が false なら名前だけ出す。
type FrameGroup struct {
	ID     int64
	Name   string
	Linked bool
}

// FrameGallery はナビのギャラリー。
type FrameGallery struct {
	ID   int64
	Name string
}

// FrameBanner はサイドバーのバナー。
type FrameBanner struct {
	ImageURL string
	LinkURL  *string
}

// DivisionRef はディビジョンの参照。
type DivisionRef struct {
	ID   int64
	Name domain.I18nText
}

// TournamentSection は大会一覧の区分。
type TournamentSection struct {
	Status      domain.TournamentStatus
	Tournaments []TournamentSummary
}

// TournamentSummary は大会一覧の 1 件。
type TournamentSummary struct {
	ID   int64
	Name domain.I18nText
}

// TournamentTopPage は大会トップ。TopHTML は DB の値そのままで、無害化は呼び出し側で行う。
type TournamentTopPage struct {
	Frame     Frame
	LogoURL   *string
	TopHTML   domain.I18nText
	Divisions []TopDivision
}

// TopDivision は大会トップのディビジョン表の 1 行。
type TopDivision struct {
	ID              int64
	Name            domain.I18nText
	ScheduleLinked  bool
	StandingsLinked bool
}

// TeamListPage はチーム一覧。
type TeamListPage struct {
	Frame     Frame
	Divisions []TeamListDivision
}

// TeamListDivision はチーム一覧のディビジョン。
type TeamListDivision struct {
	ID        int64
	Name      domain.I18nText
	Kind      domain.DivisionKind
	EntryKind domain.EntryKind
	Teams     []TeamListTeam
}

// TeamListTeam はチーム一覧の 1 チーム。Name は正式名。MemberCount はスタッフを除く選手数で、
// 個人戦なら nil。
type TeamListTeam struct {
	ID          int64
	Name        domain.I18nText
	BarStyle    domain.TeamBarStyle
	MemberCount *int
}

// ScheduleFilter はスケジュールの絞り込みと並び替え。検証は呼び出し側で済ませる。
type ScheduleFilter struct {
	// YMD は "YYYYMMDD"。nil はクエリにキーがないこと(既定を使う)、"" は全日程
	YMD *string
	// Team は大会全体ならチームの正式名(日本語)、ディビジョン別ならチーム ID の文字列。"" は絞り込まない
	Team string
	// Field はコートの完全一致。"" は絞り込まない
	Field string
	Sort  *domain.ScheduleSort
}

// SchedulePage はスケジュール。Division はディビジョン別のときだけ非 nil。
type SchedulePage struct {
	Frame    Frame
	Division *DivisionRef
	Dates    []string // "YYYY-MM-DD" の昇順
	Teams    []ScheduleTeamOption
	Fields   []string // 範囲内の試合で使われたコートの異なり値(自然順)
	Selected ScheduleSelected
	Sort     *domain.ScheduleSort
	Rows     []domain.ScheduleRow
}

// ScheduleTeamOption はチームの絞り込みの選択肢。Name は正式名。
type ScheduleTeamOption struct {
	Value string
	Name  domain.I18nText
}

// ScheduleSelected は実際に適用した絞り込み(既定値を含む)。
type ScheduleSelected struct {
	YMD   string
	Team  string
	Field string
}

// StandingsPage はディビジョン成績。
type StandingsPage struct {
	Frame        Frame
	Division     DivisionRef
	Standings    []StandingRow
	Leaders      []LeaderRow
	BlockLeaders []BlockLeaderRow
}

// StandingRow はディビジョン順位表の 1 行。Rank は teams.final_rank。
type StandingRow struct {
	Team  domain.TeamRef
	Tally domain.Tally
	Rank  *int
}

// LeaderRow は得点ランキングの 1 行。
type LeaderRow struct {
	Team    domain.TeamRef
	Player  domain.PlayerRef
	Games   int
	Assists int
	Goals   int
}

// BlockLeaderRow はブロックランキングの 1 行。
type BlockLeaderRow struct {
	Team   domain.TeamRef
	Player domain.PlayerRef
	Games  int
	Blocks int
}

// GroupPage はグループ成績。Kind に応じて Pool、Bracket、Image のどれか 1 つだけが非 nil。
type GroupPage struct {
	Frame   Frame
	Group   GroupInfo
	Pool    *PoolView
	Bracket *BracketView
	Image   *ImageView
	Matches []domain.MatchRow
}

// GroupInfo はグループの見出しに使う情報。
type GroupInfo struct {
	ID       int64
	Name     string
	Kind     domain.GroupKind
	Division DivisionRef
}

// MatrixCellKind は対戦表のセルの種類。
type MatrixCellKind string

// MatrixCellKind の値。
const (
	MatrixSelf  MatrixCellKind = "self"
	MatrixLower MatrixCellKind = "lower"
	MatrixEmpty MatrixCellKind = "empty"
	MatrixMatch MatrixCellKind = "match"
)

// MatrixCell は対戦表のセル。MatchID、Status、Text は Kind が match のときだけ意味を持つ。
type MatrixCell struct {
	Kind    MatrixCellKind
	MatchID int64
	Status  domain.MatchStatus
	Text    string
}

// PoolView はプール型グループの対戦表とリーグ結果。
type PoolView struct {
	Teams   []domain.TeamRef
	Matrix  [][]MatrixCell
	Results []PoolResultRow
}

// PoolResultRow はリーグ結果の 1 行。
type PoolResultRow struct {
	Team   domain.TeamRef
	Tally  domain.Tally
	Points int
	Rank   *int
}

// BracketView はトーナメント表。
type BracketView struct {
	Pattern string
	PtnName string
	Columns int
	Rows    [][]BracketCellView
}

// BracketCellView はトーナメント表の 1 セル。Team と Status は参照があるときだけ非 nil。
type BracketCellView struct {
	Borders domain.BracketBorders
	Kind    domain.BracketCellKind
	Text    string
	Team    *domain.TeamRef
	MatchID *int64
	Status  *domain.MatchStatus
}

// ImageView は画像だけのグループ。
type ImageView struct {
	URL    string
	Width  *int
	Height *int
}

// TeamPage はチームページ。
type TeamPage struct {
	Frame   Frame
	Team    TeamInfo
	Record  domain.Tally
	Rank    *int
	Matches []domain.MatchRow
	Players []PlayerLine
	Staff   []StaffLine
}

// TeamLayout はチームページの構成。
type TeamLayout string

// TeamLayout の値。roster_only はディスクゴルフ団体戦(成績表と対戦一覧を出さない。clone-spec D 1.2)。
const (
	TeamLayoutStandard   TeamLayout = "standard"
	TeamLayoutRosterOnly TeamLayout = "roster_only"
)

// TeamInfo はチームの見出しと紹介文。
type TeamInfo struct {
	ID        int64
	FullName  domain.I18nText
	ShortName domain.I18nText
	Comment   domain.I18nText
	Layout    TeamLayout
}

// PlayerLine はチームページの選手一覧の 1 行。
type PlayerLine struct {
	Player  domain.PlayerRef
	Games   int
	Assists int
	Goals   int
	Blocks  int
}

// StaffLine はスタッフ一覧の 1 行。
type StaffLine struct {
	ID     int64
	Role   domain.I18nText
	Name   domain.I18nText
	Gender *domain.Gender
}

// MatchPage は試合ページ。
type MatchPage struct {
	Frame   Frame
	Match   MatchInfo
	Events  []EventLine
	Lineups Lineups
	Stats   domain.MatchStats
}

// MatchInfo は試合のスコアボードの情報。
type MatchInfo struct {
	ID         int64
	Code       string
	GroupName  *string
	Date       *string
	Time       *string
	Field      *string
	Status     domain.MatchStatus
	Left       MatchTeamScore
	Right      MatchTeamScore
	Breadcrumb Breadcrumb
}

// MatchTeamScore はスコアボードの片側。Team は短縮名、FullName は正式名(ページの title に使う。
// clone-spec D 0)。チームが未確定なら Team と FullName は nil。
type MatchTeamScore struct {
	Team     *domain.TeamRef
	FullName *domain.I18nText
	Score    *int
}

// Breadcrumb は試合ページの戻り先(参照サイトではコメントアウトされている。clone-spec D 2.1)。
type Breadcrumb struct {
	DivisionID   int64
	DivisionName domain.I18nText
	GroupID      *int64
}

// EventLine は得点経過の 1 行。LeftScore と RightScore はそのイベント時点の累計。
type EventLine struct {
	ElapsedSec int
	Kind       domain.EventKind
	Side       *domain.Side
	Scorer     *domain.PlayerRef
	Assist     *domain.PlayerRef
	Callahan   bool
	LeftScore  int
	RightScore int
}

// Lineups は試合の出場選手の左右。
type Lineups struct {
	Left  []LineupLine
	Right []LineupLine
}

// LineupLine は出場選手の 1 行。
type LineupLine struct {
	Player  domain.PlayerRef
	Assists int
	Goals   int
	Blocks  int
}

// PlayerPage は選手ページ(スタッフも同じ)。
type PlayerPage struct {
	Frame     Frame
	Player    PlayerWithRole
	Team      domain.TeamRef
	Prev      PlayerWithRole
	Next      PlayerWithRole
	StatsKind domain.StatsKind
	Stats     PlayerStats
	Events    []PlayerEventLine
}

// PlayerWithRole は選手の参照と役割。Role が非 nil ならスタッフ。
type PlayerWithRole struct {
	Player domain.PlayerRef
	Role   *domain.I18nText
}

// PlayerStats は選手ページの成績表。
type PlayerStats struct {
	Games     int
	Assists   int
	Goals     int
	Callahans int
	Greatests int
	Blocks    int
}

// PlayerEventLine は選手が得点かアシストをしたイベント。
type PlayerEventLine struct {
	Code       string
	Date       *string
	Opponent   *domain.TeamRef
	ElapsedSec int
	Scorer     *domain.PlayerRef
	Assist     *domain.PlayerRef
}

// GalleryPage はギャラリー。
type GalleryPage struct {
	Frame  Frame
	ID     int64
	Name   string
	Photos []Photo
}

// Photo はギャラリーの写真。
type Photo struct {
	ThumbURL string
	FullURL  string
}
