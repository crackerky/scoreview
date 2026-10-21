import { z } from 'zod'
import { parseSavedQueue, type ScoreOp, scoreOpSchema } from './scoreOps.utils'

// 記録係の未送信の操作を、試合ごとではなく端末全体で 1 つの localStorage の値に持つ
// (docs/operations/operator-experience.md「紙を上回ることが前提」2)。試合の記録画面を閉じても、
// 担当試合の一覧など管理画面のどの画面からでも全試合ぶんを送れるようにするため。
// Storage に依存する部分は KeyValueStore で受け、テストでは Map で代える。

/** localStorage のキー。版 2(端末全体)。版 1 は試合ごとのキー(sv-score-queue:{試合 ID}) */
export const deviceQueueKey = 'sv-score-queue'
const legacyPrefix = `${deviceQueueKey}:`

/** 端末の ID(試合の記録担当を端末ごとに区別する)を保存するキー */
export const deviceIdKey = 'sv-device-id'

/** 未送信の操作が変わったときに window へ出すイベントの名前(同じタブの画面を描き直すため) */
export const queueChangeEvent = 'sv-score-queue-change'

export type KeyValueStore = Pick<Storage, 'getItem' | 'setItem' | 'removeItem' | 'key' | 'length'>

const matchQueueSchema = z.object({
  ops: z.array(scoreOpSchema),
  /** 経過 0 秒の時刻(ミリ秒。端末の時計)。api の startedAt がまだ無いときだけ使う */
  startedAt: z.number().nullable(),
})
export type MatchQueue = z.infer<typeof matchQueueSchema>

const deviceQueueSchema = z.object({
  version: z.literal(2),
  /** キーは試合 ID の文字列 */
  matches: z.record(z.string().regex(/^[0-9]+$/), matchQueueSchema),
})
export type DeviceQueue = z.infer<typeof deviceQueueSchema>

const emptyMatch: MatchQueue = { ops: [], startedAt: null }

function parseDeviceQueue(raw: string | null): DeviceQueue {
  const empty: DeviceQueue = { version: 2, matches: {} }
  if (raw === null) {
    return empty
  }
  let json: unknown
  try {
    json = JSON.parse(raw)
  } catch (error) {
    // 保存した値が JSON でない(他のアプリが同じキーを書いた等)。溜まった操作は読めないので空にする
    if (error instanceof SyntaxError) {
      return empty
    }
    throw error
  }
  const parsed = deviceQueueSchema.safeParse(json)
  return parsed.success ? parsed.data : empty
}

/** 版 1 の試合ごとのキー(古い画面が残した未送信の操作)。キーと試合 ID */
function legacyKeys(store: KeyValueStore): Array<{ key: string; matchId: string }> {
  const found: Array<{ key: string; matchId: string }> = []
  for (let index = 0; index < store.length; index++) {
    const key = store.key(index)
    const matchId = key?.startsWith(legacyPrefix) ? key.slice(legacyPrefix.length) : ''
    if (key !== null && /^[0-9]+$/.test(matchId)) {
      found.push({ key, matchId })
    }
  }
  return found
}

/** 端末の未送信の操作を読む。版 1 の試合ごとのキーがあれば合わせて読む(書くときに版 2 へ移す) */
export function readDeviceQueue(store: KeyValueStore): DeviceQueue {
  const queue = parseDeviceQueue(store.getItem(deviceQueueKey))
  for (const { key, matchId } of legacyKeys(store)) {
    const legacy = parseSavedQueue(store.getItem(key))
    const current = queue.matches[matchId] ?? emptyMatch
    const known = new Set(current.ops.map((op) => op.opId))
    queue.matches[matchId] = {
      ops: [...legacy.ops.filter((op) => !known.has(op.opId)), ...current.ops],
      startedAt: current.startedAt ?? legacy.startedAt,
    }
  }
  return queue
}

/** 端末の未送信の操作を書く。操作の無い試合は消し、版 1 のキーも消す */
export function writeDeviceQueue(store: KeyValueStore, queue: DeviceQueue): void {
  const matches: DeviceQueue['matches'] = {}
  for (const [matchId, entry] of Object.entries(queue.matches)) {
    if (entry.ops.length > 0) {
      matches[matchId] = entry
    }
  }
  store.setItem(deviceQueueKey, JSON.stringify({ version: 2, matches }))
  for (const { key } of legacyKeys(store)) {
    store.removeItem(key)
  }
}

export function matchQueueOf(queue: DeviceQueue, matchId: number): MatchQueue {
  return queue.matches[String(matchId)] ?? emptyMatch
}

/** 操作を試合の末尾に足す。startedAt はその試合の経過 0 秒の時刻(まだ無ければ) */
export function enqueueOp(
  store: KeyValueStore,
  matchId: number,
  op: ScoreOp,
  startedAt: number | null,
): DeviceQueue {
  const queue = readDeviceQueue(store)
  const current = matchQueueOf(queue, matchId)
  queue.matches[String(matchId)] = {
    ops: [...current.ops, op],
    startedAt: current.startedAt ?? startedAt,
  }
  writeDeviceQueue(store, queue)
  return queue
}

/** 送り終えた(またはあきらめた)操作を取り除く。他の画面がその間に足した操作は残す */
export function removeOp(store: KeyValueStore, matchId: number, opId: string): DeviceQueue {
  const queue = readDeviceQueue(store)
  const current = matchQueueOf(queue, matchId)
  queue.matches[String(matchId)] = {
    ...current,
    ops: current.ops.filter((op) => op.opId !== opId),
  }
  writeDeviceQueue(store, queue)
  return queue
}

/** 試合ごとの未送信の操作の数(担当試合の一覧に出す)。0 の試合は含めない */
export function pendingCounts(queue: DeviceQueue): Map<number, number> {
  const counts = new Map<number, number>()
  for (const [matchId, entry] of Object.entries(queue.matches)) {
    if (entry.ops.length > 0) {
      counts.set(Number(matchId), entry.ops.length)
    }
  }
  return counts
}

/** 送る順の次の操作。試合の中は記録した順に送る(取り消しが得点の後に届くように) */
export function nextOp(queue: DeviceQueue): { matchId: number; op: ScoreOp } | null {
  for (const [matchId, entry] of Object.entries(queue.matches)) {
    const op = entry.ops[0]
    if (op !== undefined) {
      return { matchId: Number(matchId), op }
    }
  }
  return null
}

/** 端末に溜めた未送信の操作の数(全試合。版 1 の試合ごとのキーも含める) */
export function pendingTotal(store: KeyValueStore): number {
  let total = 0
  for (const count of pendingCounts(readDeviceQueue(store)).values()) {
    total += count
  }
  return total
}

/**
 * 端末の未送信の操作を全て消す(ログアウトのとき)。溜めた操作は利用者に結び付いていないので、
 * 残すと次にログインした人のセッションで送られる(セキュリティ点検 L-9)
 */
export function clearDeviceQueue(store: KeyValueStore): void {
  writeDeviceQueue(store, { version: 2, matches: {} })
}
