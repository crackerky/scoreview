'use client'

import { formatElapsed, sideName } from '@/admin/labels.utils'
import { DEFAULT_MATCH_MINUTES, winnerLine } from '@/admin/scoreKeeper.utils'
import {
  applyOp,
  type ConfirmedScore,
  type ExceptionalResult,
  resultTypeLabels,
} from '@/admin/scoreOps.utils'
import type { AdminEvent, AdminPlayer, MatchDetail } from '@/api/admin.types'
import type { Side } from '@/api/common.types'

// 記録係のスコア入力画面(ScoreKeeper.tsx)の、下から出る確認シートと、試合前・試合後の欄。

const sides: Side[] = ['left', 'right']

/** 次の担当試合(試合を終えたら案内する) */
export type NextMatch = { id: number; label: string }

export type Sheet =
  | {
      kind: 'players'
      eventRef: string
      step: 'scorer' | 'assist'
      scorerId: number | null
      /** 記録済みの得点の付け直し(いまの得点者を出し、「得点者なし」に戻せる) */
      repick: boolean
    }
  | { kind: 'event'; eventRef: string }
  | { kind: 'eventConfirm'; eventRef: string; action: 'remove' | 'side' }
  | { kind: 'pull' }
  | {
      kind: 'finish'
      /** 最新の状態を取り直している間は確定させない */
      checking: boolean
      /** 同点のとき、勝者を選ぶ段(pick)と、選んだ結果を確かめる段(confirm) */
      step: 'pick' | 'confirm'
      winnerSide: Side | null
      notice: string | null
    }
  | { kind: 'results' }
  | { kind: 'confirm'; resultType: ExceptionalResult; winnerSide: Side | null }
  | { kind: 'long'; side: Side }
  | { kind: 'leave'; pending: number }
  | null

export function teamLabel(detail: MatchDetail, side: Side): string {
  return sideName(detail.match[side])
}

/** 得点者とアシストの候補(スタッフを除く選手。背番号の数値順) */
function pickable(roster: AdminPlayer[]): AdminPlayer[] {
  return roster
    .filter((player) => player.role === null)
    .sort((a, b) => Number(a.number) - Number(b.number) || a.number.localeCompare(b.number))
}

function playerShort(player: { number: string; name: { ja: string } }): string {
  return `${player.number ? `#${player.number} ` : ''}${player.name.ja}`
}

export function statusText(detail: MatchDetail): string {
  const { status, resultType } = detail.match
  if (status === 'scheduled') {
    return '試合前'
  }
  if (status === 'live') {
    return '試合中'
  }
  return resultType === 'normal' ? '終了' : `終了(${resultTypeLabels[resultType]})`
}

export function eventText(detail: MatchDetail, event: AdminEvent): string {
  const team = event.side === null ? '' : teamLabel(detail, event.side)
  switch (event.kind) {
    case 'goal':
      return `${team} 得点`
    case 'timeout':
      return `${team} タイムアウト`
    case 'pull':
      return `${team} がプル`
    case 'end':
      return '試合終了'
  }
}

function whoText(event: AdminEvent): string | null {
  if (event.scorer === null) {
    return null
  }
  const assist = event.assist === null ? '' : ` ← ${playerShort(event.assist)}`
  return `${playerShort(event.scorer)}${assist}${event.callahan ? ' キャラハン' : ''}`
}

export function EventRow({
  detail,
  event,
  canEdit,
  onPick,
}: {
  detail: MatchDetail
  event: AdminEvent
  canEdit: boolean
  onPick: () => void
}) {
  const time = formatElapsed(event.elapsedSec)
  const who = whoText(event)
  const body = (
    <>
      <span className="sk-event-time">{time}</span>
      {event.kind === 'goal' ? (
        <span className="sk-event-score">
          {event.leftScore}-{event.rightScore}
        </span>
      ) : null}
      <span className="sk-event-text">
        {eventText(detail, event)}
        {event.kind !== 'goal' ? null : who === null ? (
          <span className="sk-need">得点者を付ける</span>
        ) : (
          ` ${who}`
        )}
      </span>
    </>
  )
  if (!canEdit || event.kind === 'end') {
    return <div className={`sk-event ${event.kind} ${event.side ?? ''}`}>{body}</div>
  }
  return (
    <button type="button" className={`sk-event ${event.kind} ${event.side ?? ''}`} onClick={onPick}>
      {body}
    </button>
  )
}

export function StartPanel({
  detail,
  onPull,
  onResult,
}: {
  detail: MatchDetail
  onPull: (side: Side) => void
  onResult: (resultType: ExceptionalResult, winnerSide: Side | null) => void
}) {
  const ready = detail.match.left.team !== null && detail.match.right.team !== null
  return (
    <div className="sk-start">
      <h2>試合を始める</h2>
      {ready ? (
        <>
          <p className="sk-hint">最初にプル(スロー)するチームを押すと、試合が始まります。</p>
          <div className="sk-pair">
            {sides.map((side) => (
              <button
                key={side}
                type="button"
                className={`sk-big ${side}`}
                onClick={() => onPull(side)}
              >
                {teamLabel(detail, side)} がプル
              </button>
            ))}
          </div>
        </>
      ) : (
        <p className="sk-hint">対戦チームが決まっていないので、まだ記録できません。</p>
      )}
      <h2>試合をしなかったとき</h2>
      <div className="sk-results">
        {ready
          ? sides.map((side) => (
              <button
                key={side}
                type="button"
                className="secondary"
                onClick={() => onResult('forfeit', side)}
              >
                不戦勝: {teamLabel(detail, side)} の勝ち
              </button>
            ))
          : null}
        <button
          type="button"
          className="secondary"
          onClick={() => onResult('double_forfeit', null)}
        >
          両チーム不戦敗
        </button>
        <button type="button" className="secondary" onClick={() => onResult('no_contest', null)}>
          不成立(人数不足など)
        </button>
        <button type="button" className="secondary" onClick={() => onResult('unplayed', null)}>
          未消化(時間切れなど)
        </button>
      </div>
    </div>
  )
}

export function FinishedPanel({
  detail,
  isAdmin,
  nextMatch,
}: {
  detail: MatchDetail
  isAdmin: boolean
  nextMatch: NextMatch | null
}) {
  const { match } = detail
  const winner = winnerLine(detail)
  return (
    <div className="sk-done">
      {winner === null ? null : <p className="sk-winner">{winner}</p>}
      <p>
        この試合の記録は終わりました。
        {match.resultType === 'normal' || match.resultType === 'abandoned'
          ? '得点者を付け忘れた得点は、下の得点経過の行を押すと付けられます。'
          : ''}
      </p>
      {isAdmin ? (
        <p className="sk-hint">「取り消し」で試合終了を取り消し、記録を続けられます。</p>
      ) : (
        <p className="sk-hint">結果の訂正は本部(管理者)に伝えてください。</p>
      )}
      {nextMatch === null ? null : (
        <a className="sk-next" href={`/admin/matches/${nextMatch.id}/score`}>
          次の担当試合へ: {nextMatch.label}
        </a>
      )}
    </div>
  )
}

type SheetProps = {
  sheet: Exclude<Sheet, null>
  detail: MatchDetail
  elapsed: number | null
  longElapsed: boolean
  online: boolean
  onClose: () => void
  onSheet: (sheet: Sheet) => void
  onPlayers: (
    eventRef: string,
    scorerId: number | null,
    assistId: number | null,
    callahan: boolean,
  ) => void
  onPull: (side: Side) => void
  onFinish: (winnerSide: Side | null, confirmed: ConfirmedScore) => void
  onResult: (resultType: ExceptionalResult, winnerSide: Side | null) => void
  onRemove: (eventRef: string) => void
  onMoveGoal: (eventRef: string, side: Side) => void
  onLongGoal: (side: Side) => void
  onLeave: () => void
}

/** シートの下の閉じるボタンの文言と動き。アシストの段では得点者を残して閉じる(レビュー M1) */
function closeAction(props: SheetProps): { label: string; run: () => void } {
  const { sheet } = props
  if (sheet.kind === 'players' && sheet.step === 'assist' && !sheet.repick) {
    return {
      label: 'あとで付ける(アシストなしで保存)',
      run: () => props.onPlayers(sheet.eventRef, sheet.scorerId, null, false),
    }
  }
  if (sheet.kind === 'players') {
    return { label: 'あとで付ける', run: props.onClose }
  }
  if (sheet.kind === 'leave') {
    return { label: '記録画面に残る', run: props.onClose }
  }
  return { label: 'やめる', run: props.onClose }
}

export function SheetView(props: SheetProps) {
  const close = closeAction(props)
  return (
    <div className="sk-overlay">
      <div className="sk-sheet" role="dialog" aria-modal="true" aria-labelledby="sk-sheet-title">
        <SheetBody {...props} />
        <button type="button" className="secondary sk-sheet-close" onClick={close.run}>
          {close.label}
        </button>
      </div>
    </div>
  )
}

function SheetBody(props: SheetProps) {
  const { sheet, detail, onSheet } = props
  switch (sheet.kind) {
    case 'players':
      return (
        <PlayersSheet sheet={sheet} detail={detail} onSheet={onSheet} onPlayers={props.onPlayers} />
      )
    case 'event':
      return <EventSheet sheet={sheet} detail={detail} onSheet={onSheet} />
    case 'eventConfirm':
      return (
        <EventConfirmSheet
          sheet={sheet}
          detail={detail}
          onRemove={props.onRemove}
          onMoveGoal={props.onMoveGoal}
        />
      )
    case 'pull':
      return (
        <>
          <h2 id="sk-sheet-title">どちらがプル(スロー)しましたか</h2>
          <p className="sk-hint">得点の後、守る側が投げ入れて次の 1 点が始まります。</p>
          <div className="sk-pair">
            {sides.map((side) => (
              <button
                key={side}
                type="button"
                className={`sk-big ${side}`}
                onClick={() => props.onPull(side)}
              >
                {teamLabel(detail, side)}
              </button>
            ))}
          </div>
        </>
      )
    case 'finish':
      return <FinishSheet {...props} sheet={sheet} />
    case 'results':
      return <ResultsSheet detail={detail} onSheet={onSheet} />
    case 'confirm':
      return (
        <ConfirmSheet sheet={sheet} detail={detail} onSheet={onSheet} onResult={props.onResult} />
      )
    case 'long':
      return (
        <>
          <h2 id="sk-sheet-title">経過時間が試合時間を大きく超えています</h2>
          <p className="sk-confirm-score">{formatElapsed(props.elapsed ?? 0)}</p>
          <p className="sk-hint">
            試合時間は {detail.matchMinutes ?? DEFAULT_MATCH_MINUTES}{' '}
            分です。別の試合の画面を開いていないか、試合の開始を記録し忘れていないかを確かめてください。
          </p>
          <button
            type="button"
            className={`sk-big ${sheet.side}`}
            onClick={() => props.onLongGoal(sheet.side)}
          >
            確かめた。{teamLabel(detail, sheet.side)} の得点を記録する
          </button>
        </>
      )
    case 'leave':
      return (
        <>
          <h2 id="sk-sheet-title">まだ送れていない記録が {sheet.pending} 件あります</h2>
          <p className="sk-hint">
            記録はこの端末に残り、つながったら担当試合の画面からも自動で送ります。ブラウザを閉じたり、ログアウトしたりしないでください。
          </p>
          <button type="button" className="sk-big" onClick={props.onLeave}>
            担当試合の一覧へ
          </button>
        </>
      )
  }
}

function currentWho(event: AdminEvent): string {
  return whoText(event) ?? '得点者なし'
}

function PlayersSheet({
  sheet,
  detail,
  onSheet,
  onPlayers,
}: {
  sheet: Extract<Sheet, { kind: 'players' }>
  detail: MatchDetail
  onSheet: (sheet: Sheet) => void
  onPlayers: SheetProps['onPlayers']
}) {
  const event = detail.events.find((candidate) => candidate.clientId === sheet.eventRef)
  if (event === undefined || event.side === null) {
    return <h2 id="sk-sheet-title">この得点は取り消されました</h2>
  }
  const roster = pickable(detail.rosters[event.side])
  const team = teamLabel(detail, event.side)
  const scorer = roster.find((player) => player.id === sheet.scorerId)
  const assistStep = sheet.step === 'assist' && scorer !== undefined
  // 得点者の段とアシストの段で、選手のボタンを同じ位置に並べる(押し間違えを防ぐ。レビュー M2)
  return (
    <>
      <h2 id="sk-sheet-title" className="sk-sheet-heading">
        {assistStep
          ? `${scorer.number ? `#${scorer.number}` : scorer.name.ja} の得点。アシストは?`
          : `${team} の得点(${event.leftScore}-${event.rightScore})。得点したのは?`}
      </h2>
      {sheet.repick ? <p className="sk-hint sk-now">いま: {currentWho(event)}</p> : null}
      <div className="sk-players">
        {roster.map((player) => {
          const isScorer = assistStep && player.id === scorer.id
          const current = !assistStep && sheet.repick && event.scorer?.id === player.id
          return (
            <button
              key={player.id}
              type="button"
              className={`secondary${current || isScorer ? ' picked' : ''}`}
              disabled={isScorer}
              onClick={() =>
                assistStep
                  ? onPlayers(sheet.eventRef, scorer.id, player.id, false)
                  : onSheet({ ...sheet, step: 'assist', scorerId: player.id })
              }
            >
              <span className="sk-num">{player.number ? `#${player.number}` : '-'}</span>
              <span className="sk-pname">{isScorer ? '得点した人' : player.name.ja}</span>
            </button>
          )
        })}
      </div>
      <div className="sk-actions">
        {assistStep ? (
          <>
            <button
              type="button"
              className="secondary"
              onClick={() => onSheet({ ...sheet, step: 'scorer', scorerId: null })}
            >
              ← 戻る
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => onPlayers(sheet.eventRef, scorer.id, null, false)}
            >
              アシストなし
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => onPlayers(sheet.eventRef, scorer.id, null, true)}
            >
              キャラハン
            </button>
          </>
        ) : sheet.repick && event.scorer !== null ? (
          <button
            type="button"
            className="secondary"
            onClick={() => onPlayers(sheet.eventRef, null, null, false)}
          >
            得点者なしに戻す
          </button>
        ) : null}
      </div>
      {assistStep ? (
        <p className="sk-hint">
          キャラハン: 守備側が相手のエンドゾーン内でパスを取った得点(アシストなし)。
        </p>
      ) : null}
    </>
  )
}

function EventSheet({
  sheet,
  detail,
  onSheet,
}: {
  sheet: Extract<Sheet, { kind: 'event' }>
  detail: MatchDetail
  onSheet: (sheet: Sheet) => void
}) {
  const event = detail.events.find((candidate) => candidate.clientId === sheet.eventRef)
  if (event === undefined) {
    return <h2 id="sk-sheet-title">この記録は取り消されました</h2>
  }
  const goal = event.kind === 'goal' && event.side !== null
  const other = event.side === 'left' ? 'right' : 'left'
  return (
    <>
      <h2 id="sk-sheet-title">
        {formatElapsed(event.elapsedSec)} {eventText(detail, event)}
        {goal ? `(${event.leftScore}-${event.rightScore})` : ''}
      </h2>
      {goal ? <p className="sk-hint sk-now">いま: {currentWho(event)}</p> : null}
      <div className="sk-results">
        {goal ? (
          <button
            type="button"
            className="secondary"
            onClick={() =>
              onSheet({
                kind: 'players',
                eventRef: event.clientId,
                step: 'scorer',
                scorerId: null,
                repick: true,
              })
            }
          >
            得点者・アシストを付け直す
          </button>
        ) : null}
        {goal ? (
          <button
            type="button"
            className="secondary"
            onClick={() =>
              onSheet({ kind: 'eventConfirm', eventRef: event.clientId, action: 'side' })
            }
          >
            {teamLabel(detail, other)} の得点にする(チームの押し間違い)
          </button>
        ) : null}
        <button
          type="button"
          className="secondary danger"
          onClick={() =>
            onSheet({ kind: 'eventConfirm', eventRef: event.clientId, action: 'remove' })
          }
        >
          この{goal ? '得点' : '記録'}を取り消す
        </button>
      </div>
    </>
  )
}

function EventConfirmSheet({
  sheet,
  detail,
  onRemove,
  onMoveGoal,
}: {
  sheet: Extract<Sheet, { kind: 'eventConfirm' }>
  detail: MatchDetail
  onRemove: (eventRef: string) => void
  onMoveGoal: (eventRef: string, side: Side) => void
}) {
  const event = detail.events.find((candidate) => candidate.clientId === sheet.eventRef)
  if (event === undefined) {
    return <h2 id="sk-sheet-title">この記録は取り消されました</h2>
  }
  const other: Side = event.side === 'left' ? 'right' : 'left'
  // 操作の後のスコアを先に見せる(取り消しや付け替えで何点になるかを確かめてから押す)
  const after = applyOp(
    detail,
    sheet.action === 'remove'
      ? { type: 'event.remove', opId: event.clientId, eventRef: event.clientId }
      : { type: 'event.side', opId: event.clientId, eventRef: event.clientId, side: other },
  ).match
  const scoreAfter = `${teamLabel(detail, 'left')} ${after.left.score ?? 0} - ${after.right.score ?? 0} ${teamLabel(detail, 'right')}`
  const what = `${formatElapsed(event.elapsedSec)} ${eventText(detail, event)}`
  return (
    <>
      <h2 id="sk-sheet-title">
        {sheet.action === 'remove'
          ? `${what} を取り消しますか`
          : `${what} を ${teamLabel(detail, other)} の得点にしますか`}
      </h2>
      {sheet.action === 'side' ? (
        <p className="sk-hint">得点者とアシストは外れます。後から付け直せます。</p>
      ) : null}
      <p className="sk-hint">操作の後のスコア</p>
      <p className="sk-confirm-score">{scoreAfter}</p>
      <button
        type="button"
        className={`sk-big ${sheet.action === 'remove' ? 'danger-fill' : other}`}
        onClick={() =>
          sheet.action === 'remove' ? onRemove(event.clientId) : onMoveGoal(event.clientId, other)
        }
      >
        {sheet.action === 'remove' ? '取り消す' : `${teamLabel(detail, other)} の得点にする`}
      </button>
    </>
  )
}

function FinishSheet(props: SheetProps & { sheet: Extract<Sheet, { kind: 'finish' }> }) {
  const { sheet, detail, onSheet, onFinish } = props
  const left = detail.match.left.score ?? 0
  const right = detail.match.right.score ?? 0
  const confirmed = { left, right }
  const scoreLine = `${teamLabel(detail, 'left')} ${left} - ${right} ${teamLabel(detail, 'right')}`
  const header = (
    <>
      {sheet.notice === null ? null : (
        <p className="sk-banner error" role="alert">
          {sheet.notice}
        </p>
      )}
      <p className="sk-confirm-score">{scoreLine}</p>
      <p className="sk-hint">
        {sheet.checking
          ? '最新のスコアを確かめています…'
          : props.online
            ? '最新のスコアです。紙のスコアや審判と合っているか確かめてください。'
            : '通信が切れているため、この端末の記録で確かめています。'}
      </p>
      {props.longElapsed ? (
        <p className="sk-hint sk-warn">
          経過 {formatElapsed(props.elapsed ?? 0)}。試合時間(
          {detail.matchMinutes ?? DEFAULT_MATCH_MINUTES} 分)を大きく超えています。
        </p>
      ) : null}
    </>
  )
  if (left !== right) {
    return (
      <>
        <h2 id="sk-sheet-title">この点数で試合を終えますか</h2>
        {header}
        <button
          type="button"
          className="sk-big danger-fill"
          disabled={sheet.checking}
          onClick={() => onFinish(null, confirmed)}
        >
          試合終了
        </button>
      </>
    )
  }
  if (sheet.step === 'confirm') {
    // 同点の勝者は 1 タップで決めない(レビュー H2)。選んだ結果を言葉で確かめてから終える
    const result =
      sheet.winnerSide === null ? '引き分け' : `${teamLabel(detail, sheet.winnerSide)} の勝ち`
    return (
      <>
        <h2 id="sk-sheet-title">
          {result}(同点 {left}-{right})で試合を終えますか
        </h2>
        {header}
        <button
          type="button"
          className="sk-big danger-fill"
          disabled={sheet.checking}
          onClick={() => onFinish(sheet.winnerSide, confirmed)}
        >
          {result}で試合を終える
        </button>
        <button
          type="button"
          className="secondary sk-wide"
          onClick={() => onSheet({ ...sheet, step: 'pick', winnerSide: null })}
        >
          ← 選び直す
        </button>
      </>
    )
  }
  return (
    <>
      <h2 id="sk-sheet-title">同点です</h2>
      {header}
      <p className="sk-hint">
        {detail.inBracket
          ? 'トーナメントなので勝ったチームを選んでください(サドンデスなどの結果)。'
          : '引き分けで終えるか、勝ったチームを選んでください。'}
      </p>
      <div className="sk-pair">
        {sides.map((side) => (
          <button
            key={side}
            type="button"
            className={`sk-big ${side}`}
            disabled={sheet.checking}
            onClick={() => onSheet({ ...sheet, step: 'confirm', winnerSide: side })}
          >
            {teamLabel(detail, side)} の勝ち
          </button>
        ))}
      </div>
      {detail.inBracket ? null : (
        <button
          type="button"
          className="secondary sk-wide"
          disabled={sheet.checking}
          onClick={() => onSheet({ ...sheet, step: 'confirm', winnerSide: null })}
        >
          引き分けで終了
        </button>
      )}
    </>
  )
}

function ResultsSheet({
  detail,
  onSheet,
}: {
  detail: MatchDetail
  onSheet: (sheet: Sheet) => void
}) {
  const left = detail.match.left.score ?? 0
  const right = detail.match.right.score ?? 0
  const pick = (resultType: ExceptionalResult, winnerSide: Side | null = null) =>
    onSheet({ kind: 'confirm', resultType, winnerSide })
  return (
    <>
      <h2 id="sk-sheet-title">通常どおり終えられないとき</h2>
      <div className="sk-results">
        <button type="button" className="secondary" onClick={() => pick('abandoned')}>
          途中で打ち切り: このスコア({left}-{right})で確定
        </button>
        <button type="button" className="secondary" onClick={() => pick('unplayed')}>
          途中で打ち切り: 未消化にする
        </button>
        {sides.map((side) => (
          <button
            key={side}
            type="button"
            className="secondary"
            onClick={() => pick('forfeit', side)}
          >
            不戦勝: {teamLabel(detail, side)} の勝ち
          </button>
        ))}
        <button type="button" className="secondary" onClick={() => pick('double_forfeit')}>
          両チーム不戦敗
        </button>
        <button type="button" className="secondary" onClick={() => pick('no_contest')}>
          不成立(人数不足など)
        </button>
      </div>
    </>
  )
}

/** 結果の種類ごとに、勝者の指定が要るか(api の規則。docs/api.md「結果の種類のボディ」) */
function needsWinner(detail: MatchDetail, resultType: ExceptionalResult): boolean {
  if (resultType === 'forfeit' || resultType === 'no_contest') {
    return true
  }
  const tied = (detail.match.left.score ?? 0) === (detail.match.right.score ?? 0)
  return resultType === 'abandoned' && tied && detail.inBracket
}

function ConfirmSheet({
  sheet,
  detail,
  onSheet,
  onResult,
}: {
  sheet: Extract<Sheet, { kind: 'confirm' }>
  detail: MatchDetail
  onSheet: (sheet: Sheet) => void
  onResult: SheetProps['onResult']
}) {
  const { resultType, winnerSide } = sheet
  const missingWinner = needsWinner(detail, resultType) && winnerSide === null
  return (
    <>
      <h2 id="sk-sheet-title">{confirmTitle(detail, resultType, winnerSide)}</h2>
      <p className="sk-hint">{confirmDetail(detail, resultType)}</p>
      {needsWinner(detail, resultType) && resultType !== 'forfeit' ? (
        <WinnerPicker sheet={sheet} detail={detail} onSheet={onSheet} />
      ) : null}
      <button
        type="button"
        className="sk-big danger-fill"
        disabled={missingWinner}
        onClick={() => onResult(resultType, winnerSide)}
      >
        この結果で試合を終える
      </button>
    </>
  )
}

function WinnerPicker({
  sheet,
  detail,
  onSheet,
}: {
  sheet: Extract<Sheet, { kind: 'confirm' }>
  detail: MatchDetail
  onSheet: (sheet: Sheet) => void
}) {
  // 勝者を選ぶだけでは終えない(下の「この結果で試合を終える」で確定する)
  return (
    <div className="sk-pair">
      {sides.map((side) => (
        <button
          key={side}
          type="button"
          className={`sk-big ${side} ${sheet.winnerSide === side ? 'picked' : 'secondary'}`}
          aria-pressed={sheet.winnerSide === side}
          onClick={() => onSheet({ ...sheet, winnerSide: side })}
        >
          {teamLabel(detail, side)} の勝ち
        </button>
      ))}
    </div>
  )
}

function confirmTitle(
  detail: MatchDetail,
  resultType: ExceptionalResult,
  winnerSide: Side | null,
): string {
  const winner = winnerSide === null ? '' : teamLabel(detail, winnerSide)
  switch (resultType) {
    case 'forfeit':
      return `${winner} の不戦勝で終えますか`
    case 'double_forfeit':
      return '両チーム不戦敗で終えますか'
    case 'no_contest':
      return '不成立で終えますか'
    case 'abandoned':
      return '途中で打ち切って、このスコアで確定しますか'
    case 'unplayed':
      return '未消化として終えますか'
  }
}

function confirmDetail(detail: MatchDetail, resultType: ExceptionalResult): string {
  switch (resultType) {
    case 'forfeit':
      // 記録係は勝者だけを記録する。点数はその場で決めさせない(レビュー M7)
      return detail.forfeitScore === null
        ? '勝ったチームだけを記録します。スコアは大会の規定に従って本部が後で決めます。'
        : `スコアは大会の規定の ${detail.forfeitScore.winner}-${detail.forfeitScore.loser} で記録します。`
    case 'double_forfeit':
      return 'スコアは記録せず、両チームとも負けとして数えます。'
    case 'no_contest':
      return '人数不足などで試合が成り立たなかったとき。スコアは記録せず、勝ったチームだけを記録します。'
    case 'abandoned':
      return `${detail.match.left.score ?? 0}-${detail.match.right.score ?? 0} を最終スコアとして、通常の試合と同じに数えます。`
    case 'unplayed':
      return '勝敗と得点を成績に入れません。'
  }
}
