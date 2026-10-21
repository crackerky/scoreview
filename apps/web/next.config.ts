import type { NextConfig } from 'next'

// 参照サイトの自由記述 HTML とバナー画像は外部 URL を含むので、画像最適化は使わず <img> で出す
const securityHeaders = [
  { key: 'X-Content-Type-Options', value: 'nosniff' },
  { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
  { key: 'X-Frame-Options', value: 'DENY' },
  // 使わない端末の機能を、このサイトと埋め込まれたものに使わせない(セキュリティ点検 L-4)
  {
    key: 'Permissions-Policy',
    value:
      'camera=(), microphone=(), geolocation=(), payment=(), usb=(), serial=(), hid=(), ' +
      'bluetooth=(), midi=(), magnetometer=(), gyroscope=(), accelerometer=(), ' +
      'browsing-topics=()',
  },
  // HTTPS だけで開かせる。Vercel が独自ドメインに付ける既定と同じ 2 年(63072000 秒)を明示する。
  // includeSubDomains と preload は、同じドメインの他のサイトに影響するので付けない
  { key: 'Strict-Transport-Security', value: 'max-age=63072000' },
  // Content-Security-Policy はページごとに変える(管理画面は nonce)ので src/proxy.ts で付ける
]

const nextConfig: NextConfig = {
  poweredByHeader: false,
  async headers() {
    return [
      { source: '/(.*)', headers: securityHeaders },
      // 閲覧ページは毎回サーバーに確かめさせる(no-cache)が、no-store にはしない。no-store だと
      // ブラウザの戻る・進むで bfcache が効かず、スマートフォンでページを毎回取り直すため。
      // 管理画面(/admin)は個人の操作画面なので Next.js の既定(no-store)のまま。記録画面の
      // Service Worker(/sw.js)は次の項目で no-store にする(除外に加え、後の項目が優先される)
      {
        source: '/((?!admin|_next|sw\\.js$).*)',
        headers: [{ key: 'Cache-Control', value: 'no-cache' }],
      },
      // 記録画面の Service Worker(public/sw.js)。Next.js の PWA ガイドに従い、古い版を使い続けない
      // よう保存させず、読み込めるスクリプトを同じサイトに限る
      {
        source: '/sw.js',
        headers: [
          { key: 'Content-Type', value: 'application/javascript; charset=utf-8' },
          { key: 'Cache-Control', value: 'no-cache, no-store, must-revalidate' },
          { key: 'Content-Security-Policy', value: "default-src 'self'; script-src 'self'" },
        ],
      },
    ]
  },
  async redirects() {
    return [{ source: '/', destination: '/sctnmt', permanent: false }]
  },
}

export default nextConfig
