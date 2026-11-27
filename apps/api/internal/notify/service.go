package notify

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
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
	// siteURL はスコアサイトの URL(スコアサイトのボタンの行き先の前半。末尾の / は無い)
	siteURL string
	// invite は招待コードからメンバーの招待リンクを作る({招待リンク} の差し込み)
	invite func(code string) string
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

// WithSiteURL はスコアサイトの URL を渡す(スコアサイトのボタンは、その大会のページを開く)。
func WithSiteURL(siteURL string) Option {
	return func(s *Service) { s.siteURL = strings.TrimRight(siteURL, "/") }
}

// WithInviteURL は招待コードからメンバーの招待リンクを作る関数を渡す(entry.WithInviteURL と同じもの)。
// 渡さなければ {招待リンク} は空になる。
func WithInviteURL(invite func(code string) string) Option {
	return func(s *Service) { s.invite = invite }
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
	s := &Service{store: store, report: func(error) {}, now: time.Now, random: rand.Reader,
		invite: func(string) string { return "" }}
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
	preview := Preview{Heading: heading, Recipients: recipients, CanSend: s.sender != nil,
		Everyone: audience.Kind == AudienceFriends}
	preview.SampleName, preview.Variables = Examples(audience.Kind, recipients, s.invite)
	preview.Reachable, preview.Unreachable = CountReachable(recipients)
	if preview.Everyone {
		preview.Reachable, _ = s.reach(ctx, audience.Kind, recipients)
	}
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
	if err := CheckRequestKey(request.RequestKey); err != nil {
		return Batch{}, err
	}
	// 同じ画面からの 2 回目は、文と宛先と配信数を確かめ直さずに前の記録を返す。1 回目で配信数が減って
	// いるので、確かめ直すと送ったのに「足りないので送らなかった」と返してしまうため
	if id, found, err := s.store.FindBatch(ctx, request.RequestKey); err != nil {
		return Batch{}, err
	} else if found {
		return s.existing(ctx, id, tournamentID)
	}
	body, err := CheckBody(request.Body)
	if err != nil {
		return Batch{}, err
	}
	if body, err = CheckVariables(body, request.Audience.Kind); err != nil {
		return Batch{}, err
	}
	// 友だち全員には、申し込んでいない人も含めて届き、取り消せないので、運営が確かめてから送る
	if request.Audience.Kind == AudienceFriends && !request.ConfirmAll {
		return Batch{}, domain.Invalid("confirmAll", "is required to send to every friend")
	}
	if request.Buttons, err = s.buttons(tournamentID, request.Buttons); err != nil {
		return Batch{}, err
	}
	if request.SendAt != nil {
		return s.schedule(ctx, tournamentID, request, body, actor)
	}
	heading, recipients, err := s.recipients(ctx, tournamentID, request.Audience)
	if err != nil {
		return Batch{}, err
	}
	// 友だち全員の人数を LINE がまだ数えていなければ(known が偽)、人数と配信数を確かめずに送る
	reachable, known := s.reach(ctx, request.Audience.Kind, recipients)
	if known && reachable == 0 {
		return Batch{}, domain.Invalid("audience", "has no one who can receive the message")
	}
	// 足りなければ LINE が途中から受け付けなくなり、一部の人にだけ届く。読めなければ送ってみる
	if known {
		if quota, err := s.sender.Quota(ctx); err != nil {
			s.report(fmt.Errorf("read LINE quota: %w", err))
		} else if err := CheckQuota(quota, reachable); err != nil {
			return Batch{}, err
		}
	}
	text := Text(heading, body)
	batch, groups, err := s.plan(tournamentID, request, text, actor, recipients)
	if err != nil {
		return Batch{}, err
	}
	id, created, err := s.store.CreateBatch(ctx, batch)
	if err != nil {
		return Batch{}, err
	}
	if !created {
		return s.existing(ctx, id, tournamentID)
	}
	work := context.WithoutCancel(ctx)
	if err := s.deliver(work, id, groups, Message{Text: text, Buttons: request.Buttons}); err != nil {
		return Batch{}, err
	}
	return s.Batch(work, id)
}

// buttons はボタンを確かめ、スコアサイトのボタンの行き先を大会のページにする。
func (s *Service) buttons(tournamentID int64, buttons []Button) ([]Button, error) {
	checked, err := CheckButtons(buttons)
	if err != nil {
		return nil, err
	}
	for i := range checked {
		if checked[i].Kind != ButtonSite {
			continue
		}
		if s.siteURL == "" {
			return nil, domain.Invalid("buttons", "the site URL is not configured")
		}
		checked[i].URL = fmt.Sprintf("%s/tournaments/%d", s.siteURL, tournamentID)
	}
	return checked, nil
}

// existing は同じ RequestKey で記録済みの送信を返す。
func (s *Service) existing(ctx context.Context, id, tournamentID int64) (Batch, error) {
	batch, err := s.Batch(ctx, id)
	if err != nil {
		return Batch{}, err
	}
	if batch.TournamentID != tournamentID {
		return Batch{}, fmt.Errorf("%w: the request key was used for another tournament", domain.ErrConflict)
	}
	return batch, nil
}

// schedule は送る時刻を決めた送信を、宛先を持たない予約として記録する。宛先は送る時刻に数え直すので、
// 今いなくても予約できる(承認の前に前日の案内を予約しておけるように)。配信数も送る時刻に確かめる。
func (s *Service) schedule(
	ctx context.Context,
	tournamentID int64,
	request Request,
	body string,
	actor domain.Actor,
) (Batch, error) {
	now := s.now()
	if err := CheckSendAt(*request.SendAt, now); err != nil {
		return Batch{}, err
	}
	heading, _, err := s.recipients(ctx, tournamentID, request.Audience)
	if err != nil {
		return Batch{}, err
	}
	id, created, err := s.store.CreateBatch(ctx, NewBatch{TournamentID: tournamentID, RequestKey: request.RequestKey,
		Audience: request.Audience, Body: Text(heading, body), Buttons: request.Buttons, SentBy: actor.UserID,
		At: now, SendAt: request.SendAt})
	if err != nil {
		return Batch{}, err
	}
	if !created {
		return s.existing(ctx, id, tournamentID)
	}
	return s.Batch(ctx, id)
}

// plan は記録する送信と、送る前に決めたまとまりを作る。
func (s *Service) plan(
	tournamentID int64,
	request Request,
	text string,
	actor domain.Actor,
	recipients []Recipient,
) (NewBatch, []Group, error) {
	now := s.now()
	jobs, groups, err := s.jobsFor(request.Audience.Kind, recipients, text)
	if err != nil {
		return NewBatch{}, nil, err
	}
	return NewBatch{TournamentID: tournamentID, RequestKey: request.RequestKey, Audience: request.Audience,
		Body: text, Buttons: request.Buttons, SentBy: actor.UserID, At: now, ClaimedUntil: now.Add(SendLease),
		Jobs: jobs}, groups, nil
}

// jobsFor は宛先ごとの行と、まとまり(友だちを MaxMulticast 人ずつ。まとまりごとに送り直しのキー)を
// 作る。キーを記録に残すので、途中で止まっても同じまとまりを同じキーで送り直せる。
func (s *Service) jobsFor(kind AudienceKind, recipients []Recipient, text string) ([]NewJob, []Group, error) {
	// 友だち全員は一斉送信 1 回。宛先の人は分からないので「友だち全員」の 1 行を記録する
	if kind == AudienceFriends {
		key, err := retryKey(s.random)
		if err != nil {
			return nil, nil, err
		}
		return []NewJob{{Name: EveryoneName, Status: JobPending, RetryKey: key}},
			[]Group{{RetryKey: key, Everyone: true}}, nil
	}
	jobs := make([]NewJob, 0, len(recipients))
	var groups []Group
	// 差し込みがあれば人ごとに文を作り、同じ文になった人を同じまとまりにする(LINE は 1 回の送信で
	// 同じ文しか送れない)。open は文ごとの、まだ空きのあるまとまり
	personal := len(usedVariables(text)) > 0
	open := map[string]int{}
	for _, recipient := range recipients {
		job := NewJob{LineUserID: recipient.LineUserID, Name: recipient.Name,
			Teams: strings.Join(recipient.Teams, "、"), Status: JobSkipped}
		if recipient.Friend {
			filled := ""
			if personal {
				filled = Fill(text, recipient, s.invite)
			}
			at, ok := open[filled]
			if !ok || len(groups[at].UserIDs) == MaxMulticast {
				key, err := retryKey(s.random)
				if err != nil {
					return nil, nil, err
				}
				groups = append(groups, Group{RetryKey: key, Text: filled})
				at = len(groups) - 1
				open[filled] = at
			}
			group := &groups[at]
			group.LineUserIDs = append(group.LineUserIDs, recipient.LineUserID)
			group.UserIDs = append(group.UserIDs, recipient.UserID)
			job.Status, job.RetryKey, job.Body = JobPending, group.RetryKey, filled
		}
		jobs = append(jobs, job)
	}
	return jobs, groups, nil
}

// deliver はまとまりごとに送り、結果を書く。LINE が受け付けたか分からない失敗(ErrTemporary)だけ、
// 同じキーでもう 1 回送る(1 回目を LINE が受け付けていれば、2 回目は送られずに成功になる)。
// 結果を書けなければそこで止める(残りのまとまりは pending のまま残り、Resend で送れる)。
func (s *Service) deliver(ctx context.Context, batchID int64, groups []Group, base Message) error {
	for _, group := range groups {
		message := base
		if group.Text != "" {
			message.Text = group.Text
		}
		send := func() error {
			if group.Everyone {
				return s.sender.Broadcast(ctx, message, group.RetryKey)
			}
			return s.sender.Multicast(ctx, group.UserIDs, message, group.RetryKey)
		}
		sendErr := send()
		if errors.Is(sendErr, ErrTemporary) {
			if retryErr := send(); retryErr == nil {
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
		finish := s.store.FinishJobs(ctx, batchID, group.LineUserIDs, result)
		if group.Everyone {
			finish = s.store.FinishEveryone(ctx, batchID, result)
		}
		if finish != nil {
			return finish
		}
	}
	return nil
}

// Resend は Desk.Resend。送れなかったまとまりを、記録したキーと宛先のまま送る。LINE が前に受け付けた
// まとまり(結果を書く前に止まったもの)は、LINE が送らずに成功を返すので、2 回は届かない。ほかの送信が
// 送っている途中のまとまりは送らない(Store.Claim が持たせない)。
func (s *Service) Resend(ctx context.Context, id int64) (Batch, error) {
	if s.sender == nil {
		return Batch{}, fmt.Errorf("%w: the LINE settings are missing, so nothing can be sent", domain.ErrConflict)
	}
	now := s.now()
	batch, err := s.store.Batch(ctx, id, now)
	if err != nil {
		return Batch{}, err
	}
	if !now.Before(resendStart(batch).Add(ResendWindow)) {
		return Batch{}, fmt.Errorf("%w: it is too late to resend; the retry keys last 24 hours", domain.ErrConflict)
	}
	groups, err := s.store.Claim(ctx, id, now, now.Add(SendLease))
	if err != nil {
		return Batch{}, err
	}
	work := context.WithoutCancel(ctx)
	if err := s.deliver(work, id, groups, Message{Text: batch.Body, Buttons: batch.Buttons}); err != nil {
		return Batch{}, err
	}
	return s.Batch(work, id)
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
	batches, err := s.store.Batches(ctx, tournamentID, s.now())
	if err != nil {
		return History{}, err
	}
	for i := range batches {
		batches[i].CanResend = s.canResend(batches[i])
	}
	return History{Batches: batches, CanSend: s.sender != nil}, nil
}

// canResend は今送り直せる宛先があり、送り直しのキーがまだ有効か。
func (s *Service) canResend(batch Batch) bool {
	return s.sender != nil && batch.Counts.Resendable > 0 && s.now().Before(resendStart(batch).Add(ResendWindow))
}

// resendStart は送り直しの期限を数え始める時刻。予約は送る時刻(送り直しのキーは送る時刻に作るので、
// 予約した時刻から数えると、前日の案内のような先の予約は送る前に期限が切れる)、すぐ送った送信は
// 記録した時刻。
func resendStart(batch Batch) time.Time {
	if batch.SendAt != nil {
		return *batch.SendAt
	}
	return batch.CreatedAt
}

// Batch は Desk.Batch。
func (s *Service) Batch(ctx context.Context, id int64) (Batch, error) {
	batch, err := s.store.Batch(ctx, id, s.now())
	if err != nil {
		return Batch{}, err
	}
	batch.CanResend = s.canResend(batch)
	return batch, nil
}

// Cancel は Desk.Cancel。
func (s *Service) Cancel(ctx context.Context, id int64) (Batch, error) {
	cancelled, err := s.store.CancelScheduled(ctx, id)
	if err != nil {
		return Batch{}, err
	}
	batch, err := s.Batch(ctx, id)
	if err != nil {
		return Batch{}, err
	}
	if !cancelled {
		return Batch{}, fmt.Errorf("%w: only a scheduled message can be cancelled", domain.ErrConflict)
	}
	return batch, nil
}

// dispatchLimit は 1 回の定期の呼び出しで送る予約の数の上限(残りは次の呼び出しで送る)。
const dispatchLimit = 20

// Dispatch は Desk.Dispatch。予約ごとに失敗を分け、1 つの失敗で残りを止めない。
func (s *Service) Dispatch(ctx context.Context) (DispatchResult, error) {
	due, err := s.store.DueBatches(ctx, s.now(), dispatchLimit)
	if err != nil {
		return DispatchResult{}, err
	}
	var result DispatchResult
	var failed []error
	for _, batch := range due {
		sent, skipped, err := s.dispatchOne(ctx, batch)
		if err != nil {
			failed = append(failed, fmt.Errorf("dispatch notification %d: %w", batch.ID, err))
			continue
		}
		if sent {
			result.Sent++
		}
		if skipped {
			result.Skipped++
		}
	}
	return result, errors.Join(failed...)
}

// dispatchOne は送る時刻の来た予約を 1 つ送る。宛先を今の条件で数え、送れなければ理由を付けて送れ
// なかったことにする(今月の配信数が足りなければ、一部の人にだけ届くのを避けて何も送らない)。
func (s *Service) dispatchOne(ctx context.Context, due Due) (sent, skipped bool, err error) {
	skip := func(reason SkipReason) (bool, bool, error) {
		done, err := s.store.SkipScheduled(ctx, due.ID, reason)
		return false, done, err
	}
	if s.sender == nil {
		return skip(SkipNotConfigured)
	}
	recipients, err := s.store.Recipients(ctx, due.TournamentID, due.Audience)
	if err != nil {
		return false, false, err
	}
	reachable, known := s.reach(ctx, due.Audience.Kind, recipients)
	if known && reachable == 0 {
		return skip(SkipNoRecipients)
	}
	if known {
		if quota, err := s.sender.Quota(ctx); err != nil {
			s.report(fmt.Errorf("read LINE quota: %w", err))
		} else if CheckQuota(quota, reachable) != nil {
			return skip(SkipQuota)
		}
	}
	jobs, groups, err := s.jobsFor(due.Audience.Kind, recipients, due.Body)
	if err != nil {
		return false, false, err
	}
	started, err := s.store.StartScheduled(ctx, due.ID, jobs, s.now().Add(SendLease))
	if err != nil || !started {
		return false, false, err
	}
	if err := s.deliver(context.WithoutCancel(ctx), due.ID, groups, Message{Text: due.Body, Buttons: due.Buttons}); err != nil {
		return true, false, err
	}
	return true, false, nil
}

// reach は送って届く人数。友だち全員は LINE の数からの見込みで、LINE がまだ数えていないか読めなければ
// known は偽(ほかの条件は記録した友だちの数で、いつも分かる)。
func (s *Service) reach(ctx context.Context, kind AudienceKind, recipients []Recipient) (count int, known bool) {
	if kind != AudienceFriends {
		count, _ = CountReachable(recipients)
		return count, true
	}
	if s.sender == nil {
		return 0, false
	}
	friends, known, err := s.sender.FriendCount(ctx)
	if err != nil {
		s.report(fmt.Errorf("read LINE friend count: %w", err))
		return 0, false
	}
	return int(friends), known
}
