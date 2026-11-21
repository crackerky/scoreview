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

func (s *fakeStore) Batch(_ context.Context, id int64, now time.Time) (notify.Batch, error) {
	batch := s.batches[id-1]
	out := notify.Batch{ID: id, TournamentID: batch.TournamentID, Body: batch.Body, CreatedAt: batch.At}
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
	to   []string
	text string
	key  string
}

func (s *fakeSender) Multicast(_ context.Context, to []string, text, retryKey string) error {
	s.calls = append(s.calls, multicast{to: to, text: text, key: retryKey})
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
		claims: map[int64]time.Time{}}
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
