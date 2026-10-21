package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 保存済みの大会形式から format.Plan と format.State を組み立てて解決する。Plan は保存した
// テンプレートとパラメータから毎回作り直す(Generate は決定的なので、試合 CD とグループの序数が
// 保存した行と一致する)。

const (
	sqlByDivisionOrderIndex = "division_id = ? AND format_index IS NOT NULL"
	sqlOrderStageIndex      = "stage_index"
	sqlSlotsOfGroups        = "group_id IN ?"
	sqlOrderSlotPosition    = "group_id, position"
	sqlDecisionsOf          = "group_id IN ? OR stage_id IN ?"
)

// ScheduleSetting は時間割の設定(division_formats.schedule)。Start は "HH:MM"、End は終了時刻
// (なければ nil)。Date は "YYYY-MM-DD"。
type ScheduleSetting struct {
	Date          string   `json:"date"`
	Start         string   `json:"start"`
	SlotMinutes   int      `json:"slotMinutes"`
	Courts        []string `json:"courts"`
	TeamRestSlots int      `json:"teamRestSlots"`
	StageGapSlots int      `json:"stageGapSlots"`
	End           *string  `json:"end"`
}

// config は format.ScheduleConfig にする。日付と時刻は UTC の値として扱う(試合の日付と時刻は
// タイムゾーンを持たない現地時刻なので、比べるときも同じ扱いにする)。
func (s ScheduleSetting) config() (format.ScheduleConfig, error) {
	day, err := domain.ParseDay("schedule.date", s.Date)
	if err != nil {
		return format.ScheduleConfig{}, err
	}
	start, err := clockOn(day, "schedule.start", s.Start)
	if err != nil {
		return format.ScheduleConfig{}, err
	}
	if s.SlotMinutes <= 0 || s.SlotMinutes > maxSlotMinutes {
		return format.ScheduleConfig{}, domain.Invalid("schedule.slotMinutes",
			fmt.Sprintf("must be 1 to %d", maxSlotMinutes))
	}
	if len(s.Courts) == 0 {
		return format.ScheduleConfig{}, domain.Invalid("schedule.courts", "at least one court is required")
	}
	seen := map[string]bool{}
	for i, court := range s.Courts {
		if court == "" || seen[court] {
			return format.ScheduleConfig{}, domain.Invalid(fmt.Sprintf("schedule.courts[%d]", i),
				"must be unique and not empty")
		}
		seen[court] = true
	}
	cfg := format.ScheduleConfig{Start: start, SlotMinutes: s.SlotMinutes, Courts: s.Courts,
		TeamRestSlots: s.TeamRestSlots, StageGapSlots: s.StageGapSlots}
	if s.End != nil {
		end, err := clockOn(day, "schedule.end", *s.End)
		if err != nil {
			return format.ScheduleConfig{}, err
		}
		if !end.After(start) {
			// 終了が開始より前だと、置ける枠の数が負になり「制限なし」と区別できない
			return format.ScheduleConfig{}, domain.Invalid("schedule.end", "must be after schedule.start")
		}
		cfg.End = end
	}
	return cfg, nil
}

// maxSlotMinutes は 1 試合の枠の長さの上限(参照サイトの最長 90 分に余裕を持たせた値)。
const maxSlotMinutes = 240

func clockOn(day time.Time, field, clock string) (time.Time, error) {
	if err := domain.CheckClock(field, clock); err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, domain.Invalid(field, "must be HH:MM")
	}
	return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0,
		time.UTC), nil
}

// formatState は保存済みの大会形式と、それを解決した結果。
type formatState struct {
	division Division
	row      DivisionFormat
	spec     format.Spec
	plan     *format.Plan
	cfg      format.Config
	schedule *ScheduleSetting
	// groups はグループの序数ごとの行、groupIndex は groups.id から序数
	groups     map[int]formatGroupRow
	groupIndex map[int64]int
	stages     map[int]Stage
	slots      map[format.SlotKey]GroupSlot
	matches    map[string]Match
	decisions  []GroupRankDecision
	state      format.State
	res        *format.Resolution
}

// configFor は division_formats の行から順位の設定を作る。
func configFor(row DivisionFormat) (format.Config, error) {
	rule, err := format.RuleFor(format.TiebreakPreset(row.Tiebreak))
	if err != nil {
		return format.Config{}, err
	}
	return format.Config{Tiebreak: rule, Band: format.BandRule(),
		Forfeit: &format.ForfeitScore{Winner: row.ForfeitWinnerScore, Loser: row.ForfeitLoserScore}}, nil
}

// specOf は division_formats の行から Spec を作る。
func specOf(row DivisionFormat) (format.Spec, error) {
	params, err := format.DecodeParams(row.Template, row.Params)
	if err != nil {
		return format.Spec{}, fmt.Errorf("division %d format: %w", row.DivisionID, err)
	}
	return format.Spec{Template: row.Template, StartLetter: row.StartLetter, Params: params}, nil
}

// findFormat はディビジョンの形の指定を読む。無ければ ok=false。
func findFormat(ctx context.Context, db *gorm.DB, divisionID int64) (DivisionFormat, bool, error) {
	var rows []DivisionFormat
	err := db.WithContext(ctx).Where(sqlByDivision, divisionID).Limit(1).Find(&rows).Error
	if err != nil {
		return DivisionFormat{}, false, fmt.Errorf("load format of division %d: %w", divisionID, err)
	}
	if len(rows) == 0 {
		return DivisionFormat{}, false, nil
	}
	return rows[0], true, nil
}

// loadFormatState は保存済みの大会形式を読み、現在の結果と確定状態で解決する。形の指定が無い
// ディビジョンなら ok=false。
func loadFormatState(ctx context.Context, db *gorm.DB, divisionID int64) (*formatState, bool, error) {
	row, ok, err := findFormat(ctx, db, divisionID)
	if err != nil || !ok {
		return nil, ok, err
	}
	fs := &formatState{row: row, groups: map[int]formatGroupRow{}, groupIndex: map[int64]int{},
		stages: map[int]Stage{}, slots: map[format.SlotKey]GroupSlot{}, matches: map[string]Match{}}
	if fs.division, err = findInternal[Division](ctx, db, divisionID, "division"); err != nil {
		return nil, false, err
	}
	if fs.spec, err = specOf(row); err != nil {
		return nil, false, err
	}
	if fs.cfg, err = configFor(row); err != nil {
		return nil, false, err
	}
	// 不戦勝は記録したスコアで数え、スコアの無い不戦勝だけに規定スコア(記録と同じ値。
	// regulationForfeitScore)を使う
	setting, err := regulationForfeitScore(ctx, db, fs.division)
	if err != nil {
		return nil, false, err
	}
	fs.cfg.Forfeit = nil
	if setting != nil {
		fs.cfg.Forfeit = &format.ForfeitScore{Winner: setting.Winner, Loser: setting.Loser}
	}
	if fs.plan, err = format.NewEngine().Generate(fs.spec); err != nil {
		return nil, false, fmt.Errorf("regenerate format of division %d: %w", divisionID, err)
	}
	if row.Schedule != nil {
		var setting ScheduleSetting
		if err := json.Unmarshal(row.Schedule, &setting); err != nil {
			return nil, false, fmt.Errorf("division %d schedule: %w", divisionID, err)
		}
		fs.schedule = &setting
	}
	if err := fs.loadRows(ctx, db); err != nil {
		return nil, false, err
	}
	fs.state = fs.buildState()
	if fs.res, err = format.Resolve(fs.plan, fs.cfg, fs.state); err != nil {
		return nil, false, fmt.Errorf("resolve division %d: %w", divisionID, err)
	}
	return fs, true, nil
}

func (fs *formatState) loadRows(ctx context.Context, db *gorm.DB) error {
	tx := db.WithContext(ctx)
	id := fs.division.ID
	var stages []Stage
	if err := tx.Where(sqlByDivision, id).Order(sqlOrderStageIndex).Find(&stages).Error; err != nil {
		return fmt.Errorf("load stages of division %d: %w", id, err)
	}
	stageIDs := []int64{}
	for _, s := range stages {
		fs.stages[s.StageIndex] = s
		stageIDs = append(stageIDs, s.ID)
	}
	var groups []formatGroupRow
	if err := tx.Where(sqlByDivisionOrderIndex, id).Order(sqlOrderID).Find(&groups).Error; err != nil {
		return fmt.Errorf("load format groups of division %d: %w", id, err)
	}
	groupIDs := []int64{}
	for _, g := range groups {
		fs.groups[*g.FormatIndex] = g
		fs.groupIndex[g.ID] = *g.FormatIndex
		groupIDs = append(groupIDs, g.ID)
	}
	var slots []GroupSlot
	if err := tx.Where(sqlSlotsOfGroups, groupIDs).Order(sqlOrderSlotPosition).Find(&slots).Error; err != nil {
		return fmt.Errorf("load slots of division %d: %w", id, err)
	}
	for _, s := range slots {
		key := format.SlotKey{Group: fs.groupIndex[s.GroupID], Position: s.Position}
		// Plan は保存したテンプレートとパラメータから毎回作り直すので、エンジンの生成が変わると進行中の
		// 大会の形が黙って変わる。保存した出どころと照合し、食い違えば進めない
		if !fs.plan.HasSlot(key) || !format.SourceEquals(fs.plan.SlotSource(key), s.Source) {
			return fmt.Errorf("%w: 保存した大会の形(グループ %d の枠 %d)が、今のエンジンが作る形と食い違います。"+
				"形を適用し直してください", domain.ErrConflict, key.Group, key.Position)
		}
		fs.slots[key] = s
	}
	var matches []Match
	if err := tx.Where(sqlByDivision, id).Order(sqlOrderID).Find(&matches).Error; err != nil {
		return fmt.Errorf("load matches of division %d: %w", id, err)
	}
	for _, m := range matches {
		if _, ok := fs.plan.MatchByCode(m.Code); ok {
			fs.matches[m.Code] = m
		}
	}
	if err := tx.Where(sqlDecisionsOf, groupIDs, stageIDs).Order(sqlOrderID).
		Find(&fs.decisions).Error; err != nil {
		return fmt.Errorf("load rank decisions of division %d: %w", id, err)
	}
	return nil
}

// formatResultOf は試合の行を順位計算の結果にする。結果の種類(matches.result_type)は format の
// 種類に対応させる(不成立 no_contest は format の void)。Played は行の左右のチームで、勝者と敗者は
// このチームで決まる(計算上の対戦チームと食い違えば、エンジンは結果を下流へ流さない)。
func formatResultOf(m Match) format.Result {
	kind := format.ResultNormal
	switch m.ResultType {
	case domain.ResultForfeit:
		kind = format.ResultForfeit
	case domain.ResultDoubleForfeit:
		kind = format.ResultDoubleForfeit
	case domain.ResultNoContest:
		kind = format.ResultVoid
	case domain.ResultAbandoned:
		kind = format.ResultAbandoned
	case domain.ResultUnplayed:
		kind = format.ResultUnplayed
	}
	played := format.PlayedTeams{}
	if m.LeftTeamID != nil {
		played.Left = *m.LeftTeamID
	}
	if m.RightTeamID != nil {
		played.Right = *m.RightTeamID
	}
	return format.Result{Status: m.Status, Type: kind, LeftScore: m.LeftScore,
		RightScore: m.RightScore, Winner: m.WinnerSide, Played: &played}
}

func (fs *formatState) buildState() format.State {
	st := format.State{
		Seeds:           make([]int64, fs.plan.Teams),
		Results:         map[string]format.Result{},
		ConfirmedGroups: map[int]bool{},
		ConfirmedStages: map[int]bool{},
		Manual:          map[format.SlotKey]int64{},
		RankDecisions:   map[int][]format.Decision{},
		BandDecisions:   map[format.BandKey][]format.Decision{},
	}
	for key, slot := range fs.slots {
		src := fs.plan.SlotSource(key)
		if slot.TeamID == nil {
			continue
		}
		switch src.Kind {
		case format.SourceSeed:
			if src.Seed >= 1 && src.Seed <= len(st.Seeds) {
				st.Seeds[src.Seed-1] = *slot.TeamID
			}
		case format.SourceManual:
			st.Manual[key] = *slot.TeamID
		}
	}
	for code, m := range fs.matches {
		if m.Status != domain.MatchScheduled {
			st.Results[code] = formatResultOf(m)
		}
	}
	for index, g := range fs.groups {
		if g.ConfirmedAt != nil {
			st.ConfirmedGroups[index] = true
		}
	}
	stageIndex := map[int64]int{}
	for index, s := range fs.stages {
		stageIndex[s.ID] = index
		if s.ConfirmedAt != nil {
			st.ConfirmedStages[index] = true
		}
	}
	for _, d := range fs.decisions {
		decision, ok := decodeDecision(d)
		if !ok {
			continue
		}
		switch {
		case d.GroupID != nil:
			index := fs.groupIndex[*d.GroupID]
			st.RankDecisions[index] = append(st.RankDecisions[index], decision)
		case d.StageID != nil && d.BandRank != nil:
			key := format.BandKey{Stage: stageIndex[*d.StageID], Rank: *d.BandRank}
			st.BandDecisions[key] = append(st.BandDecisions[key], decision)
		}
	}
	return st
}

// decodeDecision は順位の指定の行を format.Decision にする。basis の無い行(00015 より前の指定)は、
// どの成績に対する指定か分からないので使わない(ok=false。運営者が決め直す)。
func decodeDecision(row GroupRankDecision) (format.Decision, bool) {
	var d format.Decision
	if row.Basis == nil {
		return d, false
	}
	// 書き込みは saveDecision だけで、JSON の形は決まっている
	if json.Unmarshal(row.TeamIDs, &d.Order) != nil || json.Unmarshal(row.Basis, &d.Basis) != nil {
		return d, false
	}
	d.Method = row.Reason
	return d, true
}
