import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, teams, tournaments } from '@/api/admin'
import { formats } from '@/api/format'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { createTeamsFromRosterAction } from '@/app/admin/_actions/format'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import { FormatWizard } from '@/components/admin/FormatWizard'
import '@/styles/format-admin.css'

// 大会の形を決めるウィザード。チームの登録(名簿の貼り付け)から、形の候補、プレビュー、適用まで。

export default async function FormatWizardPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const division = await loadAdmin(() => divisions.get(token, id))
  const [tournament, teamList, saved, copies] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, division.tournamentId),
      teams.list(token, id),
      formats.get(token, id),
      formats.list(token, id),
    ]),
  )
  const back = `/admin/divisions/${id}/format`
  const refs = teamList.teams.map((team) => ({ id: team.id, name: team.shortName }))
  // 保存済みの形があればそのシード順、無ければ登録順(並び順、ID の順)
  const known = new Set(saved.format?.seeds ?? [])
  const initialSeeds =
    saved.format && saved.format.seeds.length === refs.length
      ? saved.format.seeds
      : [
          ...(saved.format?.seeds ?? []),
          ...refs.map((team) => team.id).filter((teamId) => !known.has(teamId)),
        ]
  const crumbs = [
    { href: `/admin/tournaments/${tournament.id}`, label: tournament.name.ja },
    { href: `/admin/divisions/${id}`, label: division.name.ja },
  ]
  return (
    <AdminShell
      user={user}
      title={`${division.name.ja}: 大会の形を決める`}
      notice={notice}
      crumbs={crumbs}
    >
      {saved.format ? (
        <p className="adm-notice ok">
          今の形: {saved.format.title}(第 {saved.format.version} 版)。
          <a href={`/admin/divisions/${id}/progress`}>本部の進行画面へ</a>
        </p>
      ) : null}
      <details open={refs.length === 0}>
        <summary>チームを名簿から登録する(登録済み {refs.length} チーム)</summary>
        <form action={createTeamsFromRosterAction}>
          <Hidden values={{ back, divisionId: id, startOrder: refs.length }} />
          <label className="adm-field">
            <span>
              1 行に 1
              チーム。「正式名」か「正式名、短縮名」。表計算からの貼り付けもできます。上から順にシード順の初期値になります
            </span>
            <textarea name="roster" rows={8} />
          </label>
          <button type="submit">まとめて登録</button>
        </form>
      </details>
      {refs.length >= 2 ? (
        <FormatWizard
          divisionId={id}
          teams={refs}
          initialSeeds={initialSeeds}
          defaultDate={tournament.startDate ?? new Date().toISOString().slice(0, 10)}
          saved={saved.format}
          copies={copies.formats}
        />
      ) : (
        <p>チームを 2 つ以上登録すると、形の候補を出せます。</p>
      )}
    </AdminShell>
  )
}
