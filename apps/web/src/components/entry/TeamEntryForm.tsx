'use client'

import { useActionState, useId, useState } from 'react'
import type { EntryLink, EntryPastTeam } from '@/api/entry.types'
import { entryLimits } from '@/api/entry.types'
import { answerFieldName } from '@/entry/entry.utils'
import { findPastTeam, type LinkFormState, pastTeamValues } from '@/entry/entryLink.utils'
import { EntryAnswerField } from './EntryAnswerField'
import { FormError } from './FormError'
import { submitLinkForm } from './submitLinkForm'

// 代表の申込のフォーム。新しい申込と、差し戻された申込の送り直しで同じ欄を使う(送り直しは entryId を
// 付け、今の中身を入れて開く)。送れなかったときは、api の文を上に出し、書いた中身を欄に残す
// (submitLinkForm)。前に申し込んだチームがあれば、選ぶかチーム名を書くと、短縮名とメンバー表を入れる
// (docs/line-integration.md「参加者の台帳」)。

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
  /** 前にほかの大会に申し込んだチーム(新しい申込のとき) */
  pastTeams?: readonly EntryPastTeam[]
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
  pastTeams = [],
}: Props) {
  const [state, formAction] = useActionState(action, { error: null })
  const value = (name: string) => initial[name] ?? ''
  const [team, setTeam] = useState({
    fullName: value('fullName'),
    shortName: value('shortName'),
    members: value('members'),
  })
  const listId = useId()
  // 前のチームを選んだときは全部を入れる。チーム名を書いて合ったときは、まだ空の欄だけを埋める
  // (書きかけの短縮名やメンバー表を消さないため)
  // メンバーの名前の欄は畳んでおき、名前が入っているとき(送り直し、前のチームから入れたとき)は開く
  const [listOpen, setListOpen] = useState(team.members.trim() !== '')
  const choose = (past: EntryPastTeam) => {
    const values = pastTeamValues(past)
    setTeam(values)
    setListOpen((open) => open || values.members.trim() !== '')
  }
  const typeName = (fullName: string) => {
    const past = findPastTeam(pastTeams, fullName)
    const values = past ? pastTeamValues(past) : null
    setTeam((current) => ({
      fullName,
      shortName: current.shortName || (values?.shortName ?? ''),
      members: current.members || (values?.members ?? ''),
    }))
    if (values && values.members.trim() !== '') {
      setListOpen(true)
    }
  }
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
      {pastTeams.length > 0 ? (
        <div className="entry-field">
          <span>前に申し込んだチームから入れる</span>
          <div className="entry-past-teams">
            {pastTeams.map((past) => (
              <button
                key={past.fullName}
                type="button"
                className="entry-past-team"
                onClick={() => choose(past)}
              >
                {past.fullName}
                <small>{past.members.length} 人</small>
              </button>
            ))}
          </div>
        </div>
      ) : null}
      <label className="entry-field">
        <span>チーム名(正式)</span>
        <input
          type="text"
          name="fullName"
          value={team.fullName}
          onChange={(event) => typeName(event.target.value)}
          list={pastTeams.length > 0 ? listId : undefined}
          maxLength={entryLimits.maxShortText}
          autoComplete="off"
          required
        />
      </label>
      {pastTeams.length > 0 ? (
        <datalist id={listId}>
          {pastTeams.map((past) => (
            <option key={past.fullName} value={past.fullName} />
          ))}
        </datalist>
      ) : null}
      <label className="entry-field">
        <span>チーム名(短縮。日程表に出ます)</span>
        <input
          type="text"
          name="shortName"
          value={team.shortName}
          onChange={(event) => setTeam({ ...team, shortName: event.target.value })}
          maxLength={entryLimits.maxShortText}
          required
        />
      </label>
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
      {/* メンバーは申し込んだあとに招待リンクから各自で登録するので、名簿は分かる人だけが書く欄にして
          畳む */}
      <details
        className="entry-details"
        open={listOpen}
        onToggle={(event) => setListOpen(event.currentTarget.open)}
      >
        <summary>
          {registered.length > 0
            ? 'ほかのメンバーの名前を書いておく'
            : 'メンバーの名前を書いておく'}
          (分かれば。書かなくても申し込めます)
        </summary>
        <label className="entry-field">
          <span>1 行に 1 人。背番号は名前の前に</span>
          <textarea
            name="members"
            rows={6}
            placeholder={'7 山田 太郎\n12 佐藤 花子'}
            value={team.members}
            onChange={(event) => setTeam({ ...team, members: event.target.value })}
          />
        </label>
      </details>
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
