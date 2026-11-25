package store_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/notify"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

type notifyFixture struct {
	db    *gorm.DB
	at    time.Time
	codes int
}

func (f *notifyFixture) lineUser(t *testing.T, id, name string, friend bool) int64 {
	t.Helper()
	row := store.LineUser{LineUserID: id, Friend: friend, Name: name, CreatedAt: f.at, UpdatedAt: f.at}
	require.NoError(t, f.db.Create(&row).Error)
	if !friend {
		// friend の既定値は真なので、偽はあとから書く(GORM は零値を INSERT に入れない)
		require.NoError(t, f.db.Model(&row).Update("friend", false).Error)
	}
	return row.ID
}

// teamEntry は LINE からの申込を作る(rep が 0 なら運営の代理入力)。members の名前の行を、registered の
// 数だけ先頭から LINE の人に結ぶ。
func (f *notifyFixture) teamEntry(
	t *testing.T,
	division store.Division,
	name string,
	rep int64,
	status entry.Status,
	members []string,
	registered map[string]int64,
) {
	t.Helper()
	f.codes++
	row := store.TeamEntry{TournamentID: division.TournamentID, DivisionID: division.ID, TeamNameFull: name,
		TeamNameShort: name, RepName: name + "の代表", Channel: string(entry.ChannelLine), Status: string(status),
		InviteCode: fmt.Sprintf("code-%d", f.codes), SubmittedAt: f.at.Add(time.Duration(f.codes) * time.Minute),
		CreatedAt: f.at, UpdatedAt: f.at}
	if rep != 0 {
		row.RepLineUserID = &rep
	} else {
		row.Channel = string(entry.ChannelDesk)
	}
	if status == entry.StatusApproved {
		row.DecidedAt = &f.at
	}
	require.NoError(t, f.db.Create(&row).Error)
	for i, member := range members {
		line := registered[member]
		memberRow := store.MemberEntry{TeamEntryID: row.ID, Name: member, SortOrder: i, CreatedAt: f.at, UpdatedAt: f.at}
		if line != 0 {
			memberRow.LineUserID = &line
		}
		require.NoError(t, f.db.Create(&memberRow).Error)
	}
}

func names(recipients []notify.Recipient) []string {
	out := make([]string, len(recipients))
	for i, recipient := range recipients {
		out[i] = recipient.Name
	}
	return out
}

func TestNotifyStoreRecipients(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	f := &notifyFixture{db: db, at: time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)}
	tour := b.Tournament()
	open := b.Division(tour.ID)
	women := b.Division(tour.ID)
	past := b.Tournament()
	pastDivision := b.Division(past.ID)

	atami := f.lineUser(t, "Uatami", "", true)
	kobe := f.lineUser(t, "Ukobe", "神戸の代表(前回の入力)", true)
	blocked := f.lineUser(t, "Ublocked", "ブロック中", false)
	member := f.lineUser(t, "Umember", "佐藤 花子", true)
	returning := f.lineUser(t, "Ureturning", "去年の代表", true)
	f.teamEntry(t, open, "熱海", atami, entry.StatusApproved, []string{"佐藤", "鈴木"}, map[string]int64{"佐藤": member})
	f.teamEntry(t, women, "神戸", kobe, entry.StatusPending, []string{"高橋"}, map[string]int64{"高橋": kobe})
	f.teamEntry(t, open, "熱海 B", atami, entry.StatusReturned, []string{"田中"}, nil)
	f.teamEntry(t, open, "取消", blocked, entry.StatusCancelled, []string{"中村"}, nil)
	f.teamEntry(t, open, "代理", 0, entry.StatusPending, []string{"小林"}, nil)
	f.teamEntry(t, pastDivision, "去年", returning, entry.StatusApproved, nil, nil)
	f.teamEntry(t, pastDivision, "去年の熱海", atami, entry.StatusApproved, nil, nil)
	f.teamEntry(t, pastDivision, "去年の取消", blocked, entry.StatusCancelled, nil, nil)

	notices := store.NewNotifyStore(db)
	ctx := context.Background()
	read := func(audience notify.Audience) []notify.Recipient {
		t.Helper()
		recipients, err := notices.Recipients(ctx, tour.ID, audience)
		require.NoError(t, err)
		return recipients
	}

	reps := read(notify.Audience{Kind: notify.AudienceReps})
	assert.Equal(t, []string{"熱海の代表", "神戸の代表(前回の入力)"}, names(reps),
		"the same person once, the cancelled and desk entries left out")
	assert.Equal(t, []string{"熱海", "熱海 B"}, reps[0].Teams)
	assert.Equal(t, "Uatami", reps[0].UserID)
	assert.True(t, reps[0].Friend)
	assert.Equal(t, []string{"神戸の代表(前回の入力)"},
		names(read(notify.Audience{Kind: notify.AudienceReps, Status: entry.StatusPending})))
	assert.Equal(t, []string{"神戸の代表(前回の入力)"},
		names(read(notify.Audience{Kind: notify.AudienceReps, DivisionID: women.ID})))

	incomplete := read(notify.Audience{Kind: notify.AudienceIncomplete})
	assert.Equal(t, []string{"熱海の代表"}, names(incomplete), "Kobe has every member registered")
	assert.Equal(t, []string{"熱海", "熱海 B"}, incomplete[0].Teams)

	assert.Equal(t, []string{"熱海の代表", "佐藤 花子"},
		names(read(notify.Audience{Kind: notify.AudienceParticipants})), "approved entries only")
	assert.Empty(t, read(notify.Audience{Kind: notify.AudienceParticipants, DivisionID: women.ID}))

	pastReps := read(notify.Audience{Kind: notify.AudiencePastReps})
	assert.Equal(t, []string{"去年の代表"}, names(pastReps), "those who already applied here are left out")
	assert.Equal(t, []string{"去年"}, pastReps[0].Teams)

	// 取り消した申込しかない人(ブロック中)は代表として数えない
	recipients, err := notices.Recipients(ctx, past.ID, notify.Audience{Kind: notify.AudienceReps})
	require.NoError(t, err)
	assert.Equal(t, []string{"去年の代表", "去年の熱海の代表"}, names(recipients))
}

func TestNotifyStoreBatches(t *testing.T) {
	db := testdb.New(t)
	b := storetest.New(t, db)
	f := &notifyFixture{db: db, at: time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)}
	tour := b.Tournament()
	admin, err := store.CreateUser(context.Background(), db,
		store.UserFields{Email: "admin@example.com", DisplayName: "運営 一郎", Role: domain.RoleAdmin}, "hash")
	require.NoError(t, err)
	first := f.lineUser(t, "U1", "代表 1", true)
	second := f.lineUser(t, "U2", "代表 2", false)

	notices := store.NewNotifyStore(db)
	ctx := context.Background()
	lease := f.at.Add(notify.SendLease)
	batch := notify.NewBatch{TournamentID: tour.ID, RequestKey: "key-1", ClaimedUntil: lease,
		Audience: notify.Audience{Kind: notify.AudienceReps, Status: entry.StatusPending, DivisionID: 3},
		Body:     "大会\n\n明日は 8 時集合", SentBy: admin.ID, At: f.at, Jobs: []notify.NewJob{
			{LineUserID: first, Name: "代表 1", Teams: "熱海", Status: notify.JobPending, RetryKey: "group-1"},
			{LineUserID: second, Name: "代表 2", Teams: "神戸、神戸 B", Status: notify.JobSkipped},
		}}
	id, created, err := notices.CreateBatch(ctx, batch)
	require.NoError(t, err)
	assert.True(t, created)
	again, created, err := notices.CreateBatch(ctx, batch)
	require.NoError(t, err)
	assert.False(t, created, "the same request key is recorded once")
	assert.Equal(t, id, again)

	found, ok, err := notices.FindBatch(ctx, "key-1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, id, found)
	_, ok, err = notices.FindBatch(ctx, "nothing")
	require.NoError(t, err)
	assert.False(t, ok)
	claimed, err := notices.Claim(ctx, id, f.at, f.at.Add(time.Minute))
	require.NoError(t, err)
	assert.Empty(t, claimed, "the first send still holds the group")
	claimed, err = notices.Claim(ctx, id, lease.Add(time.Second), lease.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, []notify.Group{{RetryKey: "group-1", LineUserIDs: []int64{first}, UserIDs: []string{"U1"}}},
		claimed, "a stopped group can be claimed; the skipped recipient has no group")
	claimed, err = notices.Claim(ctx, id, lease.Add(time.Second), lease.Add(time.Hour))
	require.NoError(t, err)
	assert.Empty(t, claimed, "a group is claimed only once")

	sentAt := f.at.Add(time.Second)
	require.NoError(t, notices.FinishJobs(ctx, id, []int64{first}, notify.JobResult{Status: notify.JobSent, At: sentAt}))
	claimed, err = notices.Claim(ctx, id, lease.Add(2*time.Hour), lease.Add(3*time.Hour))
	require.NoError(t, err)
	assert.Empty(t, claimed, "a sent group is not claimed")

	read, err := notices.Batch(ctx, id, f.at)
	require.NoError(t, err)
	assert.Equal(t, batch.Audience, read.Audience)
	assert.Equal(t, "運営 一郎", read.SentByName)
	assert.Equal(t, notify.Counts{Sent: 1, Skipped: 1}, read.Counts)
	require.Len(t, read.Jobs, 2)
	assert.Equal(t, notify.JobSent, read.Jobs[0].Status)
	require.NotNil(t, read.Jobs[0].SentAt)
	assert.True(t, sentAt.Equal(*read.Jobs[0].SentAt))
	assert.Equal(t, "神戸、神戸 B", read.Jobs[1].Teams)

	batch.RequestKey, batch.At = "key-2", f.at.Add(time.Hour)
	newer, _, err := notices.CreateBatch(ctx, batch)
	require.NoError(t, err)
	list, err := notices.Batches(ctx, tour.ID, f.at)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, newer, list[0].ID, "newest first")
	assert.Equal(t, notify.Counts{Pending: 1, Skipped: 1}, list[0].Counts)
	assert.Equal(t, notify.Counts{Sent: 1, Skipped: 1}, list[1].Counts)
	assert.Equal(t, "運営 一郎", list[1].SentByName)

	other := f.lineUser(t, "U3", "代表 3", true)
	failedBatch := notify.NewBatch{TournamentID: tour.ID, RequestKey: "key-failed", Audience: batch.Audience,
		Body: "x", SentBy: admin.ID, At: f.at, Jobs: []notify.NewJob{
			{LineUserID: other, Name: "代表 3", Status: notify.JobPending, RetryKey: "group-2"}}}
	failedID, _, err := notices.CreateBatch(ctx, failedBatch)
	require.NoError(t, err)
	require.NoError(t, notices.FinishJobs(ctx, failedID, []int64{other},
		notify.JobResult{Status: notify.JobFailed, Error: "LINE returned 429", At: sentAt}))
	failedRead, err := notices.Batch(ctx, failedID, f.at)
	require.NoError(t, err)
	assert.Nil(t, failedRead.Jobs[0].SentAt, "nothing was sent")
	assert.Equal(t, "LINE returned 429", failedRead.Jobs[0].Error)
	assert.Equal(t, 1, failedRead.Counts.Resendable)
	claimed, err = notices.Claim(ctx, failedID, f.at, f.at.Add(time.Minute))
	require.NoError(t, err)
	assert.Len(t, claimed, 1, "a failed recipient can be resent at once")

	_, err = notices.Batch(ctx, 999999, f.at)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// 同時に送り直しても、1 つのまとまりは 1 つの送り直しにだけ渡る(同じキーの要求を LINE に同時に送らない)。
func TestNotifyStoreClaimIsExclusive(t *testing.T) {
	db := testdb.New(t)
	f := &notifyFixture{db: db, at: time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)}
	tour := storetest.New(t, db).Tournament()
	notices := store.NewNotifyStore(db)
	ctx := context.Background()
	jobs := make([]notify.NewJob, 0, 40)
	for i := range 40 {
		jobs = append(jobs, notify.NewJob{LineUserID: f.lineUser(t, fmt.Sprintf("U%d", i), "", true),
			Status: notify.JobPending, RetryKey: fmt.Sprintf("group-%d", i/10)})
	}
	id, _, err := notices.CreateBatch(ctx, notify.NewBatch{TournamentID: tour.ID, RequestKey: "key-1",
		Audience: notify.Audience{Kind: notify.AudienceReps}, Body: "x", At: f.at, ClaimedUntil: f.at, Jobs: jobs})
	require.NoError(t, err)

	later := f.at.Add(time.Minute)
	results := make([][]notify.Group, 8)
	var wait sync.WaitGroup
	for i := range results {
		wait.Go(func() {
			claimed, claimErr := notices.Claim(ctx, id, later, later.Add(notify.SendLease))
			assert.NoError(t, claimErr)
			results[i] = claimed
		})
	}
	wait.Wait()
	seen := map[string]int{}
	for _, claimed := range results {
		for _, group := range claimed {
			seen[group.RetryKey]++
			assert.Len(t, group.UserIDs, 10, "a group is claimed whole")
		}
	}
	assert.Equal(t, map[string]int{"group-0": 1, "group-1": 1, "group-2": 1, "group-3": 1}, seen)
}

// 予約は宛先を持たずに残り、送る時刻が来たら 1 回だけ宛先を書いて送ったことになる。
func TestNotifyStoreSchedules(t *testing.T) {
	db := testdb.New(t)
	f := &notifyFixture{db: db, at: time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)}
	tour := storetest.New(t, db).Tournament()
	notices := store.NewNotifyStore(db)
	ctx := context.Background()
	person := f.lineUser(t, "U1", "代表 1", true)
	eve := f.at.Add(30 * time.Hour)
	audience := notify.Audience{Kind: notify.AudienceParticipants}
	schedule := func(key string) int64 {
		t.Helper()
		id, created, err := notices.CreateBatch(ctx, notify.NewBatch{TournamentID: tour.ID, RequestKey: key,
			Audience: audience, Body: "大会\n\n明日", At: f.at, SendAt: &eve})
		require.NoError(t, err)
		require.True(t, created)
		return id
	}
	first, second, third := schedule("eve-1"), schedule("eve-2"), schedule("eve-3")

	read, err := notices.Batch(ctx, first, f.at)
	require.NoError(t, err)
	assert.Equal(t, notify.BatchScheduled, read.State)
	require.NotNil(t, read.SendAt)
	assert.True(t, eve.Equal(*read.SendAt))
	assert.Empty(t, read.Jobs)

	due, err := notices.DueBatches(ctx, eve.Add(-time.Minute), 10)
	require.NoError(t, err)
	assert.Empty(t, due, "not yet")
	due, err = notices.DueBatches(ctx, eve, 2)
	require.NoError(t, err)
	require.Len(t, due, 2, "the limit holds")
	assert.Equal(t, first, due[0].ID)
	assert.Equal(t, audience, due[0].Audience)
	assert.Equal(t, "大会\n\n明日", due[0].Body)

	jobs := []notify.NewJob{{LineUserID: person, Name: "代表 1", Status: notify.JobPending, RetryKey: "group-1"}}
	started, err := notices.StartScheduled(ctx, first, jobs, eve.Add(notify.SendLease))
	require.NoError(t, err)
	assert.True(t, started)
	started, err = notices.StartScheduled(ctx, first, jobs, eve.Add(notify.SendLease))
	require.NoError(t, err)
	assert.False(t, started, "a schedule starts once")
	read, err = notices.Batch(ctx, first, eve)
	require.NoError(t, err)
	assert.Equal(t, notify.BatchSent, read.State)
	assert.Equal(t, notify.Counts{Pending: 1}, read.Counts, "the group is held by the send that started it")

	skipped, err := notices.SkipScheduled(ctx, second, notify.SkipQuota)
	require.NoError(t, err)
	assert.True(t, skipped)
	read, err = notices.Batch(ctx, second, eve)
	require.NoError(t, err)
	assert.Equal(t, notify.BatchSkipped, read.State)
	assert.Equal(t, notify.SkipQuota, read.Note)
	cancelled, err := notices.CancelScheduled(ctx, second)
	require.NoError(t, err)
	assert.False(t, cancelled, "a skipped schedule cannot be cancelled")
	cancelled, err = notices.CancelScheduled(ctx, third)
	require.NoError(t, err)
	assert.True(t, cancelled)

	due, err = notices.DueBatches(ctx, eve.Add(time.Hour), 10)
	require.NoError(t, err)
	assert.Empty(t, due, "nothing is left to send")
	list, err := notices.Batches(ctx, tour.ID, eve)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}
