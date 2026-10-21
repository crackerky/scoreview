package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

// open はまだ結果のない試合。
func open(code string, left, right int64) format.Game {
	return format.Game{Code: code, Left: left, Right: right}
}

func TestQualify(t *testing.T) {
	// 1 は 2 勝、4 は 1 勝、2 と 3 は未勝利。残りは 1-4、2-3、3-4
	games := []format.Game{
		game("A1", 1, 2, 3, 1), game("A2", 1, 3, 3, 1), open("A3", 1, 4),
		open("A4", 2, 3), game("A5", 2, 4, 1, 3), open("A6", 3, 4),
	}
	q, err := format.Qualify([]int64{1, 2, 3, 4}, games, format.CurrentRule(true))
	require.NoError(t, err)
	assert.Equal(t, []string{"A3", "A4", "A6"}, q.Remaining)
	assert.Equal(t, 27, q.Scenarios)

	byTeam := map[int64]format.Outlook{}
	for _, o := range q.Teams {
		byTeam[o.TeamID] = o
	}
	one := byTeam[1]
	// 1 は 4 に負け、3 が 4 に勝つと 3 チームが勝ち点 6 で並び、得失点差しだいで 3 位まである
	assert.Equal(t, []format.Chance{format.ChancePossible, format.ChancePossible, format.ChancePossible,
		format.ChanceImpossible}, one.Chances)
	assert.Equal(t, []format.Condition{{Match: "A3", AtLeast: 1, Text: "次のA3に勝てば1位確定"}}, one.Conditions)
	assert.Equal(t, format.ChanceImpossible, byTeam[2].Chances[0], "2 は勝ち点 3 が最大で 1 位になれない")
	assert.Equal(t, 1, byTeam[4].Best)

	// 全試合が終われば確定
	done := []format.Game{
		game("A1", 1, 2, 3, 1), game("A2", 1, 3, 3, 1), game("A3", 1, 4, 3, 1),
		game("A4", 2, 3, 3, 1), game("A5", 2, 4, 1, 3), game("A6", 3, 4, 1, 3),
	}
	q, err = format.Qualify([]int64{1, 2, 3, 4}, done, format.CurrentRule(true))
	require.NoError(t, err)
	assert.Equal(t, 1, q.Scenarios)
	assert.Equal(t, format.ChanceCertain, q.Teams[0].Chances[0])
	assert.Equal(t, "1位確定", q.Teams[0].Conditions[0].Text)
}

func TestQualifyTooManyRemaining(t *testing.T) {
	var games []format.Game
	for a := int64(1); a <= 5; a++ {
		for b := a + 1; b <= 5; b++ {
			games = append(games, open("X", a, b))
		}
	}
	_, err := format.Qualify([]int64{1, 2, 3, 4, 5}, games, format.JFDARule())
	assert.ErrorIs(t, err, domain.ErrBadRequest)
}
