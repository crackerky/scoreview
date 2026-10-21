// 管理画面のセッション Cookie の決め事。Next.js に依存しない純粋な関数にしてテストする。

/** セッショントークンを入れる Cookie の名前 */
export const SESSION_COOKIE = 'sv_admin_session'

/** Cookie を送るパス。管理画面(と Server Actions の送信先)は /admin 以下にしかない */
export const SESSION_COOKIE_PATH = '/admin'

export type SessionCookieOptions = {
  httpOnly: true
  secure: boolean
  sameSite: 'lax'
  path: string
  expires: Date
}

/**
 * セッション Cookie の属性。
 * - httpOnly: ページのスクリプトから読めないようにする(XSS でトークンを盗ませない)
 * - secure: 本番は HTTPS だけで送る。開発(http://localhost)では付けない
 * - sameSite=lax: 他サイトからの POST に Cookie を付けない(CSRF 対策。Server Actions の Origin 検査と併用)
 * - expires: api のセッションの期限に揃える(期限後の Cookie を残さない)
 */
export function sessionCookieOptions(expiresAt: string, production: boolean): SessionCookieOptions {
  const expires = new Date(expiresAt)
  if (Number.isNaN(expires.getTime())) {
    throw new RangeError(`invalid expiresAt: ${expiresAt}`)
  }
  return { httpOnly: true, secure: production, sameSite: 'lax', path: SESSION_COOKIE_PATH, expires }
}

/**
 * ログインの後に戻る管理画面のパス。/admin 以下の、英数字と / - _ だけのパスに限る(他サイトや
 * ログイン画面そのものへ飛ばさない。オープンリダイレクトを作らない)。それ以外は /admin。
 */
export function safeNext(value: unknown): string {
  if (typeof value !== 'string' || !/^\/admin(\/[A-Za-z0-9/_-]*)?$/.test(value)) {
    return '/admin'
  }
  if (value.includes('//') || value.startsWith('/admin/login')) {
    return '/admin'
  }
  return value
}

/** ログイン画面のパス。next があればログインの後にそこへ戻る(記録画面でセッションが切れたとき) */
export function loginPath(next?: string): string {
  const back = safeNext(next)
  return back === '/admin' ? '/admin/login' : `/admin/login?${new URLSearchParams({ next: back })}`
}
