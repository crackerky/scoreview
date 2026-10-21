'use client'

import { type ScorerResult, scorerResultSchema } from '@/api/admin.types'

// 試合の記録担当(docs/api.md「試合の記録担当」)を /admin/api の Route Handler 経由で引き受ける・
// 延長する・引き継ぐ・外す。

/** 記録担当の延長の間隔(ミリ秒)。api の期限 30 秒より十分短くし、1 回取りこぼしても切れないようにする */
export const SCORER_HEARTBEAT_MS = 10_000

/** 引き受けか延長(takeover なら他の端末から引き継ぐ)。通信できなければ null、ログイン切れは 'login' */
export async function claimScorer(
  matchId: number,
  deviceId: string,
  takeover: boolean,
): Promise<ScorerResult | 'login' | null> {
  let response: Response
  try {
    response = await fetch(`/admin/api/matches/${matchId}/scorer`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ deviceId, takeover }),
    })
  } catch (error) {
    // 圏外。記録担当は分からないままにし、次の延長で確かめる
    if (error instanceof TypeError) {
      return null
    }
    throw error
  }
  if (response.status === 401) {
    return 'login'
  }
  if (!response.ok) {
    return null
  }
  return scorerResultSchema.parse(await response.json())
}

/** 記録画面を閉じるときに外す。ページを離れる間際でも届くよう keepalive で送り、結果は待たない */
export function releaseScorer(matchId: number, deviceId: string): void {
  void fetch(`/admin/api/matches/${matchId}/scorer`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ deviceId, release: true }),
    keepalive: true,
  }).catch((error: unknown) => {
    // 届かなくても api の期限(30 秒)で外れる
    if (!(error instanceof TypeError)) {
      throw error
    }
  })
}
