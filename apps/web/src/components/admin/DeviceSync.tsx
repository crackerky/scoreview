'use client'

import { usePathname } from 'next/navigation'
import { useEffect, useState } from 'react'
import { pendingCounts, queueChangeEvent, readDeviceQueue } from '@/admin/scoreQueue.utils'
import { browserStore, requestSend, sendFinishedEvent } from '@/admin/scoreSender'

// 管理画面のどの画面にも置く、端末全体の仕事(docs/operations/operator-experience.md
// 「紙を上回ることが前提」2)。
// - 記録係の画面を通信の無いところでも開けるよう、Service Worker(public/sw.js)を /admin/ に登録する
// - 端末に溜まった全試合の未送信の操作を、つながっている間は送り続ける(記録画面を開かなくても送る)
// - ログイン画面では、前の利用者が保存したページを端末から消す

// 未送信の操作を送り直す間隔(ミリ秒)。記録画面と同じ
const RETRY_INTERVAL_MS = 4000

function controller(): Promise<ServiceWorker | null> {
  if (!('serviceWorker' in navigator)) {
    return Promise.resolve(null)
  }
  return navigator.serviceWorker.ready.then((registration) => registration.active)
}

/** 担当試合の記録画面を前もって端末に保存させる(まだ開いていない試合も通信なしで開けるように) */
export function precacheScorePages(paths: string[]): void {
  void controller().then((worker) => worker?.postMessage({ type: 'precache', paths }))
}

export function DeviceSync() {
  const pathname = usePathname()

  useEffect(() => {
    // 開発サーバーでは登録しない(古い JS を返して画面の更新が見えなくなるため)
    if (process.env.NODE_ENV !== 'production' || !('serviceWorker' in navigator)) {
      return
    }
    navigator.serviceWorker
      .register('/sw.js', { scope: '/admin/', updateViaCache: 'none' })
      .catch((error: unknown) => {
        // 登録できなくても(古い端末、プライベートブラウズ)、通信がある間の記録はできる
        console.warn('service worker registration failed', error)
      })
  }, [])

  useEffect(() => {
    if (pathname === '/admin/login') {
      void controller().then((worker) => worker?.postMessage({ type: 'clear' }))
      return
    }
    void requestSend()
    const pending = () => pendingCounts(readDeviceQueue(browserStore())).size > 0
    const retry = window.setInterval(() => {
      if (document.visibilityState === 'visible' && pending()) {
        void requestSend()
      }
    }, RETRY_INTERVAL_MS)
    const onOnline = () => void requestSend()
    window.addEventListener('online', onOnline)
    return () => {
      window.clearInterval(retry)
      window.removeEventListener('online', onOnline)
    }
  }, [pathname])

  return null
}

/** この端末の未送信の操作の数(試合 ID ごと)。送信や他のタブの変更に合わせて描き直す */
export function usePendingCounts(): Map<number, number> {
  const [counts, setCounts] = useState<Map<number, number>>(new Map())
  useEffect(() => {
    const read = () => setCounts(pendingCounts(readDeviceQueue(browserStore())))
    read()
    window.addEventListener(queueChangeEvent, read)
    window.addEventListener(sendFinishedEvent, read)
    window.addEventListener('storage', read)
    return () => {
      window.removeEventListener(queueChangeEvent, read)
      window.removeEventListener(sendFinishedEvent, read)
      window.removeEventListener('storage', read)
    }
  }, [])
  return counts
}
