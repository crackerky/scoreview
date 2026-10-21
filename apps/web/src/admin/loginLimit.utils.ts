import { isIP } from 'node:net'

// ログインの IP 単位の試行制限(web 側)。Next.js に依存しない純粋なクラスと関数にしてテストする。
//
// 数えるのは関数のインスタンスの中だけ(Vercel では複数のインスタンスに分かれる)。DB で数える api の
// IP 単位の制限(15 分に 30 回の失敗。docs/api.md「認証」)と併用し、web では api に送る前の安い
// 関門にする(セキュリティ点検 M-5)。

/** 失敗を数える窓(ミリ秒)。api の LoginFailureWindow と同じ 15 分 */
export const LOGIN_FAILURE_WINDOW_MS = 15 * 60 * 1000

/**
 * 窓の中で 1 つの IP に許す失敗の数。会場の Wi-Fi で記録係が IP を共有しても、20 回の失敗には
 * 届かない。api の上限(30 回)より手前で止め、総当たりを api まで届かせない
 */
export const MAX_LOGIN_FAILURES_PER_IP = 20

/** 覚えておく IP の数の上限。多数の IP から送られてもメモリを使い切らない(古いものから忘れる) */
export const MAX_TRACKED_IPS = 10_000

/**
 * 利用者の IP。Vercel では x-forwarded-for を Vercel が利用者の IP で上書きする(偽装を防ぐため)ので
 * 先頭を使う。手元(next dev / next start)では Next.js のサーバーが接続元のアドレスを
 * x-forwarded-for に入れる(クライアントが送っていればそれを残すが、手元は開発用なので問わない)。
 * IP として読めなければ空文字(制限を掛けない。api も同じ扱い)。
 */
export function clientIpFrom(headers: Headers): string {
  const first = headers.get('x-forwarded-for')?.split(',')[0]?.trim() ?? ''
  return isIP(first) === 0 ? '' : first
}

/** IP ごとのログインの失敗を窓の中で数える */
export class LoginFailureLimiter {
  private readonly failures = new Map<string, number[]>()

  constructor(
    private readonly limit = MAX_LOGIN_FAILURES_PER_IP,
    private readonly windowMs = LOGIN_FAILURE_WINDOW_MS,
    private readonly maxTracked = MAX_TRACKED_IPS,
  ) {}

  /** ip の窓の中の失敗が上限未満なら true。空文字(IP が分からない)は数えない */
  allows(ip: string, now: number): boolean {
    if (ip === '') {
      return true
    }
    return this.recent(ip, now).length < this.limit
  }

  /** ip の失敗を 1 回記録する */
  recordFailure(ip: string, now: number): void {
    if (ip === '') {
      return
    }
    const times = [...this.recent(ip, now), now]
    // 入れ直して Map の末尾(最近)に回し、上限を超えたら先頭(最も古く使った IP)から忘れる
    this.failures.delete(ip)
    this.failures.set(ip, times)
    while (this.failures.size > this.maxTracked) {
      const oldest = this.failures.keys().next()
      if (oldest.done === true) {
        break
      }
      this.failures.delete(oldest.value)
    }
  }

  private recent(ip: string, now: number): number[] {
    const times = (this.failures.get(ip) ?? []).filter((at) => at > now - this.windowMs)
    if (times.length === 0) {
      this.failures.delete(ip)
    }
    return times
  }
}
