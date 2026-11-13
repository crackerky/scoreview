import type { ReactNode } from 'react'
import type { BoardResponse } from '@/api/public.types'
import { liveFontClass } from '@/live/fonts'
import { scorePaths } from '@/score/score.paths'
import { AutoRefresh } from './AutoRefresh'
import { ScoreTabs } from './ScoreTabs'
import '@/styles/live-tables.css'
import '@/styles/score.css'

// スコアサイトの外枠。大会の名前と 3 つの入口(スコア、タイムスケジュール、トーナメント)をいつも
// 上に出し、入った瞬間に欲しいものへ 1 回で行けるようにする。

export function ScoreShell({ board, children }: { board: BoardResponse; children: ReactNode }) {
  const { tournament } = board
  return (
    <div className={`sc ${liveFontClass}`}>
      <header className="sc-header">
        <div className="sc-topbar">
          <a className="sc-logo" href="/">
            SCORE VIEW
          </a>
          {board.matches.length > 0 ? (
            <a className="sc-live-link" href={scorePaths.live(tournament.id)}>
              <span aria-hidden="true">●</span> ライブで見る
            </a>
          ) : null}
        </div>
        <h1 className="sc-title">{tournament.name.ja}</h1>
        <ScoreTabs tournamentId={tournament.id} />
      </header>
      <main className="sc-main">{children}</main>
      {tournament.banner ? (
        <aside className="sc-sponsor" aria-label="協賛">
          {tournament.banner.linkUrl ? (
            <a href={tournament.banner.linkUrl} rel="noopener" target="_blank">
              <img src={tournament.banner.imageUrl} alt="協賛" />
            </a>
          ) : (
            <img src={tournament.banner.imageUrl} alt="協賛" />
          )}
        </aside>
      ) : null}
      <footer className="sc-footer">
        <a href="/">大会の一覧</a>
        <a href="/privacy">個人情報保護方針</a>
      </footer>
      <AutoRefresh />
    </div>
  )
}
