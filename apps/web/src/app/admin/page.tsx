import { statusLabel } from '@/admin/labels.utils'
import { courtLabel } from '@/admin/scoreKeeper.utils'
import { resultTypeLabels } from '@/admin/scoreOps.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { type SideLabels, sideLabelsOf, sideText } from '@/admin/timetable.utils'
import {
  groupTournaments,
  japaneseDate,
  jstToday,
  phaseChip,
  searchTournaments,
  tournamentPhase,
} from '@/admin/tournamentList.utils'
import { matches, tournaments } from '@/api/admin'
import type { AssignedMatch } from '@/api/admin.types'
import { formats } from '@/api/format'
import type { SearchProps } from '@/app/admin/_lib/page'
import { AdminShell, Chip, readNotice } from '@/components/admin/AdminParts'
import { PendingBadge, PendingSummary, PrecacheScorePages } from '@/components/admin/PendingQueue'

// 管理画面のトップ(Claude Design「Score View Admin」の大会一覧)。自分の担当試合(記録係の割り当てが
// 覆う試合。時刻順)と、大会を開催中、これからの大会、終わった大会に分けた一覧(記録係は担当大会だけ)。
// 記録係が開いてすぐ次の試合の記録に入れるよう、担当試合を先頭に置く。大会を作るのは別の画面。

export default async function AdminTopPage({ searchParams }: SearchProps) {
  const query = await searchParams
  const notice = readNotice(query)
  const search = typeof query.q === 'string' ? query.q : ''
  const { user, token } = await requireUser('any')
  const [list, mine] = await loadAdmin(() =>
    Promise.all([tournaments.list(token), matches.mine(token)]),
  )
  const today = jstToday(new Date())
  const groups = groupTournaments(searchTournaments(list.tournaments, search), today)
  // 担当試合の未定のチームは、準備の画面と同じ言葉(「A組1位」「F2 の勝者」)で出す。決まり方は
  // 大会の形の試合一覧にある(記録係も担当大会の部門なら読める)
  const divisionIds = [...new Set(mine.matches.map((match) => match.divisionId))]
  const formatLists = await loadAdmin(() =>
    Promise.all(divisionIds.map((divisionId) => formats.matchesIfFormatted(token, divisionId))),
  )
  const labels = sideLabelsOf(formatLists.flatMap((formatList) => formatList?.matches ?? []))
  const isAdmin = user.role === 'admin'
  return (
    <AdminShell
      user={user}
      title={isAdmin ? '大会' : '自分の担当試合'}
      notice={notice}
      primary={isAdmin ? { label: '大会を作る', href: '/admin/tournaments/new' } : undefined}
    >
      {isAdmin && mine.matches.length === 0 ? null : (
        <MyMatches list={mine.matches} labels={labels} />
      )}
      {list.tournaments.length > 5 ? (
        <form method="get" className="adm-search-row">
          <label className="adm-field">
            <span>大会名で探す</span>
            <input type="search" name="q" defaultValue={search} />
          </label>
          <button type="submit" className="secondary">
            探す
          </button>
        </form>
      ) : null}
      {groups.length === 0 ? (
        <p className="adm-muted">
          {search !== ''
            ? `「${search}」に合う大会がありません。`
            : isAdmin
              ? '大会がありません。「大会を作る」から始めてください。'
              : '担当の大会がありません。'}
        </p>
      ) : (
        groups.map((group) => (
          <section key={group.title}>
            <h2 className="adm-section-title">{group.title}</h2>
            <ul className="adm-rows">
              {group.tournaments.map((tournament) => {
                const chip = phaseChip(tournament, tournamentPhase(tournament, today))
                return (
                  <li key={tournament.id}>
                    <a className="adm-row" href={`/admin/tournaments/${tournament.id}`}>
                      <span className="adm-row-main">
                        <span className="adm-row-title">{tournament.name.ja}</span>
                        <span className="adm-row-meta">
                          {tournament.startDate ? japaneseDate(tournament.startDate) : '開催日未定'}
                          {tournament.listed ? ' ・ 公開中' : ' ・ 非公開'}
                        </span>
                      </span>
                      <Chip tone={chip.tone}>{chip.label}</Chip>
                      <span className="adm-chevron" aria-hidden="true">
                        ›
                      </span>
                    </a>
                  </li>
                )
              })}
            </ul>
          </section>
        ))
      )}
    </AdminShell>
  )
}

function MyMatches({ list, labels }: { list: AssignedMatch[]; labels: SideLabels }) {
  if (list.length === 0) {
    return (
      <>
        <PendingSummary listed={[]} />
        <p className="adm-muted">
          担当の試合はまだありません。本部(管理者)に、コートと時間帯の割り当てを頼んでください。
        </p>
      </>
    )
  }
  // 記録が終わっていない試合を先に、終わった試合を後ろにまとめる(次にやる試合が一番上に来る)
  const ordered = [
    ...list.filter((match) => match.status !== 'finished'),
    ...list.filter((match) => match.status === 'finished'),
  ]
  // 終わっていない担当試合の記録画面は、通信のあるうちに端末へ保存する(圏外でも開けるように)
  const toScore = ordered
    .filter((match) => match.status !== 'finished')
    .map((match) => `/admin/matches/${match.id}/score`)
  return (
    <>
      <PendingSummary listed={list.map((match) => match.id)} />
      <PrecacheScorePages paths={toScore} />
      <ul className="adm-matches">
        {ordered.map((match) => (
          <li key={match.id} className={match.status}>
            <a href={`/admin/matches/${match.id}/score`}>
              <span className="when">
                {[match.date?.slice(5).replace('-', '/'), match.time, courtLabel(match.field)]
                  .filter(Boolean)
                  .join(' ')}
              </span>
              <span className="teams">
                {match.code} {sideText(match.left, labels.get(match.id)?.left)} -{' '}
                {sideText(match.right, labels.get(match.id)?.right)}
              </span>
              <span className="state">
                {match.status === 'finished' && match.resultType !== 'normal'
                  ? resultTypeLabels[match.resultType]
                  : statusLabel(match.status)}
                {match.status === 'finished' && match.left.score !== null
                  ? ` ${match.left.score}-${match.right.score}`
                  : ''}
              </span>
              <span className="sub">
                {match.tournamentName} / {match.divisionName}
              </span>
              <PendingBadge matchId={match.id} />
            </a>
          </li>
        ))}
      </ul>
    </>
  )
}
