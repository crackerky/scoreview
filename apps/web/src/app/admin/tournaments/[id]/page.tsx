import { runsDayOf } from '@/admin/access.utils'
import { buildAlerts, courtCards } from '@/admin/hqBoard.utils'
import { loadBoards } from '@/admin/hqData'
import { loadAdmin, requireUser } from '@/admin/session'
import { currentStage, tournamentStages } from '@/admin/tournamentHome.utils'
import { japaneseDate, jstToday } from '@/admin/tournamentList.utils'
import { assignments, matches, teams, tournaments } from '@/api/admin'
import { entries } from '@/api/entry'
import { notifications } from '@/api/notify'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, type PrimaryAction } from '@/components/admin/AdminParts'
import { StageCard } from '@/components/admin/StageCard'
import { countByStatus } from '@/entry/entry.utils'

// 大会ホーム。大会の段取りの順(準備、公式 LINE、試合を組む、当日、その他)に段のカードを並べ、各段に
// 今の様子と次にやることを 1 つだけ出す(段の中身は tournamentHome.utils)。待っていることがある最初の
// 段の次にやることを、画面の主な操作にする。記録係と本部スタッフには当日の段と部門だけを出す。

export default async function AdminTournamentPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('any')
  // 大会の作成、設定は管理者だけ。本部と記録係の割り当ては本部スタッフもできる
  const isAdmin = user.role === 'admin'
  const canRun = runsDayOf(user)
  const [tournament, board, assignmentList, entryList, notificationList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      loadBoards(token, id),
      canRun ? assignments.list(token, id) : null,
      // 申込には代表の連絡先が入り、送る記録も管理者だけ
      isAdmin ? entries.list(token, id) : null,
      isAdmin ? notifications.list(token, id) : null,
    ]),
  )
  const divisions = await loadAdmin(() =>
    Promise.all(
      board.divisions.map(async (division) => {
        // チームの一覧は記録係には読めないので、数は本部の役割にだけ出す
        const [teamList, matchList] = await Promise.all([
          canRun ? teams.list(token, division.id) : null,
          matches.list(token, division.id),
        ])
        return { division, teams: teamList?.teams.length ?? null, matches: matchList.matches }
      }),
    ),
  )
  const allMatches = divisions.flatMap((item) => item.matches)
  const keepers = assignmentList?.assignments ?? []
  const entryCounts = entryList ? countByStatus(entryList.entries) : null
  const stages = tournamentStages({
    tournamentId: id,
    tournament,
    today: jstToday(new Date()),
    role: user.role,
    divisions: divisions.map((item) => ({
      id: item.division.id,
      name: item.division.name.ja,
      formatted: !board.unformatted.includes(item.division),
      teams: item.teams,
      matches: item.matches.length,
    })),
    entries:
      entryList && entryCounts
        ? {
            total: entryList.entries.length,
            pending: entryCounts.pending,
            approved: entryCounts.approved,
          }
        : null,
    scheduled:
      notificationList?.notifications.filter((item) => item.state === 'scheduled').length ?? null,
    keepers: keepers.length,
    freeCourts: courtCards(board.boards, keepers)
      .filter((card) => card.needsKeeper)
      .map((card) => card.name),
    hasBoard: board.boards.length > 0,
    day: {
      live: allMatches.filter((match) => match.status === 'live').length,
      finished: allMatches.filter((match) => match.status === 'finished').length,
      total: allMatches.length,
      alerts: buildAlerts(board.boards, id, keepers).length,
    },
  })
  const current = currentStage(stages)
  const primary: PrimaryAction | undefined = current?.next ?? undefined
  const kicker = [
    tournament.startDate ? japaneseDate(tournament.startDate) : '開催日未定',
    tournament.listed ? '公開中' : '非公開(準備中)',
  ].join(' ・ ')
  return (
    <AdminShell
      user={user}
      title={tournament.name.ja}
      kicker={kicker}
      notice={notice}
      primary={primary}
    >
      <div className="adm-stages">
        {stages.map((stage) => (
          <StageCard key={stage.key} stage={stage} current={stage === current} />
        ))}
      </div>
    </AdminShell>
  )
}
