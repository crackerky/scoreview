import { divisionSteps, poolLetter, seedsFromPools } from '@/admin/build.utils'
import { loadDivisionBuild } from '@/admin/buildData'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { entries } from '@/api/entry'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip } from '@/components/admin/AdminParts'
import { PoolBuilder } from '@/components/admin/PoolBuilder'

// 部門の作業場(試合を組む)。チーム → 組分け → 時間割の 3 つの段をタブにする(?step=)。組分けは
// PoolBuilder で決めて下書きに残し、時間割の段で、その組のまま形を作る画面(大会の形を決める)へ進む。

const steps = [
  { key: 'teams', label: '1 チーム' },
  { key: 'pools', label: '2 組分け' },
  { key: 'times', label: '3 時間割' },
] as const
type StepKey = (typeof steps)[number]['key']

export default async function DivisionBuildPage(props: IdPageProps) {
  const [{ id, notice }, query] = await Promise.all([readAdminPage(props), props.searchParams])
  const { user, token } = await requireUser('admin')
  const division = await loadAdmin(() => divisions.get(token, id))
  const [tournament, entryList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, division.tournamentId),
      entries.list(token, division.tournamentId),
    ]),
  )
  const data = await loadAdmin(() =>
    loadDivisionBuild(token, { id, name: division.name.ja }, entryList),
  )
  const status = divisionSteps(data.build)
  const step: StepKey = steps.some((item) => item.key === query.step)
    ? (query.step as StepKey) // 直前の some で steps のどれかと確かめた
    : status.next.step
  const base = `/admin/divisions/${id}/build`
  const initialPools =
    data.draft ?? Array.from({ length: Math.max(2, Math.round(data.teams.length / 4)) }, () => [])
  return (
    <AdminShell
      user={user}
      title={`${division.name.ja}を組む`}
      notice={notice}
      crumbs={[
        { href: `/admin/tournaments/${tournament.id}`, label: tournament.name.ja },
        { href: `/admin/tournaments/${tournament.id}/build`, label: '試合を組む' },
      ]}
      tabs={steps.map((item) => ({
        href: `${base}?step=${item.key}`,
        label: item.label,
        current: item.key === step,
      }))}
    >
      {step === 'teams' ? (
        <TeamsStep data={data} tournamentId={tournament.id} divisionId={id} />
      ) : null}
      {step === 'pools' ? (
        <>
          {data.build.formatted ? (
            <p className="adm-notice">
              この部門はもう組と試合があります({data.build.pools || '—'} 組、{data.build.matches}{' '}
              試合)。組分けを変えて時間割を作り直すと、組と試合が置き換わります(結果が入った後は作り直せません)。
            </p>
          ) : null}
          {data.teams.length < 2 ? (
            <p>チームが 2 つ以上になると組分けができます。</p>
          ) : (
            <PoolBuilder divisionId={id} teams={data.teams} initialPools={initialPools} />
          )}
        </>
      ) : null}
      {step === 'times' ? (
        <TimesStep data={data} divisionId={id} tournamentId={tournament.id} />
      ) : null}
    </AdminShell>
  )
}

type Data = Awaited<ReturnType<typeof loadDivisionBuild>>

function TeamsStep({
  data,
  tournamentId,
  divisionId,
}: {
  data: Data
  tournamentId: number
  divisionId: number
}) {
  return (
    <div className="adm-stack">
      <p className="adm-muted">
        参加が決まったチームが、そのまま組分けの候補になります。確認待ちの申込は、参加を決めると入ります。
      </p>
      <table className="adm-table">
        <thead>
          <tr>
            <th>#</th>
            <th>チーム</th>
            <th>短縮名</th>
          </tr>
        </thead>
        <tbody>
          {data.teams.map((team, index) => (
            <tr key={team.id}>
              <td className="adm-num">{index + 1}</td>
              <td>{team.fullName}</td>
              <td>{team.shortName}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {data.waiting.length > 0 ? (
        <p>
          <Chip tone="warn">確認待ち {data.waiting.length}</Chip> {data.waiting.join('、')}(
          <a href={`/admin/tournaments/${tournamentId}/entries`}>申込で参加を決める</a>)
        </p>
      ) : null}
      <p className="adm-links">
        <a href={`/admin/divisions/${divisionId}/format`}>名簿を貼ってチームを登録する</a>
      </p>
    </div>
  )
}

function TimesStep({
  data,
  divisionId,
  tournamentId,
}: {
  data: Data
  divisionId: number
  tournamentId: number
}) {
  const draft = data.build.draft
  const byId = new Map(data.teams.map((team) => [team.id, team.shortName]))
  return (
    <div className="adm-stack">
      {draft?.complete && data.draft ? (
        <>
          <p>
            組分けで決めた {data.draft.length}{' '}
            組のまま、このあとの順位決定戦と時間割(日付、コート、1 試合の時間)を決めて作ります。
          </p>
          <ul className="adm-muted">
            {data.draft.map((pool, index) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: 組は位置そのものが名前
              <li key={index}>
                {poolLetter(index)} 組:{' '}
                {pool.map((teamId) => byId.get(teamId) ?? teamId).join('、')}
              </li>
            ))}
          </ul>
          <p>
            <a
              className="adm-primary-link"
              href={`/admin/divisions/${divisionId}/format?pools=draft`}
            >
              この組分けで時間割を作る
            </a>
          </p>
          <p className="adm-muted">
            シード順は {seedsFromPools(data.draft).seeds.length} チーム(各組の上から順に A1、B1…)。
          </p>
        </>
      ) : (
        <p className="adm-notice">
          組分けがまだ終わっていません。
          <a href={`/admin/divisions/${divisionId}/build?step=pools`}>組分けへ</a>
        </p>
      )}
      {data.build.formatted ? (
        <p>
          今の時間割: {data.build.scheduled}/{data.build.matches} 試合に日時とコートがあります。
          <a href={`/admin/tournaments/${tournamentId}/timetable`}>大会全体の時間割</a>
          {' ・ '}
          <a href={`/admin/divisions/${divisionId}`}>部門の試合の一覧</a>
        </p>
      ) : null}
    </div>
  )
}
