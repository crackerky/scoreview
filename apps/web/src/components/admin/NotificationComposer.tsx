'use client'

import { type FormEvent, type ReactNode, useState } from 'react'
import {
  type Audience,
  audienceKinds,
  audienceStatuses,
  maxNotificationBody,
} from '@/api/notify.types'
import { sendNotificationAction } from '@/app/admin/_actions/notify'
import { audienceFromQuery, audienceKindLabels, audienceStatusLabels } from '@/notify/notify.utils'
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

function sameAudience(left: Audience, right: Audience): boolean {
  return (
    left.kind === right.kind && left.status === right.status && left.divisionId === right.divisionId
  )
}

// 送る画面の本体。上の宛先のフォーム(GET で人数を数え直す)と下の送るフォームは別のフォームなので、
// 宛先の選択を変えたまま送ると、数えた人数と違う宛先に送ってしまう。そこで、選択を変えたら知らせ、
// 送るフォームに印(changed)を付けて送る。Server Action は印を見たら送らずに、選び直した宛先で
// 数え直した画面に戻す(送る操作を画面で取り消すと、管理画面の「処理中」の表示と食い違うため)。
export function NotificationComposer({
  checked,
  divisions,
  heading,
  formId,
  hidden,
  draftKey,
  children,
}: Props) {
  const [selection, setSelection] = useState<Audience>(checked)
  const changed = !sameAudience(selection, checked)
  const onAudienceChange = (event: FormEvent<HTMLFormElement>) => {
    const values: Record<string, string> = {}
    for (const [key, value] of new FormData(event.currentTarget)) {
      if (typeof value === 'string') {
        values[key] = value
      }
    }
    setSelection(audienceFromQuery(values))
  }
  return (
    <>
      <h2 className="adm-section-title">宛先</h2>
      <form method="get" className="adm-stack" onChange={onAudienceChange}>
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
          <button type="submit">宛先を確かめる</button>
        </div>
      </form>
      {children}
      <h2 className="adm-section-title">文</h2>
      <form id={formId} action={sendNotificationAction} className="adm-stack">
        <Hidden
          values={{
            ...hidden,
            kind: selection.kind,
            status: selection.status,
            division: selection.divisionId === null ? '' : selection.divisionId,
            changed: changed ? '1' : '',
          }}
        />
        <MessageDraft heading={heading} maxLength={maxNotificationBody} draftKey={draftKey} />
      </form>
    </>
  )
}
