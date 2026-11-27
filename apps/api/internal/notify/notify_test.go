package notify_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/notify"
)

func tournament() entry.Tournament {
	return entry.Tournament{ID: 1, Name: "ルーキーズファイナル", Divisions: []entry.Division{{ID: 10}, {ID: 11}}}
}

func TestCheckAudience(t *testing.T) {
	valid := []notify.Audience{
		{Kind: notify.AudienceReps},
		{Kind: notify.AudienceReps, Status: entry.StatusPending, DivisionID: 10},
		{Kind: notify.AudienceIncomplete, DivisionID: 11},
		{Kind: notify.AudienceParticipants},
		{Kind: notify.AudiencePastReps},
	}
	for _, audience := range valid {
		assert.NoError(t, notify.CheckAudience(audience, tournament()), "%+v", audience)
	}
	invalid := []notify.Audience{
		{Kind: "everyone"},
		{Kind: notify.AudienceReps, Status: entry.StatusCancelled},
		{Kind: notify.AudienceIncomplete, Status: entry.StatusApproved},
		{Kind: notify.AudiencePastReps, DivisionID: 10},
		{Kind: notify.AudienceReps, DivisionID: 99},
	}
	for _, audience := range invalid {
		var problem *domain.ValidationError
		assert.ErrorAs(t, notify.CheckAudience(audience, tournament()), &problem, "%+v", audience)
	}
}

func TestCheckBody(t *testing.T) {
	body, err := notify.CheckBody("  明日は 8 時集合です\r\n\t持ち物: 白と黒のシャツ\n ")
	require.NoError(t, err)
	assert.Equal(t, "明日は 8 時集合です\n\t持ち物: 白と黒のシャツ", body)
	_, err = notify.CheckBody(" \n ")
	require.Error(t, err)
	_, err = notify.CheckBody("ベル\a")
	require.Error(t, err)
	_, err = notify.CheckBody(strings.Repeat("あ", notify.MaxBody))
	require.NoError(t, err)
	_, err = notify.CheckBody(strings.Repeat("あ", notify.MaxBody+1))
	require.Error(t, err)
}

func TestCheckRequestKey(t *testing.T) {
	require.NoError(t, notify.CheckRequestKey("0f8fad5b-d9cb-469f-a165-70867728950e"))
	require.Error(t, notify.CheckRequestKey(""))
	require.Error(t, notify.CheckRequestKey("a b"))
	require.Error(t, notify.CheckRequestKey(strings.Repeat("a", 65)))
}

func TestCheckQuota(t *testing.T) {
	limit := int64(200)
	require.NoError(t, notify.CheckQuota(notify.Quota{Limit: &limit, Used: 190}, 10))
	require.ErrorIs(t, notify.CheckQuota(notify.Quota{Limit: &limit, Used: 190}, 11), domain.ErrConflict)
	require.ErrorIs(t, notify.CheckQuota(notify.Quota{Limit: &limit, Used: 250}, 1), domain.ErrConflict)
	require.NoError(t, notify.CheckQuota(notify.Quota{Used: 99999}, 1000), "no limit")
}

// fakeStore は notify.Store を覚えるだけで実装する。宛先の結果と持ち時間は LINE の行 ID ごとに持つ。
type fakeStore struct {
	recipients []notify.Recipient
	batches    []notify.NewBatch
	byKey      map[string]int64
	results    map[int64]notify.JobResult
	claims     map[int64]time.Time
	states     map[int64]notify.BatchState
	notes      map[int64]notify.SkipReason
}

func (s *fakeStore) Tournament(_ context.Context, id int64) (entry.Tournament, error) {
	if id != 1 {
		return entry.Tournament{}, domain.NotFound(domain.ResourceTournament, id)
	}
	return tournament(), nil
}

func (s *fakeStore) Recipients(context.Context, int64, notify.Audience) ([]notify.Recipient, error) {
	return s.recipients, nil
}

func (s *fakeStore) FindBatch(_ context.Context, key string) (int64, bool, error) {
	id, ok := s.byKey[key]
	return id, ok, nil
}

func (s *fakeStore) CreateBatch(_ context.Context, batch notify.NewBatch) (int64, bool, error) {
	if id, ok := s.byKey[batch.RequestKey]; ok {
		return id, false, nil
	}
	s.batches = append(s.batches, batch)
	id := int64(len(s.batches))
	s.byKey[batch.RequestKey] = id
	s.states[id] = notify.BatchSent
	if batch.SendAt != nil {
		s.states[id] = notify.BatchScheduled
	}
	for _, job := range batch.Jobs {
		s.results[job.LineUserID] = notify.JobResult{Status: job.Status}
		if job.Status == notify.JobPending {
			s.claims[job.LineUserID] = batch.ClaimedUntil
		}
	}
	return id, true, nil
}

func (s *fakeStore) claimable(lineUserID int64, now time.Time) bool {
	switch s.results[lineUserID].Status {
	case notify.JobFailed:
		return true
	case notify.JobPending:
		return s.claims[lineUserID].Before(now)
	default:
		return false
	}
}

func (s *fakeStore) Claim(_ context.Context, id int64, now, until time.Time) ([]notify.Group, error) {
	var groups []notify.Group
	for _, job := range s.batches[id-1].Jobs {
		if job.RetryKey == "" || !s.claimable(job.LineUserID, now) {
			continue
		}
		s.results[job.LineUserID] = notify.JobResult{Status: notify.JobPending}
		s.claims[job.LineUserID] = until
		if len(groups) == 0 || groups[len(groups)-1].RetryKey != job.RetryKey {
			groups = append(groups, notify.Group{RetryKey: job.RetryKey})
		}
		group := &groups[len(groups)-1]
		group.LineUserIDs = append(group.LineUserIDs, job.LineUserID)
		group.UserIDs = append(group.UserIDs, fmt.Sprintf("U%d", job.LineUserID))
	}
	return groups, nil
}

func (s *fakeStore) FinishJobs(_ context.Context, _ int64, ids []int64, result notify.JobResult) error {
	for _, id := range ids {
		s.results[id] = result
	}
	return nil
}

func (s *fakeStore) Batches(context.Context, int64, time.Time) ([]notify.Batch, error) {
	return nil, nil
}

func (s *fakeStore) DueBatches(_ context.Context, now time.Time, limit int) ([]notify.Due, error) {
	var out []notify.Due
	for i, batch := range s.batches {
		if s.states[int64(i+1)] == notify.BatchScheduled && !batch.SendAt.After(now) && len(out) < limit {
			out = append(out, notify.Due{ID: int64(i + 1), TournamentID: batch.TournamentID,
				Audience: batch.Audience, Body: batch.Body})
		}
	}
	return out, nil
}

func (s *fakeStore) move(id int64, to notify.BatchState) bool {
	if s.states[id] != notify.BatchScheduled {
		return false
	}
	s.states[id] = to
	return true
}

func (s *fakeStore) StartScheduled(_ context.Context, id int64, jobs []notify.NewJob, until time.Time) (bool, error) {
	if !s.move(id, notify.BatchSent) {
		return false, nil
	}
	s.batches[id-1].Jobs = jobs
	for _, job := range jobs {
		s.results[job.LineUserID] = notify.JobResult{Status: job.Status}
		if job.Status == notify.JobPending {
			s.claims[job.LineUserID] = until
		}
	}
	return true, nil
}

func (s *fakeStore) SkipScheduled(_ context.Context, id int64, reason notify.SkipReason) (bool, error) {
	s.notes[id] = reason
	return s.move(id, notify.BatchSkipped), nil
}

func (s *fakeStore) CancelScheduled(_ context.Context, id int64) (bool, error) {
	return s.move(id, notify.BatchCancelled), nil
}

func (s *fakeStore) Batch(_ context.Context, id int64, now time.Time) (notify.Batch, error) {
	batch := s.batches[id-1]
	out := notify.Batch{ID: id, TournamentID: batch.TournamentID, Body: batch.Body, CreatedAt: batch.At,
		SendAt: batch.SendAt, State: s.states[id], Note: s.notes[id]}
	for _, job := range batch.Jobs {
		switch s.results[job.LineUserID].Status {
		case notify.JobSent:
			out.Counts.Sent++
		case notify.JobSkipped:
			out.Counts.Skipped++
		case notify.JobFailed:
			out.Counts.Failed++
		case notify.JobPending:
			out.Counts.Pending++
		}
		if s.claimable(job.LineUserID, now) {
			out.Counts.Resendable++
		}
	}
	return out, nil
}

// fakeSender は送ったものを覚える。failures が残っていれば、先頭の誤りで失敗する。
type fakeSender struct {
	calls    []multicast
	failures []error
	quota    notify.Quota
}

type multicast struct {
	to      []string
	text    string
	buttons []notify.Button
	key     string
}

func (s *fakeSender) Multicast(_ context.Context, to []string, message notify.Message, retryKey string) error {
	s.calls = append(s.calls, multicast{to: to, text: message.Text, buttons: message.Buttons, key: retryKey})
	if len(s.failures) > 0 {
		err := s.failures[0]
		s.failures = s.failures[1:]
		return err
	}
	return nil
}

func (s *fakeSender) Quota(context.Context) (notify.Quota, error) { return s.quota, nil }

// temporary は LINE が受け付けたか分からない失敗、refused は受け付けなかった失敗。
func temporary() error {
	return fmt.Errorf("%w: LINE /v2/bot/message/multicast returned 500: {}", notify.ErrTemporary)
}

func refused() error {
	return errors.New(`LINE /v2/bot/message/multicast returned 400: {"message":"bad"}`)
}

func friends(count int) []notify.Recipient {
	out := make([]notify.Recipient, count)
	for i := range out {
		out[i] = notify.Recipient{LineUserID: int64(i + 1), UserID: fmt.Sprintf("U%d", i+1), Friend: true,
			Name: fmt.Sprintf("代表 %d", i+1), Teams: []string{"熱海"}}
	}
	return out
}

// sequence は毎回違う送り直しのキーになる乱数の元(0 から 255 を繰り返す)。
func sequence() *bytes.Reader {
	out := make([]byte, 4096)
	for i := range out {
		out[i] = byte(i)
	}
	return bytes.NewReader(out)
}

func newService(store *fakeStore, sender *fakeSender, reported *[]error, now *time.Time) *notify.Service {
	report := func(err error) { *reported = append(*reported, err) }
	return notify.NewService(store, notify.WithSender(sender, report), notify.WithRandom(sequence()),
		notify.WithClock(func() time.Time { return *now }))
}

func newFakeStore(recipients []notify.Recipient) *fakeStore {
	return &fakeStore{recipients: recipients, byKey: map[string]int64{}, results: map[int64]notify.JobResult{},
		claims: map[int64]time.Time{}, states: map[int64]notify.BatchState{}, notes: map[int64]notify.SkipReason{}}
}

func clock() *time.Time {
	now := time.Date(2027, 2, 8, 9, 0, 0, 0, time.UTC)
	return &now
}

// 友だちには 500 人ずつ、まとまりごとに決めたキーで送り、友だちでない人は送らずに記録する。
// 同じ画面からの 2 回目は、配信数が減っていても確かめ直さずに前の記録を返す。
func TestSendInChunksAndOnce(t *testing.T) {
	recipients := append(friends(501), notify.Recipient{LineUserID: 999, UserID: "Ublocked", Name: "ブロック中"})
	store := newFakeStore(recipients)
	sender := &fakeSender{}
	var reported []error
	service := newService(store, sender, &reported, clock())
	request := notify.Request{Audience: notify.Audience{Kind: notify.AudienceReps}, Body: "明日は 8 時集合です\r\n",
		RequestKey: "key-1"}
	batch, err := service.Send(context.Background(), 1, request, domain.Actor{UserID: 7})
	require.NoError(t, err)
	assert.Equal(t, notify.Counts{Sent: 501, Skipped: 1}, batch.Counts)
	assert.Equal(t, store.batches[0].At.Add(notify.SendLease), store.batches[0].ClaimedUntil)
	assert.False(t, batch.CanResend)
	require.Len(t, sender.calls, 2)
	assert.Len(t, sender.calls[0].to, notify.MaxMulticast)
	assert.Equal(t, []string{"U501"}, sender.calls[1].to)
	assert.Equal(t, "ルーキーズファイナル\n\n明日は 8 時集合です", sender.calls[0].text)
	uuid := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	assert.Regexp(t, uuid, sender.calls[0].key)
	assert.NotEqual(t, sender.calls[0].key, sender.calls[1].key)
	require.Len(t, store.batches, 1)
	jobs := store.batches[0].Jobs
	assert.Equal(t, sender.calls[0].key, jobs[0].RetryKey, "the key is recorded before sending")
	assert.Equal(t, sender.calls[1].key, jobs[500].RetryKey)
	assert.Empty(t, jobs[501].RetryKey, "no key for someone who is not sent to")
	assert.Equal(t, int64(7), store.batches[0].SentBy)
	assert.Equal(t, "熱海", jobs[0].Teams)
	assert.Empty(t, reported)

	limit := int64(1000)
	sender.quota = notify.Quota{Limit: &limit, Used: 1000}
	request.Body = ""
	again, err := service.Send(context.Background(), 1, request, domain.Actor{UserID: 7})
	require.NoError(t, err, "the second press is answered with the first send, not re-checked")
	assert.Equal(t, batch.ID, again.ID)
	assert.Len(t, sender.calls, 2, "the same request is not sent twice")
	_, err = service.Send(context.Background(), 2, request, domain.Actor{UserID: 7})
	require.ErrorIs(t, err, domain.ErrConflict, "the key of another tournament")
}

// LINE が受け付けたか分からない失敗だけ、同じキーですぐ 1 回送り直す。受け付けなかったと分かる
// 失敗は送り直さない。
func TestSendRetriesOnlyTemporaryFailures(t *testing.T) {
	store := newFakeStore(friends(2))
	sender := &fakeSender{failures: []error{temporary()}}
	var reported []error
	service := newService(store, sender, &reported, clock())
	request := notify.Request{Audience: notify.Audience{Kind: notify.AudienceReps}, Body: "x", RequestKey: "key-1"}
	batch, err := service.Send(context.Background(), 1, request, domain.Actor{UserID: 7})
	require.NoError(t, err)
	assert.Equal(t, notify.Counts{Sent: 2}, batch.Counts)
	require.Len(t, sender.calls, 2)
	assert.Equal(t, sender.calls[0].key, sender.calls[1].key)

	sender.calls, sender.failures = nil, []error{refused()}
	request.RequestKey = "key-2"
	batch, err = service.Send(context.Background(), 1, request, domain.Actor{UserID: 7})
	require.NoError(t, err, "the batch is recorded even when LINE refuses it")
	assert.Equal(t, notify.Counts{Failed: 2, Resendable: 2}, batch.Counts)
	assert.Len(t, sender.calls, 1, "a refusal is not retried at once")
	assert.Contains(t, store.results[1].Error, "returned 400")
	assert.True(t, batch.CanResend)
	require.Len(t, reported, 1)
}

// 送れなかったまとまりは、記録したキーと宛先と文のまま、送ってから 24 時間のあいだ送り直せる。
func TestResendUsesTheRecordedGroups(t *testing.T) {
	store := newFakeStore(friends(501))
	sender := &fakeSender{failures: []error{nil, temporary(), temporary()}}
	var reported []error
	now := clock()
	service := newService(store, sender, &reported, now)
	request := notify.Request{Audience: notify.Audience{Kind: notify.AudienceReps}, Body: "x", RequestKey: "key-1"}
	batch, err := service.Send(context.Background(), 1, request, domain.Actor{UserID: 7})
	require.NoError(t, err)
	assert.Equal(t, notify.Counts{Sent: 500, Failed: 1, Resendable: 1}, batch.Counts)
	require.Len(t, sender.calls, 3)
	failedKey := sender.calls[1].key

	*now = now.Add(time.Hour)
	batch, err = service.Resend(context.Background(), batch.ID)
	require.NoError(t, err)
	assert.Equal(t, notify.Counts{Sent: 501}, batch.Counts)
	assert.False(t, batch.CanResend)
	require.Len(t, sender.calls, 4, "only the group that failed is sent again")
	assert.Equal(t, multicast{to: []string{"U501"}, text: "ルーキーズファイナル\n\nx", key: failedKey}, sender.calls[3])

	again, err := service.Resend(context.Background(), batch.ID)
	require.NoError(t, err)
	assert.Len(t, sender.calls, 4, "nothing is left to resend")
	assert.Equal(t, notify.Counts{Sent: 501}, again.Counts)

	sender.failures = []error{refused()}
	request.RequestKey = "key-2"
	late, err := service.Send(context.Background(), 1, request, domain.Actor{UserID: 7})
	require.NoError(t, err)
	*now = now.Add(notify.ResendWindow - time.Hour)
	read, err := service.Batch(context.Background(), late.ID)
	require.NoError(t, err)
	assert.True(t, read.CanResend)
	*now = now.Add(time.Hour)
	read, err = service.Batch(context.Background(), late.ID)
	require.NoError(t, err)
	assert.False(t, read.CanResend, "the retry keys have expired")
	_, err = service.Resend(context.Background(), late.ID)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestSendRefusals(t *testing.T) {
	request := notify.Request{Audience: notify.Audience{Kind: notify.AudienceReps}, Body: "x", RequestKey: "key-1"}
	ctx := context.Background()

	_, err := notify.NewService(newFakeStore(friends(1))).Send(ctx, 1, request, domain.Actor{UserID: 7})
	require.ErrorIs(t, err, domain.ErrConflict, "nothing is sent without the LINE settings")

	var reported []error
	blocked := newFakeStore([]notify.Recipient{{LineUserID: 1, UserID: "U1", Name: "ブロック中"}})
	_, err = newService(blocked, &fakeSender{}, &reported, clock()).Send(ctx, 1, request, domain.Actor{UserID: 7})
	var problem *domain.ValidationError
	require.ErrorAs(t, err, &problem, "no one can receive it")

	limit := int64(200)
	store := newFakeStore(friends(3))
	sender := &fakeSender{quota: notify.Quota{Limit: &limit, Used: 198}}
	_, err = newService(store, sender, &reported, clock()).Send(ctx, 1, request, domain.Actor{UserID: 7})
	require.ErrorIs(t, err, domain.ErrConflict, "the monthly quota is not enough")
	assert.Empty(t, store.batches)
	assert.Empty(t, sender.calls)
}

func TestPreviewCountsAndQuota(t *testing.T) {
	limit := int64(200)
	recipients := append(friends(2), notify.Recipient{LineUserID: 9, UserID: "U9", Name: "ブロック中"})
	var reported []error
	service := newService(newFakeStore(recipients), &fakeSender{quota: notify.Quota{Limit: &limit, Used: 5}},
		&reported, clock())
	preview, err := service.Preview(context.Background(), 1, notify.Audience{Kind: notify.AudienceReps})
	require.NoError(t, err)
	assert.Equal(t, "ルーキーズファイナル", preview.Heading)
	assert.Equal(t, 2, preview.Reachable)
	assert.Equal(t, 1, preview.Unreachable)
	require.NotNil(t, preview.Quota)
	assert.Equal(t, int64(5), preview.Quota.Used)
	assert.True(t, preview.CanSend)

	without, err := notify.NewService(newFakeStore(recipients)).Preview(context.Background(), 1,
		notify.Audience{Kind: notify.AudienceReps})
	require.NoError(t, err)
	assert.False(t, without.CanSend)
	assert.Nil(t, without.Quota)
	assert.Len(t, without.Recipients, 3, "the recipients can be checked without the LINE settings")
}

// 送っている途中のまとまりは、送っている送信の持ち時間が切れるまで送り直さない(同じキーの要求を
// LINE に同時に送らない)。切れたら、途中で止まったものとして送り直せる。
func TestResendWaitsForARunningSend(t *testing.T) {
	store := newFakeStore(friends(2))
	now := clock()
	var reported []error
	service := newService(store, &fakeSender{}, &reported, now)
	_, _, err := store.CreateBatch(context.Background(), notify.NewBatch{TournamentID: 1, RequestKey: "running",
		Body: "大会\n\nx", At: *now, ClaimedUntil: now.Add(notify.SendLease), Jobs: []notify.NewJob{
			{LineUserID: 1, Status: notify.JobPending, RetryKey: "group-1"},
			{LineUserID: 2, Status: notify.JobPending, RetryKey: "group-1"},
		}})
	require.NoError(t, err)
	running, err := service.Batch(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, running.CanResend, "another send still holds the group")
	sender := &fakeSender{}
	service = newService(store, sender, &reported, now)
	_, err = service.Resend(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, sender.calls)

	*now = now.Add(notify.SendLease + time.Second)
	stopped, err := service.Batch(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, stopped.CanResend, "the send stopped part way")
	resent, err := service.Resend(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, sender.calls, 1)
	assert.Equal(t, []string{"U1", "U2"}, sender.calls[0].to)
	assert.Equal(t, "group-1", sender.calls[0].key)
	assert.Equal(t, notify.Counts{Sent: 2}, resent.Counts)
}

// 時刻を決めた送信は宛先を持たない予約として残り、送る時刻が来たら、そのときの条件の宛先に送る。
func TestScheduleAndDispatch(t *testing.T) {
	store := newFakeStore(nil)
	sender := &fakeSender{}
	var reported []error
	now := clock()
	service := newService(store, sender, &reported, now)
	ctx := context.Background()
	eve := now.Add(30 * time.Hour)
	request := notify.Request{Audience: notify.Audience{Kind: notify.AudienceParticipants},
		Body: "明日は 8 時集合です", RequestKey: "key-1", SendAt: &eve}
	scheduled, err := service.Send(ctx, 1, request, domain.Actor{UserID: 7})
	require.NoError(t, err, "no one has to be approved yet")
	assert.Equal(t, notify.BatchScheduled, scheduled.State)
	assert.Equal(t, "ルーキーズファイナル\n\n明日は 8 時集合です", scheduled.Body)
	require.NotNil(t, scheduled.SendAt)
	assert.Empty(t, sender.calls)

	result, err := service.Dispatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, notify.DispatchResult{}, result, "not yet")

	store.recipients = friends(3)
	*now = eve.Add(time.Minute)
	result, err = service.Dispatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, notify.DispatchResult{Sent: 1}, result)
	require.Len(t, sender.calls, 1)
	assert.Equal(t, []string{"U1", "U2", "U3"}, sender.calls[0].to, "the recipients are counted when it is sent")
	assert.Equal(t, "ルーキーズファイナル\n\n明日は 8 時集合です", sender.calls[0].text)
	sent, err := service.Batch(ctx, scheduled.ID)
	require.NoError(t, err)
	assert.Equal(t, notify.BatchSent, sent.State)
	assert.Equal(t, notify.Counts{Sent: 3}, sent.Counts)

	result, err = service.Dispatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, notify.DispatchResult{}, result, "a schedule is sent once")
	assert.Len(t, sender.calls, 1)
}

func TestScheduleChecksTheTime(t *testing.T) {
	var reported []error
	now := clock()
	service := newService(newFakeStore(nil), &fakeSender{}, &reported, now)
	for _, at := range []time.Time{now.Add(-time.Minute), *now, now.Add(notify.MaxScheduleAhead + time.Hour)} {
		request := notify.Request{Audience: notify.Audience{Kind: notify.AudienceReps}, Body: "x",
			RequestKey: "key-1", SendAt: &at}
		_, err := service.Send(context.Background(), 1, request, domain.Actor{UserID: 7})
		var problem *domain.ValidationError
		require.ErrorAs(t, err, &problem, "%s", at)
	}
}

// 送る時刻に送れない予約は、理由を付けて送れなかったことにする。取りやめた予約は送らない。
func TestDispatchSkipsAndCancel(t *testing.T) {
	limit := int64(200)
	store := newFakeStore(nil)
	sender := &fakeSender{quota: notify.Quota{Limit: &limit, Used: 199}}
	var reported []error
	now := clock()
	service := newService(store, sender, &reported, now)
	ctx := context.Background()
	at := now.Add(time.Hour)
	schedule := func(key string) notify.Batch {
		t.Helper()
		batch, err := service.Send(ctx, 1, notify.Request{Audience: notify.Audience{Kind: notify.AudienceReps},
			Body: "x", RequestKey: key, SendAt: &at}, domain.Actor{UserID: 7})
		require.NoError(t, err)
		return batch
	}
	nobody, cancelled := schedule("nobody"), schedule("cancelled")
	cancelledRead, err := service.Cancel(ctx, cancelled.ID)
	require.NoError(t, err)
	assert.Equal(t, notify.BatchCancelled, cancelledRead.State)
	_, err = service.Cancel(ctx, cancelled.ID)
	require.ErrorIs(t, err, domain.ErrConflict, "only a scheduled message can be cancelled")

	*now = at.Add(time.Minute)
	result, err := service.Dispatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, notify.DispatchResult{Skipped: 1}, result)
	read, err := service.Batch(ctx, nobody.ID)
	require.NoError(t, err)
	assert.Equal(t, notify.BatchSkipped, read.State)
	assert.Equal(t, notify.SkipNoRecipients, read.Note)

	*now = at.Add(-2 * time.Minute)
	short := schedule("short")
	store.recipients = friends(2)
	*now = at.Add(time.Minute)
	result, err = service.Dispatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, notify.DispatchResult{Skipped: 1}, result)
	read, err = service.Batch(ctx, short.ID)
	require.NoError(t, err)
	assert.Equal(t, notify.SkipQuota, read.Note, "nothing is sent when only part would arrive")
	assert.Empty(t, sender.calls)
}

// 予約は送る時刻から 24 時間のあいだ送り直せる(予約した時刻からではない)。
func TestResendAScheduledSendFromItsSendTime(t *testing.T) {
	store := newFakeStore(nil)
	sender := &fakeSender{}
	var reported []error
	now := clock()
	service := newService(store, sender, &reported, now)
	ctx := context.Background()
	eve := now.Add(7 * 24 * time.Hour)
	scheduled, err := service.Send(ctx, 1, notify.Request{Audience: notify.Audience{Kind: notify.AudienceParticipants},
		Body: "x", RequestKey: "key-1", SendAt: &eve}, domain.Actor{UserID: 7})
	require.NoError(t, err)

	store.recipients = friends(2)
	sender.failures = []error{refused()}
	*now = eve.Add(time.Minute)
	_, err = service.Dispatch(ctx)
	require.NoError(t, err)
	read, err := service.Batch(ctx, scheduled.ID)
	require.NoError(t, err)
	assert.Equal(t, notify.Counts{Failed: 2, Resendable: 2}, read.Counts)
	assert.True(t, read.CanResend, "a week after it was scheduled, but just after it was sent")

	*now = eve.Add(time.Hour)
	resent, err := service.Resend(ctx, scheduled.ID)
	require.NoError(t, err)
	assert.Equal(t, notify.Counts{Sent: 2}, resent.Counts)
}

// ボタンは 3 つまで。申込とスコアサイトは 1 つずつで、文字を書かなければ決まった文字にする。URL は https だけ。
func TestCheckButtons(t *testing.T) {
	got, err := notify.CheckButtons([]notify.Button{{Kind: notify.ButtonEntry}, {Kind: notify.ButtonSite, Label: " 結果 "},
		{Kind: notify.ButtonURL, Label: "会場の地図", URL: " https://example.com/map "}})
	require.NoError(t, err)
	assert.Equal(t, []notify.Button{{Kind: notify.ButtonEntry, Label: "申込を開く"}, {Kind: notify.ButtonSite, Label: "結果"},
		{Kind: notify.ButtonURL, Label: "会場の地図", URL: "https://example.com/map"}}, got)
	for name, buttons := range map[string][]notify.Button{
		"too many":    {{Kind: notify.ButtonEntry}, {Kind: notify.ButtonSite}, {Kind: notify.ButtonURL, Label: "a", URL: "https://a.example"}, {Kind: notify.ButtonURL, Label: "b", URL: "https://b.example"}},
		"entry twice": {{Kind: notify.ButtonEntry}, {Kind: notify.ButtonEntry}},
		"http":        {{Kind: notify.ButtonURL, Label: "a", URL: "http://example.com"}},
		"no label":    {{Kind: notify.ButtonURL, URL: "https://example.com"}},
		"long label":  {{Kind: notify.ButtonURL, Label: strings.Repeat("あ", 21), URL: "https://example.com"}},
		"unknown":     {{Kind: "call"}},
	} {
		_, err := notify.CheckButtons(buttons)
		require.ErrorIs(t, err, domain.ErrBadRequest, name)
	}
}

// ボタンは文と一緒に届け、スコアサイトのボタンはその大会のページを開く。記録にもボタンを残す。
func TestSendWithButtons(t *testing.T) {
	store := newFakeStore(friends(2))
	sender := &fakeSender{}
	var reported []error
	now := clock()
	service := notify.NewService(store, notify.WithSender(sender, func(err error) { reported = append(reported, err) }),
		notify.WithRandom(sequence()), notify.WithClock(func() time.Time { return *now }),
		notify.WithSiteURL("https://score.example/"))
	request := notify.Request{Audience: notify.Audience{Kind: notify.AudienceReps}, Body: "明日は 8 時集合です",
		RequestKey: "key-1", Buttons: []notify.Button{{Kind: notify.ButtonSite}, {Kind: notify.ButtonEntry}}}
	_, err := service.Send(context.Background(), 1, request, domain.Actor{UserID: 7})
	require.NoError(t, err)
	want := []notify.Button{{Kind: notify.ButtonSite, Label: "スコアサイトを見る", URL: "https://score.example/tournaments/1"},
		{Kind: notify.ButtonEntry, Label: "申込を開く"}}
	require.Len(t, sender.calls, 1)
	assert.Equal(t, want, sender.calls[0].buttons)
	assert.Equal(t, want, store.batches[0].Buttons)

	// スコアサイトの URL が無ければ、スコアサイトのボタンは使えない
	_, err = newService(newFakeStore(friends(1)), &fakeSender{}, &reported, now).Send(context.Background(), 1,
		notify.Request{Audience: notify.Audience{Kind: notify.AudienceReps}, Body: "x", RequestKey: "key-2",
			Buttons: []notify.Button{{Kind: notify.ButtonSite}}}, domain.Actor{UserID: 7})
	require.ErrorIs(t, err, domain.ErrBadRequest)
}
