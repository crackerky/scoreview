import type { ReactNode } from 'react'

// <title> は各ページが <title> 要素で出す(api を 1 回だけ呼ぶため generateMetadata は使わない)。
// ここで metadata.title を決めると <title> が重複する。見た目は外枠ごとの CSS(site.css、live.css、
// admin.css)が決めるので、ここでは全体に効く CSS を読まない。

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="ja">
      <body style={{ margin: 0 }}>{children}</body>
    </html>
  )
}
