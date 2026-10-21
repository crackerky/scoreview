package store

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"
	"unicode"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// 大会形式のプレビューと適用。プレビューは保存せずに Plan と時間割を返し、適用はディビジョンの
// グループと試合を、同じ入力から作った Plan で置き換える。

const (
	sqlStartedInDivision  = "division_id = ? AND status <> ?"
	sqlOtherDivisionsOf   = "tournament_id = ? AND id <> ?"
	sqlCodesOfDivisions   = "division_id IN ?"
	sqlFormatsOfDivisions = "division_id IN ?"
)

// FormatInput は形の指定と、運営者が決めるその他の設定。Params は format.DecodeParams の戻り値、
// Seeds はシード順のチーム(数は形のチーム数と同じ)。Schedule が nil なら時間割を作らない。
type FormatInput struct {
	Template string
	Params   any
	Seeds    []int64
	Tiebreak format.TiebreakPreset
	Forfeit  format.ForfeitScore
	Schedule *ScheduleSetting
}

// FormatPreview は形の指定から作った大会の姿。保存前のプレビューと、適用後の確認に使う。
type FormatPreview struct {
	Title       string
	StartLetter string
	Teams       int
	Stages      []string
	Groups      []PreviewGroup
	Matches     []PreviewMatch
	Courts      []string
	Violations  []ScheduleViolation
	// End は最後の試合の終了時刻("HH:MM")、EndDay はそれが何日目か(1 から)。時間割が無ければ nil
	End    *string
	EndDay *int
	// FitNote は会場の時間に収まらない試合の説明(「17:00 までにあと 6 試合が収まりません」)。
	// 収まっていれば nil
	FitNote     *string
	RuleText    []string
	Advancement []string
	// Ranked は最終順位が付くチーム数、Unique はそのうち同順位でないチーム数
	Ranked int
	Unique int
	// ReplacesGroups と ReplacesMatches は適用すると消える既存のグループと試合の数
	ReplacesGroups  int
	ReplacesMatches int
}

// PreviewGroup はプレビューのグループ。Bracket はトーナメントの格子(閲覧ページと同じ形)。
type PreviewGroup struct {
	Index   int
	Letter  string
	Name    string
	Kind    domain.GroupKind
	Stage   int
	Slots   []PreviewSlot
	Bracket *BracketView
}

// PreviewSlot はグループの枠。Seed はシードの枠ならシード番号。Team はシードに登録されたチーム。
type PreviewSlot struct {
	Position int
	Label    string
	Seed     *int
	Team     *domain.TeamRef
}

// PreviewMatch はプレビューの試合。Date、Time、Field は時間割があれば入る。
type PreviewMatch struct {
	Code  string
	Group int
	Round int
	Left  PreviewSide
	Right PreviewSide
	Date  *string
	Time  *string
	Field *string
}

// PreviewSide は試合の片側。Team はチームが決まっていれば入り、Label は決まり方(「A1」「[E2-勝ち]」)。
type PreviewSide struct {
	Label string
	Team  *domain.TeamRef
}

// ScheduleViolation は時間割の制約違反(format.Violation に運営者向けの説明を付けたもの)。
type ScheduleViolation struct {
	Kind         format.ViolationKind
	Codes        []string
	ShortMinutes int
	Detail       string
}

// formatDraft は入力を検査して作った Plan と時間割。
type formatDraft struct {
	input    FormatInput
	spec     format.Spec
	plan     *format.Plan
	cfg      format.Config
	schedule *format.Schedule
	// scheduleCfg は時間割を組んだ条件(schedule があるときだけ)
	scheduleCfg format.ScheduleConfig
	teams       map[int64]Team
}

func draftFormat(ctx context.Context, db *gorm.DB, division Division, input FormatInput) (*formatDraft, error) {
	start, err := startLetterFor(ctx, db, division)
	if err != nil {
		return nil, err
	}
	spec := format.Spec{Template: input.Template, StartLetter: start, Params: input.Params}
	plan, err := format.NewEngine().Generate(spec)
	if err != nil {
		return nil, err
	}
	if len(input.Seeds) != plan.Teams {
		return nil, domain.Invalid("seeds", fmt.Sprintf("この形は %d チームです(%d チームが指定されました)",
			plan.Teams, len(input.Seeds)))
	}
	if err := checkTeamsInDivision(ctx, db, input.Seeds, division.ID); err != nil {
		return nil, err
	}
	rule, err := format.RuleFor(input.Tiebreak)
	if err != nil {
		return nil, err
	}
	if input.Forfeit.Loser < 0 || input.Forfeit.Winner <= input.Forfeit.Loser {
		return nil, domain.Invalid("forfeit", "winner score must be larger than loser score")
	}
	d := &formatDraft{input: input, spec: spec, plan: plan,
		cfg: format.Config{Tiebreak: rule, Band: format.BandRule(), Forfeit: new(input.Forfeit)}}
	if input.Schedule != nil {
		cfg, err := input.Schedule.Config()
		if err != nil {
			return nil, err
		}
		d.scheduleCfg = cfg
		// 同じ大会の他のディビジョンの試合が使うコートと時刻を避ける
		if cfg.Busy, err = otherDivisionBusy(ctx, db, division, cfg.SlotMinutes); err != nil {
			return nil, err
		}
		if d.schedule, err = format.BuildSchedule(plan, cfg); err != nil {
			return nil, err
		}
	}
	if d.teams, err = loadTeams(ctx, db, input.Seeds); err != nil {
		return nil, err
	}
	return d, nil
}

// startLetterFor はディビジョンの最初のグループ文字。大会内で通しにするため、同じ大会の他の
// ディビジョンの試合 CD と形の指定で使った文字の次にする(R3 5-5)。
func startLetterFor(ctx context.Context, db *gorm.DB, division Division) (string, error) {
	tx := db.WithContext(ctx)
	var others []int64
	if err := tx.Model(&Division{}).Where(sqlOtherDivisionsOf, division.TournamentID, division.ID).
		Pluck("id", &others).Error; err != nil {
		return "", fmt.Errorf("list divisions of tournament %d: %w", division.TournamentID, err)
	}
	if len(others) == 0 {
		return "A", nil
	}
	var codes []string
	if err := tx.Model(&Match{}).Where(sqlCodesOfDivisions, others).Pluck("code", &codes).Error; err != nil {
		return "", fmt.Errorf("list match codes: %w", err)
	}
	var used []string
	for _, code := range codes {
		used = append(used, leadingLetters(code))
	}
	var formats []DivisionFormat
	if err := tx.Where(sqlFormatsOfDivisions, others).Find(&formats).Error; err != nil {
		return "", fmt.Errorf("list formats of tournament: %w", err)
	}
	for _, f := range formats {
		spec, err := specOf(f)
		if err != nil {
			return "", err
		}
		plan, err := format.NewEngine().Generate(spec)
		if err != nil {
			return "", fmt.Errorf("regenerate format of division %d: %w", f.DivisionID, err)
		}
		for _, g := range plan.Groups {
			used = append(used, g.Letter)
		}
	}
	return format.NextLetter(used), nil
}

func leadingLetters(code string) string {
	end := 0
	for end < len(code) && code[end] < unicode.MaxASCII && unicode.IsUpper(rune(code[end])) {
		end++
	}
	return code[:end]
}

// PreviewFormat は保存せずに、形の指定から作った大会の姿を返す。
func PreviewFormat(ctx context.Context, db *gorm.DB, divisionID int64, input FormatInput) (FormatPreview, error) {
	division, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision)
	if err != nil {
		return FormatPreview{}, err
	}
	d, err := draftFormat(ctx, db, division, input)
	if err != nil {
		return FormatPreview{}, err
	}
	preview := d.preview()
	tx := db.WithContext(ctx)
	var groups, matches int64
	if err := tx.Model(&Group{}).Where(sqlByDivision, divisionID).Count(&groups).Error; err != nil {
		return FormatPreview{}, fmt.Errorf("count groups: %w", err)
	}
	if err := tx.Model(&Match{}).Where(sqlByDivision, divisionID).Count(&matches).Error; err != nil {
		return FormatPreview{}, fmt.Errorf("count matches: %w", err)
	}
	preview.ReplacesGroups, preview.ReplacesMatches = int(groups), int(matches)
	return preview, nil
}

func (d *formatDraft) seedTeam(seed int) *domain.TeamRef {
	if seed < 1 || seed > len(d.input.Seeds) {
		return nil
	}
	team, ok := d.teams[d.input.Seeds[seed-1]]
	if !ok {
		return nil
	}
	ref := teamRef(team)
	return &ref
}

func (d *formatDraft) slotTeam(key format.SlotKey) *domain.TeamRef {
	src := d.plan.SlotSource(key)
	if src.Kind != format.SourceSeed {
		return nil
	}
	return d.seedTeam(src.Seed)
}

func (d *formatDraft) side(m format.Match, side format.SideRef) PreviewSide {
	if side.From != nil {
		return PreviewSide{Label: side.From.Label()}
	}
	key := format.SlotKey{Group: m.Group, Position: side.Slot}
	return PreviewSide{Label: d.plan.SlotSource(key).Label(d.plan), Team: d.slotTeam(key)}
}

func (d *formatDraft) preview() FormatPreview {
	p := FormatPreview{Title: format.Title(d.plan), StartLetter: d.spec.StartLetter, Teams: d.plan.Teams,
		RuleText: format.DescribeRule(d.cfg.Tiebreak), Advancement: format.DescribeAdvancement(d.plan)}
	p.Ranked, p.Unique = format.FinalPlaces(d.plan)
	for _, s := range d.plan.Stages {
		p.Stages = append(p.Stages, s.Name)
	}
	for _, g := range d.plan.Groups {
		pg := PreviewGroup{Index: g.Index, Letter: g.Letter, Name: g.Name, Kind: g.Kind, Stage: g.Stage}
		for _, slot := range g.Slots {
			key := format.SlotKey{Group: g.Index, Position: slot.Position}
			ps := PreviewSlot{Position: slot.Position, Label: slot.Source.Label(d.plan), Team: d.slotTeam(key)}
			if slot.Source.Kind == format.SourceSeed {
				seed := slot.Source.Seed
				ps.Seed = &seed
			}
			pg.Slots = append(pg.Slots, ps)
		}
		if g.Layout != nil {
			view := d.gridView(g)
			pg.Bracket = &view
		}
		p.Groups = append(p.Groups, pg)
	}
	assigned := map[string]format.Assignment{}
	if d.schedule != nil {
		p.Courts = d.input.Schedule.Courts
		for _, a := range d.schedule.Assignments {
			assigned[a.Code] = a
		}
		end, day := format.EndOf(d.scheduleCfg, d.schedule)
		clock, endDay := end.Format("15:04"), day+1
		p.End, p.EndDay = &clock, &endDay
		if note := format.OverflowNote(d.scheduleCfg, d.schedule); note != "" {
			p.FitNote = &note
		}
		p.Violations = toViolations(d.schedule.Violations)
	}
	for _, m := range d.plan.Matches {
		pm := PreviewMatch{Code: m.Code, Group: m.Group, Round: m.Round,
			Left: d.side(m, m.Left), Right: d.side(m, m.Right)}
		if a, ok := assigned[m.Code]; ok {
			date, clock, court := a.Start.Format(time.DateOnly), a.Start.Format("15:04"), a.Court
			pm.Date, pm.Time, pm.Field = &date, &clock, &court
		}
		p.Matches = append(p.Matches, pm)
	}
	return p
}

// gridView はトーナメントの格子を閲覧ページの形にする。シードの枠のチームだけを入れる。
func (d *formatDraft) gridView(g format.Group) BracketView {
	view := BracketView{Pattern: g.Pattern, PtnName: g.Name, Columns: g.Layout.Layout.Columns}
	teams := map[[2]int]*domain.TeamRef{}
	for _, ref := range g.Layout.Refs {
		if ref.Game || (ref.Role != domain.SlotLeft && ref.Role != domain.SlotRight) {
			continue
		}
		m, _ := d.plan.MatchByCode(ref.Match)
		side := m.Left
		if ref.Role == domain.SlotRight {
			side = m.Right
		}
		if side.From == nil {
			teams[[2]int{ref.Row, ref.Col}] = d.slotTeam(format.SlotKey{Group: g.Index, Position: side.Slot})
		}
	}
	for r, row := range g.Layout.Layout.Rows {
		cells := make([]BracketCellView, len(row))
		for c, cell := range row {
			cells[c] = BracketCellView{Borders: cell.Borders, Kind: cell.Kind, Text: cell.Text,
				Team: teams[[2]int{r, c}]}
		}
		view.Rows = append(view.Rows, cells)
	}
	for _, ref := range g.Layout.Refs {
		if ref.Game {
			view.Rows[ref.Row][ref.Col].Text = ref.Match
		}
	}
	return view
}

func toViolations(vs []format.Violation) []ScheduleViolation {
	out := make([]ScheduleViolation, len(vs))
	for i, v := range vs {
		out[i] = ScheduleViolation{Kind: v.Kind, Codes: v.Codes, ShortMinutes: v.ShortMinutes,
			Detail: violationText(v)}
	}
	return out
}

// violationText は違反を運営者の言葉にする。
func violationText(v format.Violation) string {
	codes := v.Codes
	switch v.Kind {
	case format.ViolationUnscheduled:
		return fmt.Sprintf("%s の時刻とコートが決まっていません", codes[0])
	case format.ViolationCourtConflict:
		return fmt.Sprintf("%s と %s が同じコートで時間が重なっています(%d 分)", codes[0], codes[1], v.ShortMinutes)
	case format.ViolationTeamRest:
		return fmt.Sprintf("同じチームの %s と %s の間の休みが %d 分足りません", codes[0], codes[1], v.ShortMinutes)
	case format.ViolationDependencyOrder:
		return fmt.Sprintf("%s が、対戦相手の決まる %s より前に始まります", codes[1], codes[0])
	case format.ViolationStageGap:
		return fmt.Sprintf("段の切り替え(%s → %s)の間が %d 分足りません", codes[0], codes[1], v.ShortMinutes)
	}
	return v.Detail
}

// ApplyFormat はディビジョンのグループと試合を、形の指定から作った Plan で置き換え、形の指定を
// 保存する。結果の入った試合(試合中、終了)があれば ErrConflict(大会が始まった後に作り直さない)。
// 1 つのトランザクションで、段、グループ、枠、試合、格子、形の指定を作り、シードの枠を埋める。
func ApplyFormat(
	ctx context.Context,
	db *gorm.DB,
	actor domain.Actor,
	divisionID int64,
	input FormatInput,
) (FormatPreview, error) {
	var preview FormatPreview
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		division, err := findByID[Division](ctx, tx, divisionID, domain.ResourceDivision)
		if err != nil {
			return err
		}
		// 大会 → ディビジョンの順にロックし、結果の入った試合の検査から作り直しまでの間に試合が
		// 始まらないようにする(試合の更新もディビジョンをロックする)
		if err := lockTournament(ctx, tx, division.TournamentID); err != nil {
			return err
		}
		if err := lockDivision(ctx, tx, divisionID); err != nil {
			return err
		}
		var started int64
		if err := tx.Model(&Match{}).Where(sqlStartedInDivision, divisionID, domain.MatchScheduled).
			Count(&started).Error; err != nil {
			return fmt.Errorf("count started matches: %w", err)
		}
		if started > 0 {
			return fmt.Errorf("%w: 結果の入った試合が %d 件あるので、形を作り直せません", domain.ErrConflict, started)
		}
		d, err := draftFormat(ctx, tx, division, input)
		if err != nil {
			return err
		}
		if err := clearDivision(ctx, tx, divisionID); err != nil {
			return err
		}
		if err := d.save(ctx, tx, division); err != nil {
			return err
		}
		changed, err := syncFormatProgress(ctx, tx, divisionID)
		if err != nil {
			return err
		}
		if err := syncBracketSlots(ctx, tx, changed); err != nil {
			return err
		}
		preview = d.preview()
		return auditFormatOp(ctx, tx, division.TournamentID, actor, AuditFormatApply, nil,
			AuditFormat{Template: d.spec.Template, Teams: input.Seeds, Reason: preview.Title})
	})
	return preview, err
}

// clearDivision はディビジョンの試合、グループ、段を消す(枠、格子の対応、順位の指定は連鎖で消える)。
func clearDivision(ctx context.Context, tx *gorm.DB, divisionID int64) error {
	for _, model := range []any{&Match{}, &Group{}, &Stage{}} {
		if err := tx.WithContext(ctx).Where(sqlByDivision, divisionID).Delete(model).Error; err != nil {
			return writeError(fmt.Sprintf("clear %T of division", model), err)
		}
	}
	return nil
}

func (d *formatDraft) save(ctx context.Context, tx *gorm.DB, division Division) error {
	db := tx.WithContext(ctx)
	stageIDs := make([]int64, len(d.plan.Stages))
	for i, s := range d.plan.Stages {
		row := Stage{DivisionID: division.ID, StageIndex: i, Name: s.Name}
		if err := db.Create(&row).Error; err != nil {
			return writeError("create stage", err)
		}
		stageIDs[i] = row.ID
	}
	groupIDs := make([]int64, len(d.plan.Groups))
	slotIDs := map[format.SlotKey]int64{}
	for _, g := range d.plan.Groups {
		id, err := d.saveGroup(ctx, tx, division, g, stageIDs[g.Stage], slotIDs)
		if err != nil {
			return err
		}
		groupIDs[g.Index] = id
	}
	matchIDs := map[string]int64{}
	assigned := map[string]format.Assignment{}
	if d.schedule != nil {
		for _, a := range d.schedule.Assignments {
			assigned[a.Code] = a
		}
	}
	for _, m := range d.plan.Matches {
		id, err := d.saveMatch(ctx, tx, division, m, groupIDs[m.Group], slotIDs, assigned)
		if err != nil {
			return err
		}
		matchIDs[m.Code] = id
	}
	for _, g := range d.plan.Groups {
		if g.Layout != nil {
			if err := saveFormatGrid(ctx, tx, g, groupIDs[g.Index], matchIDs); err != nil {
				return err
			}
		}
	}
	return d.saveSpec(ctx, tx, division)
}

func (d *formatDraft) saveGroup(
	ctx context.Context,
	tx *gorm.DB,
	division Division,
	g format.Group,
	stageID int64,
	slotIDs map[format.SlotKey]int64,
) (int64, error) {
	db := tx.WithContext(ctx)
	// 英語名は日本語名と同じにする(英語のメニューとチームページの組の名前。domain/english.go)
	row := Group{DivisionID: division.ID, Name: g.Name, NameEn: g.Name, Kind: g.Kind, SortOrder: g.Index,
		Linked: true, Published: true}
	if g.Kind == domain.GroupBracket {
		layout, err := emptyBracketLayout()
		if err != nil {
			return 0, err
		}
		pattern, ptnName := g.Pattern, fmt.Sprintf("%s =%s=", g.Bracket, g.Name)
		row.BracketLayout, row.BracketPattern, row.BracketPtnName = layout, &pattern, &ptnName
	}
	if err := db.Create(&row).Error; err != nil {
		return 0, writeError("create group", err)
	}
	if err := db.Model(&formatGroupRow{}).Where(sqlByID, row.ID).
		Updates(map[string]any{"stage_id": stageID, "format_index": g.Index}).Error; err != nil {
		return 0, writeError("link group to stage", err)
	}
	for _, slot := range g.Slots {
		data, err := format.EncodeSource(slot.Source)
		if err != nil {
			return 0, err
		}
		gs := GroupSlot{GroupID: row.ID, Position: slot.Position, Source: data}
		if slot.Source.Kind == format.SourceSeed {
			team := d.input.Seeds[slot.Source.Seed-1]
			gs.TeamID = &team
		}
		if err := db.Create(&gs).Error; err != nil {
			return 0, writeError("create group slot", err)
		}
		slotIDs[format.SlotKey{Group: g.Index, Position: slot.Position}] = gs.ID
	}
	return row.ID, nil
}

func (d *formatDraft) saveMatch(
	ctx context.Context,
	tx *gorm.DB,
	division Division,
	m format.Match,
	groupID int64,
	slotIDs map[format.SlotKey]int64,
	assigned map[string]format.Assignment,
) (int64, error) {
	db := tx.WithContext(ctx)
	row := Match{DivisionID: division.ID, GroupID: &groupID, Code: m.Code,
		Status: domain.MatchScheduled, HasDetail: true}
	if a, ok := assigned[m.Code]; ok {
		day := time.Date(a.Start.Year(), a.Start.Month(), a.Start.Day(), 0, 0, 0, 0, time.UTC)
		clock, court := a.Start.Format("15:04"), a.Court
		row.MatchDate, row.StartTime, row.Field = &day, &clock, &court
	}
	slots := map[string]any{}
	set := func(side format.SideRef, code **string, outcome **domain.Outcome, column string) {
		if side.From != nil {
			c, o := side.From.Code, side.From.Outcome
			*code, *outcome = &c, &o
			return
		}
		slots[column] = slotIDs[format.SlotKey{Group: m.Group, Position: side.Slot}]
	}
	set(m.Left, &row.LeftPlaceholderCode, &row.LeftPlaceholderOutcome, "left_slot_id")
	set(m.Right, &row.RightPlaceholderCode, &row.RightPlaceholderOutcome, "right_slot_id")
	if err := db.Create(&row).Error; err != nil {
		return 0, writeError("create match", err)
	}
	if len(slots) > 0 {
		if err := db.Model(&matchSlotRow{}).Where(sqlByID, row.ID).Updates(slots).Error; err != nil {
			return 0, writeError("link match to slots", err)
		}
	}
	return row.ID, nil
}

// saveFormatGrid はトーナメントの格子に試合 ID を入れて保存し、セルと試合の対応(bracket_slots)を作る。
func saveFormatGrid(
	ctx context.Context,
	tx *gorm.DB,
	g format.Group,
	groupID int64,
	matchIDs map[string]int64,
) error {
	layout := g.Layout.Layout
	rows := make([][]domain.BracketCell, len(layout.Rows))
	for i, row := range layout.Rows {
		rows[i] = slices.Clone(row)
	}
	layout.Rows = rows
	var slots []BracketSlot
	for _, ref := range g.Layout.Refs {
		id := matchIDs[ref.Match]
		if ref.Game {
			cell := &layout.Rows[ref.Row][ref.Col]
			cell.MatchID = &id
			// 推測: 未試合の game セルの文字列は参照サイトで未観測。ApplyBracketPlan と同じく試合 CD を出す
			cell.Text = ref.Match
			continue
		}
		slots = append(slots, BracketSlot{GroupID: groupID, RowIndex: ref.Row, ColIndex: ref.Col,
			MatchID: id, Role: ref.Role})
	}
	data, err := domain.MarshalBracketLayout(layout)
	if err != nil {
		return fmt.Errorf("group %s: %w", g.Letter, err)
	}
	if err := tx.WithContext(ctx).Model(&Group{}).Where(sqlByID, groupID).
		Update("bracket_layout", data).Error; err != nil {
		return writeError("save bracket layout", err)
	}
	if len(slots) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Create(&slots).Error; err != nil {
		return writeError("save bracket slots", err)
	}
	return nil
}

func (d *formatDraft) saveSpec(ctx context.Context, tx *gorm.DB, division Division) error {
	params, err := format.EncodeParams(d.input.Params)
	if err != nil {
		return err
	}
	var schedule []byte
	if d.input.Schedule != nil {
		if schedule, err = json.Marshal(d.input.Schedule); err != nil {
			return fmt.Errorf("encode schedule: %w", err)
		}
	}
	row := DivisionFormat{DivisionID: division.ID, Template: d.spec.Template, Params: params,
		StartLetter: d.spec.StartLetter, Tiebreak: string(d.input.Tiebreak), BandRule: "per_game",
		ForfeitWinnerScore: d.input.Forfeit.Winner, ForfeitLoserScore: d.input.Forfeit.Loser,
		Schedule: schedule, Version: 1, UpdatedAt: time.Now()}
	err = tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "division_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"template": row.Template, "params": row.Params, "start_letter": row.StartLetter,
			"tiebreak": row.Tiebreak, "band_rule": row.BandRule,
			"forfeit_winner_score": row.ForfeitWinnerScore, "forfeit_loser_score": row.ForfeitLoserScore,
			"schedule": row.Schedule, "updated_at": row.UpdatedAt,
			"version": gorm.Expr("division_formats.version + 1"),
		}),
	}).Create(&row).Error
	if err != nil {
		return writeError("save division format", err)
	}
	return nil
}
