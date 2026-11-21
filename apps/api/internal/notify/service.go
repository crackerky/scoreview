package notify

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"scoreview/api/internal/domain"
)

// maxJobError は宛先ごとの行に残す LINE のエラーの文字数(応答の本文を含むので長くなりうる)。
const maxJobError = 500

// Service は Store で宛先を選んで記録し、Sender で送る Desk。
type Service struct {
	store  Store
	sender Sender
	report func(error)
	now    func() time.Time
	random io.Reader
}

var _ Desk = (*Service)(nil)

// Option は Service の構成を変える Functional Option。
type Option func(*Service)

// WithSender は送り方を渡す(LINE の設定があるときだけ。渡さなければ、宛先は確かめられるが送れない)。
// report は送れなかったときと配信数を読めなかったときのエラーを受け取る(結果は記録に残し、画面にも
// 出すので、呼ぶ側はログに残すだけでよい)。
func WithSender(sender Sender, report func(error)) Option {
	return func(s *Service) { s.sender, s.report = sender, report }
}

// WithClock は現在時刻の取り方を差し替える(テスト用)。
func WithClock(now func() time.Time) Option {
	return func(s *Service) { s.now = now }
}

// WithRandom は LINE に渡す送り直しのキーの乱数の元を差し替える(テスト用。本番は crypto/rand)。
func WithRandom(random io.Reader) Option {
	return func(s *Service) { s.random = random }
}

// NewService は store で読み書きする Service を返す。
func NewService(store Store, opts ...Option) *Service {
	s := &Service{store: store, report: func(error) {}, now: time.Now, random: rand.Reader}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Preview は Desk.Preview。配信数を LINE から読めなくても、宛先は返す(送る前の確認を止めない)。
func (s *Service) Preview(ctx context.Context, tournamentID int64, audience Audience) (Preview, error) {
	heading, recipients, err := s.recipients(ctx, tournamentID, audience)
	if err != nil {
		return Preview{}, err
	}
	preview := Preview{Heading: heading, Recipients: recipients, CanSend: s.sender != nil}
	preview.Reachable, preview.Unreachable = CountReachable(recipients)
	if s.sender != nil {
		quota, err := s.sender.Quota(ctx)
		if err != nil {
			s.report(fmt.Errorf("read LINE quota: %w", err))
		} else {
			preview.Quota = &quota
		}
	}
	return preview, nil
}

func (s *Service) recipients(
	ctx context.Context,
	tournamentID int64,
	audience Audience,
) (string, []Recipient, error) {
	tournament, err := s.store.Tournament(ctx, tournamentID)
	if err != nil {
		return "", nil, err
	}
	if err := CheckAudience(audience, tournament); err != nil {
		return "", nil, err
	}
	recipients, err := s.store.Recipients(ctx, tournamentID, audience)
	if err != nil {
		return "", nil, err
	}
	return tournament.Name, recipients, nil
}

// Send は Desk.Send。記録を作ってから送る。記録を作った後は、呼んだ側が待つのをやめても送り終えて
// 結果を書く(途中でやめると、LINE が受け付けた人の行が送る前のまま残るため)。
func (s *Service) Send(ctx context.Context, tournamentID int64, request Request, actor domain.Actor) (Batch, error) {
	if s.sender == nil {
		return Batch{}, fmt.Errorf("%w: the LINE settings are missing, so nothing can be sent", domain.ErrConflict)
	}
	body, err := CheckBody(request.Body)
	if err != nil {
		return Batch{}, err
	}
	if err := CheckRequestKey(request.RequestKey); err != nil {
		return Batch{}, err
	}
	heading, recipients, err := s.recipients(ctx, tournamentID, request.Audience)
	if err != nil {
		return Batch{}, err
	}
	reachable, _ := CountReachable(recipients)
	if reachable == 0 {
		return Batch{}, domain.Invalid("audience", "has no one who can receive the message")
	}
	// 足りなければ LINE が途中から受け付けなくなり、一部の人にだけ届く。読めなければ送ってみる
	if quota, err := s.sender.Quota(ctx); err != nil {
		s.report(fmt.Errorf("read LINE quota: %w", err))
	} else if err := CheckQuota(quota, reachable); err != nil {
		return Batch{}, err
	}
	text := Text(heading, body)
	id, created, err := s.store.CreateBatch(ctx, newBatch(tournamentID, request, text, actor, s.now(), recipients))
	if err != nil {
		return Batch{}, err
	}
	if created {
		work := context.WithoutCancel(ctx)
		if err := s.deliver(work, id, recipients, text); err != nil {
			return Batch{}, err
		}
		return s.store.Batch(work, id)
	}
	batch, err := s.store.Batch(ctx, id)
	if err != nil {
		return Batch{}, err
	}
	if batch.TournamentID != tournamentID {
		return Batch{}, fmt.Errorf("%w: the request key was used for another tournament", domain.ErrConflict)
	}
	return batch, nil
}

func newBatch(
	tournamentID int64,
	request Request,
	text string,
	actor domain.Actor,
	at time.Time,
	recipients []Recipient,
) NewBatch {
	batch := NewBatch{TournamentID: tournamentID, RequestKey: request.RequestKey, Audience: request.Audience,
		Body: text, SentBy: actor.UserID, At: at, Jobs: make([]NewJob, len(recipients))}
	for i, recipient := range recipients {
		status := JobPending
		if !recipient.Friend {
			status = JobSkipped
		}
		batch.Jobs[i] = NewJob{LineUserID: recipient.LineUserID, Name: recipient.Name,
			Teams: strings.Join(recipient.Teams, "、"), Status: status}
	}
	return batch
}

// deliver は友だちに MaxMulticast 人ずつ送り、まとまりごとに結果を書く。失敗したまとまりは同じ
// キーでもう 1 回だけ送る(1 回目を LINE が受け付けていれば、2 回目は送られずに成功になる)。
func (s *Service) deliver(ctx context.Context, batchID int64, recipients []Recipient, text string) error {
	friends := slices.DeleteFunc(slices.Clone(recipients), func(r Recipient) bool { return !r.Friend })
	for chunk := range slices.Chunk(friends, MaxMulticast) {
		ids, to := make([]int64, len(chunk)), make([]string, len(chunk))
		for i, recipient := range chunk {
			ids[i], to[i] = recipient.LineUserID, recipient.UserID
		}
		key, err := retryKey(s.random)
		if err != nil {
			return err
		}
		sendErr := s.sender.Multicast(ctx, to, text, key)
		if sendErr != nil {
			if retryErr := s.sender.Multicast(ctx, to, text, key); retryErr == nil {
				sendErr = nil
			} else {
				sendErr = errors.Join(sendErr, retryErr)
			}
		}
		result := JobResult{Status: JobSent, At: s.now()}
		if sendErr != nil {
			s.report(fmt.Errorf("send notification %d: %w", batchID, sendErr))
			result = JobResult{Status: JobFailed, Error: clip(sendErr.Error(), maxJobError), At: result.At}
		}
		if err := s.store.FinishJobs(ctx, batchID, ids, result); err != nil {
			return err
		}
	}
	return nil
}

// clip は text を最大 limit 文字にする。
func clip(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit])
}

// retryKey は LINE の X-Line-Retry-Key に渡す UUID(版 4)を作る。
// https://developers.line.biz/ja/reference/messaging-api/#retry-api-request
func retryKey(random io.Reader) (string, error) {
	var b [16]byte
	if _, err := io.ReadFull(random, b[:]); err != nil {
		return "", fmt.Errorf("make LINE retry key: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// History は Desk.History。
func (s *Service) History(ctx context.Context, tournamentID int64) (History, error) {
	if _, err := s.store.Tournament(ctx, tournamentID); err != nil {
		return History{}, err
	}
	batches, err := s.store.Batches(ctx, tournamentID)
	if err != nil {
		return History{}, err
	}
	return History{Batches: batches, CanSend: s.sender != nil}, nil
}

// Batch は Desk.Batch。
func (s *Service) Batch(ctx context.Context, id int64) (Batch, error) {
	return s.store.Batch(ctx, id)
}
