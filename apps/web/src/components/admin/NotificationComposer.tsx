'use client'

import { type KeyboardEvent, type ReactNode, useEffect, useRef, useState } from 'react'
import {
  type Audience,
  audienceKinds,
  audienceStatuses,
  maxNotificationBody,
} from '@/api/notify.types'
import { checkAudienceAction, sendNotificationAction } from '@/app/admin/_actions/notify'
import {
  audienceFromQuery,
  audienceKindLabels,
  audienceStatusLabels,
  checkedFields,
  sameAudience,
} from '@/notify/notify.utils'
import { Hidden, Select } from './AdminParts'
import { MessageDraft } from './MessageDraft'

type Props = {
  /** 人数を数えた宛先(画面の URL の条件) */
  checked: Audience
  divisions: ReadonlyArray<{ value: string; label: string }>
  heading: string
  formId: string
  /** 送るフォームの hidden 欄(戻り先、大会、送る画面ごとの乱数) */
  hidden: Record<string, string | number>
  draftKey: string
  /** 数えた人数と宛先の一覧(サーバーで描いたもの) */
  children: ReactNode
}

/** フォームで今選んでいる宛先の条件 */
function selectionOf(form: HTMLFormElement): Audience {
  const values: Record<string, string> = {}
  for (const [key, value] of new FormData(form)) {
    if (typeof value === 'string') {
      values[key] = value
    }
  }
  return audienceFromQuery(values)
}

// 送る画面の本体。宛先の条件の欄と文の欄を 1 つの送るフォームに置き、人数を数えた宛先も hidden 欄で
// 送る。Server Action は 2 つを比べ、違えば送らずに、選んでいる宛先で数え直した画面に戻す(JavaScript が
// 動かないときや、ブラウザがフォームの状態を戻したときも、数えた人数と違う宛先には送らない)。
// 「宛先を確かめる」は同じフォームの別の操作(checkAudienceAction)で、文を URL に載せずに数え直す。
// この部品は、選択を変えたことを知らせることと、Enter で送らないことだけを受け持つ。
export function NotificationComposer({
  checked,
  divisions,
  heading,
  formId,
  hidden,
  draftKey,
  children,
}: Props) {
  const formRef = useRef<HTMLFormElement>(null)
  const [selection, setSelection] = useState<Audience>(checked)
  // ブラウザが戻したフォームの状態(再読み込み、戻る)も、画面に出ている選択として読む
  useEffect(() => {
    if (formRef.current) {
      setSelection(selectionOf(formRef.current))
    }
  }, [])
  const changed = !sameAudience(selection, checked)
  // 宛先の欄で Enter を押すと、フォームの既定のボタン(画面の上の「送る」)で送られてしまうので止める
  const stopEnter = (event: KeyboardEvent<HTMLFormElement>) => {
    if (event.key === 'Enter' && !(event.target instanceof HTMLTextAreaElement)) {
      event.preventDefault()
    }
  }
  return (
    <form
      id={formId}
      ref={formRef}
      action={sendNotificationAction}
      className="adm-stack"
      onChange={(event) => setSelection(selectionOf(event.currentTarget))}
      onKeyDown={stopEnter}
    >
      <Hidden values={{ ...hidden, ...checkedFields(checked) }} />
      <h2 className="adm-section-title">宛先</h2>
      <fieldset>
        <legend>誰に送るか</legend>
        {audienceKinds.map((kind) => (
          <label key={kind} className="adm-check adm-audience">
            <input type="radio" name="kind" value={kind} defaultChecked={checked.kind === kind} />
            <span>
              {audienceKindLabels[kind].label}
              <small className="adm-muted">{audienceKindLabels[kind].hint}</small>
            </span>
          </label>
        ))}
      </fieldset>
      <div className="adm-grid">
        <Select
          label="申込の状態(申し込んだ代表のときだけ)"
          name="status"
          defaultValue={checked.status}
          options={audienceStatuses.map((status) => ({
            value: status,
            label: audienceStatusLabels[status],
          }))}
        />
        <Select
          label="部門(前の大会の代表には使いません)"
          name="division"
          defaultValue={checked.divisionId === null ? '' : String(checked.divisionId)}
          options={[{ value: '', label: 'すべての部門' }, ...divisions]}
        />
      </div>
      {changed ? (
        <p className="adm-notice error" role="status">
          宛先を変えました。「宛先を確かめる」を押すと、人数を数え直します(書いた文は消えません)。
        </p>
      ) : null}
      <div className="adm-actions">
        <button type="submit" formAction={checkAudienceAction} formNoValidate>
          宛先を確かめる
        </button>
      </div>
      {children}
      <h2 className="adm-section-title">文</h2>
      <MessageDraft heading={heading} maxLength={maxNotificationBody} draftKey={draftKey} />
    </form>
  )
}
