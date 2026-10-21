package httpapi

import (
	"encoding/json"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
	"scoreview/api/internal/store"
)

// 大会形式の管理用エンドポイントの DTO(docs/api.md「大会の形と進行」)。

type proposalInputDTO struct {
	Teams       int    `json:"teams"`
	Courts      int    `json:"courts"`
	SlotMinutes int    `json:"slotMinutes"`
	Date        string `json:"date"`
	Start       string `json:"start"`
	Sport       string `json:"sport"`
}

type proposalDTO struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Template    string          `json:"template"`
	Params      json.RawMessage `json:"params"`
	Recommended bool            `json:"recommended"`
	Note        string          `json:"note"`
	Matches     int             `json:"matches"`
	End         string          `json:"end"`
	Ranked      int             `json:"ranked"`
	Unique      int             `json:"unique"`
	Violations  int             `json:"violations"`
}

type proposalListDTO struct {
	Proposals []proposalDTO `json:"proposals"`
}

type forfeitDTO struct {
	Winner int `json:"winner"`
	Loser  int `json:"loser"`
}

type scheduleSettingDTO struct {
	Date          string   `json:"date"`
	Start         string   `json:"start"`
	SlotMinutes   int      `json:"slotMinutes"`
	Courts        []string `json:"courts"`
	TeamRestSlots int      `json:"teamRestSlots"`
	StageGapSlots int      `json:"stageGapSlots"`
	End           *string  `json:"end"`
}

type formatInputDTO struct {
	Template string              `json:"template"`
	Params   json.RawMessage     `json:"params"`
	Seeds    []int64             `json:"seeds"`
	Tiebreak string              `json:"tiebreak"`
	Forfeit  *forfeitDTO         `json:"forfeit"`
	Schedule *scheduleSettingDTO `json:"schedule"`
}

func (in formatInputDTO) toInput() (store.FormatInput, error) {
	if len(in.Params) == 0 {
		return store.FormatInput{}, domain.Invalid("params", "is required")
	}
	params, err := format.DecodeParams(in.Template, in.Params)
	if err != nil {
		return store.FormatInput{}, err
	}
	out := store.FormatInput{Template: in.Template, Params: params, Seeds: in.Seeds,
		Tiebreak: format.TiebreakPreset(in.Tiebreak), Forfeit: format.DefaultForfeitScore()}
	if out.Tiebreak == "" {
		out.Tiebreak = format.TiebreakCurrent
	}
	if in.Forfeit != nil {
		out.Forfeit = format.ForfeitScore{Winner: in.Forfeit.Winner, Loser: in.Forfeit.Loser}
	}
	if in.Schedule != nil {
		s := store.ScheduleSetting(*in.Schedule)
		out.Schedule = &s
	}
	return out, nil
}

type previewSlotDTO struct {
	Position int         `json:"position"`
	Label    string      `json:"label"`
	Seed     *int        `json:"seed"`
	Team     *teamRefDTO `json:"team"`
}

type previewGroupDTO struct {
	Index   int              `json:"index"`
	Letter  string           `json:"letter"`
	Name    string           `json:"name"`
	Kind    string           `json:"kind"`
	Stage   int              `json:"stage"`
	Slots   []previewSlotDTO `json:"slots"`
	Bracket *bracketDTO      `json:"bracket"`
}

type previewSideDTO struct {
	Label string      `json:"label"`
	Team  *teamRefDTO `json:"team"`
}

type previewMatchDTO struct {
	Code  string         `json:"code"`
	Group int            `json:"group"`
	Round int            `json:"round"`
	Left  previewSideDTO `json:"left"`
	Right previewSideDTO `json:"right"`
	Date  *string        `json:"date"`
	Time  *string        `json:"time"`
	Field *string        `json:"field"`
}

type violationDTO struct {
	Kind         string   `json:"kind"`
	Codes        []string `json:"codes"`
	ShortMinutes int      `json:"shortMinutes"`
	Detail       string   `json:"detail"`
}

type violationListDTO struct {
	Violations []violationDTO `json:"violations"`
}

type formatPreviewDTO struct {
	Title           string            `json:"title"`
	StartLetter     string            `json:"startLetter"`
	Teams           int               `json:"teams"`
	Stages          []string          `json:"stages"`
	Groups          []previewGroupDTO `json:"groups"`
	Matches         []previewMatchDTO `json:"matches"`
	Courts          []string          `json:"courts"`
	Violations      []violationDTO    `json:"violations"`
	End             *string           `json:"end"`
	RuleText        []string          `json:"ruleText"`
	Advancement     []string          `json:"advancement"`
	Ranked          int               `json:"ranked"`
	Unique          int               `json:"unique"`
	ReplacesGroups  int               `json:"replacesGroups"`
	ReplacesMatches int               `json:"replacesMatches"`
}

func nonNil[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}

func toViolations(vs []store.ScheduleViolation) []violationDTO {
	out := make([]violationDTO, len(vs))
	for i, v := range vs {
		out[i] = violationDTO{Kind: string(v.Kind), Codes: nonNil(v.Codes), ShortMinutes: v.ShortMinutes,
			Detail: v.Detail}
	}
	return out
}

func toPreviewSide(side store.PreviewSide) previewSideDTO {
	return previewSideDTO{Label: side.Label, Team: toTeamRefPtr(side.Team)}
}

func toFormatPreview(p store.FormatPreview) formatPreviewDTO {
	out := formatPreviewDTO{Title: p.Title, StartLetter: p.StartLetter, Teams: p.Teams, Stages: nonNil(p.Stages),
		Groups: []previewGroupDTO{}, Matches: []previewMatchDTO{}, Courts: nonNil(p.Courts),
		Violations: toViolations(p.Violations), End: p.End, RuleText: nonNil(p.RuleText),
		Advancement: nonNil(p.Advancement), Ranked: p.Ranked, Unique: p.Unique,
		ReplacesGroups: p.ReplacesGroups, ReplacesMatches: p.ReplacesMatches}
	for _, g := range p.Groups {
		dto := previewGroupDTO{Index: g.Index, Letter: g.Letter, Name: g.Name, Kind: string(g.Kind), Stage: g.Stage,
			Slots: []previewSlotDTO{}}
		for _, s := range g.Slots {
			dto.Slots = append(dto.Slots, previewSlotDTO{Position: s.Position, Label: s.Label, Seed: s.Seed,
				Team: toTeamRefPtr(s.Team)})
		}
		if g.Bracket != nil {
			b := toBracket(*g.Bracket)
			dto.Bracket = &b
		}
		out.Groups = append(out.Groups, dto)
	}
	for _, m := range p.Matches {
		out.Matches = append(out.Matches, previewMatchDTO{Code: m.Code, Group: m.Group, Round: m.Round,
			Left: toPreviewSide(m.Left), Right: toPreviewSide(m.Right), Date: m.Date, Time: m.Time, Field: m.Field})
	}
	return out
}

func toProposal(p format.Proposal) (proposalDTO, error) {
	params, err := format.EncodeParams(p.Spec.Params)
	if err != nil {
		return proposalDTO{}, err
	}
	return proposalDTO{ID: p.ID, Title: p.Title, Template: p.Spec.Template, Params: params,
		Recommended: p.Recommended, Note: p.Note, Matches: p.Matches, End: p.End.Format("15:04"),
		Ranked: p.Ranked, Unique: p.Unique, Violations: p.Violations}, nil
}

type savedFormatDTO struct {
	DivisionID     int64               `json:"divisionId"`
	TournamentID   int64               `json:"tournamentId"`
	TournamentName string              `json:"tournamentName"`
	DivisionName   string              `json:"divisionName"`
	Title          string              `json:"title"`
	Version        int                 `json:"version"`
	Template       string              `json:"template"`
	Params         json.RawMessage     `json:"params"`
	Tiebreak       string              `json:"tiebreak"`
	Forfeit        forfeitDTO          `json:"forfeit"`
	Schedule       *scheduleSettingDTO `json:"schedule"`
	Seeds          []int64             `json:"seeds"`
}

type savedFormatOneDTO struct {
	Format *savedFormatDTO `json:"format"`
}

type savedFormatListDTO struct {
	Formats []savedFormatDTO `json:"formats"`
}

func toSavedFormat(f store.SavedFormat) (savedFormatDTO, error) {
	params, err := format.EncodeParams(f.Params)
	if err != nil {
		return savedFormatDTO{}, err
	}
	out := savedFormatDTO{DivisionID: f.DivisionID, TournamentID: f.TournamentID, TournamentName: f.TournamentName,
		DivisionName: f.DivisionName, Title: f.Title, Version: f.Version, Template: f.Template, Params: params,
		Tiebreak: string(f.Tiebreak), Forfeit: forfeitDTO{Winner: f.Forfeit.Winner, Loser: f.Forfeit.Loser},
		Seeds: nonNil(f.Seeds)}
	if f.Schedule != nil {
		s := scheduleSettingDTO(*f.Schedule)
		s.Courts = nonNil(s.Courts)
		out.Schedule = &s
	}
	return out, nil
}

// 進行状況

type todoDTO struct {
	Kind    string  `json:"kind"`
	Text    string  `json:"text"`
	GroupID *int64  `json:"groupId"`
	StageID *int64  `json:"stageId"`
	MatchID *int64  `json:"matchId"`
	Court   *string `json:"court"`
	Minutes int     `json:"minutes"`
}

type stageProgressDTO struct {
	ID                int64  `json:"id"`
	Index             int    `json:"index"`
	Name              string `json:"name"`
	Confirmed         bool   `json:"confirmed"`
	ConfirmedUnplayed bool   `json:"confirmedUnplayed"`
	NeedsConfirm      bool   `json:"needsConfirm"`
}

type progressSideDTO struct {
	Label string      `json:"label"`
	Team  *teamRefDTO `json:"team"`
	Score *int        `json:"score"`
}

type progressRankRowDTO struct {
	tallyDTO
	Team       teamRefDTO `json:"team"`
	Rank       int        `json:"rank"`
	Points     int        `json:"points"`
	Reason     string     `json:"reason"`
	Unresolved bool       `json:"unresolved"`
}

type teamConditionDTO struct {
	Team teamRefDTO `json:"team"`
	Text string     `json:"text"`
}

type groupProgressDTO struct {
	ID                int64                `json:"id"`
	Letter            string               `json:"letter"`
	Name              string               `json:"name"`
	Kind              string               `json:"kind"`
	Stage             int                  `json:"stage"`
	State             string               `json:"state"`
	ConfirmedUnplayed bool                 `json:"confirmedUnplayed"`
	Remaining         int                  `json:"remaining"`
	Slots             []progressSideDTO    `json:"slots"`
	Rows              []progressRankRowDTO `json:"rows"`
	Ties              [][]teamRefDTO       `json:"ties"`
	Conditions        []teamConditionDTO   `json:"conditions"`
}

type progressMatchDTO struct {
	ID        int64           `json:"id"`
	Code      string          `json:"code"`
	GroupName string          `json:"groupName"`
	Date      *string         `json:"date"`
	Time      *string         `json:"time"`
	Field     *string         `json:"field"`
	Status    string          `json:"status"`
	Left      progressSideDTO `json:"left"`
	Right     progressSideDTO `json:"right"`
}

type progressFinalDTO struct {
	Rank int        `json:"rank"`
	Team teamRefDTO `json:"team"`
	Tied bool       `json:"tied"`
}

type progressDTO struct {
	DivisionID  int64              `json:"divisionId"`
	Title       string             `json:"title"`
	Version     int                `json:"version"`
	Tiebreak    string             `json:"tiebreak"`
	RuleText    []string           `json:"ruleText"`
	Advancement []string           `json:"advancement"`
	SlotMinutes int                `json:"slotMinutes"`
	Courts      []string           `json:"courts"`
	Now         string             `json:"now"`
	Todo        []todoDTO          `json:"todo"`
	Stages      []stageProgressDTO `json:"stages"`
	Groups      []groupProgressDTO `json:"groups"`
	Live        []progressMatchDTO `json:"live"`
	Next        []progressMatchDTO `json:"next"`
	Final       []progressFinalDTO `json:"final"`
}

type progressMatchListDTO struct {
	Matches []progressMatchDTO `json:"matches"`
}

func toProgressSide(side store.ProgressSide) progressSideDTO {
	return progressSideDTO{Label: side.Label, Team: toTeamRefPtr(side.Team), Score: side.Score}
}

func toProgressMatch(m store.ProgressMatch) progressMatchDTO {
	return progressMatchDTO{ID: m.ID, Code: m.Code, GroupName: m.GroupName, Date: m.Date, Time: m.Time,
		Field: m.Field, Status: string(m.Status), Left: toProgressSide(m.Left), Right: toProgressSide(m.Right)}
}

func toProgressMatches(ms []store.ProgressMatch) []progressMatchDTO {
	out := make([]progressMatchDTO, len(ms))
	for i, m := range ms {
		out[i] = toProgressMatch(m)
	}
	return out
}

func toProgress(p store.FormatProgress, now time.Time) progressDTO {
	out := progressDTO{DivisionID: p.DivisionID, Title: p.Title, Version: p.Version, Tiebreak: string(p.Tiebreak),
		RuleText: nonNil(p.RuleText), Advancement: nonNil(p.Advancement), SlotMinutes: p.SlotMinutes,
		Courts: nonNil(p.Courts), Now: formatInstant(now), Todo: []todoDTO{}, Stages: []stageProgressDTO{},
		Groups: []groupProgressDTO{}, Live: toProgressMatches(p.Live), Next: toProgressMatches(p.Next),
		Final: []progressFinalDTO{}}
	for _, t := range p.Todo {
		out.Todo = append(out.Todo, todoDTO{Kind: string(t.Kind), Text: t.Text, GroupID: t.GroupID,
			StageID: t.StageID, MatchID: t.MatchID, Court: t.Court, Minutes: t.Minutes})
	}
	for _, s := range p.Stages {
		out.Stages = append(out.Stages, stageProgressDTO(s))
	}
	for _, g := range p.Groups {
		dto := groupProgressDTO{ID: g.ID, Letter: g.Letter, Name: g.Name, Kind: string(g.Kind), Stage: g.Stage,
			State: string(g.State), ConfirmedUnplayed: g.ConfirmedUnplayed, Remaining: g.Remaining,
			Slots: []progressSideDTO{}, Rows: []progressRankRowDTO{}, Ties: [][]teamRefDTO{},
			Conditions: []teamConditionDTO{}}
		for _, s := range g.Slots {
			dto.Slots = append(dto.Slots, toProgressSide(s))
		}
		for _, r := range g.Rows {
			dto.Rows = append(dto.Rows, progressRankRowDTO{tallyDTO: toTally(r.Tally), Team: toTeamRef(r.Team),
				Rank: r.Rank, Points: r.Points, Reason: r.Reason, Unresolved: r.Unresolved})
		}
		for _, tie := range g.Ties {
			refs := make([]teamRefDTO, len(tie))
			for i, t := range tie {
				refs[i] = toTeamRef(t)
			}
			dto.Ties = append(dto.Ties, refs)
		}
		for _, c := range g.Conditions {
			dto.Conditions = append(dto.Conditions, teamConditionDTO{Team: toTeamRef(c.Team), Text: c.Text})
		}
		out.Groups = append(out.Groups, dto)
	}
	for _, f := range p.Final {
		out.Final = append(out.Final, progressFinalDTO{Rank: f.Rank, Team: toTeamRef(f.Team), Tied: f.Tied})
	}
	return out
}

// 確定、同順位の指定、訂正、時間割

type confirmInputDTO struct {
	Unplayed bool `json:"unplayed"`
}

type rankDecisionInputDTO struct {
	TeamIDs []int64 `json:"teamIds"`
	Reason  string  `json:"reason"`
}

type bandDecisionInputDTO struct {
	Rank    int     `json:"rank"`
	TeamIDs []int64 `json:"teamIds"`
	Reason  string  `json:"reason"`
}

type scoreInputDTO struct {
	Left  int `json:"left"`
	Right int `json:"right"`
}

type correctionInputDTO struct {
	Score      *scoreInputDTO `json:"score"`
	WinnerSide *domain.Side   `json:"winnerSide"`
	// Downstream は結果のある下流の試合の対戦が変わるときの承認と下流の扱い(keep / reset)。
	// null は承認しない
	Downstream *string `json:"downstream"`
}

func (in correctionInputDTO) toInput() (store.CorrectionInput, error) {
	if in.Score == nil {
		return store.CorrectionInput{}, domain.Invalid("score", "is required")
	}
	if in.WinnerSide != nil && *in.WinnerSide != domain.SideLeft && *in.WinnerSide != domain.SideRight {
		return store.CorrectionInput{}, domain.Invalid("winnerSide", "must be left or right")
	}
	return store.CorrectionInput{Score: domain.Score{Left: in.Score.Left, Right: in.Score.Right},
		WinnerSide: in.WinnerSide}, nil
}

type impactMatchDTO struct {
	MatchID     int64       `json:"matchId"`
	Code        string      `json:"code"`
	GroupName   string      `json:"groupName"`
	BeforeLeft  *teamRefDTO `json:"beforeLeft"`
	BeforeRight *teamRefDTO `json:"beforeRight"`
	AfterLeft   *teamRefDTO `json:"afterLeft"`
	AfterRight  *teamRefDTO `json:"afterRight"`
	HasResult   bool        `json:"hasResult"`
}

type impactDTO struct {
	Lines         []string         `json:"lines"`
	Matches       []impactMatchDTO `json:"matches"`
	NeedsApproval bool             `json:"needsApproval"`
}

func toImpact(i store.CorrectionImpact) impactDTO {
	out := impactDTO{Lines: nonNil(i.Lines), Matches: []impactMatchDTO{}, NeedsApproval: i.NeedsApproval}
	for _, m := range i.Matches {
		out.Matches = append(out.Matches, impactMatchDTO{MatchID: m.MatchID, Code: m.Code, GroupName: m.GroupName,
			BeforeLeft: toTeamRefPtr(m.BeforeLeft), BeforeRight: toTeamRefPtr(m.BeforeRight),
			AfterLeft: toTeamRefPtr(m.AfterLeft), AfterRight: toTeamRefPtr(m.AfterRight), HasResult: m.HasResult})
	}
	return out
}

type shiftInputDTO struct {
	Court   *string `json:"court"`
	Minutes int     `json:"minutes"`
}

type swapInputDTO struct {
	OtherMatchID int64 `json:"otherMatchId"`
}
