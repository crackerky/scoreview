'use client'

import { usePathname, useSearchParams } from 'next/navigation'
import { parseLang } from '@/i18n/lang'
import { langSwitchHref } from './langSwitch.utils'

// ヘッダーバー右端の言語切替「English / 日本語」(本サイト独自。docs/requirements.md「意図的な差分」)。
// 参照サイトの同じ位置には運営会社の英語ページへのリンク "English" があり、スコアサイト内の言語は
// URL の ?lang= を手で書き換えるしかなかった(clone-spec A 1.2)。今のページの lang だけを切り替え、
// クエリの他の値(日程の絞り込みなど)は保つ。表示中の言語はリンクにしない。

export function LangSwitch() {
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const current = parseLang(searchParams.get('lang') ?? undefined)
  const query = searchParams.toString()
  return (
    <>
      {current === 'en' ? (
        <span lang="en" aria-current="true">
          English
        </span>
      ) : (
        <a lang="en" hrefLang="en" href={langSwitchHref(pathname, query, 'en')}>
          English
        </a>
      )}
      {' / '}
      {current === 'jp' ? (
        <span lang="ja" aria-current="true">
          日本語
        </span>
      ) : (
        <a lang="ja" hrefLang="ja" href={langSwitchHref(pathname, query, 'jp')}>
          日本語
        </a>
      )}
    </>
  )
}
