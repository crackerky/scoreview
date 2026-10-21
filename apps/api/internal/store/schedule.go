package store

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

// GetTournamentSchedule は大会全体のスケジュールを返す(参照サイトの scv_search_tn)。
// filter.YMD が nil(クエリにキーがない)なら試合のある最終日を既定にする(clone-spec B 3)。
// filter.Team はチームの正式名(日本語)で、左右どちらかがそのチームの試合に絞る。
// 問い合わせは試合数によらず 7 回(大会、ナビ 4、試合、チーム)。
func GetTournamentSchedule(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	filter ScheduleFilter,
) (SchedulePage, error) {
	scope, err := loadScopeByTournamentID(ctx, db, id)
	if err != nil {
		return SchedulePage{}, err
	}
	divisionIDs := make([]int64, len(scope.divisions))
	for i, division := range scope.divisions {
		divisionIDs[i] = division.ID
	}
	var matches []Match
	if len(divisionIDs) > 0 {
		if err := db.WithContext(ctx).Where(sqlByDivisions, divisionIDs).Find(&matches).Error; err != nil {
			return SchedulePage{}, fmt.Errorf("load matches of tournament %d: %w", id, err)
		}
	}
	var teams []Team
	if err := db.WithContext(ctx).Where(sqlByTournament, id).Find(&teams).Error; err != nil {
		return SchedulePage{}, fmt.Errorf("load teams of tournament %d: %w", id, err)
	}
	// チームの選択肢は大会の全チーム(試合の有無によらない)。ディビジョン順、sort_order、id 順
	order := scope.divisionOrder()
	slices.SortStableFunc(teams, func(a, b Team) int {
		if c := cmp.Compare(order[a.DivisionID], order[b.DivisionID]); c != 0 {
			return c
		}
		if c := cmp.Compare(a.SortOrder, b.SortOrder); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	options := make([]ScheduleTeamOption, 0, len(teams))
	for _, team := range teams {
		options = append(
			options,
			ScheduleTeamOption{Value: team.NameFullJa, Name: i18n(team.NameFullJa, team.NameFullEn)},
		)
	}
	teamMatches := func(team Team) bool { return team.NameFullJa == filter.Team }

	defaultYMD := ""
	if filter.YMD == nil {
		defaultYMD = lastDate(matches)
	}
	return buildSchedule(ctx, db, scheduleInput{
		scope: scope, matches: matches, teams: teams, options: options,
		filter: filter, defaultYMD: defaultYMD, teamMatches: teamMatches,
	})
}

// GetDivisionSchedule はディビジョン別のスケジュールを返す(参照サイトの scv_search_div)。
// filter.YMD が nil なら全日程。filter.Team はチーム ID の文字列。
func GetDivisionSchedule(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	filter ScheduleFilter,
) (SchedulePage, error) {
	division, err := findByID[Division](ctx, db, id, domain.ResourceDivision)
	if err != nil {
		return SchedulePage{}, err
	}
	scope, err := loadScopeOf(ctx, db, division.TournamentID)
	if err != nil {
		return SchedulePage{}, err
	}
	var matches []Match
	if err := db.WithContext(ctx).Where(sqlByDivision, id).Find(&matches).Error; err != nil {
		return SchedulePage{}, fmt.Errorf("load matches of division %d: %w", id, err)
	}
	var teams []Team
	if err := db.WithContext(ctx).Where(sqlByDivision, id).Order(sqlOrderSortID).Find(&teams).Error; err != nil {
		return SchedulePage{}, fmt.Errorf("load teams of division %d: %w", id, err)
	}
	options := make([]ScheduleTeamOption, 0, len(teams))
	for _, team := range teams {
		options = append(options, ScheduleTeamOption{
			Value: strconv.FormatInt(team.ID, 10), Name: i18n(team.NameFullJa, team.NameFullEn),
		})
	}
	teamMatches := func(team Team) bool { return strconv.FormatInt(team.ID, 10) == filter.Team }

	page, err := buildSchedule(ctx, db, scheduleInput{
		scope: scope, matches: matches, teams: teams, options: options,
		filter: filter, teamMatches: teamMatches,
	})
	if err != nil {
		return SchedulePage{}, err
	}
	page.Division = &DivisionRef{ID: division.ID, Name: i18n(division.NameJa, division.NameEn)}
	return page, nil
}

type scheduleInput struct {
	scope   tournamentScope
	matches []Match // 範囲内の全試合(絞り込み前)
	teams   []Team  // 範囲内のチーム
	options []ScheduleTeamOption
	filter  ScheduleFilter
	// defaultYMD は filter.YMD が nil のときに使う日付("" は全日程)
	defaultYMD  string
	teamMatches func(Team) bool
}

func buildSchedule(ctx context.Context, db *gorm.DB, in scheduleInput) (SchedulePage, error) {
	teamsByID := make(map[int64]Team, len(in.teams))
	for _, team := range in.teams {
		teamsByID[team.ID] = team
	}
	// 範囲外のチームを参照する試合があっても表示できるように、足りないチームだけ追加で読む
	var missing idSet
	for _, id := range matchTeamIDs(in.matches) {
		if _, ok := teamsByID[id]; !ok {
			missing.add(id)
		}
	}
	extra, err := loadTeams(ctx, db, missing.list)
	if err != nil {
		return SchedulePage{}, err
	}
	for id, team := range extra {
		teamsByID[id] = team
	}

	ymd := in.defaultYMD
	if in.filter.YMD != nil {
		ymd = *in.filter.YMD
	}
	selected := ScheduleSelected{YMD: ymd, Team: in.filter.Team, Field: in.filter.Field}

	divisions := map[int64]domain.ScheduleDivision{}
	for i, division := range in.scope.divisions {
		divisions[division.ID] = domain.ScheduleDivision{
			ID:    division.ID,
			Name:  i18n(division.NameJa, division.NameEn),
			Kind:  division.Kind,
			Order: i,
		}
	}

	page := SchedulePage{
		Frame:    in.scope.frame,
		Dates:    matchDates(in.matches),
		Teams:    in.options,
		Fields:   matchFields(in.matches),
		Selected: selected,
		Sort:     in.filter.Sort,
		Rows:     []domain.ScheduleRow{},
	}
	for _, match := range in.matches {
		if !scheduleMatches(match, ymd, in.filter, teamsByID, in.teamMatches) {
			continue
		}
		row, err := toMatchRow(match, teamsByID)
		if err != nil {
			return SchedulePage{}, err
		}
		division, ok := divisions[match.DivisionID]
		if !ok {
			return SchedulePage{}, fmt.Errorf(
				"match %d belongs to unknown division %d",
				match.ID,
				match.DivisionID,
			)
		}
		page.Rows = append(page.Rows, domain.ScheduleRow{MatchRow: row, Division: division})
	}
	domain.SortSchedule(page.Rows, in.filter.Sort)
	return page, nil
}

// scheduleMatches は絞り込み(日付、チーム、コートの AND。clone-spec B 3)に合う試合かどうか。
func scheduleMatches(
	match Match,
	ymd string,
	filter ScheduleFilter,
	teams map[int64]Team,
	teamMatches func(Team) bool,
) bool {
	if ymd != "" && (match.MatchDate == nil || match.MatchDate.Format("20060102") != ymd) {
		return false
	}
	if filter.Field != "" && (match.Field == nil || *match.Field != filter.Field) {
		return false
	}
	if filter.Team != "" {
		hit := false
		for _, id := range []*int64{match.LeftTeamID, match.RightTeamID} {
			if id == nil {
				continue
			}
			if team, ok := teams[*id]; ok && teamMatches(team) {
				hit = true
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// matchDates は試合のある日付("YYYY-MM-DD")を重複なしの昇順で返す。
func matchDates(matches []Match) []string {
	dates := []string{}
	seen := map[string]bool{}
	for _, match := range matches {
		if d := formatDate(match); d != nil && !seen[*d] {
			seen[*d] = true
			dates = append(dates, *d)
		}
	}
	slices.Sort(dates)
	return dates
}

// lastDate は試合のある最終日を "YYYYMMDD" で返す。日付のある試合がなければ ""(全日程)。
func lastDate(matches []Match) string {
	dates := matchDates(matches)
	if len(dates) == 0 {
		return ""
	}
	return strings.ReplaceAll(dates[len(dates)-1], "-", "")
}

// matchFields は試合で使われたコートの異なり値を自然順で返す(参照サイトのコートの選択肢は
// この異なり値を並べている。取込パーサ担当が実物の HTML で確認)。
func matchFields(matches []Match) []string {
	fields := []string{}
	seen := map[string]bool{}
	for _, match := range matches {
		if match.Field != nil && !seen[*match.Field] {
			seen[*match.Field] = true
			fields = append(fields, *match.Field)
		}
	}
	slices.SortFunc(fields, domain.CompareFields)
	return fields
}
