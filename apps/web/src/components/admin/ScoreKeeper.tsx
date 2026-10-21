'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { z } from 'zod'
import { formatElapsed } from '@/admin/labels.utils'
import {
  courtLabel,
  isLongElapsed,
  restoreOp,
  type ScorerState,
  scorerStateOf,
  UNDO_COOLDOWN_MS,
  undoAllowed,
} from '@/admin/scoreKeeper.utils'
import {
  type ConfirmedScore,
  type ExceptionalResult,
  elapsedSeconds,
  newOpId,
  project,
  type ScoreOp,
  undoTarget,
} from '@/admin/scoreOps.utils'
import { queueChangeEvent } from '@/admin/scoreQueue.utils'
import { claimScorer, releaseScorer, SCORER_HEARTBEAT_MS } from '@/admin/scorerClaim'
import {
  enqueueAndSend,
  pendingOps,
  deviceId as readDeviceId,
  requestSend,
  type ScoreDroppedDetail,
  type ScoreSentDetail,
  scoreAppliedEvent,
  scoreDroppedEvent,
  sendFinishedEvent,
  storageIsDurable,
} from '@/admin/scoreSender'
import { type MatchDetail, matchDetailSchema } from '@/api/admin.types'
import type { Side } from '@/api/common.types'
import {
  EventRow,
  eventText,
  FinishedPanel,
  type NextMatch,
  type Sheet,
  SheetView,
  StartPanel,
  statusText,
  teamLabel,
} from './ScoreSheets'

// 記録係のスコア入力画面(docs/operations/operator-experience.md「記録係:間違えにくく、直しやすい入力」)。
// スマートフォン縦持ちの 1 画面に、スコアボード、取り消し、試合終了と、親指の届く下側に左右の大きな
// +1 を置く。押した瞬間に端末へ記録し(見込みの状態を画面に出す)、api へは裏で順に送る。未送信の
// 操作は端末全体の待ち行列(scoreQueue.utils.ts)に残り、管理画面のどの画面からでも送られる。
// 同じ試合を別の端末でも開いているときは、記録担当(docs/api.md「試合の記録担当」)を引き継ぐまで
// 記録させない。他の端末の記録が見えるよう、画面を表示している間は数秒ごとに状態を取り直す。

const sides: Side[] = ['left', 'right']
// 状態を取り直す間隔(ミリ秒)。他の端末の得点が数秒で見えるように。画面が隠れている間は止める
const POLL_INTERVAL_MS = 3000
// 状態の取り直しを待つ上限(ミリ秒)。会場の回線が細いとき、確認シートをいつまでも待たせない
const REFRESH_TIMEOUT_MS = 4000
// 送れなかった操作を送り直す間隔(ミリ秒)。会場の回線が戻るのを待つ。短すぎると電池を食う
const RETRY_INTERVAL_MS = 4000
// 経過時間の表示の更新間隔(ミリ秒)
const CLOCK_TICK_MS = 1000
// 「取り消しました [元に戻す]」を出しておく時間(ミリ秒)
const UNDO_TOAST_MS = 8000

const relayResponseSchema = z.object({ detail: matchDetailSchema })

/** canScore は、この利用者がこの試合を記録できるか(記録係は割り当てが覆う試合だけ) */
type Props = {
  initial: MatchDetail
  isAdmin: boolean
  canScore: boolean
  userId: number
  nextMatch: NextMatch | null
}

type Toast = { id: string; text: string; restore: ScoreOp | null }
type Notice = { id: string; message: string }

export function ScoreKeeper({ initial, isAdmin, canScore, userId, nextMatch }: Props) {
  const matchId = initial.match.id
  const [base, setBase] = useState(initial)
  const [ops, setOps] = useState<ScoreOp[]>([])
  const [queueStartedAt, setQueueStartedAt] = useState<number | null>(null)
  const [online, setOnline] = useState(true)
  const [authLost, setAuthLost] = useState(false)
  const [storageBroken, setStorageBroken] = useState(false)
  const [errors, setErrors] = useState<Notice[]>([])
  const [sheet, setSheet] = useState<Sheet>(null)
  const [toast, setToast] = useState<Toast | null>(null)
  const [undoLockedUntil, setUndoLockedUntil] = useState(0)
  const [dismissed, setDismissed] = useState<string[]>([])
  const [newRows, setNewRows] = useState(false)
  const [myDevice, setMyDevice] = useState<string | null>(null)
  // 経過時間の表示に使う現在時刻。サーバーの描画と食い違わないよう、表示は端末で読み込んだ後から
  const [now, setNow] = useState<number | null>(null)
  const lastUndoAt = useRef<number | null>(null)
  const lastAppliedAt = useRef(0)
  const longAcknowledged = useRef(false)
  const leaving = useRef(false)
  const eventsRef = useRef<HTMLOListElement>(null)
  const eventCount = useRef(initial.events.length)

  const view = useMemo(() => project(base, ops), [base, ops])
  const scorer: ScorerState =
    myDevice === null || now === null
      ? { kind: 'mine' }
      : scorerStateOf(base.scorer, myDevice, userId, now)
  const locked = !canScore || scorer.kind === 'other'

  const readQueue = useCallback(() => {
    const saved = pendingOps(matchId)
    setOps(saved.ops)
    setQueueStartedAt(saved.startedAt)
  }, [matchId])

  const refresh = useCallback(async () => {
    const started = performance.now()
    let response: Response
    try {
      response = await fetch(`/admin/api/matches/${matchId}`, {
        cache: 'no-store',
        signal: AbortSignal.timeout(REFRESH_TIMEOUT_MS),
      })
    } catch (error) {
      // 圏外か、回線が細くて間に合わない。次の取り直しを待つ
      if (error instanceof TypeError || (error instanceof DOMException && isAbort(error))) {
        setOnline(false)
        return
      }
      throw error
    }
    setOnline(true)
    if (response.status === 401) {
      setAuthLost(true)
      return
    }
    if (!response.ok) {
      return
    }
    const { detail } = relayResponseSchema.parse(await response.json())
    // 取り直しを始めた後に操作の応答が届いていれば、そちらが新しい
    if (started >= lastAppliedAt.current) {
      setBase(detail)
    }
  }, [matchId])

  const claim = useCallback(
    async (takeover: boolean) => {
      if (!canScore || myDevice === null) {
        return
      }
      const result = await claimScorer(matchId, myDevice, takeover)
      if (result === 'login') {
        setAuthLost(true)
        return
      }
      if (result !== null) {
        setBase((current) => ({ ...current, scorer: result.scorer }))
        setNow(Date.now())
      }
    },
    [canScore, matchId, myDevice],
  )

  // 端末の ID と、前回この端末に残った未送信の操作を読み、送る
  useEffect(() => {
    setMyDevice(readDeviceId())
    setStorageBroken(!storageIsDurable())
    readQueue()
    setNow(Date.now())
    void requestSend()
  }, [readQueue])

  // 送信の結果(この画面からでも、端末全体の送信からでも)を聞いて描き直す
  useEffect(() => {
    const onQueue = () => readQueue()
    const onApplied = (event: Event) => {
      const sent = (event as CustomEvent<ScoreSentDetail>).detail // scoreSender が出す型
      if (sent.matchId === matchId) {
        lastAppliedAt.current = performance.now()
        setBase(sent.detail)
        setOnline(true)
      }
    }
    const onDropped = (event: Event) => {
      const { matchId: dropped, op, failure } = (event as CustomEvent<ScoreDroppedDetail>).detail // 同上
      if (dropped !== matchId) {
        return
      }
      if (failure.code === 'score_mismatch') {
        // 確認したスコアの後に別の端末で記録が入った。最新のスコアで確認し直させる
        void refresh().then(() =>
          setSheet({
            kind: 'finish',
            checking: false,
            step: 'pick',
            winnerSide: null,
            notice: `確認した後に別の端末で記録が入り、スコアが変わっていました。試合は終わっていません。最新のスコアで確認し直してください。`,
          }),
        )
        return
      }
      setErrors((current) => [
        ...current,
        { id: op.opId, message: failureMessage(failure.status, failure.message) },
      ])
      void refresh()
    }
    const onFinished = (event: Event) => {
      const result = (event as CustomEvent<string>).detail // scoreSender の SendResult
      if (result === 'offline') {
        setOnline(false)
      } else if (result === 'login') {
        setAuthLost(true)
      } else if (result === 'done') {
        setOnline(true)
      }
      setStorageBroken(!storageIsDurable())
    }
    const onStorage = (event: StorageEvent) => {
      // 別のタブが同じ端末の待ち行列を変えた
      if (event.key === null || event.key.startsWith('sv-score-queue')) {
        readQueue()
      }
    }
    window.addEventListener(queueChangeEvent, onQueue)
    window.addEventListener(scoreAppliedEvent, onApplied)
    window.addEventListener(scoreDroppedEvent, onDropped)
    window.addEventListener(sendFinishedEvent, onFinished)
    window.addEventListener('storage', onStorage)
    return () => {
      window.removeEventListener(queueChangeEvent, onQueue)
      window.removeEventListener(scoreAppliedEvent, onApplied)
      window.removeEventListener(scoreDroppedEvent, onDropped)
      window.removeEventListener(sendFinishedEvent, onFinished)
      window.removeEventListener('storage', onStorage)
    }
  }, [matchId, readQueue, refresh])

  // 表示している間だけ、状態の取り直し、未送信の送り直し、記録担当の延長を続ける
  useEffect(() => {
    let timers: number[] = []
    const start = () => {
      stop()
      void refresh()
      void claim(false)
      timers = [
        window.setInterval(() => void refresh(), POLL_INTERVAL_MS),
        window.setInterval(() => void requestSend(), RETRY_INTERVAL_MS),
        window.setInterval(() => void claim(false), SCORER_HEARTBEAT_MS),
      ]
    }
    const stop = () => {
      for (const timer of timers) {
        window.clearInterval(timer)
      }
      timers = []
    }
    const onVisibility = () => (document.visibilityState === 'visible' ? start() : stop())
    const onOnline = () => {
      setOnline(true)
      void requestSend()
      void refresh()
    }
    const onOffline = () => setOnline(false)
    onVisibility()
    document.addEventListener('visibilitychange', onVisibility)
    window.addEventListener('online', onOnline)
    window.addEventListener('offline', onOffline)
    return () => {
      stop()
      document.removeEventListener('visibilitychange', onVisibility)
      window.removeEventListener('online', onOnline)
      window.removeEventListener('offline', onOffline)
    }
  }, [refresh, claim])

  // 画面を閉じたら記録担当を外す(他の端末がすぐ引き継げるように)
  useEffect(() => {
    if (!canScore || myDevice === null) {
      return
    }
    const onHide = () => releaseScorer(matchId, myDevice)
    window.addEventListener('pagehide', onHide)
    return () => window.removeEventListener('pagehide', onHide)
  }, [canScore, matchId, myDevice])

  // 未送信の記録があるまま画面を離れるときは確かめる(送るのはどの管理画面からでもよいが、
  // ブラウザを閉じたりログアウトしたりすると送れなくなる)
  const pending = ops.length
  useEffect(() => {
    if (pending === 0) {
      return
    }
    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      if (!leaving.current) {
        event.preventDefault()
      }
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [pending])

  const live = view.match.status === 'live'
  useEffect(() => {
    if (!live) {
      return
    }
    const tick = window.setInterval(() => setNow(Date.now()), CLOCK_TICK_MS)
    return () => window.clearInterval(tick)
  }, [live])

  // 得点経過を下へスクロールしている間に新しい記録が足されたら、上に足されたことを知らせる
  useEffect(() => {
    const count = view.events.length
    if (count > eventCount.current && (eventsRef.current?.scrollTop ?? 0) > 24) {
      setNewRows(true)
    }
    eventCount.current = count
  }, [view.events.length])

  useEffect(() => {
    if (toast === null) {
      return
    }
    const timer = window.setTimeout(() => setToast(null), UNDO_TOAST_MS)
    return () => window.clearTimeout(timer)
  }, [toast])

  /** 経過 0 秒の時刻。api の値(全端末で同じ)を優先し、無ければこの端末で記録を始めた時刻 */
  const startedAt = base.startedAt === null ? queueStartedAt : Date.parse(base.startedAt)

  const enqueue = (op: ScoreOp, start: number | null = startedAt) => {
    if (!enqueueAndSend(matchId, op, start)) {
      setStorageBroken(true)
    }
    readQueue()
  }

  const opId = () => newOpId(window.crypto)

  const addEvent = (
    kind: 'goal' | 'timeout' | 'pull' | 'end',
    side: Side | null,
    winnerSide: Side | null = null,
    expectedScore: ConfirmedScore | null = null,
  ): string => {
    const current = Date.now()
    const start = startedAt ?? current
    const id = opId()
    enqueue(
      {
        type: 'event.add',
        opId: id,
        kind,
        side,
        elapsedSec: elapsedSeconds(start, current),
        scorerId: null,
        assistId: null,
        callahan: false,
        winnerSide,
        expectedScore,
      },
      start,
    )
    setNow(current)
    return id
  }

  const elapsed = startedAt === null || now === null ? null : elapsedSeconds(startedAt, now)
  const longElapsed = isLongElapsed(elapsed, view.matchMinutes)

  const recordGoal = (side: Side) => {
    const eventRef = addEvent('goal', side)
    if (view.rosters[side].some((player) => player.role === null)) {
      setSheet({ kind: 'players', eventRef, step: 'scorer', scorerId: null, repick: false })
    }
  }

  const scoreGoal = (side: Side) => {
    if (longElapsed && !longAcknowledged.current) {
      setSheet({ kind: 'long', side })
      return
    }
    recordGoal(side)
  }

  const setPlayers = (
    eventRef: string,
    scorerId: number | null,
    assistId: number | null,
    callahan: boolean,
  ) => {
    enqueue({ type: 'event.update', opId: opId(), eventRef, scorerId, assistId, callahan })
    setSheet(null)
  }

  /** 得点経過を 1 行取り消し、「取り消しました [元に戻す]」を出す */
  const removeEvent = (eventRef: string) => {
    const event = view.events.find((candidate) => candidate.clientId === eventRef)
    if (event === undefined) {
      return
    }
    const score = event.kind === 'goal' ? ` (${event.leftScore}-${event.rightScore})` : ''
    enqueue({ type: 'event.remove', opId: opId(), eventRef })
    setToast({
      id: opId(),
      text: `取り消しました: ${eventText(view, event)}${score}`,
      restore: restoreOp(event, opId()),
    })
  }

  const undo = undoTarget(view, isAdmin)
  const runUndo = () => {
    const current = Date.now()
    if (undo === null || !undoAllowed(lastUndoAt.current, current)) {
      return
    }
    lastUndoAt.current = current
    setUndoLockedUntil(current + UNDO_COOLDOWN_MS)
    window.setTimeout(() => setUndoLockedUntil(0), UNDO_COOLDOWN_MS)
    if (undo.kind === 'reopen') {
      enqueue({
        type: 'status',
        opId: opId(),
        status: 'live',
        winnerSide: null,
        expectedScore: null,
      })
      setToast({ id: opId(), text: '試合終了を取り消しました。記録を続けられます', restore: null })
      return
    }
    removeEvent(undo.event.clientId)
  }

  const openFinish = () => {
    setSheet({ kind: 'finish', checking: true, step: 'pick', winnerSide: null, notice: null })
    // 確認シートには最新のスコアを出す(他の端末の得点が入っていないかを確かめてから終える)
    void requestSend()
      .then(() => refresh())
      .finally(() =>
        setSheet((current) =>
          current?.kind === 'finish' ? { ...current, checking: false } : current,
        ),
      )
  }

  const finish = (winnerSide: Side | null, confirmed: ConfirmedScore) => {
    addEvent('end', null, winnerSide, confirmed)
    setSheet(null)
  }

  const recordResult = (resultType: ExceptionalResult, winnerSide: Side | null) => {
    // 途中打ち切りは画面のスコアで確定するので、そのスコアを確認したものとして送る
    const expectedScore =
      resultType === 'abandoned'
        ? { left: view.match.left.score ?? 0, right: view.match.right.score ?? 0 }
        : null
    // 不戦勝のスコアは送らない(大会の規定スコアか、無ければ本部が後で決める)
    enqueue({
      type: 'result',
      opId: opId(),
      resultType,
      winnerSide,
      forfeitScore: null,
      expectedScore,
    })
    setSheet(null)
  }

  const { match } = view
  const finished = match.status === 'finished'
  const notStarted = match.status === 'scheduled' && view.events.length === 0
  const recent = [...view.events].reverse()
  const where = [match.code, courtLabel(match.field), match.time ?? '']
    .filter((part) => part !== '')
    .join(' ・ ')
  const loginHref = `/admin/login?next=${encodeURIComponent(`/admin/matches/${matchId}/score`)}`
  const warnings = view.concurrentRecords.filter(
    (record) => !dismissed.includes(record.secondClientId),
  )
  const undoLocked = undoLockedUntil > 0

  return (
    <div className="sk">
      <header className="sk-top">
        <a
          href="/admin"
          className="sk-back"
          onClick={(event) => {
            if (pending > 0) {
              event.preventDefault()
              setSheet({ kind: 'leave', pending })
            }
          }}
        >
          ← 担当試合
        </a>
        <span className="sk-where">{where}</span>
        {canScore ? <SyncBadge pending={pending} online={online} /> : null}
      </header>

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
            <span className={`sk-clock${longElapsed ? ' long' : ''}`}>
              {formatElapsed(elapsed)}
            </span>
          ) : null}
        </div>
      </section>

      <main className="sk-main">
        {locked ? null : notStarted ? (
          <StartPanel
            detail={view}
            onPull={(side) => addEvent('pull', side)}
            onResult={(resultType, winnerSide) =>
              setSheet({ kind: 'confirm', resultType, winnerSide })
            }
          />
        ) : finished ? null : (
          <div className="sk-minor">
            <button type="button" className="secondary" onClick={() => addEvent('timeout', 'left')}>
              タイムアウト
              <br />
              {teamLabel(view, 'left')}
            </button>
            <button type="button" className="secondary" onClick={() => setSheet({ kind: 'pull' })}>
              プル
              <br />
              (投げた側)
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
        )}
        {finished ? <FinishedPanel detail={view} isAdmin={isAdmin} nextMatch={nextMatch} /> : null}

        <div className="sk-events-wrap">
          <div className="sk-banners">
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
            {scorer.kind === 'other' && canScore ? (
              <p className="sk-banner error" role="alert">
                {scorer.sameUser ? 'あなたが別の端末で' : `${scorer.name} さんが`}
                この試合を記録中です。同じ得点が二重に入らないよう、この端末では記録できません。
                <button type="button" className="sk-close" onClick={() => void claim(true)}>
                  引き継ぐ
                </button>
              </p>
            ) : null}
            {authLost ? (
              <p className="sk-banner error" role="alert">
                ログインが切れました。未送信の記録はこの端末に残っています。
                {/* 未送信の記録は端末に残り、ログインし直した後に送るので、離れる確認を出さない */}
                <a
                  href={loginHref}
                  onClick={() => {
                    leaving.current = true
                  }}
                >
                  ログインし直す
                </a>
              </p>
            ) : null}
            {storageBroken ? (
              <p className="sk-banner error" role="alert">
                この端末に記録を保存できません。通信が切れたまま画面を閉じると記録が失われます。
              </p>
            ) : null}
            {warnings.map((record) => (
              <p className="sk-banner error" role="alert" key={record.secondClientId}>
                二重記録の疑い: {record.kind === 'goal' ? '得点' : 'プル'}を {record.firstUser}{' '}
                さんと {record.secondUser} さんが {record.secondsApart}{' '}
                秒差で記録しました。同じものなら、得点経過の行を押して片方を取り消してください。
                <button
                  type="button"
                  className="sk-close"
                  onClick={() => setDismissed((current) => [...current, record.secondClientId])}
                >
                  閉じる
                </button>
              </p>
            ))}
            {errors.map((error) => (
              <p className="sk-banner error" role="alert" key={error.id}>
                {error.message}
                <button
                  type="button"
                  className="sk-close"
                  onClick={() =>
                    setErrors((current) => current.filter((item) => item.id !== error.id))
                  }
                >
                  閉じる
                </button>
              </p>
            ))}
            {toast === null ? null : (
              <p className="sk-banner sk-toast" role="status">
                {toast.text}
                {toast.restore === null ? null : (
                  <button
                    type="button"
                    className="sk-close"
                    onClick={() => {
                      if (toast.restore !== null) {
                        enqueue(toast.restore)
                      }
                      setToast(null)
                    }}
                  >
                    元に戻す
                  </button>
                )}
              </p>
            )}
          </div>
          {newRows ? (
            <button
              type="button"
              className="sk-newrows"
              onClick={() => {
                eventsRef.current?.scrollTo({ top: 0, behavior: 'smooth' })
                setNewRows(false)
              }}
            >
              ↑ 新しい記録
            </button>
          ) : null}
          {recent.length > 0 ? (
            <ol
              className="sk-events"
              aria-label="得点経過(新しい順)"
              ref={eventsRef}
              onScroll={(event) => {
                if (event.currentTarget.scrollTop <= 24) {
                  setNewRows(false)
                }
              }}
            >
              {recent.map((event) => (
                <li key={event.clientId}>
                  <EventRow
                    detail={view}
                    event={event}
                    canEdit={!locked && (!finished || event.kind === 'goal')}
                    onPick={() =>
                      setSheet(
                        finished
                          ? {
                              kind: 'players',
                              eventRef: event.clientId,
                              step: 'scorer',
                              scorerId: null,
                              repick: true,
                            }
                          : { kind: 'event', eventRef: event.clientId },
                      )
                    }
                  />
                </li>
              ))}
            </ol>
          ) : null}
        </div>

        {locked || notStarted || finished ? null : (
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
        )}
      </main>

      <footer className="sk-bottom" hidden={locked}>
        <button
          type="button"
          className="sk-undo"
          disabled={undo === null || undoLocked}
          onClick={runUndo}
          aria-label={undo === null ? '取り消し(取り消せる記録がありません)' : undefined}
        >
          <span className="sk-undo-mark">↶ 取り消し</span>
          <span className="sk-undo-what">{undoText(view, undo, isAdmin)}</span>
        </button>
        {finished || notStarted ? null : (
          <>
            <button
              type="button"
              className="secondary sk-other"
              onClick={() => setSheet({ kind: 'results' })}
            >
              打ち切り
              <br />
              ・不戦勝
            </button>
            <button type="button" className="sk-finish" onClick={openFinish}>
              試合終了
            </button>
          </>
        )}
      </footer>

      {sheet === null ? null : (
        <SheetView
          sheet={sheet}
          detail={view}
          elapsed={elapsed}
          longElapsed={longElapsed}
          online={online}
          onClose={() => setSheet(null)}
          onSheet={setSheet}
          onPlayers={setPlayers}
          onPull={(side) => {
            addEvent('pull', side)
            setSheet(null)
          }}
          onFinish={finish}
          onResult={recordResult}
          onRemove={(eventRef) => {
            removeEvent(eventRef)
            setSheet(null)
          }}
          onMoveGoal={(eventRef, side) => {
            enqueue({ type: 'event.side', opId: opId(), eventRef, side })
            setSheet(null)
          }}
          onLongGoal={(side) => {
            longAcknowledged.current = true
            setSheet(null)
            recordGoal(side)
          }}
          onLeave={() => {
            leaving.current = true
            window.location.href = '/admin'
          }}
        />
      )}
    </div>
  )
}

function isAbort(error: DOMException): boolean {
  return error.name === 'AbortError' || error.name === 'TimeoutError'
}

/** 記録できなかった操作の知らせ。記録係がそのまま次の行動を決められる言葉にする */
function failureMessage(status: number, reason: string): string {
  switch (status) {
    case 403:
      return '記録できませんでした: この試合の担当ではありません。担当を本部に確認してください。'
    case 409:
      return `記録できませんでした: 試合の記録が別の端末で変わっています(${reason})。画面を最新にしました。`
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
