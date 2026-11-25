package entry_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/entry"
)

func pastTournament() entry.PastTournament {
	at := time.Date(2026, 4, 6, 4, 49, 40, 0, time.UTC)
	return entry.PastTournament{Name: "若葉杯", Divisions: []entry.PastDivision{{Name: "オープン", Kind: domain.DivisionNone}},
		Entries: []entry.PastEntry{{FullName: "千葉大学MISTRAL", RepName: "代表 一郎", RepLineID: "U" + strings.Repeat("a", 32),
			SubmittedAt: at, InviteCode: strings.Repeat("a", 26),
			Members: []entry.PastMember{{Name: "選手 甲", Number: "7", LineID: "U" + strings.Repeat("b", 32)}}}}}
}

func TestIsLineUserID(t *testing.T) {
	assert.True(t, entry.IsLineUserID("U"+strings.Repeat("0f", 16)))
	assert.False(t, entry.IsLineUserID("U"+strings.Repeat("0F", 16)), "LINE writes lower-case hex")
	assert.False(t, entry.IsLineUserID("u"+strings.Repeat("0", 32)))
	assert.False(t, entry.IsLineUserID("U"+strings.Repeat("0", 31)))
}

func TestCheckPast(t *testing.T) {
	require.NoError(t, entry.CheckPast(pastTournament()))
	broken := []func(*entry.PastTournament){
		func(p *entry.PastTournament) { p.Name = "" },
		func(p *entry.PastTournament) { p.Divisions = nil },
		func(p *entry.PastTournament) { p.Entries[0].Division = 1 },
		func(p *entry.PastTournament) { p.Entries[0].RepLineID = "U1" },
		func(p *entry.PastTournament) { p.Entries[0].InviteCode = "x" },
		func(p *entry.PastTournament) { p.Entries[0].FullName = "改行\nあり" },
		func(p *entry.PastTournament) { p.Entries[0].Members[0].Number = strings.Repeat("1", 11) },
		func(p *entry.PastTournament) {
			p.Entries[0].Members = append(p.Entries[0].Members, p.Entries[0].Members[0])
		},
	}
	for i, breakIt := range broken {
		past := pastTournament()
		breakIt(&past)
		var problem *domain.ValidationError
		assert.ErrorAs(t, entry.CheckPast(past), &problem, "case %d", i)
	}
}
