import type { ReactNode } from 'react'
import type { Gender } from '@/api/common.types'
import { playerHref } from '@/i18n/href'
import type { Lang } from '@/i18n/lang'
import { ScreenReaderNote } from './ScreenReaderNote'

// clone-spec D 1.1 / 2.3: 選手へのリンクは性別で色が変わる(男 playerLinkM、女 playerLinkF)。
// 性別のない古い大会は playerLink(2017 年の試合ページで確認)。

const genderClass: Record<Gender, string> = { M: 'playerLinkM', F: 'playerLinkF' }

// 本サイト独自: 性別は文字の色だけで示しているので、読み上げ用に補う(見た目は変えない)
const genderNotes: Record<Lang, Record<Gender, string>> = {
  jp: { M: '男性', F: '女性' },
  en: { M: 'male', F: 'female' },
}

export function playerLinkClass(gender: Gender | null): string {
  return gender === null ? 'playerLink' : genderClass[gender]
}

/** 選手ページ /pscore/scv_player/open/{id} へのリンク。中身の書式はページごとに違うので呼び出し側が渡す。 */
export function PlayerLink({
  id,
  gender,
  lang,
  children,
}: {
  id: number
  gender: Gender | null
  lang: Lang
  children: ReactNode
}) {
  return (
    <a className={playerLinkClass(gender)} href={playerHref(id, lang)}>
      {children}
      <ScreenReaderNote text={gender === null ? null : genderNotes[lang][gender]} />
    </a>
  )
}
