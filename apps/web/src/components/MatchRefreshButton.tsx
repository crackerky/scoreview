'use client'

import { type MouseEvent, useEffect, useState } from 'react'

// clone-spec D 2.1: 試合ページの更新ボタン。最初の 5 秒は「待て」アイコン(押すと alert)、
// その後「更新」アイコンに替わり、押すと ?lang=..&refpos=1 で同じページを location.replace する。
// 参照サイトは jQuery の show/hide でフェードするが、ここでは表示を切り替えるだけにする。
// 自動の再読み込みはしない(参照サイトにもない)。

/** 「待て」から「更新」に替わるまでの時間(ミリ秒)。参照サイトの setTimeout の値 */
const waitMs = 5000

type Phase = 'hidden' | 'wait' | 'ready'

type Props = {
  /** 更新で開き直す URL(refpos=1 付き) */
  reloadHref: string
  /** 待ち時間中に押したときの alert の文面 */
  waitMessage: string
  okAlt: string
  ngAlt: string
  /** refpos=1 で開かれたとき。表示後に #score_navi へ移動する(参照サイトの head の JS) */
  scrollToNavi: boolean
}

const hidden = { display: 'none' } as const

export function MatchRefreshButton({ reloadHref, waitMessage, okAlt, ngAlt, scrollToNavi }: Props) {
  // サーバーの描画では両方 display: none(参照サイトの初期 HTML と同じ)
  const [phase, setPhase] = useState<Phase>('hidden')

  useEffect(() => {
    setPhase('wait')
    const timer = window.setTimeout(() => setPhase('ready'), waitMs)
    return () => window.clearTimeout(timer)
  }, [])

  useEffect(() => {
    if (scrollToNavi) {
      window.location.href = '#score_navi'
    }
  }, [scrollToNavi])

  const reload = (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault()
    window.location.replace(reloadHref)
  }
  const warn = (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault()
    window.alert(waitMessage)
  }

  return (
    <>
      <a
        id="Refresh_OK"
        style={phase === 'ready' ? undefined : hidden}
        href={reloadHref}
        onClick={reload}
      >
        <img src="/brand/refresh.svg" alt={okAlt} />
      </a>
      {/* biome-ignore lint/a11y/useValidAnchor: 参照サイトと同じ a#Refresh_NG(遷移先がなく押すと alert だけ) */}
      <a id="Refresh_NG" style={phase === 'wait' ? undefined : hidden} href="#" onClick={warn}>
        <img src="/brand/refresh-ng.svg" alt={ngAlt} />
      </a>
    </>
  )
}
