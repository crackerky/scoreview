import type { ReactNode } from 'react'
import { ScoreShell } from '@/components/score/ScoreShell'
import { requireScoreBoard } from '@/score/score.load'

// スコアサイト(/tournaments/{id})の外枠。中のページと同じ掲示板を使う(1 回の描画で api は 1 回)。

/**
 * 描いた画面を観客のあいだで使い回す秒数(Next.js の revalidate。score.load.ts の
 * scoreRevalidateSeconds と同じ)。観客が何人いても、大会の画面はこの間隔で 1 回だけ描き直す。
 * 画面の描き直し(15 秒)より十分短くする。
 */
export const revalidate = 5

/** ビルドでは 1 つも描かない(api はビルド中に呼べない)。最初に開かれたときに描いて使い回す */
export function generateStaticParams(): Array<{ id: string }> {
  return []
}

export const viewport = { themeColor: '#05070b' }

export default async function ScoreLayout({
  children,
  params,
}: {
  children: ReactNode
  params: Promise<{ id: string }>
}) {
  const { board } = await requireScoreBoard((await params).id)
  return <ScoreShell board={board}>{children}</ScoreShell>
}
