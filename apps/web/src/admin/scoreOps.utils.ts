import { z } from 'zod'
import {
  type AdminEvent,
  type AdminPlayer,
  forfeitScoreSchema,
  MAX_ELAPSED_SEC,
  type MatchDetail,
} from '@/api/admin.types'
import {
  type PlayerRef,
  type ResultType,
  resultTypeSchema,
  type Side,
  sideSchema,
} from '@/api/common.types'

// 記録係の画面の操作(得点、取り消し、得点者の付け直し、試合の終了、結果の種類)。
// 操作は端末に溜めてから api へ順に送る(通信が切れても記録を止めない。docs/operations/
// operator-experience.md「紙を上回ることが前提」2)。各操作には端末が作る UUID(opId)を付け、
// api は同じ opId の操作を 2 回適用しない(docs/api.md「操作 ID と冪等性」)。
// 画面には、api から最後に受け取った状態に未送信の操作を重ねた「見込みの状態」を出す。

const opId = z.uuid()
const nullableId = z.number().int().positive().nullable()

const eventAddSchema = z.object({
  type: z.literal('event.add'),
  opId,
  kind: z.enum(['goal', 'timeout', 'pull', 'end']),
  side: sideSchema.nullable(),
  elapsedSec: z.number().int().min(0).max(MAX_ELAPSED_SEC),
  scorerId: nullableId,
  assistId: nullableId,
  callahan: z.boolean(),
  winnerSide: sideSchema.nullable(),
})

/** 端末に溜めて api へ送る 1 回の操作。eventRef は得点経過の clientId(または api の id の文字列) */
export const scoreOpSchema = z.discriminatedUnion('type', [
  eventAddSchema,
  z.object({ type: z.literal('event.remove'), opId, eventRef: z.string().min(1) }),
  z.object({
    type: z.literal('event.update'),
    opId,
    eventRef: z.string().min(1),
    scorerId: nullableId,
    assistId: nullableId,
    callahan: z.boolean(),
  }),
  z.object({
    type: z.literal('status'),
    opId,
    status: z.enum(['live', 'finished']),
    winnerSide: sideSchema.nullable(),
  }),
  z.object({
    type: z.literal('result'),
    opId,
    resultType: resultTypeSchema.exclude(['normal']),
    winnerSide: sideSchema.nullable(),
    forfeitScore: forfeitScoreSchema.nullable(),
  }),
])
export type ScoreOp = z.infer<typeof scoreOpSchema>
export type ExceptionalResult = Exclude<ResultType, 'normal'>

/** 端末に保存する未送信の操作。startedAt は経過時間 0 秒の時刻(ミリ秒。端末の時計) */
export const savedQueueSchema = z.object({
  version: z.literal(1),
  ops: z.array(scoreOpSchema),
  startedAt: z.number().nullable(),
})
export type SavedQueue = z.infer<typeof savedQueueSchema>

/** localStorage の値を読む。壊れていたら(古い形式、手で書き換えた等)空にする */
export function parseSavedQueue(raw: string | null): SavedQueue {
  const empty: SavedQueue = { version: 1, ops: [], startedAt: null }
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
  const parsed = savedQueueSchema.safeParse(json)
  return parsed.success ? parsed.data : empty
}

export function savedQueueKey(matchId: number): string {
  return `sv-score-queue:${matchId}`
}

/** 得点経過の指し方(clientId)。api の id は持っていても、端末の操作は clientId で指す */
export function eventRefOf(event: AdminEvent): string {
  return event.clientId
}

function playerRef(roster: AdminPlayer[], id: number | null): PlayerRef | null {
  if (id === null) {
    return null
  }
  const player = roster.find((candidate) => candidate.id === id)
  if (player === undefined) {
    return null
  }
  return { id: player.id, number: player.number, name: player.name, gender: player.gender }
}

/** 得点経過を時系列順(経過秒、同じなら seq)に並べ、各行の累計スコアを付け直す */
function withRunningScores(events: AdminEvent[]): AdminEvent[] {
  let left = 0
  let right = 0
  return [...events]
    .sort((a, b) => a.elapsedSec - b.elapsedSec || a.seq - b.seq)
    .map((event) => {
      if (event.kind === 'goal' && event.side === 'left') {
        left++
      } else if (event.kind === 'goal' && event.side === 'right') {
        right++
      }
      return { ...event, leftScore: left, rightScore: right }
    })
}

function finalScore(events: AdminEvent[]): { left: number; right: number } {
  const last = events.at(-1)
  return { left: last?.leftScore ?? 0, right: last?.rightScore ?? 0 }
}

/** スコアと、スコアに差があれば勝者を、試合の左右に書き込む */
function withScore(
  detail: MatchDetail,
  score: { left: number; right: number } | null,
  patch: Partial<MatchDetail['match']>,
): MatchDetail {
  return {
    ...detail,
    match: {
      ...detail.match,
      ...patch,
      left: { ...detail.match.left, score: score?.left ?? null },
      right: { ...detail.match.right, score: score?.right ?? null },
    },
  }
}

function sameRef(event: AdminEvent, ref: string): boolean {
  return event.clientId === ref || String(event.id) === ref
}

/**
 * 操作 1 つを試合の状態に重ねた見込みの状態。api の規則(docs/api.md「試合」)と同じに動かすが、
 * 規則に反する操作(終了した試合への得点など)は画面で出さないので、ここでは検査しない。
 * 正しい状態は送信後に api の応答で置き換わる。
 */
export function applyOp(detail: MatchDetail, op: ScoreOp): MatchDetail {
  switch (op.type) {
    case 'event.add': {
      const seq = Math.max(0, ...detail.events.map((event) => event.seq)) + 1
      const roster = op.side === null ? [] : detail.rosters[op.side]
      const added: AdminEvent = {
        id: -seq,
        clientId: op.opId,
        seq,
        elapsedSec: op.elapsedSec,
        kind: op.kind,
        side: op.side,
        scorer: playerRef(roster, op.scorerId),
        assist: playerRef(roster, op.assistId),
        callahan: op.callahan,
        leftScore: 0,
        rightScore: 0,
      }
      const events = withRunningScores([...detail.events, added])
      const score = finalScore(events)
      const finished = op.kind === 'end'
      return withScore({ ...detail, events }, score, {
        status: finished ? 'finished' : 'live',
        winnerSide: finished && score.left === score.right ? op.winnerSide : null,
        resultType: 'normal',
      })
    }
    case 'event.remove': {
      const removed = detail.events.find((event) => sameRef(event, op.eventRef))
      const events = withRunningScores(
        detail.events.filter((event) => !sameRef(event, op.eventRef)),
      )
      const reopened = removed?.kind === 'end' && !events.some((event) => event.kind === 'end')
      return withScore({ ...detail, events }, finalScore(events), {
        status: reopened ? 'live' : detail.match.status,
        winnerSide: reopened ? null : detail.match.winnerSide,
      })
    }
    case 'event.update': {
      const events = detail.events.map((event) => {
        if (!sameRef(event, op.eventRef) || event.side === null) {
          return event
        }
        const roster = detail.rosters[event.side]
        return {
          ...event,
          scorer: playerRef(roster, op.scorerId),
          assist: playerRef(roster, op.assistId),
          callahan: op.callahan,
        }
      })
      return { ...detail, events }
    }
    case 'status': {
      const events =
        op.status === 'live' ? detail.events.filter((event) => event.kind !== 'end') : detail.events
      const score = finalScore(events)
      return withScore({ ...detail, events }, score, {
        status: op.status,
        winnerSide: op.status === 'finished' && score.left === score.right ? op.winnerSide : null,
        resultType: 'normal',
      })
    }
    case 'result':
      return applyResult(detail, op.resultType, op.winnerSide, op.forfeitScore)
  }
}

function applyResult(
  detail: MatchDetail,
  resultType: ExceptionalResult,
  winnerSide: Side | null,
  forfeit: { winner: number; loser: number } | null,
): MatchDetail {
  const patch = { status: 'finished' as const, resultType, winnerSide }
  switch (resultType) {
    case 'forfeit': {
      const score = forfeit ?? detail.forfeitScore ?? { winner: 0, loser: 0 }
      return withScore(
        detail,
        winnerSide === 'right'
          ? { left: score.loser, right: score.winner }
          : { left: score.winner, right: score.loser },
        patch,
      )
    }
    case 'double_forfeit':
    case 'no_contest':
      return withScore(detail, null, patch)
    case 'abandoned':
    case 'unplayed':
      return withScore(detail, finalScore(detail.events), patch)
  }
}

/** 未送信の操作をすべて重ねた見込みの状態 */
export function project(detail: MatchDetail, ops: readonly ScoreOp[]): MatchDetail {
  return ops.reduce(applyOp, detail)
}

/** 「取り消し」で戻す操作の候補。null なら戻せるものが無い */
export type UndoTarget =
  | { kind: 'event'; event: AdminEvent }
  | { kind: 'reopen' } // 終了の取り消し(管理者だけ)
  | null

/**
 * 「取り消し」が戻すもの。試合中は最後の得点経過(得点、タイムアウト、プル)を消す。
 * 終了した試合は、管理者なら終了を取り消して試合中へ戻し、記録係は戻せない(api の規則)。
 */
export function undoTarget(detail: MatchDetail, isAdmin: boolean): UndoTarget {
  if (detail.match.status === 'finished') {
    return isAdmin ? { kind: 'reopen' } : null
  }
  const last = detail.events.at(-1)
  return last === undefined ? null : { kind: 'event', event: last }
}

/**
 * 操作 ID(UUID v4)。crypto.randomUUID は安全なコンテキスト(HTTPS か localhost)でしか使えないので、
 * 会場の LAN から http で開いた場合は getRandomValues で同じ形を作る。
 */
export function newOpId(source: Pick<Crypto, 'getRandomValues'> & Partial<Crypto>): string {
  if (typeof source.randomUUID === 'function') {
    return source.randomUUID()
  }
  const bytes = source.getRandomValues(new Uint8Array(16))
  // RFC 9562 の version 4 と variant 10 のビットを立てる
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80
  const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

/**
 * 経過秒。startedAt(ミリ秒)が無ければ 0(最初の記録が経過 0 秒になる)。端末の時計が飛んでも
 * api の上限(MAX_ELAPSED_SEC)を超えないように抑える(超えると操作が送れず、保存した列も読めない)
 */
export function elapsedSeconds(startedAt: number | null, now: number): number {
  if (startedAt === null || now <= startedAt) {
    return 0
  }
  return Math.min(MAX_ELAPSED_SEC, Math.floor((now - startedAt) / 1000))
}

/** 結果の種類の文言(画面と印刷に出す。「状態」のような内部の言葉を出さない) */
export const resultTypeLabels: Record<ResultType, string> = {
  normal: '通常',
  forfeit: '不戦勝',
  double_forfeit: '両チーム不戦敗',
  no_contest: '不成立',
  abandoned: '途中打ち切り(その時点のスコアで確定)',
  unplayed: '未消化',
}

/** api への送信の失敗の分類。retry は後で送り直す、drop はこの操作をあきらめる、login はログインし直す */
export function classifyFailure(status: number | null): 'retry' | 'drop' | 'login' {
  if (status === null || status >= 500 || status === 408 || status === 429) {
    return 'retry'
  }
  if (status === 401) {
    return 'login'
  }
  return 'drop'
}
