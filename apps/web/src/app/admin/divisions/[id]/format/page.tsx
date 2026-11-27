import { draftState, seedsFromPools } from '@/admin/build.utils'
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
// 適用まで。?pools=draft で開くと、試合を組むの組分けで決めた組のまま、そのあとの順位決定戦と時間割を
// 決める(候補は組を変えない形だけ)。

export default async function FormatWizardPage(props: IdPageProps) {
  const [{ id, notice }, query] = await Promise.all([readAdminPage(props), props.searchParams])
  const { user, token } = await requireUser('admin')
  const division = await loadAdmin(() => divisions.get(token, id))
  const [tournament, teamList, saved, copies, matchList, draft] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, division.tournamentId),
      teams.list(token, id),
      formats.get(token, id),
      formats.list(token, id),
      matches.list(token, id),
      formats.poolDraft(token, id),
    ]),
  )
  // 試合を組むの組分けで決めた組(全チームが入り、どの組も 2 チーム以上のときだけ使う)
  const fromDraft =
    query.pools === 'draft' &&
    draft.pools !== null &&
    draftState(
      draft.pools,
      teamList.teams.map((team) => team.id),
    ).complete
      ? seedsFromPools(draft.pools)
      : null
  // 結果の入った試合(試合中、終了)があれば形は作り直せない(api も 409 で断る)。ボタンを先に止める
  const startedMatches = matchList.matches.filter((match) => match.status !== 'scheduled').length
  const back = `/admin/divisions/${id}/format`
  const refs = teamList.teams.map((team) => ({ id: team.id, name: team.shortName }))
  // 保存済みの形(大会のコピーなら引き継いだ形)のシード順、無ければ登録順(並び順、ID の順)
  const seeded = saved.format?.seeds ?? saved.inherited?.seeds ?? []
  const known = new Set(seeded)
  const initialSeeds =
    fromDraft?.seeds ??
    (seeded.length === refs.length
      ? seeded
      : [...seeded, ...refs.map((team) => team.id).filter((teamId) => !known.has(teamId))])
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
          <a href={`/admin/divisions/${id}/progress`}>部門の進行へ</a>
        </p>
      ) : null}
      {fromDraft ? (
        <p className="adm-notice ok">
          試合を組むの組分けで決めた {fromDraft.poolSeeds.length}{' '}
          組のまま作ります。条件を入れて「形の候補を出す」を押すと、組のあとに続く順位決定戦の候補が出ます。
          <a href={`/admin/divisions/${id}/build?step=pools`}>組分けを直す</a>
        </p>
      ) : null}
      {query.pools === 'draft' && !fromDraft ? (
        <p className="adm-notice error">
          組分けがまだ終わっていません(全部のチームが組に入り、どの組も 2 チーム以上)。
          <a href={`/admin/divisions/${id}/build?step=pools`}>組分けへ</a>
        </p>
      ) : null}
      {saved.inherited && !fromDraft ? (
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
          divisionName={`${division.name.ja} ${division.name.en}`}
          tournamentId={tournament.id}
          teams={refs}
          initialSeeds={initialSeeds}
          defaultDate={tournament.startDate ?? new Date().toISOString().slice(0, 10)}
          saved={saved.format}
          inherited={saved.inherited}
          copies={copies.formats}
          startedMatches={startedMatches}
          fixedPools={fromDraft?.poolSeeds ?? null}
        />
      ) : (
        <p>チームを 2 つ以上登録すると、形の候補を出せます。</p>
      )}
    </AdminShell>
  )
}
