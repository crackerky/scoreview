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
    return [{ source: '/(.*)', headers: securityHeaders }]
  },
  async redirects() {
    return [{ source: '/', destination: '/sctnmt', permanent: false }]
  },
}

export default nextConfig
