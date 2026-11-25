import { timingSafeEqual } from 'node:crypto'

// 定期の呼び出し(Vercel の schedules)の確かめ。Vercel は CRON_SECRET を Authorization: Bearer で付けて
// 呼ぶ。https://vercel.com/docs/schedules/handle-runs

/** CRON_SECRET が設定されていて、Authorization がそれと一致するか(長さを含めて時間差の出ない比べ方) */
export function isCronRequest(authorization: string | null, secret: string | undefined): boolean {
  if (!secret || authorization === null) {
    return false
  }
  const expected = Buffer.from(`Bearer ${secret}`)
  const actual = Buffer.from(authorization)
  return expected.length === actual.length && timingSafeEqual(expected, actual)
}
