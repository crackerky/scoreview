package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 観戦アプリの「表で見る」(星取表とトーナメント表)。管理者が大会ごとに見せ方を決め、観戦アプリは
// 見せ方と、トーナメント表を描くための組み合わせの形を読む(docs/api.md「観戦アプリの表」)。

const sqlFormatGroupsOfDivisions = "division_id IN ? AND format_index IS NOT NULL AND published = ?"

// TournamentLiveTables は tournament_live_tables の行。
type TournamentLiveTables struct {
	TournamentID int64     `gorm:"column:tournament_id;primaryKey"`
	Shown        bool      `gorm:"column:shown"`
	PoolShown    bool      `gorm:"column:pool_shown"`
	BracketShown bool      `gorm:"column:bracket_shown"`
	ColWDL       bool      `gorm:"column:col_wdl"`
	ColGoals     bool      `gorm:"column:col_goals"`
	ColDiff      bool      `gorm:"column:col_diff"`
	ColPoints    bool      `gorm:"column:col_points"`
	ColRank      bool      `gorm:"column:col_rank"`
	Marks        string    `gorm:"column:marks"`
	OrderBy      string    `gorm:"column:order_by"`
	Live         string    `gorm:"column:live"`
	Highlight    bool      `gorm:"column:highlight"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
	UpdatedBy    *int64    `gorm:"column:updated_by"`
}

// TableName は GORM に表名を伝える。
func (TournamentLiveTables) TableName() string { return "tournament_live_tables" }

// LiveTablesConfig は表の見せ方。Shown が偽なら観戦アプリに「表で見る」の入口ごと出さない。
// Marks は maru / wdl / kanji、Order は entry / rank、Live は score / blank。
type LiveTablesConfig struct {
	Shown        bool
	PoolShown    bool
	BracketShown bool
	Columns      LiveTableColumns
	Marks        string
	Order        string
	Live         string
	Highlight    bool
}

// LiveTableColumns は星取表の右に並べる列。
type LiveTableColumns struct {
	WDL    bool
	Goals  bool
	Diff   bool
	Points bool
	Rank   bool
}

// DefaultLiveTablesConfig は行の無い大会の見せ方(tournament_live_tables の列の既定と同じ)。
func DefaultLiveTablesConfig() LiveTablesConfig {
	return LiveTablesConfig{Shown: true, PoolShown: true, BracketShown: true,
		Columns: LiveTableColumns{WDL: true, Goals: true, Diff: true, Points: true, Rank: true},
		Marks:   "maru", Order: "entry", Live: "score", Highlight: true}
}

func (c LiveTablesConfig) validate() error {
	// 値は tournament_live_tables の CHECK と同じ
	switch c.Marks {
	case "maru", "wdl", "kanji":
	default:
		return domain.Invalid("marks", "must be maru, wdl or kanji")
	}
	switch c.Order {
	case "entry", "rank":
	default:
		return domain.Invalid("order", "must be entry or rank")
	}
	switch c.Live {
	case "score", "blank":
	default:
		return domain.Invalid("live", "must be score or blank")
	}
	return nil
}

func liveTablesConfigOf(row TournamentLiveTables) LiveTablesConfig {
	return LiveTablesConfig{Shown: row.Shown, PoolShown: row.PoolShown, BracketShown: row.BracketShown,
		Columns: LiveTableColumns{WDL: row.ColWDL, Goals: row.ColGoals, Diff: row.ColDiff, Points: row.ColPoints,
			Rank: row.ColRank},
		Marks: row.Marks, Order: row.OrderBy, Live: row.Live, Highlight: row.Highlight}
}

// GetLiveTablesConfig は大会の表の見せ方を返す。決めていなければ既定。大会が無ければ NotFound。
func GetLiveTablesConfig(ctx context.Context, db *gorm.DB, tournamentID int64) (LiveTablesConfig, error) {
	if _, err := findByID[Tournament](ctx, db, tournamentID, domain.ResourceTournament); err != nil {
		return LiveTablesConfig{}, err
	}
	return loadLiveTablesConfig(ctx, db, tournamentID)
}

func loadLiveTablesConfig(ctx context.Context, db *gorm.DB, tournamentID int64) (LiveTablesConfig, error) {
	var rows []TournamentLiveTables
	if err := db.WithContext(ctx).Where(sqlByTournament, tournamentID).Limit(1).Find(&rows).Error; err != nil {
		return LiveTablesConfig{}, fmt.Errorf("load live tables of tournament %d: %w", tournamentID, err)
	}
	if len(rows) == 0 {
		return DefaultLiveTablesConfig(), nil
	}
	return liveTablesConfigOf(rows[0]), nil
}

// SaveLiveTablesConfig は大会の表の見せ方を保存し、保存した見せ方を返す。
func SaveLiveTablesConfig(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	tournamentID int64,
	config LiveTablesConfig,
	now time.Time,
) (LiveTablesConfig, error) {
	if err := config.validate(); err != nil {
		return LiveTablesConfig{}, err
	}
	if _, err := findByID[Tournament](ctx, db, tournamentID, domain.ResourceTournament); err != nil {
		return LiveTablesConfig{}, err
	}
	by := actor.UserID
	row := TournamentLiveTables{TournamentID: tournamentID, Shown: config.Shown, PoolShown: config.PoolShown,
		BracketShown: config.BracketShown, ColWDL: config.Columns.WDL, ColGoals: config.Columns.Goals,
		ColDiff: config.Columns.Diff, ColPoints: config.Columns.Points, ColRank: config.Columns.Rank,
		Marks: config.Marks, OrderBy: config.Order, Live: config.Live, Highlight: config.Highlight,
		UpdatedAt: now, UpdatedBy: &by}
	err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tournament_id"}},
		UpdateAll: true,
	}).Create(&row).Error
	if err != nil {
		return LiveTablesConfig{}, writeError("save live tables", err)
	}
	return loadLiveTablesConfig(ctx, db, tournamentID)
}

// LiveTables は観戦アプリの「表で見る」に要るもの。Divisions は大会の形を決めた部門だけ(部門の順)。
type LiveTables struct {
	Config    LiveTablesConfig
	Divisions []LiveTableDivision
}

// LiveTableDivision は部門の星取表とトーナメント表の形。
type LiveTableDivision struct {
	ID       int64
	Pools    []LiveTablePool
	Brackets []LiveBracket
}

// LiveTablePool はプール(groups.id)と、優勝を争うグループへ進む上位の数(進出圏の色。0 は色なし)。
type LiveTablePool struct {
	GroupID int64
	Advance int
}

// LiveBracket はトーナメントのグループ。Name は運営の言葉の名前(「1〜4 位を決めるトーナメント」)、
// Matches は試合番号順。
type LiveBracket struct {
	GroupID int64
	Name    string
	Matches []LiveBracketMatch
}

// LiveBracketMatch はトーナメントの 1 試合の形。チームと点は日程の行(部門と試合番号)から引く。
// WinnerPlace と LoserPlace はグループ内の順位(0 は次の試合へ進む)。
type LiveBracketMatch struct {
	Code        string
	Round       int
	WinnerPlace int
	LoserPlace  int
	Left        LiveBracketSide
	Right       LiveBracketSide
}

// LiveBracketSide は試合の片側の出どころ。同じグループの前の試合の勝者・敗者なら From、グループの
// 枠なら Seed(「予選 A 組 1位」)。
type LiveBracketSide struct {
	From *format.MatchRef
	Seed string
}

// GetLiveTables は大会の表の見せ方と、形を決めた部門の星取表とトーナメント表の形を返す。公開した
// グループだけを出す。問い合わせの数は部門の数によらない。
func GetLiveTables(ctx context.Context, db *gorm.DB, tournamentID int64) (LiveTables, error) {
	scope, err := loadScopeByTournamentID(ctx, db, tournamentID)
	if err != nil {
		return LiveTables{}, err
	}
	config, err := loadLiveTablesConfig(ctx, db, tournamentID)
	if err != nil {
		return LiveTables{}, err
	}
	out := LiveTables{Config: config, Divisions: []LiveTableDivision{}}
	if len(scope.divisions) == 0 {
		return out, nil
	}
	divisionIDs := make([]int64, len(scope.divisions))
	for i, division := range scope.divisions {
		divisionIDs[i] = division.ID
	}
	var formats []DivisionFormat
	if err := db.WithContext(ctx).Where(sqlFormatsOfDivisions, divisionIDs).Find(&formats).Error; err != nil {
		return LiveTables{}, fmt.Errorf("load formats of tournament %d: %w", tournamentID, err)
	}
	var groups []formatGroupRow
	err = db.WithContext(ctx).Select("id", "division_id", "format_index").
		Where(sqlFormatGroupsOfDivisions, divisionIDs, true).Find(&groups).Error
	if err != nil {
		return LiveTables{}, fmt.Errorf("load format groups of tournament %d: %w", tournamentID, err)
	}
	// 部門ごとの、グループの序数から groups.id
	groupIDs := map[int64]map[int]int64{}
	for _, g := range groups {
		if groupIDs[g.DivisionID] == nil {
			groupIDs[g.DivisionID] = map[int]int64{}
		}
		groupIDs[g.DivisionID][*g.FormatIndex] = g.ID
	}
	byDivision := map[int64]DivisionFormat{}
	for _, row := range formats {
		byDivision[row.DivisionID] = row
	}
	for _, division := range scope.divisions {
		row, ok := byDivision[division.ID]
		if !ok {
			continue
		}
		table, err := liveTableOf(row, groupIDs[division.ID])
		if err != nil {
			return LiveTables{}, err
		}
		out.Divisions = append(out.Divisions, table)
	}
	return out, nil
}

// liveTableOf は保存した形の指定から、部門の星取表とトーナメント表の形を作る。まだ適用していない
// グループ(groups.id の無い序数)は出さない。
func liveTableOf(row DivisionFormat, groupIDs map[int]int64) (LiveTableDivision, error) {
	spec, err := specOf(row)
	if err != nil {
		return LiveTableDivision{}, err
	}
	plan, err := format.NewEngine().Generate(spec)
	if err != nil {
		return LiveTableDivision{}, fmt.Errorf("regenerate format of division %d: %w", row.DivisionID, err)
	}
	out := LiveTableDivision{ID: row.DivisionID, Pools: []LiveTablePool{}, Brackets: []LiveBracket{}}
	for _, g := range plan.Groups {
		id, ok := groupIDs[g.Index]
		if !ok {
			continue
		}
		if g.Kind == domain.GroupPool {
			out.Pools = append(out.Pools, LiveTablePool{GroupID: id, Advance: format.TopAdvance(plan, g.Index)})
			continue
		}
		if g.Kind != domain.GroupBracket {
			continue
		}
		bracket := LiveBracket{GroupID: id, Name: format.OperatorName(g), Matches: []LiveBracketMatch{}}
		side := func(ref format.SideRef) LiveBracketSide {
			if ref.From != nil {
				from := *ref.From
				return LiveBracketSide{From: &from}
			}
			return LiveBracketSide{Seed: format.SeedName(plan, plan.SlotSource(format.SlotKey{Group: g.Index,
				Position: ref.Slot}))}
		}
		for _, m := range plan.GroupMatches(g.Index) {
			bracket.Matches = append(bracket.Matches, LiveBracketMatch{Code: m.Code, Round: m.Round,
				WinnerPlace: m.WinnerPlace, LoserPlace: m.LoserPlace, Left: side(m.Left), Right: side(m.Right)})
		}
		out.Brackets = append(out.Brackets, bracket)
	}
	return out, nil
}
