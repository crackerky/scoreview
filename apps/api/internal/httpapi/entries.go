package httpapi

import (
	"context"
	"net/http"
	"time"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

// 大会への申込(docs/api.md「申込」)。申込には代表の連絡先が入るので、読むのも変えるのも管理者だけ。
// ハンドラはボディの読み込みと応答の形への変換だけを行い、決まりは entry、記録は store に任せる。

func (s *Server) entryRoutes() {
	const base = "/api/v1/admin"
	ok, created := http.StatusOK, http.StatusCreated
	s.admin("GET "+base+"/tournaments/{id}/entry-questions", adminOnly(), ok, withPathID(s.getEntryQuestions))
	s.admin("PUT "+base+"/tournaments/{id}/entry-questions", adminOnly(), ok, withPathID(s.putEntryQuestions))
	s.admin("GET "+base+"/tournaments/{id}/entries", adminOnly(), ok, withPathID(s.listEntries))
	s.admin("POST "+base+"/tournaments/{id}/entries", adminOnly(), created, withPathID(s.createDeskEntry))
	s.admin("GET "+base+"/tournaments/{id}/entries/export", adminOnly(), ok, withPathID(s.exportEntries))
	s.admin("GET "+base+"/entries/{id}", adminOnly(), ok, withPathID(s.getEntry))
	s.admin("POST "+base+"/entries/{id}/decision", adminOnly(), ok, withPathID(s.decideEntry))
}

type entryQuestionDTO struct {
	ID       int64    `json:"id"`
	Scope    string   `json:"scope"`
	Label    string   `json:"label"`
	Kind     string   `json:"kind"`
	Required bool     `json:"required"`
	Options  []string `json:"options"`
	Answered int      `json:"answered"`
}

type entryQuestionsDTO struct {
	Questions []entryQuestionDTO `json:"questions"`
}

// entryQuestionInputDTO は保存する質問。id が null なら新しく作る。
type entryQuestionInputDTO struct {
	ID       *int64   `json:"id"`
	Scope    string   `json:"scope"`
	Label    string   `json:"label"`
	Kind     string   `json:"kind"`
	Required bool     `json:"required"`
	Options  []string `json:"options"`
}

type entryQuestionsInputDTO struct {
	Questions []entryQuestionInputDTO `json:"questions"`
}

type entryDivisionDTO struct {
	ID   int64   `json:"id"`
	Name i18nDTO `json:"name"`
}

type entrySummaryDTO struct {
	ID          int64            `json:"id"`
	Division    entryDivisionDTO `json:"division"`
	FullName    string           `json:"fullName"`
	ShortName   string           `json:"shortName"`
	RepName     string           `json:"repName"`
	Channel     string           `json:"channel"`
	Status      string           `json:"status"`
	Members     int              `json:"members"`
	Registered  int              `json:"registered"`
	SubmittedAt time.Time        `json:"submittedAt"`
	TeamID      *int64           `json:"teamId"`
	// ScorerChosen はチームの記録係を選んだか(参加が決まったチームの代表が選ぶ)
	ScorerChosen bool `json:"scorerChosen"`
}

type entryListDTO struct {
	Entries []entrySummaryDTO `json:"entries"`
}

type entryMemberDTO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Number     string `json:"number"`
	Registered bool   `json:"registered"`
	PlayerID   *int64 `json:"playerId"`
}

type entryAnswerDTO struct {
	QuestionID int64  `json:"questionId"`
	Value      string `json:"value"`
}

type entryDetailDTO struct {
	entrySummaryDTO
	TournamentID int64      `json:"tournamentId"`
	RepPhone     string     `json:"repPhone"`
	RepEmail     string     `json:"repEmail"`
	ReturnNote   string     `json:"returnNote"`
	InviteCode   string     `json:"inviteCode"`
	InviteURL    string     `json:"inviteUrl"`
	DecidedAt    *time.Time `json:"decidedAt"`
	// ScorerMemberID はチームの記録係(memberList の id)。まだ選んでいなければ null
	ScorerMemberID *int64           `json:"scorerMemberId"`
	MemberList     []entryMemberDTO `json:"memberList"`
	Answers        []entryAnswerDTO `json:"answers"`
}

// entryAdminMemberDTO は運営の画面のメンバー。本人が答えたメンバー向けの質問を足す(本人用のリンクの
// 応答には出さない。ほかのメンバーや代表に見せないため)。
type entryAdminMemberDTO struct {
	entryMemberDTO
	Answers []entryAnswerDTO `json:"answers"`
}

// entryAdminDetailDTO は運営の画面の申込。memberList を答えつきのメンバーにする(外側の欄が、埋め込んだ
// entryDetailDTO の同じ名前の欄より優先して JSON に出る)。
type entryAdminDetailDTO struct {
	entryDetailDTO
	MemberList []entryAdminMemberDTO `json:"memberList"`
}

type entryExportDTO struct {
	Questions []entryQuestionDTO    `json:"questions"`
	Entries   []entryAdminDetailDTO `json:"entries"`
}

type entryMemberInputDTO struct {
	Name   string `json:"name"`
	Number string `json:"number"`
}

type entryDraftDTO struct {
	DivisionID int64                 `json:"divisionId"`
	FullName   string                `json:"fullName"`
	ShortName  string                `json:"shortName"`
	RepName    string                `json:"repName"`
	RepPhone   string                `json:"repPhone"`
	RepEmail   string                `json:"repEmail"`
	Members    []entryMemberInputDTO `json:"members"`
	Answers    []entryAnswerDTO      `json:"answers"`
}

type entryDecisionDTO struct {
	Action string `json:"action"`
	Note   string `json:"note"`
}

func toEntryQuestions(questions []entry.Question) entryQuestionsDTO {
	out := entryQuestionsDTO{Questions: make([]entryQuestionDTO, len(questions))}
	for i, question := range questions {
		options := question.Options
		if options == nil {
			options = []string{}
		}
		out.Questions[i] = entryQuestionDTO{ID: question.ID, Scope: string(question.Scope),
			Label: question.Label, Kind: string(question.Kind), Required: question.Required,
			Options: options, Answered: question.Answered}
	}
	return out
}

func toEntrySummary(summary entry.Summary) entrySummaryDTO {
	return entrySummaryDTO{ID: summary.ID,
		Division: entryDivisionDTO{ID: summary.Division.ID, Name: toI18n(summary.Division.Name)},
		FullName: summary.FullName, ShortName: summary.ShortName, RepName: summary.RepName,
		Channel: string(summary.Channel), Status: string(summary.Status), Members: summary.Members,
		Registered: summary.Registered, SubmittedAt: summary.SubmittedAt, TeamID: summary.TeamID,
		ScorerChosen: summary.ScorerChosen}
}

func toEntryDetail(detail entry.Detail) entryDetailDTO {
	out := entryDetailDTO{entrySummaryDTO: toEntrySummary(detail.Summary), TournamentID: detail.TournamentID,
		RepPhone: detail.RepPhone, RepEmail: detail.RepEmail, ReturnNote: detail.ReturnNote,
		InviteCode: detail.InviteCode, InviteURL: detail.InviteURL, DecidedAt: detail.DecidedAt,
		ScorerMemberID: detail.ScorerMemberID,
		MemberList:     make([]entryMemberDTO, len(detail.MemberList)), Answers: toEntryAnswers(detail.Answers)}
	for i, member := range detail.MemberList {
		out.MemberList[i] = toEntryMember(member)
	}
	return out
}

func toEntryAdminDetail(detail entry.Detail) entryAdminDetailDTO {
	out := entryAdminDetailDTO{entryDetailDTO: toEntryDetail(detail),
		MemberList: make([]entryAdminMemberDTO, len(detail.MemberList))}
	for i, member := range detail.MemberList {
		out.MemberList[i] = entryAdminMemberDTO{entryMemberDTO: toEntryMember(member),
			Answers: toEntryAnswers(member.Answers)}
	}
	return out
}

func toEntryMember(member entry.Member) entryMemberDTO {
	return entryMemberDTO{ID: member.ID, Name: member.Name, Number: member.Number,
		Registered: member.Registered, PlayerID: member.PlayerID}
}

func toEntryAnswers(answers []entry.Answer) []entryAnswerDTO {
	out := make([]entryAnswerDTO, len(answers))
	for i, answer := range answers {
		out[i] = entryAnswerDTO{QuestionID: answer.QuestionID, Value: answer.Value}
	}
	return out
}

func (s *Server) getEntryQuestions(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	questions, err := s.entries.Questions(ctx, id)
	if err != nil {
		return nil, err
	}
	return toEntryQuestions(questions), nil
}

func (s *Server) putEntryQuestions(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	var body entryQuestionsInputDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	questions := make([]entry.Question, len(body.Questions))
	for i, in := range body.Questions {
		questions[i] = entry.Question{Scope: entry.Scope(in.Scope), Label: in.Label, Kind: entry.Kind(in.Kind),
			Required: in.Required, Options: in.Options}
		if in.ID != nil {
			questions[i].ID = *in.ID
		}
	}
	saved, err := s.entries.SaveQuestions(ctx, id, questions)
	if err != nil {
		return nil, err
	}
	return toEntryQuestions(saved), nil
}

func (s *Server) listEntries(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	summaries, err := s.entries.Entries(ctx, id)
	if err != nil {
		return nil, err
	}
	out := entryListDTO{Entries: make([]entrySummaryDTO, len(summaries))}
	for i, summary := range summaries {
		out.Entries[i] = toEntrySummary(summary)
	}
	return out, nil
}

// createDeskEntry は電話や紙で受けた申込を運営が代わりに入れる。
func (s *Server) createDeskEntry(ctx context.Context, id int64, r *http.Request, _ domain.User) (any, error) {
	var body entryDraftDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	draft, err := body.draft()
	if err != nil {
		return nil, err
	}
	detail, err := s.entries.Submit(ctx, id, entry.ChannelDesk, draft)
	if err != nil {
		return nil, err
	}
	return toEntryAdminDetail(detail), nil
}

// draft は申込のボディを entry の中身にする。同じ質問への答えが 2 つあれば検証エラー。
func (d entryDraftDTO) draft() (entry.Draft, error) {
	answers, err := answerMap(d.Answers)
	if err != nil {
		return entry.Draft{}, err
	}
	draft := entry.Draft{DivisionID: d.DivisionID, FullName: d.FullName, ShortName: d.ShortName,
		RepName: d.RepName, RepPhone: d.RepPhone, RepEmail: d.RepEmail,
		Members: make([]entry.MemberDraft, len(d.Members)), Answers: answers}
	for i, member := range d.Members {
		draft.Members[i] = entry.MemberDraft{Name: member.Name, Number: member.Number}
	}
	return draft, nil
}

// answerMap は答えの並びを質問 ID ごとの答えにする。
func answerMap(answers []entryAnswerDTO) (map[int64]string, error) {
	out := make(map[int64]string, len(answers))
	for _, answer := range answers {
		if _, ok := out[answer.QuestionID]; ok {
			return nil, domain.Invalid("answers", "has two answers to the same question")
		}
		out[answer.QuestionID] = answer.Value
	}
	return out, nil
}

func (s *Server) getEntry(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	detail, err := s.entries.Entry(ctx, id)
	if err != nil {
		return nil, err
	}
	return toEntryAdminDetail(detail), nil
}

func (s *Server) decideEntry(ctx context.Context, id int64, r *http.Request, user domain.User) (any, error) {
	var body entryDecisionDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	detail, err := s.entries.Decide(ctx, id, entry.Action(body.Action), body.Note, actorOf(user))
	if err != nil {
		return nil, err
	}
	return toEntryAdminDetail(detail), nil
}

func (s *Server) exportEntries(ctx context.Context, id int64, _ *http.Request, _ domain.User) (any, error) {
	export, err := s.entries.Export(ctx, id)
	if err != nil {
		return nil, err
	}
	out := entryExportDTO{Questions: toEntryQuestions(export.Questions).Questions,
		Entries: make([]entryAdminDetailDTO, len(export.Entries))}
	for i, detail := range export.Entries {
		out.Entries[i] = toEntryAdminDetail(detail)
	}
	return out, nil
}
