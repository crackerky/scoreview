import { safeAdminPath } from '@/admin/form.utils'
import { sideName } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { matches } from '@/api/admin'
import { ApiError } from '@/api/client'
import { formats } from '@/api/format'
import type { Impact } from '@/api/format.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { correctMatchAction, reopenMatchAction } from '@/app/admin/_actions/format'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import '@/styles/format-admin.css'

// 結果の入力と訂正(docs/operations/operator-experience.md「当日」5)。スコアを入れて「影響を確かめる」を
// 押すと、保存の前に順位と次の組み合わせがどう変わるかを出す。既に結果のある試合に影響するなら、
// 下流の結果の扱い(未試合に戻す / 食い違いとして残す)を選ばないと保存できない。

function DownstreamChoice({ required }: { required: boolean }) {
  return (
    <fieldset className="adm-field">
      <legend>既に結果のある下流の試合の扱い</legend>
      {required ? null : (
        <label className="adm-check">
          <input type="radio" name="downstream" value="" defaultChecked />
          <span>下流に結果のある試合があれば止める</span>
        </label>
      )}
      <label className="adm-check">
        <input type="radio" name="downstream" value="reset" required={required} />
        <span>
          下流の試合の結果を消して未試合に戻す(訂正後の順位のチームで組み直し、試合をやり直す)
        </span>
      </label>
      <label className="adm-check">
        <input type="radio" name="downstream" value="keep" required={required} />
        <span>
          下流の試合の結果は残す(「食い違い」として本部の画面に出る。結果は差し替えるまで次の試合と最終順位に使わない)
        </span>
      </label>
    </fieldset>
  )
}

function pick(value: string | string[] | undefined): string {
  return typeof value === 'string' ? value : ''
}

function scoreOf(value: string): number | null {
  return /^\d{1,3}$/.test(value) ? Number(value) : null
}

export default async function MatchResultPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const query = await props.searchParams
  const { user, token } = await requireUser('admin')
  const detail = await loadAdmin(() => matches.detail(token, id))
  const match = detail.match
  const left = scoreOf(pick(query.left))
  const right = scoreOf(pick(query.right))
  const winner = pick(query.winner)
  const winnerSide = winner === 'left' || winner === 'right' ? winner : null
  const next = safeAdminPath(pick(query.next) || `/admin/divisions/${match.divisionId}/progress`)
  const back = `/admin/matches/${id}/result`

  let impact: Impact | null = null
  let impactError: string | null = null
  if (left !== null && right !== null) {
    try {
      impact = await formats.correctionImpact(token, id, {
        score: { left, right },
        winnerSide,
        downstream: null,
      })
    } catch (error) {
      if (!(error instanceof ApiError) || error.status >= 500) {
        throw error
      }
      impactError = error.message
    }
  }
  const leftName = sideName(match.left)
  const rightName = sideName(match.right)
  const finished = match.status === 'finished'
  const crumbs = [
    { href: `/admin/divisions/${match.divisionId}/progress`, label: '本部の進行画面' },
  ]
  return (
    <AdminShell
      user={user}
      title={`${match.code} の結果を${finished ? '直す' : '入れる'}`}
      notice={notice}
      crumbs={crumbs}
    >
      <p>
        {match.date ?? ''} {match.time ?? ''} {match.field ? `コート ${match.field}` : ''}
        <br />
        <strong>
          {leftName} 対 {rightName}
        </strong>
        {finished ? (
          <>
            <br />
            今の結果: {match.left.score ?? '-'} - {match.right.score ?? '-'}
          </>
        ) : null}
      </p>
      {detail.events.length > 0 ? (
        <p className="adm-notice error">
          この試合には得点経過があります。結果は{' '}
          <a href={`/admin/matches/${id}/score`}>スコア入力画面</a> で得点経過を直してください。
        </p>
      ) : null}
      {detail.events.length > 0 && finished ? (
        <form action={reopenMatchAction}>
          <Hidden values={{ back, matchId: id }} />
          <DownstreamChoice required={false} />
          <button type="submit" className="danger">
            試合中に戻して得点経過を直す
          </button>
        </form>
      ) : null}

      <form method="get" action={back}>
        <input type="hidden" name="next" value={next} />
        <div className="hq-inline">
          <label className="adm-field">
            <span>{leftName}</span>
            <input
              type="number"
              name="left"
              min={0}
              inputMode="numeric"
              defaultValue={left ?? match.left.score ?? ''}
              required
            />
          </label>
          <label className="adm-field">
            <span>{rightName}</span>
            <input
              type="number"
              name="right"
              min={0}
              inputMode="numeric"
              defaultValue={right ?? match.right.score ?? ''}
              required
            />
          </label>
          {detail.inBracket ? (
            <label className="adm-field">
              <span>同点のときの勝者(サドンデス等)</span>
              <select name="winner" defaultValue={winnerSide ?? ''}>
                <option value="">スコアで決める</option>
                <option value="left">{leftName}</option>
                <option value="right">{rightName}</option>
              </select>
            </label>
          ) : null}
          <button type="submit">影響を確かめる</button>
        </div>
      </form>

      {impactError ? (
        <p className="adm-notice error" role="alert">
          {impactError}
        </p>
      ) : null}
      {impact !== null && left !== null && right !== null ? (
        <>
          <section className={`hq-impact${impact.needsApproval ? ' danger' : ''}`}>
            <h2>
              {left} - {right} にすると
            </h2>
            <ul>
              {impact.lines.map((line) => (
                <li key={line}>{line}</li>
              ))}
            </ul>
          </section>
          <form action={correctMatchAction}>
            <Hidden
              values={{
                back,
                next,
                matchId: id,
                left,
                right,
                winnerSide: winnerSide ?? '',
              }}
            />
            {impact.needsApproval ? <DownstreamChoice required={true} /> : null}
            <button type="submit" className={impact.needsApproval ? 'danger' : ''}>
              この結果で保存する
            </button>
          </form>
        </>
      ) : null}
    </AdminShell>
  )
}
