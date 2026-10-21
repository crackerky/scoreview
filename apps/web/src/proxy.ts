import { type NextRequest, NextResponse } from 'next/server'
import { adminCsp, browseCsp, isAdminPath } from '@/security/csp.utils'

// ページの応答に Content-Security-Policy を付ける(src/security/csp.utils.ts。セキュリティ点検 L-4)。
// 管理画面は要求ごとの nonce を作り、要求ヘッダーの CSP と x-nonce で Next.js に渡す(Next.js が
// 自分のスクリプトに nonce を付ける)。

export function proxy(request: NextRequest) {
  const dev = process.env.NODE_ENV === 'development'
  if (!isAdminPath(request.nextUrl.pathname)) {
    const response = NextResponse.next()
    response.headers.set('Content-Security-Policy', browseCsp(dev))
    return response
  }
  const nonce = Buffer.from(crypto.randomUUID()).toString('base64')
  const policy = adminCsp(nonce, dev)
  const requestHeaders = new Headers(request.headers)
  requestHeaders.set('x-nonce', nonce)
  requestHeaders.set('Content-Security-Policy', policy)
  const response = NextResponse.next({ request: { headers: requestHeaders } })
  response.headers.set('Content-Security-Policy', policy)
  return response
}

export const config = {
  matcher: [
    {
      // 静的ファイルと画像には付けない(HTML のページだけ)。先読み(prefetch)の要求は nonce を作らない。
      // Service Worker(/sw.js)は next.config.ts で専用の CSP を付ける
      source: '/((?!_next/static|_next/image|favicon.ico|brand/|sw\\.js).*)',
      missing: [
        { type: 'header', key: 'next-router-prefetch' },
        { type: 'header', key: 'purpose', value: 'prefetch' },
      ],
    },
  ],
}
