import type { ReactNode } from 'react'
import type { TournamentFrame } from '@/api/common.types'
import { brand } from '@/brand/brand.config'
import type { Lang } from '@/i18n/lang'
import { ScoreNavi } from './ScoreNavi'
import { Sidebar } from './Sidebar'
import { SiteHeader } from './SiteHeader'

// clone-spec A 1.1 / 1.6: 全ページ共通の外枠。
// 大会ページ: 背景パターン、#score_navi、#sub あり。大会一覧: 黒背景、ナビとサイドバーなし。
// 参照サイトと同じく body の背景はページ内の <style> で切り替える。

type FrameVariant =
  | { kind: 'list' }
  | {
      kind: 'tournament'
      frame: TournamentFrame
      lang: Lang
      /** サイドバーに SCORE メニューを出すか(ディビジョン成績、グループ、チーム、試合、選手、ギャラリー) */
      scoreMenu?: boolean
    }

type SiteFrameProps = {
  variant: FrameVariant
  /**
   * #score_topmenu の中身。undefined なら div ごと出さない(大会一覧と大会トップ)。
   * null なら空の div を出す(チーム一覧・ギャラリーなど。clone-spec C 0)。
   */
  topMenu?: ReactNode
  /** フッターの著作権表記の年。省略時は表示時の西暦 */
  year?: number
  children: ReactNode
}

const bodyStyles = {
  list: 'body {background: var(--brand-list-body-bg);}',
  tournament: 'body {background: var(--brand-body-color) var(--brand-body-bg) repeat top;}',
} as const

export function SiteFrame({ variant, topMenu, year, children }: SiteFrameProps) {
  const copyrightYear = year ?? new Date().getFullYear()
  return (
    <>
      <style>{bodyStyles[variant.kind]}</style>
      <div id="top">
        <div id="wrapper">
          <div id="header">
            <SiteHeader />
            {variant.kind === 'tournament' && (
              <ScoreNavi frame={variant.frame} lang={variant.lang} />
            )}
          </div>
          {variant.kind === 'tournament' && (
            <Sidebar
              frame={variant.frame}
              lang={variant.lang}
              scoreMenu={variant.scoreMenu ?? false}
            />
          )}
          <div id="contents">
            <div id="main">
              <div id="score_parts">
                {topMenu !== undefined && <div id="score_topmenu">{topMenu}</div>}
                <div id="score_view">{children}</div>
              </div>
              <br />
            </div>
          </div>
        </div>
      </div>
      <div id="footer_ue" />
      <div id="footer">
        <div className="copyright">
          {`Copyright(c) ${copyrightYear} ${brand.copyrightHolder} All Rights Reserved.`}
        </div>
      </div>
    </>
  )
}
