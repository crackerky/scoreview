package parse

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/importer/site"
)

func TestParseMatch(t *testing.T) {
	m := parseBoth(t, "match_45277_jp.html", "match_45277_en.html", ParseMatch, MergeMatchEN)

	assert.Equal(t, "A1", m.Code)
	assert.Equal(t, "Tounament A", m.GroupName)
	assert.Equal(t, "2025-10-18", m.Date)
	assert.Equal(t, ptr("12:20"), m.Time)
	assert.Equal(t, ptr(1), m.FieldNo)
	assert.Equal(t, MatchSide{TeamID: 17498, ShortName: Text{JA: "バトルロイヤル", EN: "BRH"}, Score: ptr(7)}, m.Left)
	assert.Equal(t, MatchSide{
		TeamID: 17399, ShortName: Text{JA: "何かが足りない", EN: "Something's miss"}, Score: ptr(6),
	}, m.Right)
	assert.Equal(t, &Breadcrumb{
		TournamentID: 525, TournamentShortName: "2026 Atami-B",
		DivisionID: ptr(int64(949)), GroupID: ptr(int64(6623)),
	}, m.Breadcrumb)

	// ページは新しい順。古い順に並べ直して seq を振る
	require.Len(t, m.Events, 15)
	assert.Equal(t, MatchEvent{Seq: 1, ElapsedSec: 0, Kind: "pull", Side: ptr("left")}, m.Events[0])
	assert.Equal(t, MatchEvent{
		Seq: 2, ElapsedSec: 60, Kind: "goal", Side: ptr("right"),
		ScorerID: ptr(int64(605948)), AssistID: ptr(int64(605951)), Score: ptr(1),
	}, m.Events[1])
	// 同じ経過時刻(14:00)の 2 得点は、点数の小さいほうが先
	assert.Equal(t, 840, m.Events[10].ElapsedSec)
	assert.Equal(t, 840, m.Events[11].ElapsedSec)
	assert.Equal(t, ptr(5), m.Events[10].Score)
	assert.Equal(t, ptr(6), m.Events[11].Score)
	assert.Equal(t, MatchEvent{Seq: 15, ElapsedSec: 1200, Kind: "end"}, m.Events[14])

	require.Len(t, m.Players.Left, 2)
	assert.Equal(t, int64(606024), m.Players.Left[0].PlayerID)
	assert.Equal(t, "64", m.Players.Left[0].Number)
	assert.Equal(t, ptr(0), m.Players.Left[0].Blocks)
	assert.NotEmpty(t, m.Players.Left[0].Name.EN)
	assert.Equal(t, ptr(1), m.Players.Right[1].Blocks)

	assert.Equal(t, &MatchStats{
		Left:  SideStats{Goals: 7, FromOffence: 5, FromDefence: 2, FromTurnovers: 2},
		Right: SideStats{Goals: 6, FromOffence: 5, FromDefence: 1, FromTurnovers: 1},
	}, m.Stats)
	assert.True(t, m.BlockColumn)
	assert.Equal(t, "", m.Note)
}

// フッター行の備考と、試合終了の行の ★(44803 で観測した形を 45277 に当てはめたもの)。
func TestParseMatchNoteAndEndMark(t *testing.T) {
	raw, err := io.ReadAll(fixture(t, "match_45277_jp.html"))
	require.NoError(t, err)
	page := strings.Replace(string(raw), "<td class='LM TG_MD1' width='68%'></td>",
		"<td class='LM TG_MD1' width='68%'>架空の備考</td>", 1)
	page = strings.Replace(page, `<td class='TG_EVMOJI CM  EVT' ></td>
				<td class='TG_EVMOJI CM  EVT' >試合終了</td>`, `<td class='TG_EVMOJI CM  EVT' >★</td>
				<td class='TG_EVMOJI CM  EVT' >試合終了</td>`, 1)
	m, err := ParseMatch(strings.NewReader(page), site.LangJP)
	require.NoError(t, err)
	assert.Equal(t, "架空の備考", m.Note)
	assert.Equal(t, MatchEvent{Seq: 15, ElapsedSec: 1200, Kind: "end", Side: ptr("left")}, m.Events[14])
}

// 試合終了の行がない試合(終了入力漏れ。clone-spec D 2.3)
func TestParseMatchWithoutEndEvent(t *testing.T) {
	m, err := ParseMatch(fixture(t, "match_45051_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotEmpty(t, m.Events)
	for _, ev := range m.Events {
		assert.NotEqual(t, "end", ev.Kind)
	}
	assert.Equal(t, ptr(12), m.Events[len(m.Events)-1].Score)
	assert.Equal(t, ptr(0), m.Left.Score)
}

// 2017 年: グループ名が空、タイムアウトの ★、性別なしの playerLink
func TestParseMatchOldFormat(t *testing.T) {
	m, err := ParseMatch(fixture(t, "match_14693_jp.html"), site.LangJP)
	require.NoError(t, err)
	assert.Equal(t, "", m.GroupName)
	require.NotNil(t, m.Breadcrumb)
	assert.Equal(t, ptr(int64(2147)), m.Breadcrumb.GroupID, "グループはパンくずから分かる")

	var timeouts []string
	for _, ev := range m.Events {
		if ev.Kind == "timeout" {
			require.NotNil(t, ev.Side)
			timeouts = append(timeouts, *ev.Side)
		}
	}
	assert.Equal(t, []string{"left", "right"}, timeouts)
	assert.Equal(t, "pull", m.Events[0].Kind)
	assert.Equal(t, ptr("right"), m.Events[0].Side)
	require.NotEmpty(t, m.Players.Left)
	assert.Nil(t, m.Players.Left[0].Gender)
	assert.Equal(t, 1, m.Stats.Left.Timeouts)
}

// 「キャラハンゴール」のイベント(clone-spec に未記載。525 の取込で見つかった変種)
func TestParseMatchCallahanGoal(t *testing.T) {
	m, err := ParseMatch(fixture(t, "match_45310_jp.html"), site.LangJP)
	require.NoError(t, err)
	var callahans []MatchEvent
	for _, ev := range m.Events {
		if ev.Callahan {
			callahans = append(callahans, ev)
		}
	}
	require.Len(t, callahans, 1)
	ev := callahans[0]
	assert.Equal(t, "goal", ev.Kind)
	assert.Equal(t, ptr("right"), ev.Side)
	assert.NotNil(t, ev.ScorerID)
	assert.Nil(t, ev.AssistID, "キャラハンにアシストはない")
	assert.Equal(t, ptr(3), ev.Score)
}

// 出場選手表に Block 列がない試合(clone-spec に未記載の変種)
func TestParseMatchWithoutBlockColumn(t *testing.T) {
	m, err := ParseMatch(fixture(t, "match_44347_jp.html"), site.LangJP)
	require.NoError(t, err)
	require.NotEmpty(t, m.Players.Left)
	for _, p := range append(m.Players.Left, m.Players.Right...) {
		assert.Nil(t, p.Blocks)
	}
	assert.False(t, m.BlockColumn)
	assert.Equal(t, ptr(3), m.Left.Score)
	assert.Equal(t, ptr(3), m.Right.Score, "引き分け")
}
