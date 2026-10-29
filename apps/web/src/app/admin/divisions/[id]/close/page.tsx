import { describeSide } from '@/admin/formatWizard.utils'
import { closingReason, sharedRanks } from '@/admin/hq.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions } from '@/api/admin'
import { formats } from '@/api/format'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { closeDivisionAction } from '@/app/admin/_actions/hq'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import '@/styles/format-admin.css'

// 大会を締める(レビュー H10)。残りの試合をまとめて未消化にし、決まらない順位を同順位で提案して、
// 確かめたうえで最終順位を書き、そのまま閲覧ページに公開する。

export default async function CloseDivisionPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('hq')
  const [division, closing] = await loadAdmin(() =>
    Promise.all([divisions.get(token, id), formats.closing(token, id)]),
  )
  const progress = `/admin/divisions/${id}/progress`
  const shared = sharedRanks(closing.ranks)
  return (
    <AdminShell
      user={user}
      title={`${division.name.ja}: 大会を締める`}
      notice={notice}
      crumbs={[{ href: progress, label: '本部の進行画面' }]}
    >
      {closing.closedAt ? (
        <p className="adm-notice ok">
          締めました
          {closing.published ? '。最終順位を公開しています' : '(最終順位はまだ公開していません)'}
        </p>
      ) : null}

      <section className="hq-box">
        <h2>
          {closing.remaining.length > 0
            ? `残りの ${closing.remaining.length} 試合を未消化にします`
            : '残りの試合はありません'}
        </h2>
        {closing.remaining.length > 0 ? (
          <ul className="adm-list">
            {closing.remaining.map((match) => (
              <li key={match.id}>
                {match.code} {match.time ?? '--:--'} {match.groupLabel}:{' '}
                {match.left.team?.name.ja ?? describeSide(match.left.label)} 対{' '}
                {match.right.team?.name.ja ?? describeSide(match.right.label)}
                {match.status === 'live' ? '(試合中。今のスコアのまま未消化にします)' : ''}
              </li>
            ))}
          </ul>
        ) : null}
      </section>

      <section className="hq-box">
        <h2>{closing.closedAt ? '最終順位' : 'この最終順位にします'}</h2>
        {closing.ties.length > 0 ? (
          <ul className="hq-ties">
            {closing.ties.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        ) : null}
        <table className="hq-rank">
          <thead>
            <tr>
              <th>順位</th>
              <th>チーム</th>
              <th>決め方</th>
            </tr>
          </thead>
          <tbody>
            {closing.ranks.map((row) => (
              <tr key={row.team.id}>
                <td>
                  {row.rank} 位{shared.has(row.rank) ? 'タイ' : ''}
                </td>
                <td>{row.team.name.ja}</td>
                <td className="hq-reason">{closingReason(row, shared.has(row.rank))}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {closing.closedAt ? (
        // 締めの取り消しは無いので、締めた後は締める操作を出さない
        <p className="adm-actions">
          <a href={progress}>本部の進行画面へ戻る</a>
        </p>
      ) : (
        <form action={closeDivisionAction} className="hq-box danger">
          <Hidden values={{ back: `/admin/divisions/${id}/close`, divisionId: id }} />
          {closing.remaining.map((match) => (
            <input key={match.id} type="hidden" name="matchIds" value={match.id} />
          ))}
          <label className="adm-check">
            <input type="checkbox" name="publish" defaultChecked />
            <span>閲覧ページに最終順位を公開する</span>
          </label>
          <div className="adm-actions">
            <button type="submit" className="danger">
              {closing.remaining.length > 0 ? '未消化にして大会を締める' : 'この順位で大会を締める'}
            </button>
            <a href={progress}>やめる</a>
          </div>
        </form>
      )}
    </AdminShell>
  )
}
