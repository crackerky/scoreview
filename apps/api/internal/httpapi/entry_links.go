package httpapi

import (
	"net/http"
	"time"

	"scoreview/api/internal/entry"
)

// 本人用のリンクで開く申込とメンバー登録(docs/api.md「LINE からの申込」)。ログインは無く、パスの
// 合言葉で本人を確かめる。web からの内部トークンの確認は ServeHTTP が行う。

func (s *Server) entryLinkRoutes() {
	s.handle("GET /api/v1/entry-links/{token}", s.getEntryLink)
	s.handle("POST /api/v1/entry-links/{token}/team", s.submitEntryByLink)
	s.handle("POST /api/v1/entry-links/{token}/team/{id}", s.resubmitEntryByLink)
	s.handle("POST /api/v1/entry-links/{token}/member", s.joinEntryByLink)
}

type entryLinkTournamentDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type entryLinkTeamDTO struct {
	Tournament entryLinkTournamentDTO `json:"tournament"`
	Open       bool                   `json:"open"`
	Divisions  []entryDivisionDTO     `json:"divisions"`
	Questions  []entryQuestionDTO     `json:"questions"`
	Mine       []entryDetailDTO       `json:"mine"`
	ExpiresAt  time.Time              `json:"expiresAt"`
}

type entryLinkMemberDTO struct {
	TournamentName string             `json:"tournamentName"`
	Team           entrySummaryDTO    `json:"team"`
	Questions      []entryQuestionDTO `json:"questions"`
	Unclaimed      []entryMemberDTO   `json:"unclaimed"`
	Mine           *entryMemberDTO    `json:"mine"`
	MineAnswers    []entryAnswerDTO   `json:"mineAnswers"`
	ExpiresAt      time.Time          `json:"expiresAt"`
}

type entryLinkDTO struct {
	Purpose string              `json:"purpose"`
	Team    *entryLinkTeamDTO   `json:"team"`
	Member  *entryLinkMemberDTO `json:"member"`
}

// entryResubmitDTO は差し戻した申込の送り直し。registeredIds は代表がフォームを開いたときに本人が
// 登録していたメンバーの行。
type entryResubmitDTO struct {
	entryDraftDTO
	RegisteredIDs []int64 `json:"registeredIds"`
}

type entryJoinDTO struct {
	MemberID *int64           `json:"memberId"`
	Name     string           `json:"name"`
	Number   string           `json:"number"`
	Answers  []entryAnswerDTO `json:"answers"`
}

func toEntryLinkTeam(form entry.TeamForm) *entryLinkTeamDTO {
	out := &entryLinkTeamDTO{Tournament: entryLinkTournamentDTO{ID: form.Tournament.ID, Name: form.Tournament.Name},
		Open: form.Open, Divisions: make([]entryDivisionDTO, len(form.Divisions)),
		Questions: toEntryQuestions(form.Questions).Questions, Mine: make([]entryDetailDTO, len(form.Mine)),
		ExpiresAt: form.ExpiresAt}
	for i, division := range form.Divisions {
		out.Divisions[i] = entryDivisionDTO{ID: division.ID, Name: toI18n(division.Name)}
	}
	for i, detail := range form.Mine {
		out.Mine[i] = toEntryDetail(detail)
	}
	return out
}

func toEntryLinkMember(form entry.MemberForm) *entryLinkMemberDTO {
	out := &entryLinkMemberDTO{TournamentName: form.TournamentName, Team: toEntrySummary(form.Team),
		Questions: toEntryQuestions(form.Questions).Questions,
		Unclaimed: make([]entryMemberDTO, len(form.Unclaimed)), MineAnswers: toEntryAnswers(form.MineAnswers),
		ExpiresAt: form.ExpiresAt}
	for i, member := range form.Unclaimed {
		out.Unclaimed[i] = toEntryMember(member)
	}
	if form.Mine != nil {
		mine := toEntryMember(*form.Mine)
		out.Mine = &mine
	}
	return out
}

func (s *Server) getEntryLink(r *http.Request) (any, error) {
	view, err := s.entries.LinkView(r.Context(), r.PathValue("token"))
	if err != nil {
		return nil, err
	}
	out := entryLinkDTO{Purpose: string(view.Purpose)}
	if view.Team != nil {
		out.Team = toEntryLinkTeam(*view.Team)
	}
	if view.Member != nil {
		out.Member = toEntryLinkMember(*view.Member)
	}
	return out, nil
}

func (s *Server) submitEntryByLink(r *http.Request) (any, error) {
	var body entryDraftDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	draft, err := body.draft()
	if err != nil {
		return nil, err
	}
	detail, err := s.entries.SubmitByLink(r.Context(), r.PathValue("token"), draft)
	if err != nil {
		return nil, err
	}
	return toEntryDetail(detail), nil
}

// resubmitEntryByLink は差し戻された申込を、代表が直して送り直す。
func (s *Server) resubmitEntryByLink(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	var body entryResubmitDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	draft, err := body.draft()
	if err != nil {
		return nil, err
	}
	detail, err := s.entries.ResubmitByLink(r.Context(), r.PathValue("token"), id, draft, body.RegisteredIDs)
	if err != nil {
		return nil, err
	}
	return toEntryDetail(detail), nil
}

func (s *Server) joinEntryByLink(r *http.Request) (any, error) {
	var body entryJoinDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	answers, err := answerMap(body.Answers)
	if err != nil {
		return nil, err
	}
	form, err := s.entries.JoinByLink(r.Context(), r.PathValue("token"), entry.JoinDraft{
		MemberID: body.MemberID, Name: body.Name, Number: body.Number, Answers: answers})
	if err != nil {
		return nil, err
	}
	return toEntryLinkMember(form), nil
}
