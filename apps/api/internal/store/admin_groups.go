package store

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"scoreview/api/internal/domain"
)

const (
	sqlTeamsInDivision = "id IN ? AND division_id = ?"
	sqlCodesInDivision = "division_id = ? AND code IN ?"
	// maxCodePrefixLen は試合 CD の接頭辞の長さの上限(参照サイトの CD は "A4" "B12" のような短い文字列)
	maxCodePrefixLen = 8
)

// GroupImage は画像のみの型のグループの画像。
type GroupImage struct {
	URL    string
	Width  *int
	Height *int
}

// GroupFields はグループの編集できる値。Image は kind=image のとき必須。トーナメント表の格子は
// ここでは編集せず、ApplyBracketPlan で生成する。
type GroupFields struct {
	Name      string
	Kind      domain.GroupKind
	SortOrder int
	Linked    bool
	Published bool
	Image     *GroupImage
}

// GroupTeamFields はグループに属するチーム。RankOverride は計算した順位の上書き(0 は空欄)。
type GroupTeamFields struct {
	TeamID       int64
	RankOverride *int
}

// GroupBracket は生成済みのトーナメント表の識別子(参照サイトの GeneratePtn と PtnName)。
type GroupBracket struct {
	Pattern string
	PtnName string
}

// GroupRecord は管理画面のグループ。Teams は group_teams の登録順。
type GroupRecord struct {
	ID         int64
	DivisionID int64
	GroupFields
	Bracket *GroupBracket
	Teams   []GroupTeamFields
}

func (f GroupFields) validate() error {
	if err := domain.CheckRequired("name", f.Name); err != nil {
		return err
	}
	switch f.Kind {
	case domain.GroupPool, domain.GroupBracket:
	case domain.GroupImage:
		if f.Image == nil {
			return domain.Invalid("image", "is required for an image group")
		}
	default:
		return domain.Invalid("kind", "must be pool, bracket or image")
	}
	if f.Image != nil {
		if err := domain.CheckURL("image.url", f.Image.URL); err != nil {
			return err
		}
		for field, value := range map[string]*int{"image.width": f.Image.Width, "image.height": f.Image.Height} {
			if value != nil && *value <= 0 {
				return domain.Invalid(field, "must be positive")
			}
		}
	}
	return nil
}

// emptyBracketLayout は生成前のトーナメント表(groups の CHECK が bracket に格子を要求するため)。
func emptyBracketLayout() ([]byte, error) {
	return domain.MarshalBracketLayout(
		domain.BracketLayout{Columns: 1, Rows: [][]domain.BracketCell{}},
	)
}

func (f GroupFields) apply(row *Group) error {
	row.Name, row.Kind, row.SortOrder = f.Name, f.Kind, f.SortOrder
	row.Linked, row.Published = f.Linked, f.Published
	row.ImageURL, row.ImageWidth, row.ImageHeight = nil, nil, nil
	if f.Image != nil {
		url := f.Image.URL
		row.ImageURL, row.ImageWidth, row.ImageHeight = &url, f.Image.Width, f.Image.Height
	}
	if f.Kind == domain.GroupBracket && row.BracketLayout == nil {
		layout, err := emptyBracketLayout()
		if err != nil {
			return err
		}
		row.BracketLayout = layout
	}
	return nil
}

func toGroupRecord(row Group, members []GroupTeam) GroupRecord {
	record := GroupRecord{
		ID: row.ID, DivisionID: row.DivisionID,
		GroupFields: GroupFields{
			Name:      row.Name,
			Kind:      row.Kind,
			SortOrder: row.SortOrder,
			Linked:    row.Linked,
			Published: row.Published,
		},
		Teams: make([]GroupTeamFields, len(members)),
	}
	if row.ImageURL != nil {
		record.Image = &GroupImage{
			URL:    *row.ImageURL,
			Width:  row.ImageWidth,
			Height: row.ImageHeight,
		}
	}
	if row.BracketPattern != nil || row.BracketPtnName != nil {
		record.Bracket = &GroupBracket{
			Pattern: deref(row.BracketPattern),
			PtnName: deref(row.BracketPtnName),
		}
	}
	for i, member := range members {
		record.Teams[i] = GroupTeamFields{TeamID: member.TeamID, RankOverride: member.RankOverride}
	}
	return record
}

// ListGroups はディビジョンのグループを sort_order, id 順で返す(非公開のグループも含む)。
func ListGroups(ctx context.Context, db *gorm.DB, divisionID int64) ([]GroupRecord, error) {
	if _, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision); err != nil {
		return nil, err
	}
	var rows []Group
	err := db.WithContext(ctx).Omit("bracket_layout").Where(sqlByDivision, divisionID).
		Order(sqlOrderSortID).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list groups of division %d: %w", divisionID, err)
	}
	out := make([]GroupRecord, 0, len(rows))
	for _, row := range rows {
		members, err := groupMembers(ctx, db, row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, toGroupRecord(row, members))
	}
	return out, nil
}

func groupMembers(ctx context.Context, db *gorm.DB, groupID int64) ([]GroupTeam, error) {
	var members []GroupTeam
	err := db.WithContext(ctx).
		Where(sqlByGroup, groupID).
		Order(sqlOrderGroupTeams).
		Find(&members).
		Error
	if err != nil {
		return nil, fmt.Errorf("load teams of group %d: %w", groupID, err)
	}
	return members, nil
}

// GetAdminGroup はグループを返す。無ければ GROUP_ID の NotFound。
func GetAdminGroup(ctx context.Context, db *gorm.DB, id int64) (GroupRecord, error) {
	row, err := findByID[Group](ctx, db, id, domain.ResourceGroup)
	if err != nil {
		return GroupRecord{}, err
	}
	members, err := groupMembers(ctx, db, id)
	if err != nil {
		return GroupRecord{}, err
	}
	return toGroupRecord(row, members), nil
}

// CreateGroup はディビジョンにグループを作る。kind=bracket なら空の格子で作り、ApplyBracketPlan で埋める。
func CreateGroup(
	ctx context.Context,
	db *gorm.DB,
	divisionID int64,
	fields GroupFields,
) (GroupRecord, error) {
	if err := fields.validate(); err != nil {
		return GroupRecord{}, err
	}
	if _, err := findByID[Division](ctx, db, divisionID, domain.ResourceDivision); err != nil {
		return GroupRecord{}, err
	}
	row := Group{DivisionID: divisionID}
	if err := fields.apply(&row); err != nil {
		return GroupRecord{}, err
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return GroupRecord{}, writeError("create group", err)
	}
	return GetAdminGroup(ctx, db, row.ID)
}

// UpdateGroup はグループの値を fields で置き換える(格子とチームは変えない)。
func UpdateGroup(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	fields GroupFields,
) (GroupRecord, error) {
	if err := fields.validate(); err != nil {
		return GroupRecord{}, err
	}
	row, err := findByID[Group](ctx, db, id, domain.ResourceGroup)
	if err != nil {
		return GroupRecord{}, err
	}
	if err := fields.apply(&row); err != nil {
		return GroupRecord{}, err
	}
	if err := db.WithContext(ctx).Save(&row).Error; err != nil {
		return GroupRecord{}, writeError("update group", err)
	}
	return GetAdminGroup(ctx, db, id)
}

// DeleteGroup はグループを消す。試合は残り、group_id が NULL になる(ON DELETE SET NULL)。
func DeleteGroup(ctx context.Context, db *gorm.DB, id int64) error {
	return deleteByID[Group](ctx, db, id, domain.ResourceGroup)
}

// ReplaceGroupTeams はグループのチームを teams(この順で対戦表と順位表に出す)で置き換える。
// チームはグループと同じディビジョンに属すること。
func ReplaceGroupTeams(
	ctx context.Context,
	db *gorm.DB,
	id int64,
	teams []GroupTeamFields,
) (GroupRecord, error) {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, err := findByID[Group](ctx, tx, id, domain.ResourceGroup)
		if err != nil {
			return err
		}
		ids := make([]int64, len(teams))
		for i, team := range teams {
			if team.RankOverride != nil && *team.RankOverride < 0 {
				return domain.Invalid(fmt.Sprintf("[%d].rankOverride", i), "must be 0 or more")
			}
			ids[i] = team.TeamID
		}
		if err := checkTeamsInDivision(ctx, tx, ids, group.DivisionID); err != nil {
			return err
		}
		return replaceGroupTeams(ctx, tx, id, teams)
	})
	if err != nil {
		return GroupRecord{}, err
	}
	return GetAdminGroup(ctx, db, id)
}

func replaceGroupTeams(
	ctx context.Context,
	tx *gorm.DB,
	groupID int64,
	teams []GroupTeamFields,
) error {
	if err := tx.WithContext(ctx).
		Where(sqlByGroup, groupID).
		Delete(&GroupTeam{}).
		Error; err != nil {
		return fmt.Errorf("clear teams of group %d: %w", groupID, err)
	}
	if len(teams) == 0 {
		return nil
	}
	rows := make([]GroupTeam, len(teams))
	for i, team := range teams {
		rows[i] = GroupTeam{
			GroupID:      groupID,
			TeamID:       team.TeamID,
			SortOrder:    i,
			RankOverride: team.RankOverride,
		}
	}
	if err := tx.WithContext(ctx).Create(&rows).Error; err != nil {
		return writeError("set teams of group", err)
	}
	return nil
}

// checkTeamsInDivision は ids が重複なく、すべて divisionID のチームであることを確かめる。
func checkTeamsInDivision(ctx context.Context, db *gorm.DB, ids []int64, divisionID int64) error {
	var unique idSet
	for _, id := range ids {
		unique.add(id)
	}
	if len(unique.list) != len(ids) {
		return domain.Invalid("teamIds", "must not contain duplicates")
	}
	if len(ids) == 0 {
		return nil
	}
	var count int64
	err := db.WithContext(ctx).
		Model(&Team{}).
		Where(sqlTeamsInDivision, ids, divisionID).
		Count(&count).
		Error
	if err != nil {
		return fmt.Errorf("check teams of division %d: %w", divisionID, err)
	}
	if int(count) != len(ids) {
		return domain.Invalid("teamIds", "must be teams of the same division")
	}
	return nil
}

// ApplyBracketPlan はテンプレートの設計図からトーナメント表を生成する。
//
// seeds は表の上から順の枠(シード順)で、数は plan.Teams と同じ。各枠は確定したチーム(TeamID)か、
// 未確定の参照(Placeholder。例 前のグループの試合 "[A1-勝ち]")のどちらか一方。未確定の枠は、参照先の
// 試合が終わると試合更新関数(advanceFromResult)が埋める。試合 CD は codePrefix と
// 試合の番号をつないだもの(例 "B1")で、後続の試合は前の試合を "[B1-勝ち]" の形で参照する
// (matches.*_placeholder_*)。1 つのトランザクションで、試合の行、格子(groups.bracket_layout)、
// セルと試合の対応(bracket_slots)、グループのチームを作る。グループに試合が既にあるか、
// ディビジョン内に同じ CD の試合があれば ErrConflict(作り直しは試合を消してから)。
func ApplyBracketPlan(
	ctx context.Context,
	db *gorm.DB,
	groupID int64,
	plan domain.BracketPlan,
	seeds []MatchSideFields,
	codePrefix string,
) (GroupRecord, error) {
	if strings.TrimSpace(codePrefix) == "" || len(codePrefix) > maxCodePrefixLen {
		return GroupRecord{}, domain.Invalid(
			"codePrefix",
			fmt.Sprintf("must be 1 to %d characters", maxCodePrefixLen),
		)
	}
	if len(seeds) != plan.Teams {
		return GroupRecord{}, domain.Invalid(
			"seeds",
			fmt.Sprintf("the template needs %d seeds", plan.Teams),
		)
	}
	var teamIDs []int64
	for i, seed := range seeds {
		if err := seed.validate(fmt.Sprintf("seeds[%d]", i)); err != nil {
			return GroupRecord{}, err
		}
		if (seed.TeamID == nil) == (seed.Placeholder == nil) {
			return GroupRecord{}, domain.Invalid(
				fmt.Sprintf("seeds[%d]", i),
				"must have either teamId or placeholder",
			)
		}
		if seed.TeamID != nil {
			teamIDs = append(teamIDs, *seed.TeamID)
		}
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, err := findByID[Group](ctx, tx, groupID, domain.ResourceGroup)
		if err != nil {
			return err
		}
		if err := checkTeamsInDivision(ctx, tx, teamIDs, group.DivisionID); err != nil {
			return err
		}
		if err := checkBracketIsFree(ctx, tx, group, plan, codePrefix); err != nil {
			return err
		}
		matchIDs, err := createPlannedMatches(ctx, tx, group, plan, seeds, codePrefix)
		if err != nil {
			return err
		}
		if err := saveBracketLayout(ctx, tx, group, plan, matchIDs, codePrefix); err != nil {
			return err
		}
		members := make([]GroupTeamFields, len(teamIDs))
		for i, teamID := range teamIDs {
			members[i] = GroupTeamFields{TeamID: teamID}
		}
		if err := replaceGroupTeams(ctx, tx, groupID, members); err != nil {
			return err
		}
		return syncBracketGroup(ctx, tx, groupID)
	})
	if err != nil {
		return GroupRecord{}, err
	}
	return GetAdminGroup(ctx, db, groupID)
}

func plannedCode(prefix string, match domain.PlannedMatch) string {
	return fmt.Sprintf("%s%d", prefix, match.Number)
}

func checkBracketIsFree(
	ctx context.Context,
	tx *gorm.DB,
	group Group,
	plan domain.BracketPlan,
	prefix string,
) error {
	var inGroup int64
	if err := tx.WithContext(ctx).
		Model(&Match{}).
		Where(sqlByGroup, group.ID).
		Count(&inGroup).
		Error; err != nil {
		return fmt.Errorf("count matches of group %d: %w", group.ID, err)
	}
	if inGroup > 0 {
		return fmt.Errorf(
			"%w: group already has matches; delete them before generating",
			domain.ErrConflict,
		)
	}
	codes := make([]string, len(plan.Matches))
	for i, match := range plan.Matches {
		codes[i] = plannedCode(prefix, match)
	}
	var taken []string
	err := tx.WithContext(ctx).Model(&Match{}).Where(sqlCodesInDivision, group.DivisionID, codes).
		Pluck("code", &taken).Error
	if err != nil {
		return fmt.Errorf("check match codes: %w", err)
	}
	if len(taken) > 0 {
		return fmt.Errorf("%w: match codes already used in the division: %s",
			domain.ErrConflict, strings.Join(taken, ", "))
	}
	return nil
}

func createPlannedMatches(
	ctx context.Context,
	tx *gorm.DB,
	group Group,
	plan domain.BracketPlan,
	seeds []MatchSideFields,
	prefix string,
) ([]int64, error) {
	ids := make([]int64, len(plan.Matches))
	side := func(entrant domain.PlannedEntrant) (*int64, *string, *domain.Outcome, error) {
		switch {
		case entrant.Seed != nil:
			seed := seeds[*entrant.Seed]
			if seed.Placeholder != nil {
				code, outcome := seed.Placeholder.Code, seed.Placeholder.Outcome
				return nil, &code, &outcome, nil
			}
			return seed.TeamID, nil, nil, nil
		case entrant.From != nil && entrant.From.Match < len(plan.Matches):
			code := plannedCode(prefix, plan.Matches[entrant.From.Match])
			outcome := entrant.From.Outcome
			return nil, &code, &outcome, nil
		default:
			return nil, nil, nil, fmt.Errorf(
				"bracket plan %s has an entrant without seed or source",
				plan.Pattern,
			)
		}
	}
	for i, planned := range plan.Matches {
		row := Match{
			DivisionID: group.DivisionID, GroupID: &group.ID, Code: plannedCode(prefix, planned),
			Status: domain.MatchScheduled, HasDetail: true,
		}
		var err error
		if row.LeftTeamID, row.LeftPlaceholderCode, row.LeftPlaceholderOutcome, err = side(
			planned.Left,
		); err != nil {
			return nil, err
		}
		if row.RightTeamID, row.RightPlaceholderCode, row.RightPlaceholderOutcome, err = side(
			planned.Right,
		); err != nil {
			return nil, err
		}
		if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, writeError("create bracket match", err)
		}
		ids[i] = row.ID
	}
	return ids, nil
}

func saveBracketLayout(
	ctx context.Context,
	tx *gorm.DB,
	group Group,
	plan domain.BracketPlan,
	matchIDs []int64,
	prefix string,
) error {
	layout := plan.Layout
	for _, game := range plan.Games {
		cell := &layout.Rows[game.Row][game.Col]
		id := matchIDs[game.Match]
		cell.MatchID = &id
		// 推測: 未試合の game セルの文字列は参照サイトで未観測。スコアが入るまでは試合 CD を出す
		// (スコアが入れば store.buildBracket がスコアで置き換える)
		cell.Text = plannedCode(prefix, plan.Matches[game.Match])
	}
	data, err := domain.MarshalBracketLayout(layout)
	if err != nil {
		return fmt.Errorf("bracket plan %s: %w", plan.Pattern, err)
	}
	pattern := plan.Pattern
	// 参照サイトの PtnName は "{テンプレ名} =グループ名="(clone-spec C 2.3)
	ptnName := fmt.Sprintf("%s =%s=", plan.Name, group.Name)
	err = tx.WithContext(ctx).Model(&Group{}).Where(sqlByID, group.ID).Updates(map[string]any{
		"kind":             domain.GroupBracket,
		"bracket_pattern":  pattern,
		"bracket_ptn_name": ptnName,
		"bracket_layout":   data,
	}).Error
	if err != nil {
		return writeError("save bracket layout", err)
	}
	slots := make([]BracketSlot, len(plan.Slots))
	for i, slot := range plan.Slots {
		slots[i] = BracketSlot{
			GroupID:  group.ID,
			RowIndex: slot.Row,
			ColIndex: slot.Col,
			MatchID:  matchIDs[slot.Match],
			Role:     slot.Role,
		}
	}
	if len(slots) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Create(&slots).Error; err != nil {
		return writeError("save bracket slots", err)
	}
	return nil
}
