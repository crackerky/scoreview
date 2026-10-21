import type { ReactNode } from 'react'
import '@/brand/brand.css'
import '@/styles/legacy.css'
import '@/styles/print.css'
import '@/styles/site-notes.css'
import '@/styles/mobile.css'

// <title> は各ページが <title> 要素で出す(clone-spec A 1.7 の規則。api を 1 回だけ呼ぶため
// generateMetadata は使わない)。ここで metadata.title を決めると <title> が重複する。

export default function RootLayout({ children }: { children: ReactNode }) {
  // 参照サイトは lang=en でも <html lang="ja"> のまま。ここも ja 固定にする
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  )
}
