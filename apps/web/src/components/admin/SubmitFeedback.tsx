'use client'

import { useEffect } from 'react'
import {
  BUSY_ATTRIBUTE,
  BUSY_TIMEOUT_MS,
  isActionResponse,
  SKIP_ATTRIBUTE,
  submitDecision,
} from '@/admin/submitFeedback.utils'

// 管理画面のフォームを送ってから応答が来るまで、押したボタンに「処理中」を出し、同じボタンの
// 押し直しを止める。関数と DB が止まっているところからの最初の操作は数秒かかり、ボタンが何も
// 反応しないと「押しても進まない」と受け取られて押し直される(作成の二重送信にもなる)ため。
// フォームごとに部品を差し替えず、document の submit を見て全フォームに効かせる。
// 応答の到着は Resource Timing で知る(Server Action は今の URL への fetch。fetch を書き換えない)。

function clearBusy() {
  for (const element of document.querySelectorAll(`[${BUSY_ATTRIBUTE}="true"]`)) {
    element.removeAttribute(BUSY_ATTRIBUTE)
  }
}

export function SubmitFeedback() {
  useEffect(() => {
    let observer: PerformanceObserver | null = null
    let timer: number | undefined
    const stopWatching = () => {
      observer?.disconnect()
      observer = null
      window.clearTimeout(timer)
    }
    const release = () => {
      stopWatching()
      clearBusy()
    }
    // React より先に見る(capture)。押し直しを React の Server Action の処理より前に止めるため
    const onSubmit = (event: SubmitEvent) => {
      const form = event.target
      const button = event.submitter
      if (!(form instanceof HTMLFormElement) || !(button instanceof HTMLElement)) {
        return
      }
      const decision = submitDecision({
        submitterBusy: button.getAttribute(BUSY_ATTRIBUTE) === 'true',
        methodAttribute: form.getAttribute('method'),
        skip: form.hasAttribute(SKIP_ATTRIBUTE),
      })
      if (decision === 'block') {
        event.preventDefault()
        event.stopPropagation()
        return
      }
      if (decision === 'ignore') {
        return
      }
      button.setAttribute(BUSY_ATTRIBUTE, 'true')
      stopWatching()
      const pageUrl = window.location.href
      const submittedAt = performance.now()
      observer = new PerformanceObserver((list) => {
        const done = list
          .getEntries()
          .some(
            (entry) =>
              entry instanceof PerformanceResourceTiming &&
              isActionResponse(entry, { pageUrl, submittedAt }, window.location.origin),
          )
        if (done) {
          release()
        }
      })
      observer.observe({ type: 'resource' })
      timer = window.setTimeout(release, BUSY_TIMEOUT_MS)
    }
    // bfcache から戻ったページに処理中の表示を残さない
    const onPageShow = () => release()
    document.addEventListener('submit', onSubmit, true)
    window.addEventListener('pageshow', onPageShow)
    return () => {
      document.removeEventListener('submit', onSubmit, true)
      window.removeEventListener('pageshow', onPageShow)
      release()
    }
  }, [])
  return null
}
