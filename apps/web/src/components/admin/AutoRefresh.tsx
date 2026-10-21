'use client'

import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

// 本部の進行画面を一定の間隔で取り直す(レビュー M3)。タブが見えていない間は止め、見えたらすぐ
// 取り直す。入力欄を操作している間は取り直さない(入力中の値や開いた選択肢を消さないため)。

/** 取り直す間隔(ミリ秒)。結果の入力は 1 試合 20 分の枠で数件ずつ来るので 30 秒で足りる */
const refreshIntervalMs = 30_000

function editing(): boolean {
  const active = document.activeElement
  return (
    active instanceof HTMLInputElement ||
    active instanceof HTMLSelectElement ||
    active instanceof HTMLTextAreaElement
  )
}

export function AutoRefresh({ updatedAt }: { updatedAt: string }) {
  const router = useRouter()
  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === 'visible' && !editing()) {
        router.refresh()
      }
    }
    const timer = window.setInterval(refresh, refreshIntervalMs)
    document.addEventListener('visibilitychange', refresh)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', refresh)
    }
  }, [router])
  return (
    <span className="hq-updated" role="status">
      {updatedAt}(30 秒ごとに自動で更新)
    </span>
  )
}
