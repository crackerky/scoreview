import { formatElapsed, matchStatusOptions, sideName, statusLabel } from '@/admin/labels.utils'
import { loadAdmin, requireUser } from '@/admin/session'
import { matches } from '@/api/admin'
import type { AdminPlayer, MatchDetail } from '@/api/admin.types'
import type { Side } from '@/api/common.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import {
  addEventAction,
  putPlayerStatsAction,
  removeEventAction,
  setStatusAction,
} from '@/app/admin/_actions/match'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Check, Field, Hidden, Select } from '@/components/admin/AdminParts'

// JavaScript なしでも記録できる簡易のスコア入力画面(docs/requirements.md A-05)。ふだんの記録は
// /admin/matches/{id}/score(端末に溜めて送る画面)で行い、ここはスクリプトが動かない端末の退避先。
// 得点者とアシストを大きなボタン(radio)で選んで「得点」を 1 タップで送り、送信ごとに画面を描き直す。

const sideLabel: Record<Side, string> = { left: '左', right: '右' }

/** 得点者とアシストの候補(スタッフを除く選手。背番号の数値順) */
function pickable(roster: AdminPlayer[]): AdminPlayer[] {
  return roster
    .filter((player) => player.role === null)
    .sort((a, b) => Number(a.number) - Number(b.number) || a.number.localeCompare(b.number))
}

function playerLabel(player: AdminPlayer): string {
  return `${player.number ? `#${player.number} ` : ''}${player.name.ja}`
}

export default async function AdminScorePage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('any')
  const detail = await loadAdmin(() => matches.detail(token, id))
  const { match } = detail
  const back = `/admin/matches/${id}/score/basic`
  const title = `${match.code || '試合'} スコア入力(簡易)`
  const crumbs = [{ href: `/admin/divisions/${match.divisionId}`, label: '試合一覧' }]
  const finished = match.status === 'finished'
  return (
    <AdminShell user={user} title={title} notice={notice} crumbs={crumbs}>
      <div className="adm-scoreboard" aria-live="polite">
        <span className="team">{sideName(match.left)}</span>
        <span>
          <span className="score">
            {match.left.score ?? 0} - {match.right.score ?? 0}
          </span>
          <br />
          <span className={`adm-badge ${match.status}`}>{statusLabel(match.status)}</span>
        </span>
        <span className="team">{sideName(match.right)}</span>
      </div>
      <p className="adm-actions">
        <a href={`/admin/matches/${id}/score`}>ふだんの記録画面に戻る</a>
        <a href={`/pscore/scv_match/open/${id}`}>閲覧ページを開く</a>
      </p>

      {finished ? (
        <p className="adm-muted">
          試合は終了しています。訂正するときは管理者が「試合終了」を取り消してください。
        </p>
      ) : (
        <>
          <div className="adm-sides">
            <SidePanel detail={detail} side="left" back={back} />
            <SidePanel detail={detail} side="right" back={back} />
          </div>
          <EndForm detail={detail} back={back} />
        </>
      )}

      <h2>得点経過(新しい順)</h2>
      <EventList detail={detail} back={back} />

      <h2>ブロック数とグレイテスト</h2>
      <StatsForm detail={detail} back={back} />

      <h2>状態の変更</h2>
      <form action={setStatusAction}>
        <Hidden values={{ back, id }} />
        <p className="adm-muted">
          得点経過を付けない試合は、ここでスコアを入れて終了にします(得点経過がある試合のスコアは経過から決まります)。
        </p>
        <div className="adm-grid">
          <Select
            label="状態"
            name="status"
            options={matchStatusOptions}
            defaultValue={match.status}
          />
          <Select
            label="勝者(同点のとき)"
            name="winnerSide"
            options={[
              { value: '', label: 'なし' },
              { value: 'left', label: `左 ${sideName(match.left)}` },
              { value: 'right', label: `右 ${sideName(match.right)}` },
            ]}
            defaultValue={match.winnerSide ?? ''}
          />
          <Field label="左のスコア" name="leftScore" type="number" />
          <Field label="右のスコア" name="rightScore" type="number" />
        </div>
        <button type="submit" className="secondary">
          状態を変える
        </button>
      </form>
    </AdminShell>
  )
}

function SidePanel({ detail, side, back }: { detail: MatchDetail; side: Side; back: string }) {
  const { match } = detail
  const teamSide = match[side]
  const roster = pickable(detail.rosters[side])
  const values = { back, id: match.id, side }
  if (teamSide.team === null) {
    return (
      <section className={`adm-side ${side}`}>
        <h2>{sideName(teamSide)}</h2>
        <p className="adm-muted">チームが未確定なので入力できません。</p>
      </section>
    )
  }
  return (
    <section className={`adm-side ${side}`}>
      <h2>
        {sideLabel[side]}: {teamSide.team.name.ja}
      </h2>
      <form action={addEventAction}>
        <Hidden values={{ ...values, kind: 'goal' }} />
        <fieldset className="adm-picks">
          <legend>得点者</legend>
          {roster.map((player) => (
            <label key={player.id}>
              <input type="radio" name="scorerId" value={player.id} />
              {playerLabel(player)}
            </label>
          ))}
        </fieldset>
        <fieldset className="adm-picks">
          <legend>アシスト</legend>
          <label>
            <input type="radio" name="assistId" value="" defaultChecked />
            なし
          </label>
          {roster.map((player) => (
            <label key={player.id}>
              <input type="radio" name="assistId" value={player.id} />
              {playerLabel(player)}
            </label>
          ))}
        </fieldset>
        <Check label="キャラハンゴール" name="callahan" />
        <Field
          label="経過(分:秒。空欄なら最初の記録からの経過)"
          name="elapsed"
          placeholder="例 12:30"
        />
        <button type="submit" className="adm-goal">
          得点 {teamSide.team.name.ja}
        </button>
      </form>
      <div className="adm-quick">
        <form action={addEventAction}>
          <Hidden values={{ ...values, kind: 'timeout' }} />
          <button type="submit" className="secondary">
            タイムアウト
          </button>
        </form>
        <form action={addEventAction}>
          <Hidden values={{ ...values, kind: 'pull' }} />
          <button type="submit" className="secondary">
            プル({sideLabel[side]}が投げる)
          </button>
        </form>
      </div>
    </section>
  )
}

function EndForm({ detail, back }: { detail: MatchDetail; back: string }) {
  const { match } = detail
  const tied = (match.left.score ?? 0) === (match.right.score ?? 0)
  return (
    <form action={addEventAction}>
      <fieldset>
        <legend>試合終了</legend>
        <Hidden values={{ back, id: match.id, kind: 'end' }} />
        {tied ? (
          <fieldset className="adm-picks">
            <legend>
              同点です。
              {detail.inBracket
                ? 'トーナメントなので勝者を選んでください'
                : '勝者がいれば選んでください'}
            </legend>
            {detail.inBracket ? null : (
              <label>
                <input type="radio" name="winnerSide" value="" defaultChecked />
                引き分け
              </label>
            )}
            <label>
              <input type="radio" name="winnerSide" value="left" />
              {sideName(match.left)}
            </label>
            <label>
              <input type="radio" name="winnerSide" value="right" />
              {sideName(match.right)}
            </label>
          </fieldset>
        ) : null}
        <button type="submit" className="danger">
          試合終了を記録
        </button>
      </fieldset>
    </form>
  )
}

const kindLabel = { goal: '得点', timeout: 'タイムアウト', pull: 'プル', end: '試合終了' } as const

function EventList({ detail, back }: { detail: MatchDetail; back: string }) {
  const events = [...detail.events].reverse()
  if (events.length === 0) {
    return <p className="adm-muted">まだ記録がありません。</p>
  }
  return (
    <ol className="adm-events">
      {events.map((event, index) => (
        <li key={event.id}>
          <span>
            {formatElapsed(event.elapsedSec)} {kindLabel[event.kind]}
            {event.side ? `(${sideLabel[event.side]})` : ''}
            {event.scorer ? ` ${event.scorer.name.ja}` : ''}
            {event.assist ? ` ← ${event.assist.name.ja}` : ''}
            {event.callahan ? ' キャラハン' : ''}
            {event.kind === 'goal' ? ` ${event.leftScore}-${event.rightScore}` : ''}
          </span>
          <form action={removeEventAction}>
            <Hidden values={{ back, id: detail.match.id, eventId: event.id }} />
            <button type="submit" className={index === 0 ? 'danger' : 'secondary'}>
              取り消し
            </button>
          </form>
        </li>
      ))}
    </ol>
  )
}

function StatsForm({ detail, back }: { detail: MatchDetail; back: string }) {
  const stats = new Map(detail.playerStats.map((stat) => [`${stat.side}-${stat.playerId}`, stat]))
  const sides: Side[] = ['left', 'right']
  return (
    <form action={putPlayerStatsAction}>
      <Hidden values={{ back, id: detail.match.id }} />
      {sides.map((side) => (
        <div key={side} className="adm-scroll">
          <table className="adm-table">
            <caption>{sideName(detail.match[side])}</caption>
            <thead>
              <tr>
                <th>選手</th>
                <th>出場</th>
                <th>ブロック</th>
                <th>グレイテスト</th>
              </tr>
            </thead>
            <tbody>
              {pickable(detail.rosters[side]).map((player) => {
                const key = `${side}-${player.id}`
                const stat = stats.get(key)
                return (
                  <tr key={key}>
                    <td>{playerLabel(player)}</td>
                    <td>
                      <input
                        type="checkbox"
                        name={`played-${key}`}
                        defaultChecked={stat !== undefined}
                        aria-label={`${player.name.ja} の出場`}
                      />
                    </td>
                    <td>
                      <input
                        type="number"
                        min={0}
                        name={`blocks-${key}`}
                        defaultValue={stat?.blocks ?? 0}
                        aria-label={`${player.name.ja} のブロック数`}
                      />
                    </td>
                    <td>
                      <input
                        type="number"
                        min={0}
                        name={`greatests-${key}`}
                        defaultValue={stat?.greatests ?? 0}
                        aria-label={`${player.name.ja} のグレイテスト`}
                      />
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      ))}
      <button type="submit">選手記録を保存</button>
    </form>
  )
}
