import { sideName, statusLabel, tournamentStatusOptions } from '@/admin/labels.utils'
import { courtLabel } from '@/admin/scoreKeeper.utils'
import { resultTypeLabels } from '@/admin/scoreOps.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { matches, tournaments } from '@/api/admin'
import type { AssignedMatch } from '@/api/admin.types'
import { createTournamentAction } from '@/app/admin/_actions/tournament'
import type { SearchProps } from '@/app/admin/_lib/page'
import { AdminShell, Field, Hidden, readNotice } from '@/components/admin/AdminParts'
import { PendingBadge, PendingSummary, PrecacheScorePages } from '@/components/admin/PendingQueue'

// 管理画面のトップ。自分の担当試合(記録係の割り当てが覆う試合。時刻順)、大会の一覧(記録係は担当大会
// だけ)、管理者なら大会の作成。記録係が開いてすぐ次の試合の記録に入れるよう、担当試合を先頭に置く。
export default async function AdminTopPage({ searchParams }: SearchProps) {
  const notice = readNotice(await searchParams)
  const { user, token } = await requireUser('any')
  const [list, mine] = await loadAdmin(() =>
    Promise.all([tournaments.list(token), matches.mine(token)]),
  )
  const statusLabel = (value: string) =>
    tournamentStatusOptions.find((option) => option.value === value)?.label ?? value
  return (
    <AdminShell
      user={user}
      title={user.role === 'admin' ? '大会一覧' : '自分の担当試合'}
      notice={notice}
    >
      {user.role === 'admin' && mine.matches.length === 0 ? null : (
        <MyMatches list={mine.matches} />
      )}
      {user.role === 'admin' ? null : <h2>担当の大会</h2>}
      {list.tournaments.length === 0 ? (
        <p className="adm-muted">
          {user.role === 'admin' ? '大会がありません。' : '担当の大会がありません。'}
        </p>
      ) : (
        <ul className="adm-list">
          {list.tournaments.map((tournament) => (
            <li key={tournament.id}>
              <a href={`/admin/tournaments/${tournament.id}`}>
                {tournament.name.ja}
                <span className="adm-muted">
                  ({statusLabel(tournament.status)}
                  {tournament.startDate ? ` ${tournament.startDate}` : ''}
                  {tournament.listed ? '' : ' 一覧に非掲載'})
                </span>
              </a>
            </li>
          ))}
        </ul>
      )}
      {user.role === 'admin' ? (
        <>
          <h2>大会を作る</h2>
          <form action={createTournamentAction}>
            <Hidden
              values={{
                back: '/admin',
                status: 'entry_open',
                teamListLabel: 'teams',
                groupLabel: 'groups',
                listed: 'on',
                showSchedule: 'on',
              }}
            />
            <div className="adm-grid">
              <Field label="大会名(日本語)" name="nameJa" required />
              <Field label="大会名(英語)" name="nameEn" />
              <Field label="開催日" name="startDate" type="date" />
            </div>
            <button type="submit">作成</button>
          </form>
        </>
      ) : null}
    </AdminShell>
  )
}

function MyMatches({ list }: { list: AssignedMatch[] }) {
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
                {match.code} {sideName(match.left)} - {sideName(match.right)}
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
