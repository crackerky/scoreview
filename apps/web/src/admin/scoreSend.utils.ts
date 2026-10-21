import { z } from 'zod'
import { type MatchDetail, matchDetailSchema } from '@/api/admin.types'
import { classifyFailure, type ScoreOp } from './scoreOps.utils'
import { type KeyValueStore, nextOp, readDeviceQueue, removeOp } from './scoreQueue.utils'

// 端末に溜まった全試合の未送信の操作を、試合の中は記録した順に 1 つずつ api(/admin/api の
// Route Handler)へ送る。送り先と保存先は引数で受け、テストでは偽物に代える。

const relayResponseSchema = z.object({ detail: matchDetailSchema })
const relayErrorSchema = z.object({ error: z.object({ code: z.string(), message: z.string() }) })

/** 送れずにあきらめた操作の理由(api の応答) */
export type SendFailure = { status: number; code: string; message: string }

export type SendDeps = {
  store: KeyValueStore
  /** 操作 1 つを送る。通信の失敗(圏外など)は TypeError を投げる(fetch と同じ) */
  post: (matchId: number, op: ScoreOp) => Promise<{ status: number; body: unknown }>
}

export type SendHandlers = {
  /** api が操作を受け付けた(detail は操作の後の試合) */
  applied: (matchId: number, op: ScoreOp, detail: MatchDetail) => void
  /** api が操作を断った(試合が終わっていた、スコアが食い違ったなど)。操作は捨てた */
  dropped: (matchId: number, op: ScoreOp, failure: SendFailure) => void
}

/**
 * 送り終えたら done、圏外なら offline、api が一時的に応えないなら retry、ログインが切れたら login。
 * offline、retry、login のときは送れなかった操作を残す(後で送り直す)。
 */
export type SendResult = 'done' | 'offline' | 'retry' | 'login'

export async function sendPending(deps: SendDeps, handlers: SendHandlers): Promise<SendResult> {
  for (;;) {
    const next = nextOp(readDeviceQueue(deps.store))
    if (next === null) {
      return 'done'
    }
    const { matchId, op } = next
    let response: { status: number; body: unknown }
    try {
      response = await deps.post(matchId, op)
    } catch (error) {
      // fetch の TypeError は通信の失敗(圏外など)。操作は残して後で送り直す
      if (error instanceof TypeError) {
        return 'offline'
      }
      throw error
    }
    if (response.status >= 200 && response.status < 300) {
      const { detail } = relayResponseSchema.parse(response.body)
      removeOp(deps.store, matchId, op.opId)
      handlers.applied(matchId, op, detail)
      continue
    }
    const outcome = classifyFailure(response.status)
    if (outcome !== 'drop') {
      return outcome
    }
    const body = relayErrorSchema.safeParse(response.body)
    removeOp(deps.store, matchId, op.opId)
    handlers.dropped(matchId, op, {
      status: response.status,
      code: body.success ? body.data.error.code : 'unknown',
      message: body.success ? body.data.error.message : `HTTP ${response.status}`,
    })
  }
}
