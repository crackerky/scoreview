'use client'

import { useActionState } from 'react'
import type { EntryMember } from '@/api/entry.types'
import type { LinkFormState } from '@/entry/entryLink.utils'
import { FormError } from './FormError'
import { submitLinkForm } from './submitLinkForm'

// 代表が、招待リンクを使わずに登録したメンバー 1 人を外すボタン。押し間違えないよう、開いてから外す。
// 外せなかったときは理由を出す。

type Props = {
  token: string
  entryId: number
  member: EntryMember
  action: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}

export function RemoveMember({ token, entryId, member, action }: Props) {
  const [state, formAction] = useActionState(action, { error: null })
  return (
    <details className="entry-remove">
      <summary>外す</summary>
      <form action={formAction} onSubmit={submitLinkForm(formAction)}>
        <FormError error={state.error} attempt={state} />
        <input type="hidden" name="token" value={token} />
        <input type="hidden" name="entryId" value={entryId} />
        <input type="hidden" name="memberId" value={member.id} />
        <button type="submit" className="entry-button secondary">
          {member.name} さんをこのチームから外す
        </button>
      </form>
    </details>
  )
}
