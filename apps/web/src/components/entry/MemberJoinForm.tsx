'use client'

import { useActionState, useState } from 'react'
import type { EntryLinkMember } from '@/api/entry.types'
import { entryLimits } from '@/api/entry.types'
import type { LinkFormState } from '@/entry/entryLink.utils'
import { EntryAnswerField } from './EntryAnswerField'
import { FormError } from './FormError'
import { submitLinkForm } from './submitLinkForm'

// メンバー登録のフォーム。代表が書いた名簿から自分を選ぶと、名前と背番号の欄にその行を入れる(本人が
// 確かめて直す。docs/line-integration.md「代表エントリーとメンバーエントリー」)。もう登録した人には、
// 登録した中身を入れて開く。送れなかったときは、api の文を上に出し、書いた中身を欄に残す
// (submitLinkForm)。

type Props = {
  token: string
  member: EntryLinkMember
  action: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
}

export function MemberJoinForm({ token, member, action }: Props) {
  const mine = member.mine
  const saved = new Map(member.mineAnswers.map((answer) => [answer.questionId, answer.value]))
  const [state, formAction] = useActionState(action, { error: null })
  const [picked, setPicked] = useState<string>(mine ? String(mine.id) : '')
  const [name, setName] = useState(mine?.name ?? '')
  const [number, setNumber] = useState(mine?.number ?? '')
  const choose = (value: string) => {
    setPicked(value)
    const row = member.unclaimed.find((item) => String(item.id) === value)
    setName(row?.name ?? '')
    setNumber(row?.number ?? '')
  }
  return (
    <form action={formAction} onSubmit={submitLinkForm(formAction)} className="entry-form">
      <FormError error={state.error} attempt={state} />
      <input type="hidden" name="token" value={token} />
      {mine ? (
        <input type="hidden" name="member" value={mine.id} />
      ) : member.unclaimed.length > 0 ? (
        <fieldset className="entry-choices">
          <legend>代表の方が書いた名簿から、自分を選んでください</legend>
          {member.unclaimed.map((row) => (
            <label key={row.id} className="entry-choice">
              <input
                type="radio"
                name="member"
                value={row.id}
                checked={picked === String(row.id)}
                onChange={() => choose(String(row.id))}
                required
              />
              <span>
                {row.name}
                {row.number ? `(背番号 ${row.number})` : ''}
              </span>
            </label>
          ))}
          <label className="entry-choice">
            <input
              type="radio"
              name="member"
              value="new"
              checked={picked === 'new'}
              onChange={() => choose('new')}
              required
            />
            <span>名簿に自分がいない</span>
          </label>
        </fieldset>
      ) : null}
      <label className="entry-field">
        <span>名前(フルネーム)</span>
        <input
          type="text"
          name="name"
          value={name}
          onChange={(event) => setName(event.target.value)}
          maxLength={entryLimits.maxShortText}
          autoComplete="name"
          required
        />
      </label>
      <label className="entry-field">
        <span>背番号(決まっていなければ空のまま)</span>
        <input
          type="text"
          name="number"
          value={number}
          onChange={(event) => setNumber(event.target.value)}
          inputMode="numeric"
          maxLength={entryLimits.maxNumberText}
        />
      </label>
      {member.questions.map((question) => (
        <EntryAnswerField
          key={question.id}
          question={question}
          fieldClass="entry-field"
          enforceRequired
          defaultValue={saved.get(question.id) ?? ''}
        />
      ))}
      <button type="submit" className="entry-button">
        {mine ? '登録を直す' : '登録する'}
      </button>
    </form>
  )
}
