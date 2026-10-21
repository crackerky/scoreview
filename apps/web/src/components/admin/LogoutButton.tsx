'use client'

import type { FormEvent } from 'react'
import { pendingTotal } from '@/admin/scoreQueue.utils'
import { browserStore, discardPending } from '@/admin/scoreSender'
import { logoutAction } from '@/app/admin/_actions/auth'

// ログアウトのボタン。記録係の画面がこの端末に溜めた未送信の操作(端末全体のキュー)は利用者に
// 結び付いていないので、ログアウトの前に消す(次にこの端末でログインした人のセッションで送らない。
// セキュリティ点検 L-9)。未送信の操作が残っていれば、消えることを確かめてからログアウトする。

function confirmDiscard(): boolean {
  const pending = pendingTotal(browserStore())
  if (pending === 0) {
    return true
  }
  return window.confirm(
    `まだ送れていない記録が ${pending} 件あります。ログアウトすると、この端末から消えます。` +
      'ログアウトしますか?',
  )
}

export function LogoutButton() {
  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    if (!confirmDiscard()) {
      event.preventDefault()
      return
    }
    discardPending()
  }
  return (
    <form action={logoutAction} onSubmit={onSubmit}>
      <button type="submit" className="secondary">
        ログアウト
      </button>
    </form>
  )
}
