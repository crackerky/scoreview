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
