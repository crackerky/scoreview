package entry

import (
	"fmt"

	"scoreview/api/internal/domain"
)

// 差し戻した申込を代表が直して送り直す(docs/line-integration.md「申込とチームを分ける」)。同じ申込を
// 直すので、招待コードと、メンバーがもう済ませた本人登録はそのまま残る。

// CanResubmit は申込を送り直してよいかを確かめる。送り直せるのは差し戻した申込だけ。運営が直して
// ほしいと返したものなので、受付を締めた後でも、大会が終わるまでは受ける。
func CanResubmit(status Status, tournament domain.TournamentStatus) error {
	if status != StatusReturned {
		return fmt.Errorf("%w: only a returned entry can be resubmitted; it is %s", domain.ErrConflict, status)
	}
	if tournament == domain.TournamentFinished {
		return fmt.Errorf("%w: the tournament has finished", domain.ErrConflict)
	}
	return nil
}

// PlannedMember は送り直しで残すか足すメンバーの行。ID が 0 なら新しく足す行。
type PlannedMember struct {
	ID        int64
	Name      string
	Number    string
	SortOrder int
}

// MemberPlan は送り直しでのメンバーの行の変え方。Rows は送り直した後の全員(並び順)、Remove は消す行。
type MemberPlan struct {
	Rows   []PlannedMember
	Remove []int64
}

// PlanMembers は、代表が直したメンバーの並び drafts(CheckDraft を通したもの)を、今の行 current に
// 合わせる。
//
//   - 本人が登録した行は、名前と背番号を本人が決めたまま先頭に残す(代表の書き直しで、本人の登録を
//     消したり書き換えたりしない)。drafts に同じ名前があれば、その人として読み飛ばす。
//   - 残りの drafts は書いた順に並べ、本人が登録していない今の行に同じ名前があればその行を使い回す
//     (メンバーが開いている登録の画面の名簿を、なるべく変えないため)。無ければ新しい行にする。
//   - drafts に無くなった、本人が登録していない行は消す。
//
// drafts の背番号が、本人が登録した人の背番号と重なれば検証エラー(項目は members[drafts の位置].number)。
func PlanMembers(current []Member, drafts []MemberDraft) (MemberPlan, error) {
	plan := MemberPlan{Rows: []PlannedMember{}, Remove: []int64{}}
	registered := map[string]bool{}
	numbers := map[string]bool{}
	unclaimed := map[string][]int64{}
	for _, member := range current {
		if !member.Registered {
			unclaimed[member.Name] = append(unclaimed[member.Name], member.ID)
			continue
		}
		registered[member.Name] = true
		if member.Number != "" {
			numbers[member.Number] = true
		}
		plan.Rows = append(plan.Rows, PlannedMember{ID: member.ID, Name: member.Name, Number: member.Number,
			SortOrder: len(plan.Rows) + 1})
	}
	for i, draft := range drafts {
		if registered[draft.Name] {
			continue
		}
		if draft.Number != "" && numbers[draft.Number] {
			return MemberPlan{}, domain.Invalid(fmt.Sprintf("members[%d].number", i), "is used by another member")
		}
		row := PlannedMember{Name: draft.Name, Number: draft.Number, SortOrder: len(plan.Rows) + 1}
		if ids := unclaimed[draft.Name]; len(ids) > 0 {
			row.ID, unclaimed[draft.Name] = ids[0], ids[1:]
		}
		plan.Rows = append(plan.Rows, row)
	}
	if len(plan.Rows) > MaxMembers {
		return MemberPlan{}, domain.Invalid("members", fmt.Sprintf("must be at most %d", MaxMembers))
	}
	for _, member := range current {
		if !member.Registered && !planned(plan.Rows, member.ID) {
			plan.Remove = append(plan.Remove, member.ID)
		}
	}
	return plan, nil
}

// SameRegistered は、本人が登録した行が registered(代表がフォームを開いたときの登録。並びは問わない)と
// 同じかどうか。
func SameRegistered(current []Member, registered []int64) bool {
	seen := make(map[int64]bool, len(registered))
	for _, id := range registered {
		seen[id] = true
	}
	count := 0
	for _, member := range current {
		if member.Registered {
			if !seen[member.ID] {
				return false
			}
			count++
		}
	}
	return count == len(seen)
}

func planned(rows []PlannedMember, id int64) bool {
	for _, row := range rows {
		if row.ID == id {
			return true
		}
	}
	return false
}
