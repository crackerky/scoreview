import 'server-only'
import { notFound } from 'next/navigation'
import { cache } from 'react'
import { ApiError } from '@/api/client'
import type { EntryLink } from '@/api/entry.types'
import { entryLinks } from '@/api/entryLink'
import { isLinkToken } from './entryLink.utils'

// 本人用のリンクの画面(/entry/{合言葉})のデータの読み方。ページはここだけを呼ぶ。

/**
 * 合言葉の画面に要るものを読む。形の違う合言葉、無いか期限の切れたリンクは notFound()(404。画面は
 * app/entry/[token]/not-found.tsx)。メタデータとページで 1 回だけ読むよう cache で包む。
 */
export const loadEntryLink = cache(async (token: string): Promise<EntryLink> => {
  if (!isLinkToken(token)) {
    notFound()
  }
  try {
    return await entryLinks.get(token)
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      notFound()
    }
    throw error
  }
})
