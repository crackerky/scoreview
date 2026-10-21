import type { Metadata } from 'next'
import type { ReactNode } from 'react'
import '@/brand/brand.css'
import '@/styles/legacy.css'

export const metadata: Metadata = {
  title: 'scoreview',
}

export default function RootLayout({ children }: { children: ReactNode }) {
  // 参照サイトは lang=en でも <html lang="ja"> のまま。ここも ja 固定にする
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  )
}
