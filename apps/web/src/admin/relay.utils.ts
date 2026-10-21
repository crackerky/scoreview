// 記録係の画面の Route Handler(scoreRelay.ts)の入力の検査。Next.js に依存しない純粋な関数にしてテストする。

/** 同じサイトからの JSON の POST だけを受ける(他サイトのフォームやスクリプトからの送信を断る) */
export function isSameOriginJson(request: Request): boolean {
  const origin = request.headers.get('origin')
  const host = request.headers.get('host')
  if (origin === null || host === null) {
    return false
  }
  let originHost: string
  try {
    originHost = new URL(origin).host
  } catch (error) {
    // Origin ヘッダーが URL の形でない(壊れたクライアント)。同じサイトとは見なさない
    if (error instanceof TypeError) {
      return false
    }
    throw error
  }
  const json = request.headers.get('content-type')?.startsWith('application/json') ?? false
  return originHost === host && json
}

/** パスの試合 ID。数字でなければ null */
export function parseMatchId(value: string): number | null {
  return /^[0-9]+$/.test(value) ? Number(value) : null
}
