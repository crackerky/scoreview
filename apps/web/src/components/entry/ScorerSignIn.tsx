'use client'

import { useActionState, useEffect, useRef } from 'react'
import type { LinkFormState } from '@/entry/entryLink.utils'
import { FormError } from './FormError'
import { submitLinkForm } from './submitLinkForm'

// チームの記録係のリンクで開く画面のボタン。画面を開いたら 1 回だけ自動で送り、記録の画面(match があれば
// その試合のスコアの画面、無ければ担当の試合の一覧)に入る。入る操作を画面の中のスクリプトで送るのは、
// LINE などがリンクを先読みしただけでは入らないようにするため(先読みはスクリプトを動かさない)。
// スクリプトが動かないときや、入れなかったときのために、ボタンも残す。入れなかったときは理由を上に出す。

type Props = {
  token: string
  match: number | null
  action: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}

export function ScorerSignIn({ token, match, action }: Props) {
  const [state, formAction] = useActionState(action, { error: null })
  const formRef = useRef<HTMLFormElement>(null)
  // 開発中の二重の描画でも 1 回だけ送る
  const sent = useRef(false)
  useEffect(() => {
    if (!sent.current && formRef.current) {
      sent.current = true
      formRef.current.requestSubmit()
    }
  }, [])
  return (
    <form
      ref={formRef}
      action={formAction}
      onSubmit={submitLinkForm(formAction)}
      className="entry-form"
    >
      <FormError error={state.error} attempt={state} />
      <input type="hidden" name="token" value={token} />
      {match === null ? null : <input type="hidden" name="match" value={match} />}
      <button type="submit" className="entry-button">
        {match === null ? '担当の試合の一覧を開く' : '試合のスコアの画面を開く'}
      </button>
    </form>
  )
}
