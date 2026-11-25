package levent_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/japanese"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
	"scoreview/api/internal/levent"
)

// 書き出しの形は Lイベントの「代表者のみ」と「選手全員」に合わせる。名前と ID は架空。

func uid(n byte) string {
	return "U" + strings.Repeat(string("0123456789abcdef"[n%16]), 32)
}

const repsHeader = "名前,LINE名,性別,生年月日,申し込み日時,ステータス,申し込み人数,UID,チーム名（大学名含む）,部門," +
	"チーム代表者連絡先（メールアドレス）\n"

func repsCSV() string {
	return repsHeader +
		"代表 一郎,いち,男性,2004-01-01,2026-04-06 04:49:40 UTC,0,1," + uid(1) + ",MISTRAL (千葉大学),オープン,a@example.com\n" +
		"代表 二子,ふた,女性,2004-01-01,2026-04-07 04:49:40 UTC,0,1," + uid(2) + ",千葉大学MISTRAL,ウィメン,\n" +
		"代表 三郎,さぶ,男性,2004-01-01,2026-04-08 04:49:40 UTC,0,1," + uid(3) + ",文教大学 AIRS,オープン部門,\n" +
		"代表 四郎,しろ,男性,2004-01-01,2026-04-09 04:49:40 UTC,0,1," + uid(4) + ",文教大学　AIRS ,オープン,\n"
}

const membersHeader = "名前,LINE名,性別,生年月日,申し込み日時,ステータス,申し込み人数,UID,チーム名,チーム代表者," +
	"メールアドレス（代表者のみ）,背番号（空欄可）\n"

func membersCSV() string {
	return "\xef\xbb\xbf" + membersHeader +
		"選手 甲,こう,男性,2005-01-01,2026-04-10 01:00:00 UTC,0,1," + uid(5) + ",千葉大学MISTRAL（オープン）,選手,,７\n" +
		"選手 乙,おつ,女性,2005-01-01,2026-04-10 02:00:00 UTC,0,1," + uid(6) + ",千葉大学MISTRAL（ウィメン）,選手,,\n" +
		"代表 一郎,いち,男性,2004-01-01,2026-04-10 03:00:00 UTC,0,1," + uid(1) + ",千葉大学MISTRAL（オープン）,代表者,a@example.com,10\n" +
		"選手 丙,へい,男性,2005-01-01,2026-04-10 04:00:00 UTC,0,1," + uid(7) + ",文教大学AlRS（オープン）,選手,,ないです、背番号は未定\n" +
		"選手 丁,てい,男性,2005-01-01,2026-04-10 05:00:00 UTC,0,1," + uid(8) + ",知らないチーム（オープン）,選手,,\n" +
		"選手 甲改,こう,男性,2005-01-01,2026-04-11 01:00:00 UTC,0,1," + uid(5) + ",千葉大学MISTRAL（オープン）,選手,,8\n"
}

func shiftJIS(t *testing.T, text string) io.Reader {
	t.Helper()
	encoded, err := japanese.ShiftJIS.NewEncoder().String(text)
	require.NoError(t, err)
	return strings.NewReader(encoded)
}

func read(t *testing.T, aliases map[string]string) (entry.PastTournament, levent.Report) {
	t.Helper()
	day := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	past, report, err := levent.Read(shiftJIS(t, repsCSV()), []io.Reader{strings.NewReader(membersCSV())},
		levent.Options{Name: "若葉杯", StartDate: &day, Aliases: aliases, Random: bytes.NewReader(make([]byte, 1024))})
	require.NoError(t, err)
	return past, report
}

func TestReadMatchesMembersToTeams(t *testing.T) {
	past, report := read(t, map[string]string{"文教大学AlRS": "文教大学 AIRS"})
	assert.Equal(t, []entry.PastDivision{{Name: "オープン", Kind: domain.DivisionNone},
		{Name: "ウィメン", Kind: domain.DivisionWomen}}, past.Divisions)
	require.Len(t, past.Entries, 3, "the second AIRS entry in the same division is merged")
	assert.Equal(t, []string{"文教大学　AIRS"}, report.Merged)

	mistral := past.Entries[0]
	assert.Equal(t, "MISTRAL (千葉大学)", mistral.FullName, "the name is kept as the representative wrote it")
	assert.Equal(t, 0, mistral.Division)
	assert.Equal(t, "代表 一郎", mistral.RepName)
	assert.Equal(t, "a@example.com", mistral.RepEmail)
	assert.Equal(t, uid(1), mistral.RepLineID)
	assert.Equal(t, time.Date(2026, 4, 6, 4, 49, 40, 0, time.UTC), mistral.SubmittedAt)
	assert.True(t, entry.IsInviteCode(mistral.InviteCode))
	// 「千葉大学MISTRAL(オープン)」は語の順と括弧が違っても同じチーム。後の登録で名前と背番号を直す
	assert.Equal(t, []entry.PastMember{
		{Name: "選手 甲改", Number: "8", LineID: uid(5), SubmittedAt: time.Date(2026, 4, 11, 1, 0, 0, 0, time.UTC)},
		{Name: "代表 一郎", Number: "10", LineID: uid(1), SubmittedAt: time.Date(2026, 4, 10, 3, 0, 0, 0, time.UTC)},
	}, mistral.Members)
	assert.Equal(t, 1, report.Duplicates)

	assert.Equal(t, 1, past.Entries[1].Division)
	require.Len(t, past.Entries[1].Members, 1, "the women's team gets the (ウィメン) member")
	assert.Equal(t, uid(6), past.Entries[1].Members[0].LineID)

	airs := past.Entries[2]
	require.Len(t, airs.Members, 1, "the alias finds the team")
	assert.Empty(t, airs.Members[0].Number, "a long note is not a number")
	assert.Equal(t, 1, report.DroppedNumbers)
	assert.Equal(t, []levent.Unmatched{{Team: "知らないチーム（オープン）", Count: 1}}, report.Unmatched)
	require.NoError(t, entry.CheckPast(past))
}

func TestReadReportsMembersWithoutATeam(t *testing.T) {
	_, report := read(t, nil)
	assert.Len(t, report.Unmatched, 2, "without the alias the AIRS member has no team")
}

func TestReadNeedsTheColumns(t *testing.T) {
	_, _, err := levent.Read(strings.NewReader("名前,UID\n"), nil, levent.Options{Name: "x"})
	require.ErrorContains(t, err, "申し込み日時")
	bad := repsHeader + "代表,x,男性,2004-01-01,昨日,0,1," + uid(1) + ",チーム,オープン,\n"
	_, _, err = levent.Read(strings.NewReader(bad), nil, levent.Options{Name: "x"})
	require.ErrorContains(t, err, "line 2")
}
