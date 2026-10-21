import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, matches, teams, tournaments } from '@/api/admin'
import { formats } from '@/api/format'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { deleteDivisionTeamsAction } from '@/app/admin/_actions/format'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import { FormatWizard } from '@/components/admin/FormatWizard'
import { RosterImport } from '@/components/admin/RosterImport'
import '@/styles/format-admin.css'

// 大会の形を決めるウィザード。チームの登録(名簿の貼り付けとプレビュー)から、形の候補、プレビュー、
// 適用まで。

export default async function FormatWizardPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const division = await loadAdmin(() => divisions.get(token, id))
  const [tournament, teamList, saved, copies, matchList] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, division.tournamentId),
      teams.list(token, id),
      formats.get(token, id),
      formats.list(token, id),
      matches.list(token, id),
    ]),
  )
  // 結果の入った試合(試合中、終了)があれば形は作り直せない(api も 409 で断る)。ボタンを先に止める
  const startedMatches = matchList.matches.filter((match) => match.status !== 'scheduled').length
  const back = `/admin/divisions/${id}/format`
  const refs = teamList.teams.map((team) => ({ id: team.id, name: team.shortName }))
  // 保存済みの形(大会のコピーなら引き継いだ形)のシード順、無ければ登録順(並び順、ID の順)
  const seeded = saved.format?.seeds ?? saved.inherited?.seeds ?? []
  const known = new Set(seeded)
  const initialSeeds =
    seeded.length === refs.length
      ? seeded
      : [...seeded, ...refs.map((team) => team.id).filter((teamId) => !known.has(teamId))]
  const crumbs = [
    { href: `/admin/tournaments/${tournament.id}`, label: tournament.name.ja },
    { href: `/admin/divisions/${id}`, label: division.name.ja },
  ]
  const teamOptions = teamList.teams.map((team) => ({
    id: team.id,
    fullName: team.fullName.ja,
    shortName: team.shortName.ja,
  }))
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
      {saved.inherited ? (
        <p className="adm-notice ok">
          コピー元「{saved.inherited.tournamentName} {saved.inherited.divisionName}
          」の形と時間割の設定を初期値にしています(日付は今回の開催日から)。
        </p>
      ) : null}
      <details open={refs.length === 0}>
        <summary>チームを名簿から登録する(登録済み {refs.length} チーム)</summary>
        <RosterImport
          kind="teams"
          back={back}
          divisionId={id}
          startOrder={refs.length}
          existing={teamList.teams.map((team) => team.fullName.ja)}
        />
        {refs.length > 0 && matchList.matches.length === 0 ? (
          <form action={deleteDivisionTeamsAction} className="adm-actions">
            <Hidden values={{ back, divisionId: id }} />
            <button type="submit" className="danger">
              登録した {refs.length} チームをまとめて削除する(貼り直すとき)
            </button>
          </form>
        ) : null}
        {refs.length > 0 && matchList.matches.length > 0 ? (
          <p className="adm-muted">
            試合を作った後はチームをまとめて削除できません(試合の対戦チームが消えるため)。
          </p>
        ) : null}
      </details>
      {refs.length > 0 ? (
        <details>
          <summary>選手名簿を貼り付けて登録する</summary>
          <RosterImport kind="players" back={back} divisionId={id} teams={teamOptions} />
        </details>
      ) : null}
      {refs.length >= 2 ? (
        <FormatWizard
          divisionId={id}
          teams={refs}
          initialSeeds={initialSeeds}
          defaultDate={tournament.startDate ?? new Date().toISOString().slice(0, 10)}
          saved={saved.format}
          inherited={saved.inherited}
          copies={copies.formats}
          startedMatches={startedMatches}
        />
      ) : (
        <p>チームを 2 つ以上登録すると、形の候補を出せます。</p>
      )}
    </AdminShell>
  )
}
