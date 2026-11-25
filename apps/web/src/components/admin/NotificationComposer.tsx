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
  checkedTimingFields,
  sameAudience,
  sameTiming,
  type Timing,
  timingFromQuery,
} from '@/notify/notify.utils'
import { Hidden, Select } from './AdminParts'
import { MessageDraft } from './MessageDraft'

type Props = {
  /** 人数を数えた宛先(画面の URL の条件) */
  checked: Audience
  /** 確かめた送る時刻(画面の URL の時刻) */
  checkedTime: Timing
  divisions: ReadonlyArray<{ value: string; label: string }>
  heading: string
  formId: string
  /** 送るフォームの hidden 欄(戻り先、大会、送る画面ごとの乱数) */
  hidden: Record<string, string | number>
  draftKey: string
  /** 前日と当日の案内のひな形を選んで開いたときの文(それ以外は空) */
  initialBody: string
  /** 数えた人数と宛先の一覧(サーバーで描いたもの) */
  children: ReactNode
}

/** フォームで今選んでいる宛先の条件と送る時刻 */
function selectionOf(form: HTMLFormElement): { audience: Audience; timing: Timing } {
  const values: Record<string, string> = {}
  for (const [key, value] of new FormData(form)) {
    if (typeof value === 'string') {
      values[key] = value
    }
  }
  return { audience: audienceFromQuery(values), timing: timingFromQuery(values) }
}

// 送る画面の本体。宛先の条件、送る時刻、文の欄を 1 つの送るフォームに置き、人数を数えた条件と確かめた
// 時刻も hidden 欄で送る。Server Action は比べて、違えば送らずに、選んでいる条件で数え直した画面に戻す
// (JavaScript が動かないときや、ブラウザがフォームの状態を戻したときも、確かめていない送信をしない)。
// 「宛先と時刻を確かめる」は同じフォームの別の操作(checkAudienceAction)で、文を URL に載せずに数え直す。
// この部品は、選択を変えたことを知らせることと、Enter で送らないことだけを受け持つ。
export function NotificationComposer({
  checked,
  checkedTime,
  divisions,
  heading,
  formId,
  hidden,
  draftKey,
  initialBody,
  children,
}: Props) {
  const formRef = useRef<HTMLFormElement>(null)
  const [selection, setSelection] = useState({ audience: checked, timing: checkedTime })
  // ブラウザが戻したフォームの状態(再読み込み、戻る)も、画面に出ている選択として読む
  useEffect(() => {
    if (formRef.current) {
      setSelection(selectionOf(formRef.current))
    }
  }, [])
  const changed =
    !sameAudience(selection.audience, checked) || !sameTiming(selection.timing, checkedTime)
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
      <Hidden
        values={{ ...hidden, ...checkedFields(checked), ...checkedTimingFields(checkedTime) }}
      />
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
      <fieldset>
        <legend>いつ送るか</legend>
        <label className="adm-check">
          <input
            type="radio"
            name="timing"
            value="now"
            defaultChecked={checkedTime.kind === 'now'}
          />
          <span>すぐ送る</span>
        </label>
        <label className="adm-check adm-audience">
          <input
            type="radio"
            name="timing"
            value="later"
            defaultChecked={checkedTime.kind === 'later'}
          />
          <span>
            日時を決めて送る(予約)
            <small className="adm-muted">
              送る時刻に、そのときの条件で宛先を数え直して送ります。15
              分ほど遅れることがあります。送るまでは取りやめられます
            </small>
          </span>
        </label>
        <label className="adm-field">
          <span>送る日時(日本時間)</span>
          <input type="datetime-local" name="at" defaultValue={checkedTime.at} />
        </label>
      </fieldset>
      {changed ? (
        <p className="adm-notice error" role="status">
          宛先か送る時刻を変えました。「宛先と時刻を確かめる」を押すと、数え直します(書いた文は消えません)。
        </p>
      ) : null}
      <div className="adm-actions">
        <button type="submit" formAction={checkAudienceAction} formNoValidate>
          宛先と時刻を確かめる
        </button>
      </div>
      {children}
      <h2 className="adm-section-title">文</h2>
      <MessageDraft
        heading={heading}
        maxLength={maxNotificationBody}
        draftKey={draftKey}
        initialBody={initialBody}
      />
    </form>
  )
}
