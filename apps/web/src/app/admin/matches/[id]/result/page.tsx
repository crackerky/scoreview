import { operatorErrorText } from '@/admin/errorText.utils'
import { safeAdminPath } from '@/admin/form.utils'
import { forfeitScoreText } from '@/admin/hq.utils'
import { resultTypeLabels } from '@/admin/scoreOps.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { sideText } from '@/admin/timetable.utils'
import { matches } from '@/api/admin'
import type { MatchDetail } from '@/api/admin.types'
import { ApiError } from '@/api/client'
import { formats } from '@/api/format'
import type { Impact } from '@/api/format.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { correctMatchAction, reopenMatchAction, setResultAction } from '@/app/admin/_actions/hq'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Hidden } from '@/components/admin/AdminParts'
import '@/styles/format-admin.css'

// 本部の結果の入力と訂正(docs/operations/operator-experience.md「当日」5)。スコアを入れて「影響を
// 確かめる」を押すと、保存の前に順位と次の組み合わせがどう変わるかを出す。
// - 画面を開いたときのスコア(seen)を保存に添え、この間に別の人が結果を変えていたら保存を止めて、
//   今の結果と入力を並べて上書きするかを聞く(楽観的ロック。レビュー H7)
// - 記録係の得点経過がある試合は、得点経過を残したまま最終スコアで確定できる(H5)
// - 試合をしなかった試合は不戦勝などを選んで記録する(H3)
// - 下流の扱いを選ばずに保存したら、その欄を赤で囲んで理由を出す(M6)

function pick(value: string | string[] | undefined): string {
  return typeof value === 'string' ? value : ''
}

function scoreOf(value: string): number | null {
  return /^\d{1,3}$/.test(value) ? Number(value) : null
}

type Score = { left: number; right: number }

/** 試合の今のスコア(スコアなしは 0-0。api の楽観的ロックの比べ方と同じ) */
function currentScore(detail: MatchDetail): Score {
  return { left: detail.match.left.score ?? 0, right: detail.match.right.score ?? 0 }
}

/** 得点経過の合計(最後の得点経過の累計) */
function eventsScore(detail: MatchDetail): Score | null {
  const last = detail.events.at(-1)
  return last ? { left: last.leftScore, right: last.rightScore } : null
}

function resultText(detail: MatchDetail): string {
  const match = detail.match
  const status = { scheduled: '試合前', live: '試合中', finished: '終了' }[match.status]
  const kind = match.resultType === 'normal' ? '' : ` ${resultTypeLabels[match.resultType]}`
  const score =
    match.left.score === null || match.right.score === null
      ? 'スコアなし'
      : `${match.left.score}-${match.right.score}`
  return `${status}${kind} ${score}`
}

function DownstreamChoice({ missing }: { missing: boolean }) {
  return (
    <fieldset className={`adm-field hq-approval${missing ? ' missing' : ''}`}>
      <legend>既に結果のある下流の試合の扱い(選んでください)</legend>
      {missing ? (
        <p className="hq-approval-why" role="alert">
          この変更で、既に結果のある試合の対戦チームが変わります。下流の試合をどう扱うかを選んでから、もう一度押してください。
        </p>
      ) : null}
      <label className="adm-check">
        <input type="radio" name="downstream" value="reset" required />
        <span>
          下流の試合の結果を消して未試合に戻す(訂正後の順位のチームで組み直し、試合をやり直す)
        </span>
      </label>
      <label className="adm-check">
        <input type="radio" name="downstream" value="keep" required />
        <span>
          下流の試合の結果は残す(「食い違い」として本部の画面に出る。結果は差し替えるまで次の試合と最終順位に使わない)
        </span>
      </label>
    </fieldset>
  )
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
  const keepEvents = pick(query.keep) === '1'
  const changed = pick(query.changed) === '1'
  const approval = pick(query.approval)
  const next = safeAdminPath(pick(query.next) || `/admin/divisions/${match.divisionId}/progress`)
  const back = `/admin/matches/${id}/result`
  const now = currentScore(detail)
  // 画面を開いたときのスコア。影響を確かめて開き直しても最初の値を引き継ぐ
  const seen = /^\d+-\d+$/.test(pick(query.seen)) ? pick(query.seen) : `${now.left}-${now.right}`
  const recorded = eventsScore(detail)
  const hasEvents = detail.events.length > 0
  const decided = match.left.team !== null && match.right.team !== null

  let impact: Impact | null = null
  let impactError: string | null = null
  if (left !== null && right !== null && !changed && decided) {
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
      impactError = operatorErrorText(error.code, error.message)
    }
  }
  const leftName = sideText(match.left, undefined)
  const rightName = sideText(match.right, undefined)
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
        <br />
        今の結果: {resultText(detail)}
      </p>

      {changed ? (
        <ChangedMeanwhile
          detail={detail}
          input={left !== null && right !== null ? { left, right } : null}
          winnerSide={winnerSide}
          keepEvents={keepEvents}
          next={next}
          back={back}
          resultType={pick(query.resultType)}
        />
      ) : null}

      {decided ? null : (
        <p className="adm-notice error">
          対戦チームが決まっていないので、まだ結果は入れられません。前の試合の結果か、プールの確定を待ってください。
        </p>
      )}

      {decided && hasEvents && recorded ? (
        <section className="hq-box">
          <h2>記録係の得点経過があります</h2>
          <p>
            得点経過の合計は {recorded.left}-{recorded.right} です
            {detail.scorer ? `(${detail.scorer.name} さんが記録中)` : ''}。
            記録の途中で止まった試合は、得点経過を残したまま最終スコアで確定できます(得点経過と違うことは変更の履歴に残り、閲覧ページは最終スコアを出します)。
          </p>
          <p className="adm-actions">
            <a href={`/admin/matches/${id}/score`}>スコア入力画面を開く</a>
          </p>
        </section>
      ) : null}

      {decided && hasEvents && finished ? (
        <form action={reopenMatchAction} className="hq-box">
          <Hidden values={{ back, matchId: id }} />
          <p>得点経過そのものを直すときは、試合中に戻してスコア入力画面で直します。</p>
          <DownstreamChoice missing={approval === 'reopen'} />
          <button type="submit" className="danger">
            試合中に戻して得点経過を直す
          </button>
        </form>
      ) : null}

      {decided && !changed ? (
        <form method="get" action={back}>
          <h2>スコアで結果を入れる</h2>
          <Hidden values={{ next, seen }} />
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
          </div>
          {hasEvents ? (
            <label className="adm-check">
              <input type="checkbox" name="keep" value="1" defaultChecked={keepEvents} required />
              <span>得点経過を残したまま、この最終スコアで確定する</span>
            </label>
          ) : null}
          <button type="submit">影響を確かめる</button>
        </form>
      ) : null}

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
                seen,
                matchId: id,
                left,
                right,
                winnerSide: winnerSide ?? '',
                keepEvents: keepEvents ? '1' : '',
              }}
            />
            {impact.needsApproval ? <DownstreamChoice missing={approval === 'missing'} /> : null}
            <button type="submit" className={impact.needsApproval ? 'danger' : ''}>
              この結果で保存する
            </button>
          </form>
        </>
      ) : null}

      {decided && !finished && !changed ? (
        <ExceptionalResult detail={detail} back={back} next={next} seen={seen} />
      ) : null}
    </AdminShell>
  )
}

/** この間に別の人が結果を変えていた: 今の結果と入力を並べ、上書きするかを聞く(H7) */
function ChangedMeanwhile({
  detail,
  input,
  winnerSide,
  keepEvents,
  next,
  back,
  resultType,
}: {
  detail: MatchDetail
  input: Score | null
  winnerSide: 'left' | 'right' | null
  keepEvents: boolean
  next: string
  back: string
  resultType: string
}) {
  const now = currentScore(detail)
  const mine =
    input !== null
      ? `${input.left}-${input.right}`
      : resultType in resultTypeLabels
        ? resultTypeLabels[resultType as keyof typeof resultTypeLabels] // 直前の in で値を確かめた
        : '(入力)'
  return (
    <section className="hq-box danger" role="alert">
      <h2>
        この間に結果が {detail.match.left.score ?? '-'}-{detail.match.right.score ?? '-'}
        に変わりました。上書きしますか
      </h2>
      <table className="hq-rank">
        <tbody>
          <tr>
            <th>今の結果(ほかの人が保存)</th>
            <td>{resultText(detail)}</td>
          </tr>
          <tr>
            <th>あなたの入力</th>
            <td>{mine}</td>
          </tr>
        </tbody>
      </table>
      <div className="adm-actions">
        {input !== null ? (
          <a
            href={`${back}?${new URLSearchParams({
              left: String(input.left),
              right: String(input.right),
              ...(winnerSide ? { winner: winnerSide } : {}),
              ...(keepEvents ? { keep: '1' } : {}),
              seen: `${now.left}-${now.right}`,
              next,
            }).toString()}`}
          >
            今の結果を確かめてから、自分の入力で上書きする
          </a>
        ) : null}
        <a href={next}>上書きしない(今の結果のまま)</a>
      </div>
    </section>
  )
}

/** 試合をしなかった試合の結果(不戦勝、両チーム不戦敗、不成立、未消化)。H3 */
function ExceptionalResult({
  detail,
  back,
  next,
  seen,
}: {
  detail: MatchDetail
  back: string
  next: string
  seen: string
}) {
  const leftName = sideText(detail.match.left, undefined)
  const rightName = sideText(detail.match.right, undefined)
  return (
    <form action={setResultAction} className="hq-box">
      <h2>試合をしなかったとき</h2>
      <Hidden values={{ back, next, seen, matchId: detail.match.id }} />
      <fieldset className="adm-field">
        <legend>結果の種類</legend>
        <label className="adm-check">
          <input type="radio" name="resultType" value="forfeit" required />
          <span>不戦勝(相手が来なかった)。{forfeitScoreText(detail.forfeitScore)}</span>
        </label>
        <label className="adm-check">
          <input type="radio" name="resultType" value="double_forfeit" required />
          <span>
            両チーム不戦敗(どちらも来なかった)。両チームの負けとして数え、スコアは記録しません
          </span>
        </label>
        <label className="adm-check">
          <input type="radio" name="resultType" value="no_contest" required />
          <span>不成立(人数不足など)。勝敗だけを記録します</span>
        </label>
        <label className="adm-check">
          <input type="radio" name="resultType" value="unplayed" required />
          <span>未消化(時間切れなどで行わなかった)。順位には数えません</span>
        </label>
      </fieldset>
      <label className="adm-field">
        <span>勝ったチーム(不戦勝と不成立のとき)</span>
        <select name="resultWinner" defaultValue="">
          <option value="">選んでください</option>
          <option value="left">{leftName}</option>
          <option value="right">{rightName}</option>
        </select>
      </label>
      <button type="submit">この結果で記録する</button>
    </form>
  )
}
