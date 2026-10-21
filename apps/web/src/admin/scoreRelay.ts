import 'server-only'
import { ZodError } from 'zod'
import { matches } from '@/api/admin'
import type { MatchDetail } from '@/api/admin.types'
import { ApiError } from '@/api/client'
import { describeZodError } from './form.utils'
import { isSameOriginJson, parseMatchId } from './relay.utils'
import type { ScoreOp } from './scoreOps.utils'
import { scoreOpSchema } from './scoreOps.utils'
import { readSessionToken } from './session'

// 記録係の画面(Client Component)から呼ぶ Route Handler の中身。ブラウザは api を直接呼べない
// (api は公開しない。scoreview-design 7)ので、/admin 以下の Route Handler が Cookie のセッションを
// 確かめてから api へ中継する。応答は { detail } か { error: { code, message } }。

export function errorResponse(status: number, code: string, message: string): Response {
  return Response.json({ error: { code, message } }, { status })
}

function applyToApi(token: string, matchId: number, op: ScoreOp): Promise<MatchDetail> {
  switch (op.type) {
    case 'event.add':
      return matches.addEvent(token, matchId, {
        clientId: op.opId,
        kind: op.kind,
        side: op.side,
        elapsedSec: op.elapsedSec,
        scorerId: op.scorerId,
        assistId: op.assistId,
        callahan: op.callahan,
        winnerSide: op.winnerSide,
      })
    case 'event.remove':
      return matches.removeEventByRef(token, matchId, op.eventRef, op.opId)
    case 'event.update':
      return matches.updateEvent(token, matchId, op.eventRef, {
        clientId: op.opId,
        scorerId: op.scorerId,
        assistId: op.assistId,
        callahan: op.callahan,
      })
    case 'status':
      return matches.setStatus(token, matchId, {
        clientId: op.opId,
        status: op.status,
        winnerSide: op.winnerSide,
      })
    case 'result':
      return matches.setResult(token, matchId, {
        clientId: op.opId,
        resultType: op.resultType,
        winnerSide: op.winnerSide,
        forfeitScore: op.forfeitScore,
      })
  }
}

/** api の失敗を同じステータスの応答にする。api に届かなかった失敗は 502 にする(端末が送り直す) */
function relayFailure(error: unknown): Response {
  if (error instanceof ApiError) {
    return errorResponse(error.status, error.code, error.message)
  }
  if (error instanceof ZodError) {
    return errorResponse(502, 'bad_gateway', describeZodError(error))
  }
  if (error instanceof TypeError) {
    // fetch が api に届かなかった(api の再起動中など)
    return errorResponse(502, 'bad_gateway', error.message)
  }
  throw error
}

/** GET: 試合のスコア入力画面のデータ(最新の状態に取り直すため) */
export async function relayDetail(matchIdText: string): Promise<Response> {
  const matchId = parseMatchId(matchIdText)
  if (matchId === null) {
    return errorResponse(404, 'not_found', 'MATCH_ID')
  }
  const token = await readSessionToken()
  if (token === null) {
    return errorResponse(401, 'unauthorized', 'login required')
  }
  try {
    return Response.json({ detail: await matches.detail(token, matchId) })
  } catch (error) {
    return relayFailure(error)
  }
}

/** POST: 操作 1 つを api へ送る */
export async function relayOp(request: Request, matchIdText: string): Promise<Response> {
  if (!isSameOriginJson(request)) {
    return errorResponse(403, 'forbidden', 'cross-site or non-JSON request')
  }
  const matchId = parseMatchId(matchIdText)
  if (matchId === null) {
    return errorResponse(404, 'not_found', 'MATCH_ID')
  }
  const token = await readSessionToken()
  if (token === null) {
    return errorResponse(401, 'unauthorized', 'login required')
  }
  let body: unknown
  try {
    body = await request.json()
  } catch (error) {
    if (error instanceof SyntaxError) {
      return errorResponse(400, 'bad_request', 'body must be JSON')
    }
    throw error
  }
  const parsed = scoreOpSchema.safeParse(body)
  if (!parsed.success) {
    return errorResponse(400, 'bad_request', describeZodError(parsed.error))
  }
  try {
    return Response.json({ detail: await applyToApi(token, matchId, parsed.data) })
  } catch (error) {
    return relayFailure(error)
  }
}
