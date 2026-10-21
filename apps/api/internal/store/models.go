package store

import (
	"time"

	"scoreview/api/internal/domain"
)

// このファイルの型は表 1 つに 1 つの GORM モデルで、列名を gorm タグで明示する。
// 書き込み(取込ツール、シード、テストの fixture)だけに使う。読み取り関数はこれを外に返さず、
// domain の型か views.go のビュー型に変換して返す。httpapi からは使わない。
// スキーマの正本は migrations/00001_init.sql。

// Tournament は tournaments の行。
type Tournament struct {
	ID            int64                   `gorm:"column:id;primaryKey"`
	NameJa        string                  `gorm:"column:name_ja"`
	NameEn        string                  `gorm:"column:name_en"`
	ShortName     string                  `gorm:"column:short_name"`
	Status        domain.TournamentStatus `gorm:"column:status"`
	Listed        bool                    `gorm:"column:listed"`
	StartDate     *time.Time              `gorm:"column:start_date"`
	LogoURL       *string                 `gorm:"column:logo_url"`
	TopHTMLJa     string                  `gorm:"column:top_html_ja"`
	TopHTMLEn     string                  `gorm:"column:top_html_en"`
	TeamListLabel string                  `gorm:"column:team_list_label"` // teams | entries
	GroupLabel    string                  `gorm:"column:group_label"`     // groups | results
	ShowSchedule  bool                    `gorm:"column:show_schedule"`
	CreatedAt     time.Time               `gorm:"column:created_at"`
	UpdatedAt     time.Time               `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (Tournament) TableName() string { return "tournaments" }

// TournamentBanner は tournament_banners の行。
type TournamentBanner struct {
	ID           int64   `gorm:"column:id;primaryKey"`
	TournamentID int64   `gorm:"column:tournament_id"`
	ImageURL     string  `gorm:"column:image_url"`
	LinkURL      *string `gorm:"column:link_url"`
	SortOrder    int     `gorm:"column:sort_order"`
}

// TableName は GORM に表名を伝える。
func (TournamentBanner) TableName() string { return "tournament_banners" }

// Division は divisions の行。
type Division struct {
	ID                 int64               `gorm:"column:id;primaryKey"`
	TournamentID       int64               `gorm:"column:tournament_id"`
	NameJa             string              `gorm:"column:name_ja"`
	NameEn             string              `gorm:"column:name_en"`
	Kind               domain.DivisionKind `gorm:"column:kind"`
	SortOrder          int                 `gorm:"column:sort_order"`
	ScheduleEnabled    bool                `gorm:"column:schedule_enabled"`
	StandingsPublished bool                `gorm:"column:standings_published"`
	EntryKind          domain.EntryKind    `gorm:"column:entry_kind"`
	StatsKind          domain.StatsKind    `gorm:"column:stats_kind"`
	// ShowSOTG はチームページに SOTG スコア一覧(値は空)を出すか(migrations/00010)
	ShowSOTG bool `gorm:"column:show_sotg"`
	// MenuSuffix は SCORE メニューのディビジョン名の後ろの略号("Ult" など。空なら Ult。00011)
	MenuSuffix string    `gorm:"column:menu_suffix"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (Division) TableName() string { return "divisions" }

// Group は groups の行。BracketLayout は domain.MarshalBracketLayout の出力。
type Group struct {
	ID             int64            `gorm:"column:id;primaryKey"`
	DivisionID     int64            `gorm:"column:division_id"`
	Name           string           `gorm:"column:name"`
	NameEn         string           `gorm:"column:name_en"`
	Kind           domain.GroupKind `gorm:"column:kind"`
	SortOrder      int              `gorm:"column:sort_order"`
	Linked         bool             `gorm:"column:linked"`
	Published      bool             `gorm:"column:published"`
	BracketPattern *string          `gorm:"column:bracket_pattern"`
	BracketPtnName *string          `gorm:"column:bracket_ptn_name"`
	BracketLayout  []byte           `gorm:"column:bracket_layout;type:jsonb"`
	ImageURL       *string          `gorm:"column:image_url"`
	ImageWidth     *int             `gorm:"column:image_width"`
	ImageHeight    *int             `gorm:"column:image_height"`
	CreatedAt      time.Time        `gorm:"column:created_at"`
	UpdatedAt      time.Time        `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (Group) TableName() string { return "groups" }

// Team は teams の行。
type Team struct {
	ID           int64               `gorm:"column:id;primaryKey"`
	TournamentID int64               `gorm:"column:tournament_id"`
	DivisionID   int64               `gorm:"column:division_id"`
	NameFullJa   string              `gorm:"column:name_full_ja"`
	NameShortJa  string              `gorm:"column:name_short_ja"`
	NameFullEn   string              `gorm:"column:name_full_en"`
	NameShortEn  string              `gorm:"column:name_short_en"`
	CommentJa    string              `gorm:"column:comment_ja"`
	CommentEn    string              `gorm:"column:comment_en"`
	BarStyle     domain.TeamBarStyle `gorm:"column:bar_style"`
	SortOrder    int                 `gorm:"column:sort_order"`
	FinalRank    *int                `gorm:"column:final_rank"`
	CreatedAt    time.Time           `gorm:"column:created_at"`
	UpdatedAt    time.Time           `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (Team) TableName() string { return "teams" }

// GroupTeam は group_teams の行。RankOverride の 0 は「空欄で表示」。
type GroupTeam struct {
	GroupID      int64 `gorm:"column:group_id;primaryKey"`
	TeamID       int64 `gorm:"column:team_id;primaryKey"`
	SortOrder    int   `gorm:"column:sort_order"`
	RankOverride *int  `gorm:"column:rank_override"`
}

// TableName は GORM に表名を伝える。
func (GroupTeam) TableName() string { return "group_teams" }

// Player は players の行。RoleJa か RoleEn があればスタッフ。
type Player struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	TeamID    int64     `gorm:"column:team_id"`
	Number    string    `gorm:"column:number"`
	NameJa    string    `gorm:"column:name_ja"`
	NameEn    string    `gorm:"column:name_en"`
	Gender    *string   `gorm:"column:gender"`
	RoleJa    *string   `gorm:"column:role_ja"`
	RoleEn    *string   `gorm:"column:role_en"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (Player) TableName() string { return "players" }

// IsStaff はスタッフ(役割が登録されている)かどうか。
func (p Player) IsStaff() bool { return p.RoleJa != nil || p.RoleEn != nil }

// Match は matches の行。StartTime は "HH:MM"(書き込み)または "HH:MM:SS"(読み取り)。
type Match struct {
	ID                      int64              `gorm:"column:id;primaryKey"`
	DivisionID              int64              `gorm:"column:division_id"`
	GroupID                 *int64             `gorm:"column:group_id"`
	Code                    string             `gorm:"column:code"`
	MatchDate               *time.Time         `gorm:"column:match_date"`
	StartTime               *string            `gorm:"column:start_time"`
	Field                   *string            `gorm:"column:field"` // コート。"A" などもある
	LeftTeamID              *int64             `gorm:"column:left_team_id"`
	RightTeamID             *int64             `gorm:"column:right_team_id"`
	LeftPlaceholderCode     *string            `gorm:"column:left_placeholder_code"`
	LeftPlaceholderOutcome  *domain.Outcome    `gorm:"column:left_placeholder_outcome"`
	RightPlaceholderCode    *string            `gorm:"column:right_placeholder_code"`
	RightPlaceholderOutcome *domain.Outcome    `gorm:"column:right_placeholder_outcome"`
	LeftScore               *int               `gorm:"column:left_score"`
	RightScore              *int               `gorm:"column:right_score"`
	Status                  domain.MatchStatus `gorm:"column:status"`
	HasDetail               bool               `gorm:"column:has_detail"`
	WinnerSide              *domain.Side       `gorm:"column:winner_side"`
	// LeftLabelEn / RightLabelEn は対戦相手の決まり方の英語表記("[A1-Win]" など)。英語版で
	// チームの英語名が空のとき代わりに出す(migrations/00010)
	LeftLabelEn  string `gorm:"column:left_label_en"`
	RightLabelEn string `gorm:"column:right_label_en"`
	// GroupLabel は試合ページのグループ欄の文字。nil ならグループの名前(migrations/00010)
	GroupLabel *string   `gorm:"column:group_label"`
	Note       string    `gorm:"column:note"`
	HideBlocks bool      `gorm:"column:hide_blocks"`
	HideStats  bool      `gorm:"column:hide_stats"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

// TableName は GORM に表名を伝える。
func (Match) TableName() string { return "matches" }

// MatchEvent は match_events の行。
type MatchEvent struct {
	ID         int64            `gorm:"column:id;primaryKey"`
	MatchID    int64            `gorm:"column:match_id"`
	Seq        int              `gorm:"column:seq"`
	ElapsedSec int              `gorm:"column:elapsed_sec"`
	Kind       domain.EventKind `gorm:"column:kind"`
	Side       *domain.Side     `gorm:"column:side"`
	ScorerID   *int64           `gorm:"column:scorer_id"`
	AssistID   *int64           `gorm:"column:assist_id"`
	Callahan   bool             `gorm:"column:callahan"` // キャラハンゴール(goal のときだけ)
	CreatedAt  time.Time        `gorm:"column:created_at"`
}

// TableName は GORM に表名を伝える。
func (MatchEvent) TableName() string { return "match_events" }

// MatchPlayerStat は match_player_stats の行。行があることが「その試合に出場した」ことを表す。
type MatchPlayerStat struct {
	MatchID   int64       `gorm:"column:match_id;primaryKey"`
	PlayerID  int64       `gorm:"column:player_id;primaryKey"`
	Side      domain.Side `gorm:"column:side"`
	Blocks    int         `gorm:"column:blocks"`
	Greatests int         `gorm:"column:greatests"`
}

// TableName は GORM に表名を伝える。
func (MatchPlayerStat) TableName() string { return "match_player_stats" }

// Gallery は galleries の行。
type Gallery struct {
	ID           int64  `gorm:"column:id;primaryKey"`
	TournamentID int64  `gorm:"column:tournament_id"`
	Name         string `gorm:"column:name"`
	NameEn       string `gorm:"column:name_en"`
	SortOrder    int    `gorm:"column:sort_order"`
}

// TableName は GORM に表名を伝える。
func (Gallery) TableName() string { return "galleries" }

// GalleryPhoto は gallery_photos の行。
type GalleryPhoto struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	GalleryID int64  `gorm:"column:gallery_id"`
	ThumbURL  string `gorm:"column:thumb_url"`
	FullURL   string `gorm:"column:full_url"`
	SortOrder int    `gorm:"column:sort_order"`
}

// TableName は GORM に表名を伝える。
func (GalleryPhoto) TableName() string { return "gallery_photos" }
