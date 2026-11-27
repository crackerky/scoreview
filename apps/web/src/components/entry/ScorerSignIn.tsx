'use client'

import { useActionState } from 'react'
import type { LinkFormState } from '@/entry/entryLink.utils'
import { FormError } from './FormError'
import { submitLinkForm } from './submitLinkForm'

// チームの記録係のリンクで開く画面のボタン。押すと記録の画面(自分の担当試合の一覧)に入る。入れなかった
// ときは理由を上に出す。

type Props = {
  token: string
  action: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}

export function ScorerSignIn({ token, action }: Props) {
  const [state, formAction] = useActionState(action, { error: null })
  return (
    <form action={formAction} onSubmit={submitLinkForm(formAction)} className="entry-form">
      <FormError error={state.error} attempt={state} />
      <input type="hidden" name="token" value={token} />
      <button type="submit" className="entry-button">
        記録の画面を開く
      </button>
    </form>
  )
}
