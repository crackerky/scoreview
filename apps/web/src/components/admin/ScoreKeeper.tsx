'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { z } from 'zod'
import { formatElapsed, sideName } from '@/admin/labels.utils'
import {
  classifyFailure,
  type ExceptionalResult,
  elapsedSeconds,
  newOpId,
  parseSavedQueue,
  project,
  resultTypeLabels,
  type SavedQueue,
  type ScoreOp,
  savedQueueKey,
  undoTarget,
} from '@/admin/scoreOps.utils'
import {
  type AdminEvent,
  type AdminPlayer,
  type ForfeitScore,
  type MatchDetail,
  matchDetailSchema,
} from '@/api/admin.types'
import type { Side } from '@/api/common.types'

// 記録係のスコア入力画面(docs/operations/operator-experience.md「記録係:間違えにくく、直しやすい入力」)。
// スマートフォン縦持ちの 1 画面に、スコアボード、左右の大きな +1、取り消し、試合終了を置く。
// 押した瞬間に端末へ記録し(見込みの状態を画面に出す)、api へは裏で順に送る。通信が切れている間の
// 操作は localStorage に残り、つながったら送る。各操作の opId で api 側が重複を捨てる。

const sides: Side[] = ['left', 'right']
// 送れなかった操作を送り直す間隔(ミリ秒)。会場の回線が戻るのを待つ。短すぎると電池を食う
const RETRY_INTERVAL_MS = 4000
// 経過時間の表示の更新間隔(ミリ秒)
const CLOCK_TICK_MS = 1000

const relayResponseSchema = z.object({ detail: matchDetailSchema })
const relayErrorSchema = z.object({ error: z.object({ code: z.string(), message: z.string() }) })

type Sheet =
  | { kind: 'players'; eventRef: string; step: 'scorer' | 'assist'; scorerId: number | null }
  | { kind: 'pull' }
  | { kind: 'finish' }
  | { kind: 'results' }
  | { kind: 'confirm'; resultType: ExceptionalResult; winnerSide: Side | null }
  | null

/** canScore は、この利用者がこの試合を記録できるか(記録係は割り当てが覆う試合だけ) */
type Props = { initial: MatchDetail; userId: number; isAdmin: boolean; canScore: boolean }

function teamLabel(detail: MatchDetail, side: Side): string {
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

function statusText(detail: MatchDetail): string {
  const { status, resultType } = detail.match
  if (status === 'scheduled') {
    return '試合前'
  }
  if (status === 'live') {
    return '試合中'
  }
  return resultType === 'normal' ? '終了' : `終了(${resultTypeLabels[resultType]})`
}

function eventText(detail: MatchDetail, event: AdminEvent): string {
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

function loadQueue(userId: number, matchId: number): SavedQueue {
  try {
    return parseSavedQueue(window.localStorage.getItem(savedQueueKey(userId, matchId)))
  } catch (error) {
    // プライベートブラウズ等で localStorage が使えない。溜めずに送るだけになる
    if (error instanceof DOMException) {
      return parseSavedQueue(null)
    }
    throw error
  }
}

export function ScoreKeeper({ initial, userId, isAdmin, canScore }: Props) {
  const matchId = initial.match.id
  const [base, setBase] = useState(initial)
  const [ops, setOps] = useState<ScoreOp[]>([])
  const opsRef = useRef<ScoreOp[]>([])
  const startedAtRef = useRef<number | null>(null)
  const [online, setOnline] = useState(true)
  const [authLost, setAuthLost] = useState(false)
  const [storageBroken, setStorageBroken] = useState(false)
  const [errors, setErrors] = useState<Array<{ id: string; message: string }>>([])
  const [sheet, setSheet] = useState<Sheet>(null)
  // 経過時間の表示に使う現在時刻。サーバーの描画と食い違わないよう、表示は端末で読み込んだ後から
  const [now, setNow] = useState<number | null>(null)
  const flushing = useRef(false)

  const view = useMemo(() => project(base, ops), [base, ops])

  const persist = useCallback(() => {
    const saved: SavedQueue = { version: 1, ops: opsRef.current, startedAt: startedAtRef.current }
    try {
      window.localStorage.setItem(savedQueueKey(userId, matchId), JSON.stringify(saved))
      setStorageBroken(false)
    } catch (error) {
      // 容量超過やプライベートブラウズ。画面で知らせ、送信は続ける
      if (error instanceof DOMException) {
        setStorageBroken(true)
        return
      }
      throw error
    }
  }, [userId, matchId])

  const setQueue = useCallback(
    (next: ScoreOp[]) => {
      opsRef.current = next
      setOps(next)
      persist()
    },
    [persist],
  )

  const refresh = useCallback(async () => {
    try {
      const response = await fetch(`/admin/api/matches/${matchId}`, { cache: 'no-store' })
      if (response.ok) {
        setBase(relayResponseSchema.parse(await response.json()).detail)
      }
    } catch (error) {
      // 取り直しは次の送信の応答でも行われるので、通信の失敗は待つだけにする
      if (!(error instanceof TypeError)) {
        throw error
      }
    }
  }, [matchId])

  const flush = useCallback(async () => {
    if (flushing.current) {
      return
    }
    flushing.current = true
    try {
      while (opsRef.current.length > 0) {
        const op = opsRef.current[0]
        if (op === undefined) {
          break
        }
        let response: Response
        try {
          response = await fetch(`/admin/api/matches/${matchId}/ops`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(op),
          })
        } catch (error) {
          // fetch の TypeError は通信の失敗(圏外など)。操作は残して後で送り直す
          if (error instanceof TypeError) {
            setOnline(false)
            return
          }
          throw error
        }
        setOnline(true)
        if (response.ok) {
          setBase(relayResponseSchema.parse(await response.json()).detail)
          setQueue(opsRef.current.filter((queued) => queued.opId !== op.opId))
          continue
        }
        const outcome = classifyFailure(response.status)
        if (outcome === 'retry') {
          return
        }
        if (outcome === 'login') {
          setAuthLost(true)
          return
        }
        const body = relayErrorSchema.safeParse(await response.json())
        const reason = body.success ? body.data.error.message : `HTTP ${response.status}`
        setErrors((current) => [
          ...current,
          { id: op.opId, message: failureMessage(response.status, reason) },
        ])
        setQueue(opsRef.current.filter((queued) => queued.opId !== op.opId))
        await refresh()
      }
    } finally {
      flushing.current = false
    }
  }, [matchId, refresh, setQueue])

  // 前回この端末に残った未送信の操作を読み、送る
  useEffect(() => {
    const saved = loadQueue(userId, matchId)
    opsRef.current = saved.ops
    startedAtRef.current = saved.startedAt
    setOps(saved.ops)
    setNow(Date.now())
    void flush()
  }, [userId, matchId, flush])

  useEffect(() => {
    const retry = window.setInterval(() => {
      if (opsRef.current.length > 0) {
        void flush()
      }
    }, RETRY_INTERVAL_MS)
    const onOnline = () => {
      setOnline(true)
      void flush()
    }
    const onOffline = () => setOnline(false)
    window.addEventListener('online', onOnline)
    window.addEventListener('offline', onOffline)
    return () => {
      window.clearInterval(retry)
      window.removeEventListener('online', onOnline)
      window.removeEventListener('offline', onOffline)
    }
  }, [flush])

  const live = view.match.status === 'live'
  useEffect(() => {
    if (!live) {
      return
    }
    const tick = window.setInterval(() => setNow(Date.now()), CLOCK_TICK_MS)
    return () => window.clearInterval(tick)
  }, [live])

  /** 経過 0 秒の時刻。この端末で記録を始めていればそれを、無ければ api の値を使う */
  const startedAt = (): number | null => {
    if (startedAtRef.current !== null) {
      return startedAtRef.current
    }
    return base.startedAt === null ? null : Date.parse(base.startedAt)
  }

  const enqueue = (op: ScoreOp) => {
    setQueue([...opsRef.current, op])
    void flush()
  }

  const opId = () => newOpId(window.crypto)

  const addEvent = (
    kind: 'goal' | 'timeout' | 'pull' | 'end',
    side: Side | null,
    winnerSide: Side | null = null,
  ): string => {
    const current = Date.now()
    if (startedAt() === null) {
      startedAtRef.current = current
    }
    const id = opId()
    enqueue({
      type: 'event.add',
      opId: id,
      kind,
      side,
      elapsedSec: elapsedSeconds(startedAt(), current),
      scorerId: null,
      assistId: null,
      callahan: false,
      winnerSide,
    })
    setNow(current)
    return id
  }

  const scoreGoal = (side: Side) => {
    const eventRef = addEvent('goal', side)
    if (pickable(view.rosters[side]).length > 0) {
      setSheet({ kind: 'players', eventRef, step: 'scorer', scorerId: null })
    }
  }

  const setPlayers = (eventRef: string, scorerId: number | null, assistId: number | null) => {
    enqueue({
      type: 'event.update',
      opId: opId(),
      eventRef,
      scorerId,
      assistId,
      callahan: false,
    })
    setSheet(null)
  }

  const undo = undoTarget(view, isAdmin)
  const runUndo = () => {
    if (undo === null) {
      return
    }
    if (undo.kind === 'reopen') {
      enqueue({ type: 'status', opId: opId(), status: 'live', winnerSide: null })
      return
    }
    enqueue({ type: 'event.remove', opId: opId(), eventRef: undo.event.clientId })
  }

  const finish = (winnerSide: Side | null) => {
    addEvent('end', null, winnerSide)
    setSheet(null)
  }

  const recordResult = (
    resultType: ExceptionalResult,
    winnerSide: Side | null,
    forfeitScore: ForfeitScore | null,
  ) => {
    enqueue({ type: 'result', opId: opId(), resultType, winnerSide, forfeitScore })
    setSheet(null)
  }

  const { match } = view
  const finished = match.status === 'finished'
  const notStarted = match.status === 'scheduled' && view.events.length === 0
  const elapsed = startedAt() === null || now === null ? null : elapsedSeconds(startedAt(), now)
  const recent = [...view.events].reverse()
  const where = [match.code, match.field ? `コート ${match.field}` : '', match.time ?? '']
    .filter((part) => part !== '')
    .join(' ・ ')

  return (
    <div className="sk">
      <header className="sk-top">
        <a href="/admin" className="sk-back">
          ← 担当試合
        </a>
        <span className="sk-where">{where}</span>
        <SyncBadge pending={ops.length} online={online} />
      </header>

      {online ? null : (
        <p className="sk-banner" role="status">
          通信が切れています。記録はこの端末に保存し、つながったら自動で送ります。
        </p>
      )}
      {canScore ? null : (
        <p className="sk-banner error" role="alert">
          この試合はあなたの担当ではないので、記録できません(見るだけです)。担当を本部に確認してください。
        </p>
      )}
      {authLost ? (
        <p className="sk-banner error" role="alert">
          ログインが切れました。未送信の記録はこの端末に残っています。
          <a href="/admin/login">ログインし直す</a>
        </p>
      ) : null}
      {storageBroken ? (
        <p className="sk-banner error" role="alert">
          この端末に記録を保存できません。通信が切れると記録が失われます。
        </p>
      ) : null}
      {errors.map((error) => (
        <p className="sk-banner error" role="alert" key={error.id}>
          {error.message}
          <button
            type="button"
            className="sk-close"
            onClick={() => setErrors((current) => current.filter((item) => item.id !== error.id))}
          >
            閉じる
          </button>
        </p>
      ))}

      <section className="sk-board" aria-live="polite" aria-label="スコア">
        {sides.map((side) => (
          <div key={side} className={`sk-team ${side}`}>
            <span className="sk-name">{teamLabel(view, side)}</span>
            <span className="sk-score">{match[side].score ?? '-'}</span>
          </div>
        ))}
        <div className="sk-state">
          <span className={`sk-chip ${match.status}`}>{statusText(view)}</span>
          {elapsed !== null && !finished ? (
            <span className="sk-clock">{formatElapsed(elapsed)}</span>
          ) : null}
        </div>
      </section>

      <main className="sk-main">
        {!canScore ? null : notStarted ? (
          <StartPanel
            detail={view}
            onPull={(side) => addEvent('pull', side)}
            onResult={(resultType, winnerSide) =>
              setSheet({ kind: 'confirm', resultType, winnerSide })
            }
          />
        ) : finished ? (
          <FinishedPanel detail={view} isAdmin={isAdmin} />
        ) : (
          <>
            <div className="sk-plus">
              {sides.map((side) => (
                <button
                  key={side}
                  type="button"
                  className={`sk-goal ${side}`}
                  disabled={view.match[side].team === null}
                  onClick={() => scoreGoal(side)}
                >
                  <span className="sk-goal-mark">+1</span>
                  <span className="sk-goal-team">{teamLabel(view, side)}</span>
                </button>
              ))}
            </div>
            <div className="sk-minor">
              <button
                type="button"
                className="secondary"
                onClick={() => addEvent('timeout', 'left')}
              >
                タイムアウト
                <br />
                {teamLabel(view, 'left')}
              </button>
              <button
                type="button"
                className="secondary"
                onClick={() => setSheet({ kind: 'pull' })}
              >
                プル
              </button>
              <button
                type="button"
                className="secondary"
                onClick={() => addEvent('timeout', 'right')}
              >
                タイムアウト
                <br />
                {teamLabel(view, 'right')}
              </button>
            </div>
          </>
        )}

        {recent.length > 0 ? (
          <ol className="sk-events" aria-label="得点経過(新しい順)">
            {recent.map((event) => (
              <li key={event.clientId}>
                <EventRow
                  detail={view}
                  event={event}
                  onPick={() =>
                    canScore &&
                    setSheet({
                      kind: 'players',
                      eventRef: event.clientId,
                      step: 'scorer',
                      scorerId: null,
                    })
                  }
                />
              </li>
            ))}
          </ol>
        ) : null}
      </main>

      <footer className="sk-bottom" hidden={!canScore}>
        <button
          type="button"
          className="sk-undo"
          disabled={undo === null}
          onClick={runUndo}
          aria-label={undo === null ? '取り消し(取り消せる記録がありません)' : undefined}
        >
          <span className="sk-undo-mark">↶ 取り消し</span>
          <span className="sk-undo-what">{undoText(view, undo, isAdmin)}</span>
        </button>
        {finished ? null : (
          <>
            {notStarted ? null : (
              <button
                type="button"
                className="secondary sk-other"
                onClick={() => setSheet({ kind: 'results' })}
              >
                打ち切り
                <br />
                など
              </button>
            )}
            {notStarted ? null : (
              <button
                type="button"
                className="sk-finish"
                onClick={() => setSheet({ kind: 'finish' })}
              >
                試合終了
              </button>
            )}
          </>
        )}
      </footer>

      {sheet === null ? null : (
        <SheetView
          sheet={sheet}
          detail={view}
          onClose={() => setSheet(null)}
          onSheet={setSheet}
          onPlayers={setPlayers}
          onPull={(side) => {
            addEvent('pull', side)
            setSheet(null)
          }}
          onFinish={finish}
          onResult={recordResult}
        />
      )}
    </div>
  )
}

/** 記録できなかった操作の知らせ。記録係がそのまま次の行動を決められる言葉にする */
function failureMessage(status: number, reason: string): string {
  switch (status) {
    case 403:
      return '記録できませんでした: この試合の担当ではありません。担当を本部に確認してください。'
    case 409:
      return '記録できませんでした: 試合の記録が別の端末で変わっています。画面を最新にしました。'
    case 404:
      return '記録できませんでした: 試合か得点経過が見つかりません(削除された可能性があります)。'
    default:
      return `記録できませんでした: 入力を確かめてください(${reason})。`
  }
}

function undoText(
  detail: MatchDetail,
  undo: ReturnType<typeof undoTarget>,
  isAdmin: boolean,
): string {
  if (undo === null) {
    return detail.match.status === 'finished' && !isAdmin ? '終了後の訂正は本部へ' : 'なし'
  }
  if (undo.kind === 'reopen') {
    return '試合終了を取り消す'
  }
  const { event } = undo
  const score = event.kind === 'goal' ? ` (${event.leftScore}-${event.rightScore})` : ''
  return `${eventText(detail, event)}${score}`
}

function SyncBadge({ pending, online }: { pending: number; online: boolean }) {
  if (pending === 0 && online) {
    return <span className="sk-sync ok">送信済み</span>
  }
  return (
    <span className={`sk-sync ${online ? 'wait' : 'offline'}`} role="status">
      {pending === 0 ? '通信なし' : `未送信 ${pending} 件`}
    </span>
  )
}

function EventRow({
  detail,
  event,
  onPick,
}: {
  detail: MatchDetail
  event: AdminEvent
  onPick: () => void
}) {
  const time = formatElapsed(event.elapsedSec)
  if (event.kind !== 'goal') {
    return (
      <div className="sk-event">
        <span className="sk-event-time">{time}</span>
        <span>{eventText(detail, event)}</span>
      </div>
    )
  }
  const who = event.scorer
    ? `${playerShort(event.scorer)}${event.assist ? ` ← ${playerShort(event.assist)}` : ''}`
    : null
  return (
    <button type="button" className={`sk-event goal ${event.side ?? ''}`} onClick={onPick}>
      <span className="sk-event-time">{time}</span>
      <span className="sk-event-score">
        {event.leftScore}-{event.rightScore}
      </span>
      <span className="sk-event-text">
        {eventText(detail, event)}
        {who === null ? <span className="sk-need">得点者を付ける</span> : ` ${who}`}
      </span>
    </button>
  )
}

function StartPanel({
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

function FinishedPanel({ detail, isAdmin }: { detail: MatchDetail; isAdmin: boolean }) {
  const { match } = detail
  return (
    <div className="sk-done">
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
    </div>
  )
}

type SheetProps = {
  sheet: Exclude<Sheet, null>
  detail: MatchDetail
  onClose: () => void
  onSheet: (sheet: Sheet) => void
  onPlayers: (eventRef: string, scorerId: number | null, assistId: number | null) => void
  onPull: (side: Side) => void
  onFinish: (winnerSide: Side | null) => void
  onResult: (
    resultType: ExceptionalResult,
    winnerSide: Side | null,
    forfeitScore: ForfeitScore | null,
  ) => void
}

function SheetView(props: SheetProps) {
  const { sheet, onClose } = props
  return (
    <div className="sk-overlay">
      <div className="sk-sheet" role="dialog" aria-modal="true" aria-labelledby="sk-sheet-title">
        <SheetBody {...props} sheet={sheet} />
        <button type="button" className="secondary sk-sheet-close" onClick={onClose}>
          {sheet.kind === 'players' ? 'あとで付ける' : 'やめる'}
        </button>
      </div>
    </div>
  )
}

function SheetBody({ sheet, detail, onSheet, onPlayers, onPull, onFinish, onResult }: SheetProps) {
  switch (sheet.kind) {
    case 'players':
      return <PlayersSheet sheet={sheet} detail={detail} onSheet={onSheet} onPlayers={onPlayers} />
    case 'pull':
      return (
        <>
          <h2 id="sk-sheet-title">どちらがプル(スロー)しますか</h2>
          <div className="sk-pair">
            {sides.map((side) => (
              <button
                key={side}
                type="button"
                className={`sk-big ${side}`}
                onClick={() => onPull(side)}
              >
                {teamLabel(detail, side)}
              </button>
            ))}
          </div>
        </>
      )
    case 'finish':
      return <FinishSheet detail={detail} onFinish={onFinish} />
    case 'results':
      return <ResultsSheet detail={detail} onSheet={onSheet} />
    case 'confirm':
      return <ConfirmSheet sheet={sheet} detail={detail} onSheet={onSheet} onResult={onResult} />
  }
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
  onPlayers: (eventRef: string, scorerId: number | null, assistId: number | null) => void
}) {
  const event = detail.events.find((candidate) => candidate.clientId === sheet.eventRef)
  if (event === undefined || event.side === null) {
    return <h2 id="sk-sheet-title">この得点は取り消されました</h2>
  }
  const roster = pickable(detail.rosters[event.side])
  const team = teamLabel(detail, event.side)
  if (sheet.step === 'scorer') {
    return (
      <>
        <h2 id="sk-sheet-title">
          {team} の得点({event.leftScore}-{event.rightScore})。得点したのは?
        </h2>
        <div className="sk-players">
          {roster.map((player) => (
            <button
              key={player.id}
              type="button"
              className="secondary"
              onClick={() => onSheet({ ...sheet, step: 'assist', scorerId: player.id })}
            >
              <span className="sk-num">{player.number ? `#${player.number}` : '-'}</span>
              <span className="sk-pname">{player.name.ja}</span>
            </button>
          ))}
        </div>
      </>
    )
  }
  return (
    <>
      <h2 id="sk-sheet-title">アシスト(パスを出した人)は?</h2>
      <div className="sk-players">
        <button
          type="button"
          className="secondary"
          onClick={() => onPlayers(sheet.eventRef, sheet.scorerId, null)}
        >
          <span className="sk-pname">なし</span>
        </button>
        {roster
          .filter((player) => player.id !== sheet.scorerId)
          .map((player) => (
            <button
              key={player.id}
              type="button"
              className="secondary"
              onClick={() => onPlayers(sheet.eventRef, sheet.scorerId, player.id)}
            >
              <span className="sk-num">{player.number ? `#${player.number}` : '-'}</span>
              <span className="sk-pname">{player.name.ja}</span>
            </button>
          ))}
      </div>
    </>
  )
}

function FinishSheet({
  detail,
  onFinish,
}: {
  detail: MatchDetail
  onFinish: (winnerSide: Side | null) => void
}) {
  const left = detail.match.left.score ?? 0
  const right = detail.match.right.score ?? 0
  const scoreLine = `${teamLabel(detail, 'left')} ${left} - ${right} ${teamLabel(detail, 'right')}`
  if (left !== right) {
    return (
      <>
        <h2 id="sk-sheet-title">この点数で試合を終えますか</h2>
        <p className="sk-confirm-score">{scoreLine}</p>
        <button type="button" className="sk-big danger-fill" onClick={() => onFinish(null)}>
          試合終了
        </button>
      </>
    )
  }
  return (
    <>
      <h2 id="sk-sheet-title">同点です</h2>
      <p className="sk-confirm-score">{scoreLine}</p>
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
            onClick={() => onFinish(side)}
          >
            {teamLabel(detail, side)} の勝ち
          </button>
        ))}
      </div>
      {detail.inBracket ? null : (
        <button type="button" className="secondary sk-wide" onClick={() => onFinish(null)}>
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
  const [winner, setWinner] = useState(detail.forfeitScore?.winner.toString() ?? '')
  const [loser, setLoser] = useState(detail.forfeitScore?.loser.toString() ?? '')
  const { resultType, winnerSide } = sheet
  const askForfeit = resultType === 'forfeit' && detail.forfeitScore === null
  const forfeit =
    resultType === 'forfeit' ? (detail.forfeitScore ?? parseForfeit(winner, loser)) : null
  const missingWinner = needsWinner(detail, resultType) && winnerSide === null
  const ready = !missingWinner && (resultType !== 'forfeit' || forfeit !== null)
  return (
    <>
      <h2 id="sk-sheet-title">{confirmTitle(detail, resultType, winnerSide)}</h2>
      <p className="sk-hint">{confirmDetail(detail, resultType, forfeit)}</p>
      {needsWinner(detail, resultType) && resultType !== 'forfeit' ? (
        <div className="sk-pair">
          {sides.map((side) => (
            <button
              key={side}
              type="button"
              className={`sk-big ${side} ${winnerSide === side ? 'picked' : 'secondary'}`}
              aria-pressed={winnerSide === side}
              onClick={() => onSheet({ ...sheet, winnerSide: side })}
            >
              {teamLabel(detail, side)} の勝ち
            </button>
          ))}
        </div>
      ) : null}
      {askForfeit ? (
        <div className="sk-forfeit">
          <p className="sk-hint">
            大会に不戦勝のスコアが設定されていません。大会の規程のスコアを入れてください。
          </p>
          <label>
            勝ったチーム
            <input
              type="number"
              inputMode="numeric"
              min={1}
              value={winner}
              onChange={(event) => setWinner(event.target.value)}
            />
          </label>
          <label>
            負けたチーム
            <input
              type="number"
              inputMode="numeric"
              min={0}
              value={loser}
              onChange={(event) => setLoser(event.target.value)}
            />
          </label>
        </div>
      ) : null}
      <button
        type="button"
        className="sk-big danger-fill"
        disabled={!ready}
        onClick={() => onResult(resultType, winnerSide, askForfeit ? forfeit : null)}
      >
        この結果で試合を終える
      </button>
    </>
  )
}

function parseForfeit(winner: string, loser: string): ForfeitScore | null {
  const win = Number(winner)
  const lose = Number(loser)
  if (winner === '' || loser === '' || !Number.isInteger(win) || !Number.isInteger(lose)) {
    return null
  }
  return lose >= 0 && win > lose ? { winner: win, loser: lose } : null
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

function confirmDetail(
  detail: MatchDetail,
  resultType: ExceptionalResult,
  forfeit: ForfeitScore | null,
): string {
  switch (resultType) {
    case 'forfeit':
      return forfeit === null
        ? '規定のスコアで記録します。'
        : `スコアは ${forfeit.winner}-${forfeit.loser} で記録します(大会の規定)。`
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
