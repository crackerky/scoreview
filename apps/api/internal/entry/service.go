package entry

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"scoreview/api/internal/domain"
)

// Service は Store で記録を読み書きし、申込の決まりを当てる Desk。
type Service struct {
	store  Store
	now    func() time.Time
	random io.Reader
}

var _ Desk = (*Service)(nil)

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

// NewService は store で読み書きする Service を返す。
func NewService(store Store, opts ...Option) *Service {
	s := &Service{store: store, now: time.Now, random: rand.Reader}
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
	return s.store.Get(ctx, id)
}

// Submit は Desk.Submit。
func (s *Service) Submit(
	ctx context.Context,
	tournamentID int64,
	channel Channel,
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
	return s.store.Create(ctx, tournamentID, NewEntry{
		DivisionID: checked.DivisionID, FullName: checked.FullName, ShortName: checked.ShortName,
		RepName: checked.RepName, RepPhone: checked.RepPhone, RepEmail: checked.RepEmail,
		Members: checked.Members, Channel: channel, InviteCode: code, Answers: answers, At: s.now(),
	})
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
	decision := Decision{From: current.Status, To: next, Note: checkedNote, DecidedBy: actor.UserID,
		At: s.now()}
	if next == StatusApproved {
		approval := ApprovalOf(current)
		decision.Approval = &approval
	}
	detail, err := s.store.Decide(ctx, id, decision)
	if err != nil {
		return Detail{}, fmt.Errorf("%s entry %d: %w", action, id, err)
	}
	return detail, nil
}
