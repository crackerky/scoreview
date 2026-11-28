'use client'

import { useActionState, useState } from 'react'
import type { EntryLinkMember } from '@/api/entry.types'
import { entryLimits } from '@/api/entry.types'
import type { LinkFormState } from '@/entry/entryLink.utils'
import { EntryAnswerField } from './EntryAnswerField'
import { FormError } from './FormError'
import { submitLinkForm } from './submitLinkForm'

/** 性別の選択肢(選手の性別と同じ M / F) */
const genderChoices = [
  { value: 'M', label: '男性' },
  { value: 'F', label: '女性' },
] as const

// メンバー登録のフォーム。本人が自分の名前(フルネーム)を書いて登録する。代表が名簿を書いていれば、
// その中から自分を選ぶこともできるが、選ばなくてよい(LINE の名前と本名は合わないことが多く、名簿の
// 書き方も人によるので、書く方を先に出す。名簿に同じ名前の行が 1 つだけあれば、api がその行で登録する)。
// 名簿から選ぶと、名前と背番号の欄にその行を入れる(本人が確かめて直す)。もう登録した人には、登録した
// 中身を入れて開く。まだ登録していない人には、前に入れた名前(参加者の台帳)を入れておく。送れなかった
// ときは、api の文を上に出し、書いた中身を欄に残す(submitLinkForm)。大会が性別と生年月日を聞くときは
// その欄を出し、このチームで答えた値か前回の答えを入れておく(運営の画面にだけ出る)。

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
  const [name, setName] = useState(mine?.name ?? member.profileName)
  const [number, setNumber] = useState(mine?.number ?? '')
  const choose = (value: string) => {
    setPicked(value)
    const row = member.unclaimed.find((item) => String(item.id) === value)
    if (row) {
      setName(row.name)
      setNumber(row.number)
    }
  }
  return (
    <form action={formAction} onSubmit={submitLinkForm(formAction)} className="entry-form">
      <FormError error={state.error} attempt={state} />
      <input type="hidden" name="token" value={token} />
      {mine ? <input type="hidden" name="member" value={mine.id} /> : null}
      <label className="entry-field">
        <span>自分の名前(フルネーム)</span>
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
      {member.personal.gender ? (
        <fieldset className="entry-choices">
          <legend>性別</legend>
          {genderChoices.map((choice) => (
            <label key={choice.value} className="entry-choice">
              <input
                type="radio"
                name="gender"
                value={choice.value}
                defaultChecked={member.minePersonal.gender === choice.value}
                required
              />
              <span>{choice.label}</span>
            </label>
          ))}
        </fieldset>
      ) : null}
      {member.personal.birthdate ? (
        <label className="entry-field">
          <span>生年月日</span>
          <input
            type="date"
            name="birthdate"
            defaultValue={member.minePersonal.birthdate ?? ''}
            min="1900-01-01"
            autoComplete="bday"
            required
          />
        </label>
      ) : null}
      {member.personal.gender || member.personal.birthdate ? (
        <p className="entry-muted">
          性別と生年月日は大会の運営だけが見ます。次の大会の登録では、ここで入れた内容が最初から入ります。
        </p>
      ) : null}
      {!mine && member.unclaimed.length > 0 ? (
        <details className="entry-details">
          <summary>代表が書いた名簿から選ぶ(選ばなくても登録できます)</summary>
          <fieldset className="entry-choices">
            <legend>名簿に自分がいれば選んでください。名前と背番号の欄に入ります</legend>
            <label className="entry-choice">
              <input
                type="radio"
                name="member"
                value=""
                checked={picked === ''}
                onChange={() => setPicked('')}
              />
              <span>選ばない</span>
            </label>
            {member.unclaimed.map((row) => (
              <label key={row.id} className="entry-choice">
                <input
                  type="radio"
                  name="member"
                  value={row.id}
                  checked={picked === String(row.id)}
                  onChange={() => choose(String(row.id))}
                />
                <span>
                  {row.name}
                  {row.number ? `(背番号 ${row.number})` : ''}
                </span>
              </label>
            ))}
          </fieldset>
        </details>
      ) : null}
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
