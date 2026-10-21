'use client'

import type { FormEvent } from 'react'
import { savedQueues } from '@/admin/scoreOps.utils'
import { logoutAction } from '@/app/admin/_actions/auth'

// ログアウトのボタン。ログアウトの前に、記録係の画面がこの端末に溜めた未送信の操作を消す
// (次にこの端末でログインした人のセッションで送らない。セキュリティ点検 L-9)。未送信の操作が
// 残っていれば、消えることを確かめてからログアウトする。

/** 溜めた操作を消してよいか。キャンセルなら false(ログアウトしない) */
function clearSavedQueues(): boolean {
  let queues: ReturnType<typeof savedQueues>
  try {
    queues = savedQueues(window.localStorage)
  } catch (error) {
    // プライベートブラウズ等で localStorage が使えない。溜まった操作も無いのでそのままログアウトする
    if (error instanceof DOMException) {
      return true
    }
    throw error
  }
  const pending = queues.reduce((sum, queue) => sum + queue.ops, 0)
  if (
    pending > 0 &&
    !window.confirm(
      `まだ送れていない記録が ${pending} 件あります。ログアウトすると、この端末から消えます。` +
        'ログアウトしますか?',
    )
  ) {
    return false
  }
  for (const queue of queues) {
    window.localStorage.removeItem(queue.key)
  }
  return true
}

export function LogoutButton() {
  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    if (!clearSavedQueues()) {
      event.preventDefault()
    }
  }
  return (
    <form action={logoutAction} onSubmit={onSubmit}>
      <button type="submit" className="secondary">
        ログアウト
      </button>
    </form>
  )
}
