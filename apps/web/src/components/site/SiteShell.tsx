import type { ReactNode } from 'react'
import { liveFontClass } from '@/live/fonts'
import '@/styles/site.css'

// サイトの入口(大会一覧 /、個人情報保護方針、見つからない画面)の外枠。観戦アプリと同じ暗い画面と
// 文字で、スマートフォンの幅に合わせる。

export function SiteShell({ children }: { children: ReactNode }) {
  return (
    <div className={`site ${liveFontClass}`}>
      <div className="site-screen">
        <header className="site-header">
          <a className="site-logo" href="/">
            SCORE VIEW
          </a>
        </header>
        <main className="site-main">{children}</main>
        <footer className="site-footer">
          <a href="/privacy">個人情報保護方針</a>
        </footer>
      </div>
    </div>
  )
}
