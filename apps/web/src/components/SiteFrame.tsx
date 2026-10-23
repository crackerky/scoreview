import type { ReactNode } from 'react'
import type { TournamentFrame } from '@/api/common.types'
import { brand } from '@/brand/brand.config'
import { formatAsOf } from '@/format/format.utils'
import { type Lang, pickText } from '@/i18n/lang'
import { ScoreNavi } from './ScoreNavi'
import { Sidebar } from './Sidebar'
import { SiteHeader } from './SiteHeader'
import { suspensionText } from './suspension.utils'

// clone-spec A 1.1 / 1.6: 全ページ共通の外枠。
// 大会ページ: 背景パターン、#score_navi、#sub あり。大会一覧: 黒背景、ナビとサイドバーなし。
// 参照サイトと同じく body の背景はページ内の <style> で切り替える。
// 本サイト独自: 外側の #header、#main、#footer は div ではなく header / main / footer 要素にし
// (ランドマーク)、#main の先頭に視覚的に隠した h1 を置く。CSS は id で当たるので見た目は同じ。

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
  /**
   * 本文の上に「HH:MM 時点」を小さく出す(本サイト独自。速報の新しさを示す)。日程、グループ、チーム、
   * 試合のページで渡す。省略時は出さない
   */
  asOf?: Date
  /**
   * 視覚的に隠した h1 の、大会名の後ろに付ける文字(「チーム一覧」やチーム名)。支援技術でページの
   * 内容が分かるようにする。大会一覧では渡さない(サイト名の一覧の見出しになる)。案内ページ
   * (InfoPage)では、大会名の代わりにページの題名そのものになる
   */
  heading?: string
  children: ReactNode
}

const bodyStyles = {
  list: 'body {background: var(--brand-list-body-bg);}',
  tournament: 'body {background: var(--brand-body-color) var(--brand-body-bg) repeat top;}',
} as const

/** 視覚的に隠した h1 の文字。英語名の無い大会は日本語名を使う(見出しを空にしない)。 */
function pageHeading(variant: FrameVariant, heading: string | undefined): string {
  if (variant.kind === 'list') {
    return heading === undefined || heading === '' ? brand.tournamentListTitle : heading
  }
  const name = pickText(variant.lang, variant.frame.name) || variant.frame.name.ja
  return heading === undefined || heading === '' ? name : `${name} ${heading}`
}

export function SiteFrame({ variant, topMenu, year, asOf, heading, children }: SiteFrameProps) {
  const copyrightYear = year ?? new Date().getFullYear()
  return (
    <>
      <style>{bodyStyles[variant.kind]}</style>
      <div id="top">
        <div id="wrapper">
          <header id="header">
            <SiteHeader />
            {variant.kind === 'tournament' && (
              <ScoreNavi frame={variant.frame} lang={variant.lang} />
            )}
          </header>
          {variant.kind === 'tournament' && (
            <Sidebar
              frame={variant.frame}
              lang={variant.lang}
              scoreMenu={variant.scoreMenu ?? false}
            />
          )}
          <div id="contents">
            <main id="main">
              <h1 className="sv_sr" data-scoreview-addition="">
                {pageHeading(variant, heading)}
              </h1>
              <div id="score_parts">
                {topMenu !== undefined && <div id="score_topmenu">{topMenu}</div>}
                <div id="score_view">
                  {variant.kind === 'tournament' && variant.frame.suspension !== null && (
                    <p className="sv_suspension" role="status" data-scoreview-addition="">
                      {suspensionText(variant.frame.suspension, variant.lang)}
                    </p>
                  )}
                  {asOf !== undefined && variant.kind === 'tournament' && (
                    <p className="sv_as_of" data-scoreview-addition="">
                      {formatAsOf(asOf, variant.lang)}
                    </p>
                  )}
                  {children}
                </div>
              </div>
              <br />
            </main>
          </div>
        </div>
      </div>
      <div id="footer_ue" />
      <footer id="footer">
        <div className="copyright">
          {`Copyright(c) ${copyrightYear} ${brand.copyrightHolder} All Rights Reserved.`}
        </div>
      </footer>
    </>
  )
}
