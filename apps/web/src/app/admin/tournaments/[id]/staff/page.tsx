import { loadAdmin, requireUser } from '@/admin/session'
import { assignments, divisions, matches, tournaments, users } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell } from '@/components/admin/AdminParts'
import { AssignmentSection } from '@/components/admin/AssignmentSection'

// 記録係の割り当て(大会ホームの入口から)。コートと時間帯、または試合ごとに割り当てる
export default async function TournamentStaffPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [tournament, divisionList, assignmentList, userList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      divisions.list(token, id),
      assignments.list(token, id),
      users.list(token),
    ]),
  )
  const matchLists = await loadAdmin(() =>
    Promise.all(divisionList.divisions.map((division) => matches.list(token, division.id))),
  )
  const crumbs = [{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]
  return (
    <AdminShell user={user} title="記録係" notice={notice} crumbs={crumbs}>
      <AssignmentSection
        tournamentId={id}
        back={`/admin/tournaments/${id}/staff`}
        assignments={assignmentList.assignments}
        staff={userList.users.filter((item) => item.role === 'staff')}
        matches={matchLists.flatMap((list) => list.matches)}
      />
    </AdminShell>
  )
}
