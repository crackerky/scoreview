import { statusLabel, tournamentStatusOptions } from '@/admin/labels.utils'
import { courtLabel } from '@/admin/scoreKeeper.utils'
import { resultTypeLabels } from '@/admin/scoreOps.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { type SideLabels, sideLabelsOf, sideText } from '@/admin/timetable.utils'
import { matches, tournaments } from '@/api/admin'
import type { AdminTournament, AssignedMatch } from '@/api/admin.types'
import { formats } from '@/api/format'
import { copyTournamentAction, createTournamentAction } from '@/app/admin/_actions/tournament'
import type { SearchProps } from '@/app/admin/_lib/page'
import { AdminShell, Check, Field, Hidden, readNotice } from '@/components/admin/AdminParts'
import { PendingBadge, PendingSummary, PrecacheScorePages } from '@/components/admin/PendingQueue'

// 管理画面のトップ。自分の担当試合(記録係の割り当てが覆う試合。時刻順)、大会の一覧(記録係は担当大会
// だけ。名前で絞り込み、開催日か名前で並べる)、管理者なら大会の作成とコピー。記録係が開いてすぐ次の
// 試合の記録に入れるよう、担当試合を先頭に置く。

const sortOptions = [
  { value: 'date-desc', label: '開催日の新しい順' },
  { value: 'date-asc', label: '開催日の古い順' },
  { value: 'name', label: '名前の順' },
] as const

/** 一覧の絞り込み(名前の部分一致)と並べ替え。開催日の無い大会は最後 */
function filterTournaments(list: AdminTournament[], query: string, sort: string) {
  const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean)
  const hit = list.filter((tournament) => {
    const name = `${tournament.name.ja} ${tournament.name.en} ${tournament.shortName}`.toLowerCase()
    return words.every((word) => name.includes(word))
  })
  const byDate = (a: AdminTournament, b: AdminTournament) => {
    if (a.startDate === null || b.startDate === null) {
      return a.startDate === b.startDate ? 0 : a.startDate === null ? 1 : -1
    }
    return a.startDate.localeCompare(b.startDate)
  }
  return hit.sort((a, b) => {
    if (sort === 'name') {
      return a.name.ja.localeCompare(b.name.ja, 'ja')
    }
    const order = byDate(a, b)
    const dated = a.startDate !== null && b.startDate !== null
    return (sort === 'date-asc' || !dated ? order : -order) || b.id - a.id
  })
}

export default async function AdminTopPage({ searchParams }: SearchProps) {
  const query = await searchParams
  const notice = readNotice(query)
  const search = typeof query.q === 'string' ? query.q : ''
  const sort = typeof query.sort === 'string' ? query.sort : 'date-desc'
  const { user, token } = await requireUser('any')
  const [list, mine] = await loadAdmin(() =>
    Promise.all([tournaments.list(token), matches.mine(token)]),
  )
  const shown = filterTournaments(list.tournaments, search, sort)
  // 担当試合の未定のチームは、準備の画面と同じ言葉(「A組1位」「F2 の勝者」)で出す。決まり方は
  // 大会の形の試合一覧にある(記録係も担当大会の部門なら読める)
  const divisionIds = [...new Set(mine.matches.map((match) => match.divisionId))]
  const formatLists = await loadAdmin(() =>
    Promise.all(divisionIds.map((divisionId) => formats.matchesIfFormatted(token, divisionId))),
  )
  const labels = sideLabelsOf(formatLists.flatMap((formatList) => formatList?.matches ?? []))
  const statusLabel = (value: string) =>
    tournamentStatusOptions.find((option) => option.value === value)?.label ?? value
  return (
    <AdminShell
      user={user}
      title={user.role === 'admin' ? '大会一覧' : '自分の担当試合'}
      notice={notice}
    >
      {user.role === 'admin' && mine.matches.length === 0 ? null : (
        <MyMatches list={mine.matches} labels={labels} />
      )}
      {user.role === 'admin' ? null : <h2>担当の大会</h2>}
      {list.tournaments.length > 5 ? (
        <form method="get" className="adm-grid adm-search">
          <label className="adm-field">
            <span>大会名で探す</span>
            <input type="search" name="q" defaultValue={search} />
          </label>
          <label className="adm-field">
            <span>並び</span>
            <select name="sort" defaultValue={sort}>
              {sortOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
          <button type="submit" className="secondary">
            探す
          </button>
        </form>
      ) : null}
      {shown.length === 0 ? (
        <p className="adm-muted">
          {search !== ''
            ? `「${search}」に合う大会がありません。`
            : user.role === 'admin'
              ? '大会がありません。'
              : '担当の大会がありません。'}
        </p>
      ) : (
        <ul className="adm-list">
          {shown.map((tournament) => (
            <li key={tournament.id}>
              <a href={`/admin/tournaments/${tournament.id}`}>
                {tournament.name.ja}
                <span className="adm-muted">
                  ({statusLabel(tournament.status)}
                  {tournament.startDate ? ` ${tournament.startDate}` : ''}
                  {tournament.listed ? '' : ' 非公開(準備中)'})
                </span>
              </a>
            </li>
          ))}
        </ul>
      )}
      {user.role === 'admin' ? (
        <>
          <h2>大会を作る</h2>
          <p className="adm-muted">
            作った大会は準備中(公開の一覧に載りません)。準備ができたら大会の画面の「公開する」を押してください。
          </p>
          <form action={createTournamentAction}>
            <Hidden
              values={{
                back: '/admin',
                status: 'entry_open',
                teamListLabel: 'teams',
                groupLabel: 'groups',
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
          {list.tournaments.length > 0 ? (
            <>
              <h2>この大会をコピーして新しい大会を作る</h2>
              <p className="adm-muted">
                部門、大会の形、不戦勝の規定スコア、お知らせ欄、バナー、時間割の設定を引き継ぎます。試合と結果は引き継ぎません。
              </p>
              <form action={copyTournamentAction}>
                <Hidden values={{ back: '/admin' }} />
                <div className="adm-grid">
                  <label className="adm-field">
                    <span>コピー元の大会</span>
                    <select name="sourceId" required>
                      {list.tournaments.map((tournament) => (
                        <option key={tournament.id} value={tournament.id}>
                          {tournament.name.ja}
                          {tournament.startDate ? `(${tournament.startDate})` : ''}
                        </option>
                      ))}
                    </select>
                  </label>
                  <Field label="新しい大会の開催日" name="startDate" type="date" required />
                  <Field label="新しい大会の名前(空ならコピー元と同じ)" name="nameJa" />
                </div>
                <Check label="チームと選手も引き継ぐ" name="withTeams" />
                <button type="submit">コピーして作る</button>
              </form>
            </>
          ) : null}
        </>
      ) : null}
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
