package entry

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"scoreview/api/internal/domain"
)

// Service は Store で記録を読み書きし、申込の決まりを当てる Desk と Links。
type Service struct {
	store    Store
	now      func() time.Time
	random   io.Reader
	invite   func(code string) string
	notifier Notifier
	report   func(error)
}

var (
	_ Desk  = (*Service)(nil)
	_ Links = (*Service)(nil)
)

// Option は Service の構成を変える Functional Option。
type Option func(*Service)

// WithClock は現在時刻の取り方を差し替える(テスト用)。
func WithClock(now func() time.Time) Option {
	return func(s *Service) { s.now = now }
}

// WithRandom は招待コードの乱数の元を差し替える(テスト用。本番は crypto/rand)。
func WithRandom(random io.Reader) Option {
	return func(s *Service) { s.random = random }
}

// WithInviteURL は招待コードからメンバーの招待リンクを作る関数を渡す(公式アカウントのベーシック ID を
// 知っている LINE の層が作る)。渡さなければ招待リンクは空。
func WithInviteURL(invite func(code string) string) Option {
	return func(s *Service) { s.invite = invite }
}

// WithNotifier は代表への知らせ方を渡す(LINE の設定があるときだけ)。report は知らせを送れなかったときの
// エラーを受け取る(申込の操作は失敗にしないので、呼ぶ側がログに残す)。
func WithNotifier(notifier Notifier, report func(error)) Option {
	return func(s *Service) { s.notifier, s.report = notifier, report }
}

// NewService は store で読み書きする Service を返す。
func NewService(store Store, opts ...Option) *Service {
	s := &Service{store: store, now: time.Now, random: rand.Reader,
		invite: func(string) string { return "" }}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Questions は Desk.Questions。
func (s *Service) Questions(ctx context.Context, tournamentID int64) ([]Question, error) {
	if _, err := s.store.Tournament(ctx, tournamentID); err != nil {
		return nil, err
	}
	return s.store.Questions(ctx, tournamentID)
}

// SaveQuestions は Desk.SaveQuestions。
func (s *Service) SaveQuestions(
	ctx context.Context,
	tournamentID int64,
	questions []Question,
) ([]Question, error) {
	if _, err := s.store.Tournament(ctx, tournamentID); err != nil {
		return nil, err
	}
	current, err := s.store.Questions(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	checked, err := CheckQuestions(questions, current)
	if err != nil {
		return nil, err
	}
	return s.store.ReplaceQuestions(ctx, tournamentID, checked)
}

// Entries は Desk.Entries。
func (s *Service) Entries(ctx context.Context, tournamentID int64) ([]Summary, error) {
	if _, err := s.store.Tournament(ctx, tournamentID); err != nil {
		return nil, err
	}
	return s.store.List(ctx, tournamentID)
}

// Entry は Desk.Entry。
func (s *Service) Entry(ctx context.Context, id int64) (Detail, error) {
	detail, err := s.store.Get(ctx, id)
	return s.withInvite(detail), err
}

// withInvite は申込に招待リンクを付ける。
func (s *Service) withInvite(detail Detail) Detail {
	if detail.InviteCode != "" {
		detail.InviteURL = s.invite(detail.InviteCode)
	}
	return detail
}

// Submit は Desk.Submit。
func (s *Service) Submit(
	ctx context.Context,
	tournamentID int64,
	channel Channel,
	draft Draft,
) (Detail, error) {
	return s.submit(ctx, tournamentID, channel, nil, draft)
}

func (s *Service) submit(
	ctx context.Context,
	tournamentID int64,
	channel Channel,
	rep *int64,
	draft Draft,
) (Detail, error) {
	tournament, err := s.store.Tournament(ctx, tournamentID)
	if err != nil {
		return Detail{}, err
	}
	if err := CanSubmit(tournament.Status, channel); err != nil {
		return Detail{}, err
	}
	questions, err := s.store.Questions(ctx, tournamentID)
	if err != nil {
		return Detail{}, err
	}
	checked, answers, err := CheckDraft(draft, tournament, questions)
	if err != nil {
		return Detail{}, err
	}
	code, err := NewInviteCode(s.random)
	if err != nil {
		return Detail{}, err
	}
	newEntry := NewEntry{
		DivisionID: checked.DivisionID, FullName: checked.FullName, ShortName: checked.ShortName,
		RepName: checked.RepName, RepPhone: checked.RepPhone, RepEmail: checked.RepEmail,
		Members: checked.Members, Channel: channel, RepLineUserID: rep, InviteCode: code,
		Answers: answers, At: s.now(),
	}
	if rep != nil {
		profile := RepProfile(checked)
		newEntry.Profile = &profile
	}
	detail, err := s.store.Create(ctx, tournamentID, newEntry)
	if err != nil {
		return Detail{}, err
	}
	detail = s.withInvite(detail)
	s.notify(ctx, NoticeSubmitted, TournamentRef{ID: tournament.ID, Name: tournament.Name}, detail)
	return detail, nil
}

// notify は申込の代表に知らせる。相手は LINE から出した申込の代表だけ。送れなかったときは操作を失敗に
// せず、report に渡す。
func (s *Service) notify(ctx context.Context, kind NoticeKind, tournament TournamentRef, detail Detail) {
	if s.notifier == nil || detail.RepLineUserID == nil {
		return
	}
	notice := Notice{Kind: kind, LineUserID: *detail.RepLineUserID, Tournament: tournament, Entry: detail}
	if err := s.notifier.Notify(ctx, notice); err != nil {
		s.report(fmt.Errorf("notify %s of entry %d: %w", kind, detail.ID, err))
	}
}

// Decide は Desk.Decide。今の状態を読んで決まりに照らし、読んだ状態のままのときだけ変える(読んでから
// 変えるまでに別の運営が変えていたら Store が domain.ErrConflict を返す)。
func (s *Service) Decide(
	ctx context.Context,
	id int64,
	action Action,
	note string,
	actor domain.Actor,
) (Detail, error) {
	checkedNote, err := CheckNote(action, note)
	if err != nil {
		return Detail{}, err
	}
	current, err := s.store.Get(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	next, err := Next(current.Status, action)
	if err != nil {
		return Detail{}, err
	}
	if err := CanReturn(action, current); err != nil {
		return Detail{}, err
	}
	decision := Decision{From: current.Status, To: next, Note: checkedNote, DecidedBy: actor.UserID,
		At: s.now()}
	if next == StatusApproved {
		approval := ApprovalOf(current)
		decision.Approval = &approval
	}
	if action == ActionReopen {
		decision.Reopen = &Reopen{DivisionID: current.Division.ID}
	}
	detail, err := s.store.Decide(ctx, id, decision)
	if err != nil {
		return Detail{}, fmt.Errorf("%s entry %d: %w", action, id, err)
	}
	detail = s.withInvite(detail)
	if s.notifier != nil && detail.RepLineUserID != nil {
		tournament, err := s.store.Tournament(ctx, detail.TournamentID)
		if err != nil {
			s.report(fmt.Errorf("notify %s of entry %d: %w", action, id, err))
		} else {
			s.notify(ctx, DecisionNotice(action), TournamentRef{ID: tournament.ID, Name: tournament.Name}, detail)
		}
	}
	return detail, nil
}

// Export は Desk.Export。
func (s *Service) Export(ctx context.Context, tournamentID int64) (Export, error) {
	if _, err := s.store.Tournament(ctx, tournamentID); err != nil {
		return Export{}, err
	}
	questions, err := s.store.Questions(ctx, tournamentID)
	if err != nil {
		return Export{}, err
	}
	details, err := s.store.Details(ctx, tournamentID)
	if err != nil {
		return Export{}, err
	}
	for i := range details {
		details[i] = s.withInvite(details[i])
	}
	return Export{Questions: questions, Entries: details}, nil
}

// IssueTeamLinks は Links.IssueTeamLinks。
func (s *Service) IssueTeamLinks(ctx context.Context, lineUserID int64) ([]TournamentLink, error) {
	tournaments, err := s.store.LinkTournaments(ctx, lineUserID)
	if err != nil {
		return nil, err
	}
	out := make([]TournamentLink, len(tournaments))
	for i, tournament := range tournaments {
		issued, err := s.issue(ctx, Link{LineUserID: lineUserID, Purpose: LinkTeam,
			TournamentID: tournament.ID}, "")
		if err != nil {
			return nil, err
		}
		out[i] = TournamentLink{Tournament: tournament, Token: issued.Token, ExpiresAt: issued.ExpiresAt}
	}
	return out, nil
}

// IssueMemberLink は Links.IssueMemberLink。
func (s *Service) IssueMemberLink(
	ctx context.Context,
	lineUserID int64,
	inviteCode string,
) (IssuedLink, error) {
	team, err := s.store.FindByInvite(ctx, inviteCode)
	if err != nil {
		return IssuedLink{}, err
	}
	if err := CanJoin(team.Status); err != nil {
		return IssuedLink{}, err
	}
	return s.issue(ctx, Link{LineUserID: lineUserID, Purpose: LinkMember, TeamEntryID: team.ID},
		team.FullName)
}

// IssueJoinedLinks は Links.IssueJoinedLinks。
func (s *Service) IssueJoinedLinks(ctx context.Context, lineUserID int64) ([]JoinedLink, error) {
	teams, err := s.store.JoinedTeams(ctx, lineUserID)
	if err != nil {
		return nil, err
	}
	out := make([]JoinedLink, len(teams))
	for i, team := range teams {
		issued, err := s.issue(ctx, Link{LineUserID: lineUserID, Purpose: LinkMember,
			TeamEntryID: team.TeamEntryID}, team.TeamName)
		if err != nil {
			return nil, err
		}
		out[i] = JoinedLink{Team: team, Token: issued.Token, ExpiresAt: issued.ExpiresAt}
	}
	return out, nil
}

// IssueScorerLinks は Links.IssueScorerLinks。
func (s *Service) IssueScorerLinks(ctx context.Context, lineUserID int64) ([]ScorerLink, error) {
	teams, err := s.store.ScorerTeams(ctx, lineUserID)
	if err != nil {
		return nil, err
	}
	out := make([]ScorerLink, len(teams))
	for i, team := range teams {
		issued, err := s.issue(ctx, Link{LineUserID: lineUserID, Purpose: LinkScorer,
			TeamEntryID: team.TeamEntryID}, team.TeamName)
		if err != nil {
			return nil, err
		}
		out[i] = ScorerLink{Team: team, Token: issued.Token, ExpiresAt: issued.ExpiresAt}
	}
	return out, nil
}

// scorerTeam はリンクの人が、リンクのチームの記録係のままかを確かめて、そのチームを返す。代表が記録係を
// 選び直したり、申込を取り消したり、大会が終わったりしたら、リンクは使えない(domain.NotFound)。
func (s *Service) scorerTeam(ctx context.Context, link Link) (ScorerTeam, error) {
	teams, err := s.store.ScorerTeams(ctx, link.LineUserID)
	if err != nil {
		return ScorerTeam{}, err
	}
	for _, team := range teams {
		if team.TeamEntryID == link.TeamEntryID {
			return team, nil
		}
	}
	return ScorerTeam{}, domain.NotFound(domain.ResourceEntryLink, 0)
}

// ScorerSignIn は Desk.ScorerSignIn。
func (s *Service) ScorerSignIn(ctx context.Context, token string) (int64, error) {
	link, err := s.link(ctx, token, LinkScorer)
	if err != nil {
		return 0, err
	}
	team, err := s.scorerTeam(ctx, link)
	if err != nil {
		return 0, err
	}
	return s.store.EnsureScorer(ctx, link.LineUserID, team.Tournament.ID, team.MemberName)
}

func (s *Service) issue(ctx context.Context, link Link, teamName string) (IssuedLink, error) {
	token, err := domain.NewToken(s.random)
	if err != nil {
		return IssuedLink{}, err
	}
	link.ExpiresAt = s.now().Add(LinkLifetime)
	if err := s.store.SaveLink(ctx, domain.HashToken(token), link); err != nil {
		return IssuedLink{}, err
	}
	return IssuedLink{Token: token, ExpiresAt: link.ExpiresAt, TeamName: teamName}, nil
}

// openLink は合言葉のリンクを読み、期限を確かめる。
func (s *Service) openLink(ctx context.Context, token string) (Link, error) {
	link, err := s.store.FindLink(ctx, domain.HashToken(token))
	if err != nil {
		return Link{}, err
	}
	if err := CheckLink(link, s.now()); err != nil {
		return Link{}, err
	}
	return link, nil
}

// link は合言葉のリンクを読み、期限と目的を確かめる。
func (s *Service) link(ctx context.Context, token string, purpose LinkPurpose) (Link, error) {
	link, err := s.openLink(ctx, token)
	if err != nil {
		return Link{}, err
	}
	if err := CheckPurpose(link, purpose); err != nil {
		return Link{}, err
	}
	return link, nil
}

// LinkView は Desk.LinkView。
func (s *Service) LinkView(ctx context.Context, token string) (LinkView, error) {
	link, err := s.openLink(ctx, token)
	if err != nil {
		return LinkView{}, err
	}
	if link.Purpose == LinkMember {
		form, err := s.memberForm(ctx, link)
		if err != nil {
			return LinkView{}, err
		}
		return LinkView{Purpose: LinkMember, Member: &form}, nil
	}
	if link.Purpose == LinkScorer {
		team, err := s.scorerTeam(ctx, link)
		if err != nil {
			return LinkView{}, err
		}
		return LinkView{Purpose: LinkScorer, Scorer: &ScorerView{Team: team, ExpiresAt: link.ExpiresAt}}, nil
	}
	form, err := s.teamForm(ctx, link)
	if err != nil {
		return LinkView{}, err
	}
	return LinkView{Purpose: LinkTeam, Team: &form}, nil
}

func (s *Service) teamForm(ctx context.Context, link Link) (TeamForm, error) {
	tournament, err := s.store.Tournament(ctx, link.TournamentID)
	if err != nil {
		return TeamForm{}, err
	}
	questions, err := s.store.Questions(ctx, link.TournamentID)
	if err != nil {
		return TeamForm{}, err
	}
	mine, err := s.store.RepEntries(ctx, link.TournamentID, link.LineUserID)
	if err != nil {
		return TeamForm{}, err
	}
	for i := range mine {
		mine[i] = s.withInvite(mine[i])
	}
	profile, err := s.store.Profile(ctx, link.LineUserID)
	if err != nil {
		return TeamForm{}, err
	}
	past, err := s.store.PastTeams(ctx, link.LineUserID, link.TournamentID, MaxPastTeams)
	if err != nil {
		return TeamForm{}, err
	}
	return TeamForm{
		Tournament: TournamentRef{ID: tournament.ID, Name: tournament.Name},
		Open:       tournament.Status == domain.TournamentEntryOpen,
		Divisions:  tournament.Divisions,
		Questions:  scopeOf(questions, ScopeTeam),
		Mine:       mine,
		Profile:    profile,
		PastTeams:  past,
		ExpiresAt:  link.ExpiresAt,
	}, nil
}

func (s *Service) memberForm(ctx context.Context, link Link) (MemberForm, error) {
	detail, err := s.store.Get(ctx, link.TeamEntryID)
	if err != nil {
		return MemberForm{}, err
	}
	tournament, err := s.store.Tournament(ctx, detail.TournamentID)
	if err != nil {
		return MemberForm{}, err
	}
	questions, err := s.store.Questions(ctx, detail.TournamentID)
	if err != nil {
		return MemberForm{}, err
	}
	form := MemberForm{TournamentName: tournament.Name, Team: detail.Summary,
		Questions: scopeOf(questions, ScopeMember), Unclaimed: []Member{},
		Mine: MemberOf(detail.MemberList, link.LineUserID), MineAnswers: []Answer{}, ExpiresAt: link.ExpiresAt}
	repRows := RepRows(detail, link.LineUserID)
	for _, member := range detail.MemberList {
		if !member.Registered && !repRows[member.ID] {
			form.Unclaimed = append(form.Unclaimed, member)
		}
	}
	if form.Mine != nil {
		answers, err := s.store.MemberAnswers(ctx, form.Mine.ID)
		if err != nil {
			return MemberForm{}, err
		}
		form.MineAnswers = answers
	}
	profile, err := s.store.Profile(ctx, link.LineUserID)
	if err != nil {
		return MemberForm{}, err
	}
	if profile != nil {
		form.ProfileName = profile.Name
	}
	return form, nil
}

func scopeOf(questions []Question, scope Scope) []Question {
	out := []Question{}
	for _, question := range questions {
		if question.Scope == scope {
			out = append(out, question)
		}
	}
	return out
}

// SubmitByLink は Desk.SubmitByLink。
func (s *Service) SubmitByLink(ctx context.Context, token string, draft Draft) (Detail, error) {
	link, err := s.link(ctx, token, LinkTeam)
	if err != nil {
		return Detail{}, err
	}
	rep := link.LineUserID
	return s.submit(ctx, link.TournamentID, ChannelLine, &rep, draft)
}

// ResubmitByLink は Desk.ResubmitByLink。申込がこの代表のものかは、この代表がこの大会に出した申込の
// 中にあるかで確かめる(ほかの人の申込 ID を指しても、見つからないことにする)。
func (s *Service) ResubmitByLink(
	ctx context.Context,
	token string,
	entryID int64,
	draft Draft,
	registered []int64,
) (Detail, error) {
	link, err := s.link(ctx, token, LinkTeam)
	if err != nil {
		return Detail{}, err
	}
	mine, err := s.store.RepEntries(ctx, link.TournamentID, link.LineUserID)
	if err != nil {
		return Detail{}, err
	}
	current := findDetail(mine, entryID)
	if current == nil {
		return Detail{}, domain.NotFound(domain.ResourceEntry, entryID)
	}
	tournament, err := s.store.Tournament(ctx, link.TournamentID)
	if err != nil {
		return Detail{}, err
	}
	if err := CanResubmit(current.Status, tournament.Status); err != nil {
		return Detail{}, err
	}
	questions, err := s.store.Questions(ctx, link.TournamentID)
	if err != nil {
		return Detail{}, err
	}
	checked, answers, err := CheckDraft(draft, tournament, questions)
	if err != nil {
		return Detail{}, err
	}
	detail, err := s.store.Resubmit(ctx, entryID, Resubmission{Draft: checked, Answers: answers,
		Registered: registered, RepLineUserID: link.LineUserID, Profile: RepProfile(checked), At: s.now()})
	if err != nil {
		return Detail{}, fmt.Errorf("resubmit entry %d: %w", entryID, err)
	}
	detail = s.withInvite(detail)
	s.notify(ctx, NoticeResubmitted, TournamentRef{ID: tournament.ID, Name: tournament.Name}, detail)
	return detail, nil
}

// ChooseScorerByLink は Desk.ChooseScorerByLink。代表が、参加が決まった自分の申込のチームの記録係を
// 選ぶ(memberID が nil なら外す)。選ばれたメンバーには、公式 LINE で記録係になったことと記録の画面の
// 開き方を知らせる。
func (s *Service) ChooseScorerByLink(ctx context.Context, token string, entryID int64, memberID *int64) (Detail, error) {
	link, err := s.link(ctx, token, LinkTeam)
	if err != nil {
		return Detail{}, err
	}
	mine, err := s.store.RepEntries(ctx, link.TournamentID, link.LineUserID)
	if err != nil {
		return Detail{}, err
	}
	current := findDetail(mine, entryID)
	if current == nil {
		return Detail{}, domain.NotFound(domain.ResourceEntry, entryID)
	}
	if err := CanChooseScorer(*current, memberID); err != nil {
		return Detail{}, err
	}
	detail, err := s.store.SetScorer(ctx, entryID, memberID, s.now())
	if err != nil {
		return Detail{}, fmt.Errorf("choose scorer of entry %d: %w", entryID, err)
	}
	detail = s.withInvite(detail)
	chosen := scorerOf(detail)
	if s.notifier != nil && chosen != nil && chosen.LineUserID != nil &&
		!sameScorer(current.ScorerMemberID, memberID) {
		tournament, err := s.store.Tournament(ctx, link.TournamentID)
		if err != nil {
			s.report(fmt.Errorf("notify scorer of entry %d: %w", entryID, err))
			return detail, nil
		}
		notice := Notice{Kind: NoticeScorerChosen, LineUserID: *chosen.LineUserID,
			Tournament: TournamentRef{ID: tournament.ID, Name: tournament.Name}, Entry: detail}
		if err := s.notifier.Notify(ctx, notice); err != nil {
			s.report(fmt.Errorf("notify scorer of entry %d: %w", entryID, err))
		}
	}
	return detail, nil
}

// scorerOf は申込の記録係のメンバー。選んでいなければ nil。
func scorerOf(detail Detail) *Member {
	if detail.ScorerMemberID == nil {
		return nil
	}
	for i := range detail.MemberList {
		if detail.MemberList[i].ID == *detail.ScorerMemberID {
			return &detail.MemberList[i]
		}
	}
	return nil
}

// sameScorer は記録係が変わっていないか(同じ人を選び直したときは知らせない)。
func sameScorer(before, after *int64) bool {
	if before == nil || after == nil {
		return before == after
	}
	return *before == *after
}

func findDetail(details []Detail, id int64) *Detail {
	for i := range details {
		if details[i].ID == id {
			return &details[i]
		}
	}
	return nil
}

// JoinByLink は Desk.JoinByLink。
func (s *Service) JoinByLink(ctx context.Context, token string, draft JoinDraft) (MemberForm, error) {
	link, err := s.link(ctx, token, LinkMember)
	if err != nil {
		return MemberForm{}, err
	}
	detail, err := s.store.Get(ctx, link.TeamEntryID)
	if err != nil {
		return MemberForm{}, err
	}
	if err := CanJoin(detail.Status); err != nil {
		return MemberForm{}, err
	}
	questions, err := s.store.Questions(ctx, detail.TournamentID)
	if err != nil {
		return MemberForm{}, err
	}
	mine := MemberOf(detail.MemberList, link.LineUserID)
	checked, answers, err := CheckJoin(draft, detail.MemberList, mine, RepRows(detail, link.LineUserID), questions)
	if err != nil {
		return MemberForm{}, err
	}
	err = s.store.Join(ctx, Join{TeamEntryID: link.TeamEntryID, LineUserID: link.LineUserID,
		MemberID: checked.MemberID, Name: checked.Name, Number: checked.Number, Answers: answers,
		Profile: MemberProfile(checked), At: s.now()})
	if err != nil {
		return MemberForm{}, err
	}
	return s.memberForm(ctx, link)
}
