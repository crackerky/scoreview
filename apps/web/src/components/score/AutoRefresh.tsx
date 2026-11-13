'use client'

import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

// 開いている画面を一定の間隔で描き直す(点と順位を新しくする)。画面が見えていないあいだは止め、
// 戻ってきたらすぐ描き直す。描き直しはサーバーの描画の使い回し(ページの revalidate)から返るので、
// 観客が増えても api を読む回数は増えない。

/** 描き直す間隔(ミリ秒)。観戦アプリの取り直しと同じ 15 秒 */
const refreshMs = 15_000

export function AutoRefresh() {
  const router = useRouter()
  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === 'visible') {
        router.refresh()
      }
    }
    const timer = window.setInterval(refresh, refreshMs)
    document.addEventListener('visibilitychange', refresh)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', refresh)
    }
  }, [router])
  return null
}
