'use client'

import { useActionState } from 'react'
import type { EntryDetail } from '@/api/entry.types'
import type { LinkFormState } from '@/entry/entryLink.utils'
import { FormError } from './FormError'
import { submitLinkForm } from './submitLinkForm'

// 参加が決まったチームの記録係を選ぶフォーム(代表の画面)。記録係は自分たちの試合のスコアを、公式 LINE で
// 「記録」と送って開く画面から入れるので、LINE で登録したメンバーだけを選べる。選び直しと外すこともできる。

type Props = {
  token: string
  entry: EntryDetail
  action: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}

export function ScorerForm({ token, entry, action }: Props) {
  const [state, formAction] = useActionState(action, { error: null })
  const registered = entry.memberList.filter((member) => member.registered)
  const current = entry.memberList.find((member) => member.id === entry.scorerMemberId)
  return (
    <div className="entry-scorer">
      <h3 className="entry-subheading">記録係</h3>
      <p className="entry-muted">
        {current
          ? `${current.name} さんが記録係です。自分たちの試合のスコアを入れてもらいます。`
          : '試合のスコアを入れる人を 1 人選んでください。LINE で登録したメンバーから選べます。'}
      </p>
      {registered.length === 0 ? (
        <p className="entry-muted">
          まだ誰も LINE で登録していません。先にメンバーに登録のお願いを送ってください。
        </p>
      ) : (
        <form action={formAction} onSubmit={submitLinkForm(formAction)} className="entry-form">
          <FormError error={state.error} attempt={state} />
          <input type="hidden" name="token" value={token} />
          <input type="hidden" name="entryId" value={entry.id} />
          <label className="entry-field">
            <span>記録係にする人</span>
            <select name="memberId" defaultValue={current ? String(current.id) : ''}>
              <option value="">選ばない</option>
              {registered.map((member) => (
                <option key={member.id} value={member.id}>
                  {member.number ? `${member.number} ` : ''}
                  {member.name}
                </option>
              ))}
            </select>
          </label>
          <button type="submit" className="entry-button">
            記録係を決める
          </button>
        </form>
      )}
    </div>
  )
}
