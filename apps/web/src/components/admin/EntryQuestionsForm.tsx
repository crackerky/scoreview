'use client'

import { useActionState } from 'react'
import type { AdminFormState } from '@/admin/form.utils'
import { type EntryQuestion, entryKinds, entryScopes } from '@/api/entry.types'
import { FormError } from '@/components/entry/FormError'
import { submitLinkForm } from '@/components/entry/submitLinkForm'
import { kindLabels, scopeLabels } from '@/entry/entry.utils'
import { Hidden } from './AdminParts'

// 申込で聞く質問のフォーム。1 行が 1 問で、下の空の行に書くと足し、文言を消すか「消す」に印を付けると消す。
// 誤りで保存できなかったときに、書いた欄が全部消えないよう、送信は onSubmit から自分で行い、誤りは
// フォームの先頭に文で出す(submitLinkForm と同じ形。保存できたら画面ごと読み直す)。

type Props = {
  formId: string
  back: string
  tournamentId: number
  /** 今の質問と、足すための空の行(null) */
  rows: Array<EntryQuestion | null>
  action: (state: AdminFormState, formData: FormData) => Promise<AdminFormState>
}

export function EntryQuestionsForm({ formId, back, tournamentId, rows, action }: Props) {
  const [state, formAction] = useActionState(action, { error: null })
  return (
    <form
      id={formId}
      action={formAction}
      onSubmit={submitLinkForm(formAction)}
      className="adm-stack"
    >
      <FormError error={state.error} attempt={state} className="adm-notice error" />
      <Hidden values={{ back, tournamentId, rows: rows.length }} />
      {rows.map((question, index) => (
        <QuestionRow
          // 空の行は質問が無いので並びの位置で見分ける(保存すると画面ごと描き直す)
          key={question?.id ?? `new-${index}`}
          index={index}
          question={question}
        />
      ))}
    </form>
  )
}

function QuestionRow({ index, question }: { index: number; question: EntryQuestion | null }) {
  const name = (key: string) => `q${index}.${key}`
  // 答えのある質問は、答え方と答える人を変えると集めた答えの意味が変わるので選ばせない
  const locked = question !== null && question.answered > 0
  return (
    <fieldset className="adm-card">
      <legend>
        {question === null ? '質問を足す' : `質問 ${index + 1}`}
        {locked ? `(答え ${question.answered} 件)` : ''}
      </legend>
      {question === null ? null : <input type="hidden" name={name('id')} value={question.id} />}
      <div className="adm-grid">
        <label className="adm-field">
          <span>質問の文</span>
          <input type="text" name={name('label')} defaultValue={question?.label ?? ''} />
        </label>
        <ChoiceField
          label="答える人"
          name={name('scope')}
          locked={locked}
          value={question?.scope ?? 'team'}
          options={entryScopes.map((scope) => ({ value: scope, label: scopeLabels[scope] }))}
        />
        <ChoiceField
          label="答え方"
          name={name('kind')}
          locked={locked}
          value={question?.kind ?? 'short_text'}
          options={entryKinds.map((kind) => ({ value: kind, label: kindLabels[kind] }))}
        />
        <label className="adm-field">
          <span>順</span>
          <input type="number" name={name('order')} defaultValue={index + 1} step={1} />
        </label>
      </div>
      <label className="adm-field">
        <span>選択肢(答え方が「選択」のとき。1 行に 1 つ)</span>
        <textarea
          name={name('options')}
          rows={3}
          defaultValue={question?.options.join('\n') ?? ''}
        />
      </label>
      <div className="adm-actions">
        <label className="adm-check">
          <input
            type="checkbox"
            name={name('required')}
            defaultChecked={question?.required ?? false}
          />
          <span>必須にする</span>
        </label>
        {question === null ? null : (
          <label className="adm-check">
            <input type="checkbox" name={name('remove')} />
            <span>この質問を消す{locked ? `(答え ${question.answered} 件も消えます)` : ''}</span>
          </label>
        )}
      </div>
    </fieldset>
  )
}

/** 選ぶ欄。locked なら選ばせず、今の値を hidden で送って呼び名だけを出す */
function ChoiceField({
  label,
  name,
  locked,
  value,
  options,
}: {
  label: string
  name: string
  locked: boolean
  value: string
  options: Array<{ value: string; label: string }>
}) {
  if (locked) {
    return (
      <div className="adm-field">
        <span>{label}</span>
        <input type="hidden" name={name} value={value} />
        <span>{options.find((option) => option.value === value)?.label ?? value}</span>
      </div>
    )
  }
  return (
    <label className="adm-field">
      <span>{label}</span>
      <select name={name} defaultValue={value}>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  )
}
