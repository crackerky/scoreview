'use client'

import { type MouseEvent, useEffect, useState } from 'react'

// clone-spec B 9: 更新ボタン。最初の 5 秒は「待て」アイコン(Refresh_NG。押すと alert)を出し、
// その後「更新」アイコン(Refresh_OK)に切り替える。自動で再読込はしない。
// 参照サイトは jQuery の show/hide のアニメーション(500ms / 800ms)で切り替えるが、ここでは即時に切り替える。

/** NG から OK に切り替えるまでの時間(ミリ秒)。参照サイトの setTimeout の値 */
const waitMs = 5000

type RefreshButtonsProps = {
  /** OK を押したときに開く URL(現在の絞り込みで同じページを開き直す) */
  reloadHref: string
  /** NG を押したときの alert の文面 */
  waitMessage: string
  okAlt: string
  ngAlt: string
}

const hidden = { display: 'none' } as const

export function RefreshButtons({ reloadHref, waitMessage, okAlt, ngAlt }: RefreshButtonsProps) {
  const [ready, setReady] = useState(false)
  useEffect(() => {
    const timer = window.setTimeout(() => setReady(true), waitMs)
    return () => window.clearTimeout(timer)
  }, [])
  const onWaitClick = (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault()
    window.alert(waitMessage)
  }
  // NG の href も再読込先にしておく(JavaScript がないときは押せば開き直せる)
  return (
    <>
      <a id="Refresh_OK" style={ready ? undefined : hidden} href={reloadHref}>
        <img src="/brand/refresh.svg" alt={okAlt} />
      </a>
      <a id="Refresh_NG" style={ready ? hidden : undefined} href={reloadHref} onClick={onWaitClick}>
        <img src="/brand/refresh-ng.svg" alt={ngAlt} />
      </a>
      {' '}
    </>
  )
}
