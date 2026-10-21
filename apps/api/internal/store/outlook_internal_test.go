package store

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/format"
)

// 勝ち上がり条件の文: これより上になれない順位は「以上」と書かない(レビュー L6)。
func TestOutlookTextExactRank(t *testing.T) {
	look := format.Outlook{Best: 2, Worst: 3, Conditions: []format.Condition{{Match: "A3", AtLeast: 2}}}
	assert.Equal(t, "次の A3 に勝てば 2 位が確定", outlookText(look))
	look = format.Outlook{Best: 1, Worst: 3, Conditions: []format.Condition{{Match: "A3", AtLeast: 2}}}
	assert.Equal(t, "次の A3 に勝てば 2 位以上が確定", outlookText(look))
}
