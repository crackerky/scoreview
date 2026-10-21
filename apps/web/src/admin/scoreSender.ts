'use client'

import type { MatchDetail } from '@/api/admin.types'
import { newOpId, type ScoreOp } from './scoreOps.utils'
import {
  deviceIdKey,
  deviceQueueKey,
  enqueueOp,
  type KeyValueStore,
  queueChangeEvent,
  readDeviceQueue,
} from './scoreQueue.utils'
import { type SendFailure, type SendResult, sendPending } from './scoreSend.utils'

// ブラウザで端末全体の未送信の操作を送る(scoreSend.utils.ts の sendPending を fetch と localStorage
// でつなぐ)。記録画面と担当試合の一覧など、管理画面のどの画面からでも呼べる。結果は window の
// イベントで知らせ、画面はそれを聞いて描き直す(送った画面と表示している画面が別でもよいように)。

/** api が操作を受け付けたときのイベント。detail は ScoreSentDetail */
export const scoreAppliedEvent = 'sv-score-applied'
/** api が操作を断ったときのイベント。detail は ScoreDroppedDetail */
export const scoreDroppedEvent = 'sv-score-dropped'
/** 送信を終えたときのイベント。detail は SendResult */
export const sendFinishedEvent = 'sv-score-send-finished'

export type ScoreSentDetail = { matchId: number; op: ScoreOp; detail: MatchDetail }
export type ScoreDroppedDetail = { matchId: number; op: ScoreOp; failure: SendFailure }

// 同じタブの複数の画面(記録画面と端末全体の送信)が同時に送らないようにする
let running: Promise<SendResult> | null = null
let again = false
// localStorage に書けなかった(プライベートブラウズ、容量超過)ときの代わり。このタブを開いている間だけ残る
let fallback: KeyValueStore | null = null

/** メモリに置く KeyValueStore(localStorage が使えないときの代わり) */
export function memoryStore(): KeyValueStore {
  const values = new Map<string, string>()
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => {
      values.set(key, value)
    },
    removeItem: (key) => {
      values.delete(key)
    },
    key: (index) => [...values.keys()][index] ?? null,
    get length() {
      return values.size
    },
  }
}

/** 端末の保存先。localStorage が使えなければメモリ(画面を閉じると消える) */
export function browserStore(): KeyValueStore {
  if (fallback !== null) {
    return fallback
  }
  try {
    return window.localStorage
  } catch (error) {
    if (error instanceof DOMException) {
      fallback = memoryStore()
      return fallback
    }
    throw error
  }
}

/** 未送信の操作をこの端末に保存できているか(false なら画面を閉じると失われる) */
export function storageIsDurable(): boolean {
  return browserStore() !== fallback
}

function emit(name: string, detail: unknown): void {
  window.dispatchEvent(new CustomEvent(name, { detail }))
}

async function post(matchId: number, op: ScoreOp): Promise<{ status: number; body: unknown }> {
  const response = await fetch(`/admin/api/matches/${matchId}/ops`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(op),
  })
  let body: unknown = null
  try {
    body = await response.json()
  } catch (error) {
    // 本文が JSON でない(プロキシのエラー画面など)。状態コードだけで判断する
    if (!(error instanceof SyntaxError)) {
      throw error
    }
  }
  return { status: response.status, body }
}

async function sendOnce(store: KeyValueStore): Promise<SendResult> {
  const run = () =>
    sendPending(
      { store, post },
      {
        applied: (matchId, op, detail) => {
          emit(queueChangeEvent, null)
          emit(scoreAppliedEvent, { matchId, op, detail } satisfies ScoreSentDetail)
        },
        dropped: (matchId, op, failure) => {
          emit(queueChangeEvent, null)
          emit(scoreDroppedEvent, { matchId, op, failure } satisfies ScoreDroppedDetail)
        },
      },
    )
  // 別のタブも同じ端末の操作を送るので、Web Locks があればタブをまたいで 1 つずつにする
  if (typeof navigator !== 'undefined' && navigator.locks !== undefined) {
    return navigator.locks.request('sv-score-send', run)
  }
  return run()
}

/**
 * 端末の未送信の操作を全試合ぶん送る。送っている最中に呼ばれたら、終わった後にもう 1 回送る
 * (その間に足された操作を取りこぼさない)。
 */
export function requestSend(): Promise<SendResult> {
  const store = browserStore()
  if (running !== null) {
    again = true
    return running
  }
  running = (async () => {
    let result: SendResult
    do {
      again = false
      result = await sendOnce(store)
    } while (again && result === 'done')
    return result
  })().finally(() => {
    running = null
  })
  void running.then((result) => emit(sendFinishedEvent, result))
  return running
}

/** 操作を端末に溜めて送る。端末に保存できなければ(容量超過)メモリに溜め、false を返す */
export function enqueueAndSend(matchId: number, op: ScoreOp, startedAt: number | null): boolean {
  const store = browserStore()
  try {
    enqueueOp(store, matchId, op, startedAt)
  } catch (error) {
    if (!(error instanceof DOMException)) {
      throw error
    }
    // 容量超過。溜まっている操作ごとメモリへ移して続ける(画面で知らせる)
    const memory = memoryStore()
    memory.setItem(deviceQueueKey, JSON.stringify(readDeviceQueue(store)))
    fallback = memory
    enqueueOp(memory, matchId, op, startedAt)
  }
  emit(queueChangeEvent, null)
  void requestSend()
  return storageIsDurable()
}

/** 試合の未送信の操作(画面に重ねて見込みの状態を出すため) */
export function pendingOps(matchId: number): { ops: ScoreOp[]; startedAt: number | null } {
  return readDeviceQueue(browserStore()).matches[String(matchId)] ?? { ops: [], startedAt: null }
}

/** この端末の ID(試合の記録担当を端末ごとに区別する)。無ければ作って保存する */
export function deviceId(): string {
  const store = browserStore()
  const saved = store.getItem(deviceIdKey)
  if (saved !== null) {
    return saved
  }
  const created = newOpId(window.crypto)
  try {
    store.setItem(deviceIdKey, created)
  } catch (error) {
    // 保存できなければ、この画面を開いている間だけの ID になる(開き直すと別の端末に見える)
    if (!(error instanceof DOMException)) {
      throw error
    }
  }
  return created
}
