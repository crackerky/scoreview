import 'server-only'
import { notFound } from 'next/navigation'
import { cache } from 'react'
import { parseMatchId as parsePathId } from '@/admin/relay.utils'
import { ApiError } from '@/api/client'
import { getBoard } from '@/api/public'
import type { BoardResponse } from '@/api/public.types'
import type { LiveData } from '@/live/live.types'
import { buildLiveData } from '@/live/live.utils'

// スコアサイトのデータの読み方。ページとレイアウトはここだけを呼び、api の呼び方を知らない。

/**
 * 掲示板を観客のあいだで使い回す秒数。ページの revalidate(app/tournaments/[id]/layout.tsx)と同じに
 * する。Next.js は revalidate に定数の式しか読まないので、そちらは数字で書いている
 */
export const scoreRevalidateSeconds = 5

export type ScoreBoard = {
  board: BoardResponse
  /** 観戦アプリと同じ形の試合と組(表の部品がこの形を使う) */
  data: LiveData
}

/**
 * 大会の掲示板。大会が無ければ null。1 回の描画の中ではレイアウトとページで同じ結果を使う
 * (React の cache。api を 2 回呼ばない)。
 */
export const loadScoreBoard = cache(async (tournamentId: number): Promise<ScoreBoard | null> => {
  try {
    const board = await getBoard(tournamentId, { revalidateSeconds: scoreRevalidateSeconds })
    return { board, data: buildLiveData(board, null) }
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return null
    }
    throw error
  }
})

/** パスの大会 ID から掲示板を読む。ID が数字でないか大会が無ければ「見つからない」画面にする */
export async function requireScoreBoard(rawId: string): Promise<ScoreBoard> {
  const id = parsePathId(rawId)
  const score = id === null ? null : await loadScoreBoard(id)
  if (!score) {
    notFound()
  }
  return score
}
