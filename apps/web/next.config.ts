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
    return [{ source: '/(.*)', headers: securityHeaders }]
  },
  async redirects() {
    return [{ source: '/', destination: '/sctnmt', permanent: false }]
  },
}

export default nextConfig
