'use client'

import { useEffect, useRef, useState } from 'react'
import type { NotificationButtonInput } from '@/api/notify.types'
import {
  type DraftButtons,
  defaultButtonLabels,
  fillExample,
  noDraftButtons,
  usesVariables,
} from '@/notify/notify.utils'
import { LineBubble } from './LineBubble'

const buttonFields = ['buttonEntry', 'buttonSite', 'linkLabel', 'linkUrl'] as const

/** 欄の値を、届く形の見本のボタンにする(api と同じ順と文字。URL の確かめは api がする) */
function previewButtons(fields: DraftButtons): NotificationButtonInput[] {
  const out: NotificationButtonInput[] = []
  if (fields.buttonEntry) {
    out.push({ kind: 'entry', label: defaultButtonLabels.entry, url: '' })
  }
  if (fields.buttonSite) {
    out.push({ kind: 'site', label: defaultButtonLabels.site, url: '' })
  }
  if (fields.linkUrl.trim() !== '') {
    const label = fields.linkLabel.trim()
    out.push({
      kind: 'url',
      label: label === '' ? 'リンクの文字' : label,
      url: fields.linkUrl.trim(),
    })
  }
  return out
}

/** タブに残したボタンの欄を読む(無いか読めなければ null) */
function savedButtons(key: string): DraftButtons | null {
  try {
    if (window.sessionStorage.getItem(`${key}:buttonEntry`) === null) {
      return null
    }
    return {
      buttonEntry: window.sessionStorage.getItem(`${key}:buttonEntry`) === 'on',
      buttonSite: window.sessionStorage.getItem(`${key}:buttonSite`) === 'on',
      linkLabel: window.sessionStorage.getItem(`${key}:linkLabel`) ?? '',
      linkUrl: window.sessionStorage.getItem(`${key}:linkUrl`) ?? '',
    }
  } catch {
    // 保存が使えないブラウザでは、ボタンを残さないだけで送れる
    return null
  }
}

/** ボタンの欄をタブに残す(宛先を確かめるために画面を読み直しても消さない) */
function saveButtons(key: string, fields: DraftButtons) {
  try {
    for (const name of buttonFields) {
      const value = fields[name]
      window.sessionStorage.setItem(
        `${key}:${name}`,
        typeof value === 'boolean' ? (value ? 'on' : '') : value,
      )
    }
  } catch {
    // 保存が使えないブラウザでは、ボタンを残さないだけで送れる
  }
}

// 送る文の欄と、公式 LINE で届く形の見本。見本の 1 行目の大会名は api が付けるもの(heading)。
// 書いた文はブラウザのタブに残し(draftKey)、宛先を確かめるために画面を読み直しても消さない。
// initialBody(前日と当日の案内のひな形を選んで開いたとき)があれば、残した文の代わりにそれを入れる。
// 文の下のボタンの欄も同じように残す。ひな形で開いたときは、ひな形のボタン(initialButtons)にする。
// 差し込み(variables)は、押すと文の欄のカーソルの位置に {チーム名} などを入れ、見本では sampleName の
// 人の値に置き換えて見せる。
export function MessageDraft({
  heading,
  maxLength,
  draftKey,
  initialBody = '',
  initialButtons = noDraftButtons,
  variables = [],
  sampleName = '',
}: {
  heading: string
  maxLength: number
  draftKey: string
  initialBody?: string
  initialButtons?: DraftButtons
  variables?: ReadonlyArray<{ name: string; example: string }>
  sampleName?: string
}) {
  const textRef = useRef<HTMLTextAreaElement>(null)
  const [body, setBody] = useState(initialBody)
  const [buttons, setButtons] = useState(initialButtons)
  useEffect(() => {
    if (initialBody !== '') {
      saveButtons(draftKey, initialButtons)
      try {
        window.sessionStorage.setItem(draftKey, initialBody)
      } catch {
        // 保存が使えないブラウザ(プライベートの窓など)では、文を残さないだけで送れる
      }
      return
    }
    const kept = savedButtons(draftKey)
    if (kept !== null) {
      setButtons(kept)
    }
    try {
      const saved = window.sessionStorage.getItem(draftKey)
      if (saved !== null) {
        setBody(saved)
      }
    } catch {
      // 保存が使えないブラウザ(プライベートの窓など)では、文を残さないだけで送れる
    }
  }, [draftKey, initialBody, initialButtons])
  const changeButtons = (next: DraftButtons) => {
    setButtons(next)
    saveButtons(draftKey, next)
  }
  const change = (value: string) => {
    setBody(value)
    try {
      window.sessionStorage.setItem(draftKey, value)
    } catch {
      // 保存が使えないブラウザでは、文を残さないだけで送れる
    }
  }
  // 押した差し込みをカーソルの位置に入れ、カーソルを入れた後ろに置く(続けて書けるように)
  const insert = (name: string) => {
    const field = textRef.current
    const token = `{${name}}`
    const start = field?.selectionStart ?? body.length
    const end = field?.selectionEnd ?? body.length
    change(body.slice(0, start) + token + body.slice(end))
    window.requestAnimationFrame(() => {
      field?.focus()
      field?.setSelectionRange(start + token.length, start + token.length)
    })
  }
  const length = [...body].length
  const written = body.trim() === '' ? 'ここに送る文が入ります' : body.trim()
  const personal =
    sampleName !== '' &&
    usesVariables(
      written,
      variables.map((variable) => variable.name),
    )
  return (
    <div className="adm-message">
      <label className="adm-field">
        <span>送る文</span>
        <textarea
          ref={textRef}
          name="body"
          rows={8}
          maxLength={maxLength}
          required
          value={body}
          onChange={(event) => change(event.target.value)}
        />
      </label>
      <p className="adm-muted">
        {length} / {maxLength} 字。1 行目に大会名を付けて送ります。
      </p>
      {variables.length === 0 ? null : (
        <>
          <div className="adm-variables">
            <span>差し込む</span>
            {variables.map((variable) => (
              <button
                key={variable.name}
                type="button"
                className="secondary"
                onClick={() => insert(variable.name)}
              >
                {variable.name}
              </button>
            ))}
          </div>
          <p className="adm-muted">送る人ごとに、その人のチームの値に置き換えて送ります。</p>
        </>
      )}
      <fieldset>
        <legend>文の下のボタン(押すと開くもの。3 つまで)</legend>
        <label className="adm-check adm-audience">
          <input
            type="checkbox"
            name="buttonEntry"
            checked={buttons.buttonEntry}
            onChange={(event) => changeButtons({ ...buttons, buttonEntry: event.target.checked })}
          />
          <span>
            {defaultButtonLabels.entry}
            <small className="adm-muted">
              押した人に、その人の申込のリンクが公式 LINE から届きます
            </small>
          </span>
        </label>
        <label className="adm-check adm-audience">
          <input
            type="checkbox"
            name="buttonSite"
            checked={buttons.buttonSite}
            onChange={(event) => changeButtons({ ...buttons, buttonSite: event.target.checked })}
          />
          <span>
            {defaultButtonLabels.site}
            <small className="adm-muted">この大会のスコアサイトのページを開きます</small>
          </span>
        </label>
        <div className="adm-grid">
          <label className="adm-field">
            <span>リンクのボタンの文字(20 字まで)</span>
            <input
              type="text"
              name="linkLabel"
              maxLength={20}
              placeholder="会場の地図"
              value={buttons.linkLabel}
              onChange={(event) => changeButtons({ ...buttons, linkLabel: event.target.value })}
            />
          </label>
          <label className="adm-field">
            <span>リンクの URL(https で始まるもの)</span>
            <input
              type="url"
              name="linkUrl"
              inputMode="url"
              pattern="https://.+"
              title="https で始まる URL を入れてください"
              placeholder="https://"
              value={buttons.linkUrl}
              onChange={(event) => changeButtons({ ...buttons, linkUrl: event.target.value })}
            />
          </label>
        </div>
      </fieldset>
      <figure className="adm-bubble-preview">
        <figcaption>{personal ? `届く形(${sampleName}さんの場合)` : '届く形'}</figcaption>
        <LineBubble
          text={`${heading}\n\n${fillExample(written, variables)}`}
          buttons={previewButtons(buttons)}
        />
      </figure>
    </div>
  )
}

/** 送り終えた画面で、送る画面に残した文を消す(次に送る画面を開いたとき、送った文が残らないように) */
export function ForgetDraft({ draftKey }: { draftKey: string }) {
  useEffect(() => {
    try {
      window.sessionStorage.removeItem(draftKey)
      for (const name of buttonFields) {
        window.sessionStorage.removeItem(`${draftKey}:${name}`)
      }
    } catch {
      // 保存が使えないブラウザでは、もともと何も残っていない
    }
  }, [draftKey])
  return null
}
