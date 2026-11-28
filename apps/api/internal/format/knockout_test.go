package format_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

func knockoutPlan(t *testing.T, teams int) *format.Plan {
	t.Helper()
	plan, err := format.NewEngine().Generate(format.Spec{
		Template: format.TemplateKnockout, Params: format.KnockoutParams{Teams: teams},
	})
	require.NoError(t, err, "%d チーム", teams)
	return plan
}

// 2〜16 チームのどれでも、全チームが 1 回ずつ枠に入り、試合の結果はどれも次の試合か順位につながる
// (行き先の無い勝者や敗者がいない)。
func TestKnockoutPlacesEveryTeam(t *testing.T) {
	for teams := 2; teams <= 16; teams++ {
		plan := knockoutPlan(t, teams)
		require.Len(t, plan.Groups, 1)
		group := plan.Groups[0]
		assert.Equal(t, teams, group.Size(), "%d チーム", teams)
		seeds := map[int]bool{}
		for _, slot := range group.Slots {
			seeds[slot.Source.Seed] = true
		}
		assert.Len(t, seeds, teams, "シードは 1〜%d が 1 回ずつ", teams)
		fed := map[string]bool{}
		for _, m := range plan.Matches {
			for _, side := range []format.SideRef{m.Left, m.Right} {
				if side.From != nil {
					fed[fmt.Sprintf("%s/%s", side.From.Code, side.From.Outcome)] = true
				}
			}
		}
		placed := 0
		for _, m := range plan.Matches {
			for outcome, place := range map[domain.Outcome]int{
				domain.OutcomeWin: m.WinnerPlace, domain.OutcomeLose: m.LoserPlace,
			} {
				goesOn := fed[fmt.Sprintf("%s/%s", m.Code, outcome)]
				assert.True(t, goesOn != (place > 0), "%d チームの %s の %s は次の試合か順位のどちらか一方へ",
					teams, m.Code, outcome)
				if place > 0 {
					placed++
				}
			}
		}
		assert.Equal(t, teams, placed, "%d チーム全員に順位が付く", teams)
		ranked, _ := format.FinalPlaces(plan)
		assert.Equal(t, teams, ranked)
	}
}

// 4、8、16 チームは基の型そのまま。3 チームは 3 位決定戦の代わりに 1 回戦の敗者が 3 位、6 チームは
// 1・2 位のシードが 1 回戦を免除される。
func TestKnockoutByes(t *testing.T) {
	for teams, matches := range map[int]int{2: 1, 3: 2, 4: 4, 8: 12, 16: 26} {
		assert.Len(t, knockoutPlan(t, teams).Matches, matches, "%d チーム", teams)
	}
	three := knockoutPlan(t, 3)
	places := map[int]bool{}
	for _, m := range three.Matches {
		places[m.WinnerPlace], places[m.LoserPlace] = true, true
	}
	assert.True(t, places[1] && places[2] && places[3], "3 チームは 1〜3 位が付く")

	six := knockoutPlan(t, 6)
	seedOf := map[int]int{}
	for _, slot := range six.Groups[0].Slots {
		seedOf[slot.Position] = slot.Source.Seed
	}
	firstRound := map[int]bool{}
	for _, m := range six.Matches {
		if m.Round == 1 {
			firstRound[seedOf[m.Left.Slot]], firstRound[seedOf[m.Right.Slot]] = true, true
		}
	}
	assert.False(t, firstRound[1], "1 位のシードは 1 回戦を免除")
	assert.False(t, firstRound[2], "2 位のシードは 1 回戦を免除")
	assert.True(t, firstRound[3] && firstRound[6])
}

func TestKnockoutRejectsOutOfRangeTeams(t *testing.T) {
	for _, teams := range []int{1, 17} {
		_, err := format.NewEngine().Generate(format.Spec{
			Template: format.TemplateKnockout, Params: format.KnockoutParams{Teams: teams},
		})
		require.ErrorIs(t, err, domain.ErrBadRequest, "%d チーム", teams)
	}
}

// 「トーナメントだけ」は 3〜16 チームで候補に出て、時間割も作れる。
func TestProposeKnockout(t *testing.T) {
	start := time.Date(2026, 12, 5, 9, 0, 0, 0, time.UTC)
	got, err := format.Propose(format.ProposalInput{Teams: 6, Courts: 2, SlotMinutes: 60, Start: start,
		Sport: format.SportSeven})
	require.NoError(t, err)
	var knockout *format.Proposal
	for i := range got {
		if got[i].ID == "ko" {
			knockout = &got[i]
		}
	}
	require.NotNil(t, knockout, "6 チームでトーナメントだけの候補が出る")
	assert.Equal(t, "6 チームのトーナメント", knockout.Title)
	assert.Equal(t, 6, knockout.Ranked)
	assert.Equal(t, format.TemplateKnockout, knockout.Spec.Template)

	big, err := format.Propose(format.ProposalInput{Teams: 20, Courts: 4, SlotMinutes: 60, Start: start,
		Sport: format.SportSeven})
	require.NoError(t, err)
	for _, p := range big {
		assert.NotEqual(t, "ko", p.ID, "17 チーム以上はトーナメントだけの候補を出さない")
	}
}

func TestKnockoutParamsJSON(t *testing.T) {
	data, err := format.EncodeParams(format.KnockoutParams{Teams: 6})
	require.NoError(t, err)
	assert.JSONEq(t, `{"teams":6}`, string(data))
	got, err := format.DecodeParams(format.TemplateKnockout, data)
	require.NoError(t, err)
	assert.Equal(t, format.KnockoutParams{Teams: 6}, got)
	assert.Equal(t, 6, format.TeamsOf(got))
	_, err = format.DecodeParams(format.TemplateKnockout, []byte(`{"teams":6,"pools":2}`))
	require.ErrorIs(t, err, domain.ErrBadRequest)
}

// 全試合を上のシード(左)の勝ちで終えると、どのチーム数でも全員に最終順位が付き、1 位は 1 位のシード。
func TestKnockoutResolvesFinalRanks(t *testing.T) {
	for _, teams := range []int{3, 5, 6, 7, 11, 16} {
		plan := knockoutPlan(t, teams)
		state := format.State{Results: map[string]format.Result{}}
		for i := range teams {
			state.Seeds = append(state.Seeds, int64(100+i+1))
		}
		var res *format.Resolution
		for range plan.Matches {
			var err error
			res, err = format.Resolve(plan, format.DefaultConfig(), state)
			require.NoError(t, err)
			for _, m := range plan.Matches {
				teamsOf := res.Matches[m.Code]
				if _, played := state.Results[m.Code]; !played && teamsOf.Left.Resolved() && teamsOf.Right.Resolved() {
					state.Results[m.Code] = score(15, 10)
				}
			}
		}
		require.Len(t, state.Results, len(plan.Matches), "%d チームの全試合に両チームがそろう", teams)
		res, err := format.Resolve(plan, format.DefaultConfig(), state)
		require.NoError(t, err)
		assert.Len(t, res.Final, teams, "%d チーム全員に最終順位", teams)
		for _, row := range res.Final {
			if row.Rank == 1 {
				assert.Equal(t, int64(101), row.TeamID, "%d チームの 1 位は 1 位のシード", teams)
			}
		}
	}
}
