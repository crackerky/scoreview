import type { ReactNode } from 'react'
import { SiteFrame } from './SiteFrame'
import { infoPageTitle } from './title.utils'

// 大会に属さないサイトの案内ページ(このサイトについて、見方ガイド、個人情報保護方針)の外枠。
// 参照サイトのヘッダーのリンクは運営会社のサイトへ出ていくが、本サイトはサイトの中に置く
// (docs/requirements.md「意図的な差分」)。外枠は大会一覧と同じ(黒背景、ナビとサイドバーなし)で、
// 見出しは参照サイトの .text_title と .TeamListMIDA の見た目を使う。日本語だけで書く。

type InfoPageProps = {
  title: string
  children: ReactNode
}

export function InfoPage({ title, children }: InfoPageProps) {
  return (
    <>
      <title>{infoPageTitle(title)}</title>
      <SiteFrame variant={{ kind: 'list' }}>
        <div className="text_title">{title}</div>
        <div className="sv_doc">{children}</div>
      </SiteFrame>
    </>
  )
}

/** 案内ページの節。id はヘッダーのリンクなどから #id で飛ぶ先 */
export function InfoSection({
  id,
  heading,
  children,
}: {
  id?: string
  heading: string
  children: ReactNode
}) {
  return (
    <section id={id}>
      <h2 className="TeamListMIDA">{heading}</h2>
      {children}
    </section>
  )
}
