'use client'

import { useActionState } from 'react'
import type { EntryLink } from '@/api/entry.types'
import { entryLimits } from '@/api/entry.types'
import { answerFieldName } from '@/entry/entry.utils'
import type { LinkFormState } from '@/entry/entryLink.utils'
import { EntryAnswerField } from './EntryAnswerField'
import { FormError } from './FormError'
import { submitLinkForm } from './submitLinkForm'

// 代表の申込のフォーム。新しい申込と、差し戻された申込の送り直しで同じ欄を使う(送り直しは entryId を
// 付け、今の中身を入れて開く)。送れなかったときは、api の文を上に出し、書いた中身を欄に残す
// (submitLinkForm)。

type Team = NonNullable<EntryLink['team']>

type Props = {
  token: string
  divisions: Team['divisions']
  questions: Team['questions']
  action: (state: LinkFormState, formData: FormData) => Promise<LinkFormState>
  submitLabel: string
  /** 欄に最初から入れる値(送り直しのときの今の中身) */
  initial?: Record<string, string>
  /** 送り直す申込 */
  entryId?: number
  /** メンバー表の上に出す、本人がもう登録した人(送り直しのとき。代表の欄では直せない) */
  registered?: ReadonlyArray<{ id: number; name: string; number: string }>
}

export function TeamEntryForm({
  token,
  divisions,
  questions,
  action,
  submitLabel,
  initial = {},
  entryId,
  registered = [],
}: Props) {
  const [state, formAction] = useActionState(action, { error: null })
  const value = (name: string) => initial[name] ?? ''
  return (
    <form action={formAction} onSubmit={submitLinkForm(formAction)} className="entry-form">
      <FormError error={state.error} attempt={state} />
      <input type="hidden" name="token" value={token} />
      {entryId === undefined ? null : <input type="hidden" name="entryId" value={entryId} />}
      {registered.map((member) => (
        <input key={member.id} type="hidden" name="registered" value={member.id} />
      ))}
      <label className="entry-field">
        <span>部門</span>
        <select name="divisionId" required defaultValue={value('divisionId')}>
          <option value="" disabled>
            選んでください
          </option>
          {divisions.map((division) => (
            <option key={division.id} value={division.id}>
              {division.name.ja}
            </option>
          ))}
        </select>
      </label>
      <TextField label="チーム名(正式)" name="fullName" value={value('fullName')} required />
      <TextField
        label="チーム名(短縮。日程表に出ます)"
        name="shortName"
        value={value('shortName')}
        required
      />
      <TextField
        label="代表の名前(フルネーム)"
        name="repName"
        value={value('repName')}
        autoComplete="name"
        required
      />
      <TextField
        label="電話番号"
        name="repPhone"
        type="tel"
        value={value('repPhone')}
        autoComplete="tel"
      />
      <TextField
        label="メールアドレス"
        name="repEmail"
        type="email"
        value={value('repEmail')}
        autoComplete="email"
      />
      {registered.length > 0 ? (
        <div className="entry-field">
          <span>本人が登録したメンバー(本人が直します)</span>
          <ul className="entry-list">
            {registered.map((member) => (
              <li key={member.id}>
                {member.number ? `${member.number} ` : ''}
                {member.name}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      <label className="entry-field">
        <span>
          {registered.length > 0 ? 'ほかのメンバー' : 'メンバー'}
          (1 行に 1
          人。背番号は名前の前に。メンバー本人が招待リンクから登録するので、空でもかまいません)
        </span>
        <textarea
          name="members"
          rows={6}
          placeholder={'7 山田 太郎\n12 佐藤 花子'}
          defaultValue={value('members')}
        />
      </label>
      {questions.map((question) => (
        <EntryAnswerField
          key={question.id}
          question={question}
          fieldClass="entry-field"
          enforceRequired
          defaultValue={value(answerFieldName(question.id))}
        />
      ))}
      <button type="submit" className="entry-button">
        {submitLabel}
      </button>
    </form>
  )
}

function TextField({
  label,
  name,
  value,
  type = 'text',
  autoComplete,
  required = false,
}: {
  label: string
  name: string
  value: string
  type?: 'text' | 'tel' | 'email'
  autoComplete?: string
  required?: boolean
}) {
  return (
    <label className="entry-field">
      <span>{label}</span>
      <input
        type={type}
        name={name}
        defaultValue={value}
        maxLength={entryLimits.maxShortText}
        autoComplete={autoComplete}
        required={required}
      />
    </label>
  )
}
