package httpapi

import (
	"net/http"
	"time"

	"scoreview/api/internal/auth"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/store"
)

// 本人用のリンクで開く申込とメンバー登録(docs/api.md「LINE からの申込」)。ログインは無く、パスの
// 合言葉で本人を確かめる。web からの内部トークンの確認は ServeHTTP が行う。

func (s *Server) entryLinkRoutes() {
	s.handle("GET /api/v1/entry-links/{token}", s.getEntryLink)
	s.handle("POST /api/v1/entry-links/{token}/team", s.submitEntryByLink)
	s.handle("POST /api/v1/entry-links/{token}/team/{id}", s.resubmitEntryByLink)
	s.handle("POST /api/v1/entry-links/{token}/member", s.joinEntryByLink)
	s.handle("POST /api/v1/entry-links/{token}/team/{id}/scorer", s.chooseScorerByLink)
	s.handle("POST /api/v1/entry-links/{token}/session", s.scorerSession)
	s.handle("POST /api/v1/entry-links/{token}/find", s.chooseTeamByLink)
	s.handle("POST /api/v1/entry-links/{token}/team/{id}/members/{memberId}/remove", s.removeMemberByLink)
}

// entryLinkFindDTO はメンバーがチームを探す画面(大会の、取り消していない申込のチーム)。
type entryLinkFindDTO struct {
	TournamentName string             `json:"tournamentName"`
	Teams          []entryFindTeamDTO `json:"teams"`
	ExpiresAt      time.Time          `json:"expiresAt"`
}

type entryFindTeamDTO struct {
	EntryID   int64            `json:"entryId"`
	FullName  string           `json:"fullName"`
	ShortName string           `json:"shortName"`
	Division  entryDivisionDTO `json:"division"`
}

// entryFindChoiceDTO はチームを探す画面で選んだチーム。
type entryFindChoiceDTO struct {
	EntryID int64 `json:"entryId"`
}

// entryIssuedLinkDTO は発行した本人用のリンクの合言葉(web はそのリンクの画面へ移る)。
type entryIssuedLinkDTO struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (s *Server) chooseTeamByLink(r *http.Request) (any, error) {
	var body entryFindChoiceDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	issued, err := s.entries.ChooseTeamByLink(r.Context(), r.PathValue("token"), body.EntryID)
	if err != nil {
		return nil, err
	}
	return entryIssuedLinkDTO{Token: issued.Token, ExpiresAt: issued.ExpiresAt}, nil
}

// removeMemberByLink は代表が、招待リンクを使わずに登録したメンバーを外す。
func (s *Server) removeMemberByLink(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	memberID, err := pathInt(r, "memberId")
	if err != nil {
		return nil, err
	}
	detail, err := s.entries.RemoveMemberByLink(r.Context(), r.PathValue("token"), id, memberID)
	if err != nil {
		return nil, err
	}
	return toEntryDetail(detail), nil
}

type entryLinkTournamentDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type entryProfileDTO struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type entryPastTeamDTO struct {
	FullName  string                `json:"fullName"`
	ShortName string                `json:"shortName"`
	Members   []entryMemberInputDTO `json:"members"`
}

type entryLinkTeamDTO struct {
	Tournament entryLinkTournamentDTO `json:"tournament"`
	Open       bool                   `json:"open"`
	Divisions  []entryDivisionDTO     `json:"divisions"`
	Questions  []entryQuestionDTO     `json:"questions"`
	Mine       []entryDetailDTO       `json:"mine"`
	Profile    *entryProfileDTO       `json:"profile"`
	PastTeams  []entryPastTeamDTO     `json:"pastTeams"`
	ExpiresAt  time.Time              `json:"expiresAt"`
}

type entryLinkMemberDTO struct {
	TournamentName string             `json:"tournamentName"`
	Team           entrySummaryDTO    `json:"team"`
	Questions      []entryQuestionDTO `json:"questions"`
	Unclaimed      []entryMemberDTO   `json:"unclaimed"`
	Mine           *entryMemberDTO    `json:"mine"`
	MineAnswers    []entryAnswerDTO   `json:"mineAnswers"`
	ProfileName    string             `json:"profileName"`
	ExpiresAt      time.Time          `json:"expiresAt"`
	Found          bool               `json:"found"`
	// Personal は大会が本人に聞く項目、MinePersonal はその初期値(このチームで答えた値か前回の入力)
	Personal     entryPersonalAskDTO `json:"personal"`
	MinePersonal entryPersonalDTO    `json:"minePersonal"`
}

type entryLinkDTO struct {
	Purpose string              `json:"purpose"`
	Team    *entryLinkTeamDTO   `json:"team"`
	Member  *entryLinkMemberDTO `json:"member"`
	Scorer  *entryLinkScorerDTO `json:"scorer"`
	Find    *entryLinkFindDTO   `json:"find"`
}

// entryLinkScorerDTO はチームの記録係のリンクで開く画面(どのチームの記録係か)。
type entryLinkScorerDTO struct {
	TeamName       string    `json:"teamName"`
	TournamentName string    `json:"tournamentName"`
	MemberName     string    `json:"memberName"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

// entryResubmitDTO は差し戻した申込の送り直し。registeredIds は代表がフォームを開いたときに本人が
// 登録していたメンバーの行。
type entryResubmitDTO struct {
	entryDraftDTO
	RegisteredIDs []int64 `json:"registeredIds"`
}

// entryScorerDTO はチームの記録係を選ぶ(memberId が null なら外す)。
type entryScorerDTO struct {
	MemberID *int64 `json:"memberId"`
}

func (s *Server) chooseScorerByLink(r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	var body entryScorerDTO
	if err := decodeBody(r, &body); err != nil {
		return nil, err
	}
	detail, err := s.entries.ChooseScorerByLink(r.Context(), r.PathValue("token"), id, body.MemberID)
	if err != nil {
		return nil, err
	}
	return toEntryDetail(detail), nil
}

// scorerSession はチームの記録係のリンクから、記録の画面に入るセッションを発行する(応答はログインと同じ形)。
func (s *Server) scorerSession(r *http.Request) (any, error) {
	userID, err := s.entries.ScorerSignIn(r.Context(), r.PathValue("token"))
	if err != nil {
		return nil, err
	}
	session, err := auth.StartSession(r.Context(), s.db, userID, s.now())
	if err != nil {
		return nil, err
	}
	ids, err := store.StaffTournamentIDs(r.Context(), s.db, session.User.ID)
	if err != nil {
		return nil, err
	}
	return loginDTO{Token: session.Token, ExpiresAt: formatInstant(session.ExpiresAt),
		User: toAdminUser(session.User, ids)}, nil
}

type entryJoinDTO struct {
	MemberID *int64           `json:"memberId"`
	Name     string           `json:"name"`
	Number   string           `json:"number"`
	Answers  []entryAnswerDTO `json:"answers"`
	// Gender と Birthdate は大会が聞く項目の答え(聞かない大会では省略か空)
	Gender    string `json:"gender"`
	Birthdate string `json:"birthdate"`
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
	if form.Profile != nil {
		out.Profile = &entryProfileDTO{Name: form.Profile.Name, Phone: form.Profile.Phone, Email: form.Profile.Email}
	}
	out.PastTeams = make([]entryPastTeamDTO, len(form.PastTeams))
	for i, team := range form.PastTeams {
		members := make([]entryMemberInputDTO, len(team.Members))
		for j, member := range team.Members {
			members[j] = entryMemberInputDTO{Name: member.Name, Number: member.Number}
		}
		out.PastTeams[i] = entryPastTeamDTO{FullName: team.FullName, ShortName: team.ShortName, Members: members}
	}
	return out
}

func toEntryLinkMember(form entry.MemberForm) *entryLinkMemberDTO {
	out := &entryLinkMemberDTO{TournamentName: form.TournamentName, Team: toEntrySummary(form.Team),
		Questions: toEntryQuestions(form.Questions).Questions,
		Unclaimed: make([]entryMemberDTO, len(form.Unclaimed)), MineAnswers: toEntryAnswers(form.MineAnswers),
		ProfileName: form.ProfileName, ExpiresAt: form.ExpiresAt, Found: form.Found,
		Personal: toEntryPersonalAsk(form.Ask), MinePersonal: toEntryPersonal(form.MinePersonal)}
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
	if view.Scorer != nil {
		out.Scorer = &entryLinkScorerDTO{TeamName: view.Scorer.Team.TeamName,
			TournamentName: view.Scorer.Team.Tournament.Name, MemberName: view.Scorer.Team.MemberName,
			ExpiresAt: view.Scorer.ExpiresAt}
	}
	if view.Find != nil {
		out.Find = &entryLinkFindDTO{TournamentName: view.Find.TournamentName,
			Teams: make([]entryFindTeamDTO, len(view.Find.Teams)), ExpiresAt: view.Find.ExpiresAt}
		for i, team := range view.Find.Teams {
			out.Find.Teams[i] = entryFindTeamDTO{EntryID: team.EntryID, FullName: team.FullName,
				ShortName: team.ShortName, Division: entryDivisionDTO{ID: team.Division.ID,
					Name: toI18n(team.Division.Name)}}
		}
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
		MemberID: body.MemberID, Name: body.Name, Number: body.Number, Answers: answers,
		Gender: body.Gender, Birthdate: body.Birthdate})
	if err != nil {
		return nil, err
	}
	return toEntryLinkMember(form), nil
}
