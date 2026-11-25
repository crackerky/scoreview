package store_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/notify"
	"scoreview/api/internal/store"
	"scoreview/api/internal/store/storetest"
	"scoreview/api/internal/testdb"
)

func lineID(letter string) string { return "U" + strings.Repeat(letter, 32) }

// 前の大会を移すと、終わった一覧に載せない大会に承認済みの申込が入り、参加者の台帳の人が結ばれる。
// 切り替えた後に代表が「エントリー」と送ると、前のチームと前回の入力が申込の画面に出る。
func TestImportPast(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	submitted := time.Date(2026, 4, 6, 4, 49, 40, 0, time.UTC)
	later := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	// この人はもうこちらで申し込み直していて、台帳の入力のほうが新しい
	recent := store.LineUser{LineUserID: lineID("c"), Friend: true, Name: "新しい名前", Email: "new@example.com",
		ProfileUpdatedAt: &later, CreatedAt: later, UpdatedAt: later}
	require.NoError(t, db.Create(&recent).Error)

	day := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	past := entry.PastTournament{Name: "若葉杯2026", StartDate: &day, Divisions: []entry.PastDivision{
		{Name: "オープン", Kind: domain.DivisionNone}, {Name: "ウィメン", Kind: domain.DivisionWomen}},
		Entries: []entry.PastEntry{
			{Division: 0, FullName: "千葉大学MISTRAL", RepName: "代表 一郎", RepEmail: "rep@example.com",
				RepLineID: lineID("a"), SubmittedAt: submitted, InviteCode: strings.Repeat("a", 26),
				Members: []entry.PastMember{
					{Name: "代表 一郎", Number: "10", LineID: lineID("a"), SubmittedAt: submitted.Add(time.Hour)},
					{Name: "古い名前", Number: "7", LineID: lineID("c"), SubmittedAt: submitted.Add(time.Hour)},
				}},
			{Division: 1, FullName: "千葉大学MISTRAL", RepName: "代表 二子", RepLineID: lineID("b"),
				SubmittedAt: submitted, InviteCode: strings.Repeat("b", 26)},
		}}
	require.NoError(t, entry.CheckPast(past))
	entries := store.NewEntryStore(db)
	id, err := entries.ImportPast(ctx, past)
	require.NoError(t, err)

	var tournament store.Tournament
	require.NoError(t, db.Where("id = ?", id).Take(&tournament).Error)
	assert.Equal(t, domain.TournamentFinished, tournament.Status)
	assert.False(t, tournament.Listed, "a moved tournament is not shown on the site")
	require.NotNil(t, tournament.StartDate)
	assert.Equal(t, "2026-05-31", tournament.StartDate.Format(time.DateOnly))

	list, err := entries.List(ctx, id)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, entry.StatusApproved, list[0].Status)
	assert.Equal(t, entry.ChannelLine, list[0].Channel)
	assert.Equal(t, 2, list[0].Members)
	assert.Equal(t, 2, list[0].Registered)
	assert.Equal(t, "ウィメン", list[1].Division.Name.Ja)

	var people []store.LineUser
	require.NoError(t, db.Where("line_user_id IN ?", []string{lineID("a"), lineID("c")}).Order("line_user_id").
		Find(&people).Error)
	require.Len(t, people, 2)
	assert.Equal(t, "代表 一郎", people[0].Name)
	assert.Equal(t, "rep@example.com", people[0].Email, "the member row has no email, so the rep's is kept")
	assert.True(t, people[0].Friend)
	assert.Equal(t, "新しい名前", people[1].Name, "a newer input is not overwritten by the moved one")
	assert.Equal(t, "new@example.com", people[1].Email)

	// 次の大会: 代表の前のチームと、前の大会の代表への知らせの宛先
	b := storetest.New(t, db)
	next := b.Tournament(func(row *store.Tournament) { row.Status = domain.TournamentEntryOpen })
	teams, err := entries.PastTeams(ctx, people[0].ID, next.ID, entry.MaxPastTeams)
	require.NoError(t, err)
	require.Len(t, teams, 1)
	assert.Equal(t, "千葉大学MISTRAL", teams[0].FullName)
	assert.Len(t, teams[0].Members, 2)
	reps, err := store.NewNotifyStore(db).Recipients(ctx, next.ID, notify.Audience{Kind: notify.AudiencePastReps})
	require.NoError(t, err)
	assert.Len(t, reps, 2)

	// 同じ書き出しをもう一度取り込んでも、台帳の人は増えない(大会は別に作る)
	past.Entries[0].InviteCode, past.Entries[1].InviteCode = strings.Repeat("c", 26), strings.Repeat("d", 26)
	_, err = entries.ImportPast(ctx, past)
	require.NoError(t, err)
	var count int64
	require.NoError(t, db.Model(&store.LineUser{}).Count(&count).Error)
	assert.Equal(t, int64(3), count)
}
