package store

import (
	"context"
	"fmt"
	"slices"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// SQL 断片の定数。sqlsafety_test が第 1 引数をリテラルか定数に限っているので、使い回すものはここに置く。
const (
	sqlByID           = "id = ?"
	sqlByIDs          = "id IN ?"
	sqlByTournament   = "tournament_id = ?"
	sqlByTournaments  = "tournament_id IN ?"
	sqlByDivision     = "division_id = ?"
	sqlByDivisions    = "division_id IN ?"
	sqlByTeam         = "team_id = ?"
	sqlByTeams        = "team_id IN ?"
	sqlByMatch        = "match_id = ?"
	sqlByMatches      = "match_id IN ?"
	sqlOrderSortID    = "sort_order, id"
	sqlOrderID        = "id"
	sqlOrderEvents    = "elapsed_sec, seq"
	sqlGoalKind       = "kind = ?"
	sqlEventsOfPlayer = "(scorer_id = ? OR assist_id = ?)"
)

// findByID は id の行を 1 つ読む。無ければ domain.NotFound(resource, id) を返す。
func findByID[T any](
	ctx context.Context,
	db *gorm.DB,
	id int64,
	resource domain.Resource,
) (T, error) {
	var row T
	// Take は無い行を ErrRecordNotFound として GORM のログに警告で出すので、404 のたびに
	// ログが汚れないよう Find で読み、件数で判定する
	result := db.WithContext(ctx).Where(sqlByID, id).Limit(1).Find(&row)
	if result.Error != nil {
		return row, fmt.Errorf("find %s [%d]: %w", resource, id, result.Error)
	}
	if result.RowsAffected == 0 {
		return row, domain.NotFound(resource, id)
	}
	return row, nil
}

// findInternal は別の行から参照されている id の行を読む。無ければ FK が壊れているので
// NotFound ではなく内部エラーにする(利用者が指定した ID ではないため)。
func findInternal[T any](ctx context.Context, db *gorm.DB, id int64, what string) (T, error) {
	var row T
	if err := db.WithContext(ctx).Where(sqlByID, id).Take(&row).Error; err != nil {
		return row, fmt.Errorf("load %s %d: %w", what, id, err)
	}
	return row, nil
}

// loadTeams は ids のチームを読み、ID から引ける map にする。ids が空なら問い合わせない。
func loadTeams(ctx context.Context, db *gorm.DB, ids []int64) (map[int64]Team, error) {
	teams := map[int64]Team{}
	if len(ids) == 0 {
		return teams, nil
	}
	var rows []Team
	if err := db.WithContext(ctx).Where(sqlByIDs, ids).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load teams: %w", err)
	}
	for _, team := range rows {
		teams[team.ID] = team
	}
	return teams, nil
}

// loadPlayers は ids の選手を読み、ID から引ける map にする。ids が空なら問い合わせない。
func loadPlayers(ctx context.Context, db *gorm.DB, ids []int64) (map[int64]Player, error) {
	players := map[int64]Player{}
	if len(ids) == 0 {
		return players, nil
	}
	var rows []Player
	if err := db.WithContext(ctx).Where(sqlByIDs, ids).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load players: %w", err)
	}
	for _, player := range rows {
		players[player.ID] = player
	}
	return players, nil
}

// matchTeamIDs は試合が参照するチーム ID を重複なしで返す。
func matchTeamIDs(matches []Match) []int64 {
	var ids idSet
	for _, match := range matches {
		ids.addPtr(match.LeftTeamID)
		ids.addPtr(match.RightTeamID)
	}
	return ids.list
}

// idSet は順序を保つ ID の集合。
type idSet struct {
	seen map[int64]bool
	list []int64
}

func (s *idSet) add(id int64) {
	if s.seen == nil {
		s.seen = map[int64]bool{}
	}
	if !s.seen[id] {
		s.seen[id] = true
		s.list = append(s.list, id)
	}
}

func (s *idSet) addPtr(id *int64) {
	if id != nil {
		s.add(*id)
	}
}

func i18n(ja, en string) domain.I18nText { return domain.I18nText{Ja: ja, En: en} }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// teamRef は表で使うチームの参照(短縮名。clone-spec D 1.3)。
func teamRef(team Team) domain.TeamRef {
	return domain.TeamRef{ID: team.ID, Name: i18n(team.NameShortJa, team.NameShortEn)}
}

// teamRefByID は teams から参照を作る。ID が nil なら nil。map に無い ID はプログラムの誤り
// (呼び出し側が読み込み漏れ)なので、黙って nil にせずエラーにする。
func teamRefByID(teams map[int64]Team, id *int64) (*domain.TeamRef, error) {
	if id == nil {
		return nil, nil
	}
	team, ok := teams[*id]
	if !ok {
		return nil, fmt.Errorf("team %d is referenced but not loaded", *id)
	}
	ref := teamRef(team)
	return &ref, nil
}

func playerRef(player Player) domain.PlayerRef {
	ref := domain.PlayerRef{
		ID:     player.ID,
		Number: player.Number,
		Name:   i18n(player.NameJa, player.NameEn),
	}
	if player.Gender != nil {
		g := domain.Gender(*player.Gender)
		ref.Gender = &g
	}
	return ref
}

func playerRefByID(players map[int64]Player, id *int64) (*domain.PlayerRef, error) {
	if id == nil {
		return nil, nil
	}
	player, ok := players[*id]
	if !ok {
		return nil, fmt.Errorf("player %d is referenced but not loaded", *id)
	}
	ref := playerRef(player)
	return &ref, nil
}

func playerRole(player Player) *domain.I18nText {
	if !player.IsStaff() {
		return nil
	}
	role := i18n(deref(player.RoleJa), deref(player.RoleEn))
	return &role
}

// formatDate は DATE を "YYYY-MM-DD" にする。
func formatDate(match Match) *string {
	if match.MatchDate == nil {
		return nil
	}
	s := match.MatchDate.Format("2006-01-02")
	return &s
}

// formatClock は TIME の "HH:MM:SS" を "HH:MM" にする(秒は出さない。docs/api.md 共通)。
func formatClock(match Match) *string {
	if match.StartTime == nil {
		return nil
	}
	s := *match.StartTime
	if len(s) > len("15:04") {
		s = s[:len("15:04")]
	}
	return &s
}

func matchSide(
	teams map[int64]Team,
	teamID *int64,
	code *string,
	outcome *domain.Outcome,
	score *int,
) (
	domain.MatchSide, error,
) {
	team, err := teamRefByID(teams, teamID)
	if err != nil {
		return domain.MatchSide{}, err
	}
	side := domain.MatchSide{Team: team, Score: score}
	if team == nil && code != nil && outcome != nil {
		side.Placeholder = &domain.Placeholder{Code: *code, Outcome: *outcome}
	}
	return side, nil
}

// toMatchRow は試合の行を日程や結果一覧の 1 行にする。teams に左右のチームが入っていること。
func toMatchRow(match Match, teams map[int64]Team) (domain.MatchRow, error) {
	left, err := matchSide(
		teams,
		match.LeftTeamID,
		match.LeftPlaceholderCode,
		match.LeftPlaceholderOutcome,
		match.LeftScore,
	)
	if err != nil {
		return domain.MatchRow{}, fmt.Errorf("match %d left: %w", match.ID, err)
	}
	right, err := matchSide(
		teams,
		match.RightTeamID,
		match.RightPlaceholderCode,
		match.RightPlaceholderOutcome,
		match.RightScore,
	)
	if err != nil {
		return domain.MatchRow{}, fmt.Errorf("match %d right: %w", match.ID, err)
	}
	left.LabelEn, right.LabelEn = match.LeftLabelEn, match.RightLabelEn
	return domain.MatchRow{
		ID:        match.ID,
		HasDetail: match.HasDetail,
		Code:      match.Code,
		Date:      formatDate(match),
		Time:      formatClock(match),
		Field:     match.Field,
		Status:    match.Status,
		Left:      left,
		Right:     right,
	}, nil
}

// compareMatchTime は試合を既定の並び(日付、時刻、コートの自然順、未定は後ろ、ID)で比べる。
// コートは文字列なので SQL の ORDER BY では自然順にならず、アプリで並べる。
func compareMatchTime(a, b Match) int {
	key := func(match Match) domain.MatchRow {
		return domain.MatchRow{
			ID:    match.ID,
			Date:  formatDate(match),
			Time:  formatClock(match),
			Field: match.Field,
		}
	}
	return domain.CompareMatchRows(key(a), key(b))
}

// sortMatches は試合を compareMatchTime の順に並べる。
func sortMatches(matches []Match) {
	slices.SortFunc(matches, compareMatchTime)
}

func toMatchRows(matches []Match, teams map[int64]Team) ([]domain.MatchRow, error) {
	rows := make([]domain.MatchRow, 0, len(matches))
	for _, match := range matches {
		row, err := toMatchRow(match, teams)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// toResult は成績の集計に使う形にする。
func toResult(match Match) domain.Result {
	result := domain.Result{
		LeftScore: match.LeftScore, RightScore: match.RightScore, Status: match.Status, WinnerSide: match.WinnerSide,
		ResultType: match.ResultType,
	}
	if match.LeftTeamID != nil {
		result.LeftTeamID = *match.LeftTeamID
	}
	if match.RightTeamID != nil {
		result.RightTeamID = *match.RightTeamID
	}
	return result
}

func toResults(matches []Match) []domain.Result {
	out := make([]domain.Result, len(matches))
	for i, match := range matches {
		out[i] = toResult(match)
	}
	return out
}
