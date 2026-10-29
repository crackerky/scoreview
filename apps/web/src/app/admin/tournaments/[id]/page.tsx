import { runsDayOf } from '@/admin/access.utils'
import { buildAlerts, courtCards } from '@/admin/hqBoard.utils'
import { loadBoards } from '@/admin/hqData'
import { loadAdmin, requireUser } from '@/admin/session'
import { japaneseDate, jstToday } from '@/admin/tournamentList.utils'
import { assignments, groups, matches, teams, tournaments } from '@/api/admin'
import type { AdminDivision, AdminMatch } from '@/api/admin.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, EntryCard, type PrimaryAction, Stat } from '@/components/admin/AdminParts'

// 大会ホーム(Claude Design「Score View Admin」)。上の画面は要約と入口だけにし、フォームを置かない。
// 数字の要約(試合中、終わった試合、本部の知らせ)と、本部、部門、時間割、記録係、写真とバナー、
// 大会の設定への入口。記録係(staff)には部門と時間割だけを出す。

type DivisionSummary = {
  division: AdminDivision
  teams: number | null
  groups: number | null
  matches: AdminMatch[]
}

export default async function AdminTournamentPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('any')
  // 大会の作成、設定は管理者だけ。本部と記録係の割り当ては本部スタッフもできる
  const isAdmin = user.role === 'admin'
  const canRun = runsDayOf(user)
  const today = jstToday(new Date())
  const [tournament, board, assignmentList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      loadBoards(token, id),
      canRun ? assignments.list(token, id) : null,
    ]),
  )
  const summaries: DivisionSummary[] = await loadAdmin(() =>
    Promise.all(
      board.divisions.map(async (division) => {
        // チームと組の一覧は記録係には読めないので、数は本部の役割にだけ出す
        const [teamList, groupList, matchList] = await Promise.all([
          canRun ? teams.list(token, division.id) : null,
          canRun ? groups.list(token, division.id) : null,
          matches.list(token, division.id),
        ])
        return {
          division,
          teams: teamList?.teams.length ?? null,
          groups: groupList?.groups.length ?? null,
          matches: matchList.matches,
        }
      }),
    ),
  )
  const allMatches = summaries.flatMap((summary) => summary.matches)
  const live = allMatches.filter((match) => match.status === 'live').length
  const finished = allMatches.filter((match) => match.status === 'finished').length
  const keepers = assignmentList?.assignments ?? []
  const alerts = buildAlerts(board.boards, id, keepers, today)
  const freeCourts = courtCards(board.boards, keepers, today)
    .filter((card) => card.keeper === null)
    .map((card) => card.name)
  const kicker = [
    tournament.startDate ? japaneseDate(tournament.startDate) : '開催日未定',
    tournament.listed ? '公開中' : '非公開(準備中)',
  ].join(' ・ ')
  const hasBoard = board.boards.length > 0
  const primary: PrimaryAction | undefined = hasBoard
    ? { label: '本部を開く', href: `/admin/tournaments/${id}/hq` }
    : isAdmin && board.divisions.length === 0
      ? { label: '部門を作る', href: `/admin/tournaments/${id}/divisions/new` }
      : undefined
  return (
    <AdminShell
      user={user}
      title={tournament.name.ja}
      kicker={kicker}
      notice={notice}
      primary={primary}
    >
      <div className="adm-stats">
        <Stat label="いま試合中" value={String(live)} tone={live > 0 ? 'live' : undefined} />
        <Stat label="終わった試合" value={`${finished} / ${allMatches.length}`} />
        <Stat
          label="本部の知らせ"
          value={String(alerts.length)}
          tone={alerts.length > 0 ? 'warn' : undefined}
        />
      </div>
      <div className="adm-entries">
        {hasBoard ? (
          <EntryCard
            href={`/admin/tournaments/${id}/hq`}
            label="本部"
            sub="当日の進行。結果の入力、遅れの調整、組の確定"
            badge={alerts.length > 0 ? `知らせ ${alerts.length}` : undefined}
            highlight
          />
        ) : null}
        {summaries.map((summary) => (
          <EntryCard
            key={summary.division.id}
            href={`/admin/divisions/${summary.division.id}`}
            label={summary.division.name.ja}
            sub={divisionSub(summary, board.unformatted.includes(summary.division))}
            badge={
              isAdmin && board.unformatted.includes(summary.division) ? '形を決める' : undefined
            }
          />
        ))}
        {isAdmin ? (
          <EntryCard
            href={`/admin/tournaments/${id}/divisions/new`}
            label="＋ 部門を作る"
            sub="オープン、女子など、順位を別に出す部門を足します"
          />
        ) : null}
        <EntryCard
          href={`/admin/tournaments/${id}/timetable`}
          label="時間割と印刷"
          sub="コートごとの時間割と、掲示用の印刷"
        />
        {canRun ? (
          <EntryCard
            href={`/admin/tournaments/${id}/staff`}
            label="記録係"
            sub={`割り当て ${keepers.length} 件。当日の交代もここから`}
            badge={hasBoard && freeCourts.length > 0 ? `${freeCourts.join('・')}が空き` : undefined}
            tone="live"
          />
        ) : null}
        {isAdmin ? (
          <>
            <EntryCard
              href={`/admin/tournaments/${id}/media`}
              label="写真とバナー"
              sub="スポンサーのバナーと、大会のギャラリー"
            />
            <EntryCard
              href={`/admin/tournaments/${id}/settings`}
              label="大会の設定"
              sub="公開、名前と日付、お知らせ、削除"
            />
          </>
        ) : null}
      </div>
      <p className="adm-muted">
        <a href={`/scview/${id}`}>閲覧ページを開く</a> <a href={`/live/${id}`}>観戦アプリを開く</a>
      </p>
    </AdminShell>
  )
}

function divisionSub(summary: DivisionSummary, unformatted: boolean): string {
  const counts = [
    summary.teams === null ? null : `${summary.teams}チーム`,
    summary.groups === null ? null : `${summary.groups}組`,
    `${summary.matches.length}試合`,
  ]
    .filter(Boolean)
    .join(' ・ ')
  return unformatted ? `${counts}。大会の形がまだ決まっていません` : counts
}
