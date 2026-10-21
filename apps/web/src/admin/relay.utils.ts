// 記録係の画面の Route Handler(scoreRelay.ts)の入力の検査。Next.js に依存しない純粋な関数にしてテストする。

/**
 * 同じサイトからの JSON の POST だけを受ける(他サイトのフォームやスクリプトからの送信を断る)。
 * Origin を、リクエストの Host ヘッダーではなく設定で決めたサイトのオリジン(allowedOrigins。
 * siteOrigins が作る)と比べる。Host はリクエストの送り手が決める値なので基準にしない
 * (セキュリティ点検 I-4)。
 */
export function isSameOriginJson(request: Request, allowedOrigins: readonly string[]): boolean {
  const origin = request.headers.get('origin')
  if (origin === null) {
    return false
  }
  const json = request.headers.get('content-type')?.startsWith('application/json') ?? false
  return json && allowedOrigins.includes(originOf(origin) ?? '')
}

/** URL のオリジン(scheme://host:port)。URL の形でなければ null */
function originOf(url: string): string | null {
  try {
    const { origin } = new URL(url)
    // file: などオリジンを持たない URL は "null" になる
    return origin === 'null' ? null : origin
  } catch (error) {
    // URL の形でない(壊れたクライアント、書き換えた設定)。どのオリジンとも一致させない
    if (error instanceof TypeError) {
      return null
    }
    throw error
  }
}

/**
 * 手元の開発で PUBLIC_SITE_URL が無いときのサイトの URL。next dev の既定のポート。
 * 公開環境では PUBLIC_SITE_URL を必ず設定する(未設定でも起動は止めない。手元で設定なしに動かすため)。
 */
export const DEV_SITE_URL = 'http://localhost:3000'

/**
 * 管理画面の Route Handler が受け付けるオリジン。PUBLIC_SITE_URL(公開 URL。未設定なら
 * DEV_SITE_URL)と、Vercel ではデプロイごとの URL(VERCEL_URL。プレビューを開いたときのため。
 * Vercel が設定する値で、リクエストからは変えられない)。
 */
export function siteOrigins(env: {
  PUBLIC_SITE_URL?: string | undefined
  VERCEL_URL?: string | undefined
}): string[] {
  const origins = [originOf(env.PUBLIC_SITE_URL || DEV_SITE_URL)]
  if (env.VERCEL_URL) {
    origins.push(originOf(`https://${env.VERCEL_URL}`))
  }
  return origins.filter((origin): origin is string => origin !== null)
}

/** パスの試合 ID。数字でなければ null */
export function parseMatchId(value: string): number | null {
  return /^[0-9]+$/.test(value) ? Number(value) : null
}
