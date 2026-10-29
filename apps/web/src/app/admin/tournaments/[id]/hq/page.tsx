import { updatedAtText } from '@/admin/hq.utils'
import {
  buildAlerts,
  type CourtCard,
  courtCards,
  type DivisionBoard,
  decided,
  type GroupCard,
  groupCards,
  type HqAlert,
  matchesByTime,
  resultHref,
  sideName,
} from '@/admin/hqBoard.utils'
import { loadBoards } from '@/admin/hqData'
import { courtLabel } from '@/admin/scoreKeeper.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { japaneseDate, jstToday } from '@/admin/tournamentList.utils'
import { assignments, tournaments } from '@/api/admin'
import type { ProgressMatch } from '@/api/format.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, type ChipTone, type ShellTab } from '@/components/admin/AdminParts'
import { AutoRefresh } from '@/components/admin/AutoRefresh'
import { StandingsCard } from '@/components/admin/Standings'

// 大会単位の本部(Claude Design「Score View Admin」)。全部門を 1 画面にまとめ、タブで「いま」(知らせ、
// コート、組)、「試合」(時刻ごとの全試合)、「組」(順位表)を切り替える。知らせからその作業の画面へ
// 直接進む。同順位の決定、段の確定、棄権、大会の締め、中断は部門ごとの進行画面で行う。

const tabs = [
  { id: 'now', label: 'いま' },
  { id: 'matches', label: '試合' },
  { id: 'pools', label: '組' },
] as const
type Tab = (typeof tabs)[number]['id']

const clock = new Intl.DateTimeFormat('ja-JP', {
  timeZone: 'Asia/Tokyo',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})

export default async function TournamentHqPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const query = await props.searchParams
  const tab: Tab = tabs.find((item) => item.id === query.tab)?.id ?? 'now'
  const { user, token } = await requireUser('any')
  const isAdmin = user.role === 'admin'
  const now = new Date()
  const today = jstToday(now)
  const [tournament, board, assignmentList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      loadBoards(token, id),
      isAdmin ? assignments.list(token, id) : null,
    ]),
  )
  const keepers = assignmentList?.assignments ?? []
  const alerts = buildAlerts(board.boards, id, keepers, today)
  const base = `/admin/tournaments/${id}/hq`
  const shellTabs: ShellTab[] = tabs.map((item) => ({
    href: item.id === 'now' ? base : `${base}?tab=${item.id}`,
    label: item.label,
    current: item.id === tab,
    count: item.id === 'now' ? alerts.length : undefined,
  }))
  const crumbs = [{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]
  const several = board.boards.length > 1
  return (
    <AdminShell
      user={user}
      title="本部"
      kicker={`${clock.format(now)} 現在 ・ ${several ? '全部門' : (board.boards[0]?.divisionName ?? '')}`}
      notice={notice}
      crumbs={crumbs}
      tabs={board.boards.length > 0 ? shellTabs : undefined}
      primary={
        tab === 'now' && board.boards.length > 0
          ? { label: '結果を入れる', href: `${base}?tab=matches` }
          : undefined
      }
    >
      {board.boards.length === 0 ? (
        <NoBoard tournamentId={id} />
      ) : (
        <>
          <p className="adm-muted">
            <AutoRefresh updatedAt={updatedAtText(now)} />
          </p>
          {tab === 'now' ? (
            <NowTab
              tournamentId={id}
              alerts={alerts}
              courts={courtCards(board.boards, keepers, today)}
              groups={groupCards(board.boards, id)}
              showKeepers={isAdmin}
            />
          ) : null}
          {tab === 'matches' ? <MatchesTab boards={board.boards} tournamentId={id} /> : null}
          {tab === 'pools' ? <PoolsTab boards={board.boards} tournamentId={id} /> : null}
          <DivisionLinks boards={board.boards} />
        </>
      )}
    </AdminShell>
  )
}

function NoBoard({ tournamentId }: { tournamentId: number }) {
  return (
    <div className="adm-card">
      <strong>まだ本部で見るものがありません</strong>
      <span className="adm-muted">
        本部は、部門の大会の形(組と順位決定戦と時間割)を決めたあとに使います。
      </span>
      <a className="adm-go" href={`/admin/tournaments/${tournamentId}`}>
        大会ホームで部門の形を決める
      </a>
    </div>
  )
}

function NowTab({
  tournamentId,
  alerts,
  courts,
  groups,
  showKeepers,
}: {
  tournamentId: number
  alerts: HqAlert[]
  courts: CourtCard[]
  groups: GroupCard[]
  showKeepers: boolean
}) {
  return (
    <div className="adm-hq-cols">
      <div className="adm-hq-main">
        <section>
          <h2 className="adm-block-title">知らせ</h2>
          {alerts.length === 0 ? (
            <div className="adm-card adm-muted">いま対応が必要なことはありません。</div>
          ) : (
            <div className="adm-stack">
              {alerts.map((alert) => (
                <div key={alert.key} className="adm-card">
                  <div className="adm-alert-head">
                    <Chip tone={alert.tone}>{alert.chip}</Chip>
                    <strong>{alert.title}</strong>
                  </div>
                  <span className="adm-alert-body">{alert.body}</span>
                  <div className="adm-links">
                    {alert.links.map((link) => (
                      <a key={link.href} className="adm-go" href={link.href}>
                        {link.label}
                      </a>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
        <section>
          <h2 className="adm-block-title">コート</h2>
          <div className="adm-courts">
            {courts.map((court) => (
              <CourtBox
                key={court.court}
                court={court}
                showKeeper={showKeepers}
                tournamentId={tournamentId}
              />
            ))}
          </div>
        </section>
      </div>
      <section className="adm-hq-side">
        <h2 className="adm-block-title">組</h2>
        <div className="adm-stack">
          {groups.map((group) => (
            <div key={`${group.divisionId}-${group.id}`} className="adm-card">
              <div className="adm-alert-head">
                <strong>{group.name}</strong>
                <Chip tone={group.tone}>{group.chip}</Chip>
              </div>
              <span className="adm-muted">{group.sub}</span>
              {group.link ? (
                <a className="adm-go" href={group.link.href}>
                  {group.link.label}
                </a>
              ) : null}
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}

function teamsText(match: ProgressMatch): string {
  return `${sideName(match.left)} 対 ${sideName(match.right)}`
}

function scoreText(match: ProgressMatch): string {
  return match.left.score === null || match.right.score === null
    ? ''
    : `${match.left.score}-${match.right.score}`
}

function CourtBox({
  court,
  showKeeper,
  tournamentId,
}: {
  court: CourtCard
  showKeeper: boolean
  tournamentId: number
}) {
  const live = court.live
  return (
    <div className="adm-court">
      <div className="adm-court-head">
        <strong>{court.name}</strong>
        {court.delay !== null ? <Chip tone="warn">{court.delay}分遅れ</Chip> : null}
        {showKeeper ? (
          <span className={`adm-court-keeper${court.keeper ? '' : ' none'}`}>
            {court.keeper ? `記録係 ${court.keeper}` : '記録係なし'}
          </span>
        ) : null}
      </div>
      {live ? (
        <a className="adm-court-now" href={resultHref(live.id, tournamentId)}>
          <span className="adm-court-state live">
            <span className="adm-live-dot" aria-hidden="true" />
            試合中 <span className="adm-muted">{live.groupLabel}</span>
          </span>
          <span className="adm-court-match">
            <span>{teamsText(live)}</span>
            <span className="adm-num adm-court-score">{scoreText(live)}</span>
          </span>
        </a>
      ) : (
        <div className="adm-court-now">
          <span className="adm-court-state">空き</span>
          <span className="adm-muted">次の試合まで空いています</span>
        </div>
      )}
      <div className="adm-court-next">
        次
        {court.next
          ? `${court.next.time ?? ''} ${court.next.groupLabel}　${teamsText(court.next)}`
          : 'なし'}
      </div>
    </div>
  )
}

const statusChip: Record<ProgressMatch['status'], { label: string; tone: ChipTone }> = {
  live: { label: '試合中', tone: 'live' },
  finished: { label: '終了', tone: 'done' },
  scheduled: { label: '予定', tone: 'sched' },
}

function MatchesTab({ boards, tournamentId }: { boards: DivisionBoard[]; tournamentId: number }) {
  const groups = matchesByTime(boards)
  const dates = new Set(groups.map((group) => group.date))
  return (
    <div className="adm-stack-lg">
      {groups.map((group) => (
        <section key={group.key}>
          <div className="adm-time-head">
            <span className="adm-num adm-time">{group.time}</span>
            {dates.size > 1 && group.date ? (
              <span className="adm-muted">{japaneseDate(group.date)}</span>
            ) : null}
          </div>
          <ul className="adm-rows">
            {group.rows.map((match) => {
              const chip = statusChip[match.status]
              return (
                <li key={match.id} className="adm-match-row">
                  <span className="adm-match-where">
                    <strong>{courtLabel(match.field) || match.code}</strong>
                    <span className="adm-muted">{match.groupLabel}</span>
                  </span>
                  <span className="adm-match-teams">{teamsText(match)}</span>
                  <span className="adm-match-end">
                    <span className="adm-num adm-match-score">{scoreText(match)}</span>
                    <Chip tone={chip.tone}>
                      {match.status === 'live' ? (
                        <span className="adm-live-dot" aria-hidden="true" />
                      ) : null}
                      {chip.label}
                    </Chip>
                    {decided(match) ? (
                      <a className="adm-go" href={resultHref(match.id, tournamentId)}>
                        {match.status === 'finished' ? '直す' : '結果を入れる'}
                      </a>
                    ) : (
                      <span className="adm-match-wait">対戦待ち</span>
                    )}
                  </span>
                </li>
              )
            })}
          </ul>
        </section>
      ))}
    </div>
  )
}

function PoolsTab({ boards, tournamentId }: { boards: DivisionBoard[]; tournamentId: number }) {
  const several = boards.length > 1
  return (
    <div className="adm-pools">
      {boards.flatMap((board) =>
        board.progress.groups
          .filter((group) => group.kind === 'pool')
          .map((group) => (
            <StandingsCard
              key={group.id}
              group={group}
              title={several ? `${board.divisionName} ${group.label}` : group.label}
              confirmHref={`/admin/tournaments/${tournamentId}/hq/confirm/${group.id}`}
            />
          )),
      )}
    </div>
  )
}

/** 部門ごとの進行画面(同順位の決定、段の確定、棄権、締め、中断はそこで行う) */
function DivisionLinks({ boards }: { boards: DivisionBoard[] }) {
  return (
    <p className="adm-muted adm-division-links">
      詳しい進行(同順位の決定、棄権、大会の締め、雷・雨・暑さでの中断):
      {boards.map((board) => (
        <a key={board.divisionId} href={`/admin/divisions/${board.divisionId}/progress`}>
          {board.divisionName}の進行画面
        </a>
      ))}
    </p>
  )
}
