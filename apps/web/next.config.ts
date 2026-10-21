import type { NextConfig } from 'next'

// 参照サイトの自由記述 HTML とバナー画像は外部 URL を含むので、画像最適化は使わず <img> で出す
const securityHeaders = [
  { key: 'X-Content-Type-Options', value: 'nosniff' },
  { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
  { key: 'X-Frame-Options', value: 'DENY' },
]

const nextConfig: NextConfig = {
  poweredByHeader: false,
  async headers() {
    return [
      { source: '/(.*)', headers: securityHeaders },
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
