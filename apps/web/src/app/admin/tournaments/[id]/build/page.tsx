import { divisionSteps, type Step } from '@/admin/build.utils'
import { loadTournamentBuild } from '@/admin/buildData'
import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, type ChipTone } from '@/components/admin/AdminParts'

// 試合を組む(大会)。部門ごとに、チーム → 組分け → 時間割の進み具合を 1 行に並べ、次にやることの
// ボタンを置く。組分けが終わって初めて組(プール A など)ができ、その後で時間割を作れる。

const tones: Record<Step['tone'], ChipTone> = { ok: 'ok', warn: 'warn', todo: 'done' }

function StepCell({ label, step }: { label: string; step: Step }) {
  return (
    <span className="bld-step">
      <small>{label}</small>
      <Chip tone={tones[step.tone]}>{step.label}</Chip>
      <small>{step.sub}</small>
    </span>
  )
}

export default async function TournamentBuildPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [tournament, rows] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), loadTournamentBuild(token, id)]),
  )
  return (
    <AdminShell
      user={user}
      title="試合を組む"
      notice={notice}
      crumbs={[{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]}
    >
      <p className="adm-muted">
        部門ごとに、チーム → 組分け → 時間割の順に進めます。組分けが終わると組(プール A
        など)ができ、その後で時間割を作れます。
      </p>
      {rows.length === 0 ? (
        <p>
          部門がまだありません。<a href={`/admin/tournaments/${id}/divisions/new`}>部門を作る</a>
        </p>
      ) : (
        <div className="bld-divs">
          {rows.map(({ build }) => {
            const steps = divisionSteps(build)
            return (
              <div key={build.id} className="bld-div">
                <span className="bld-div-name">{build.name}</span>
                <StepCell label="1 チーム" step={steps.teams} />
                <StepCell label="2 組分け" step={steps.pools} />
                <StepCell label="3 時間割" step={steps.times} />
                <a
                  className="adm-primary-link bld-next"
                  href={`/admin/divisions/${build.id}/build?step=${steps.next.step}`}
                >
                  {steps.next.label}
                </a>
              </div>
            )
          })}
        </div>
      )}
      <p className="adm-links">
        <a href={`/admin/tournaments/${id}/timetable`}>大会全体の時間割</a>
        {' ・ '}
        <a href={`/admin/tournaments/${id}/entries`}>申込</a>
      </p>
    </AdminShell>
  )
}
