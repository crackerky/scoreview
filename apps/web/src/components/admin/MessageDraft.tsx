'use client'

import { useEffect, useState } from 'react'

// 送る文の欄と、公式 LINE で届く形の見本。見本の 1 行目の大会名は api が付けるもの(heading)。
// 書いた文はブラウザのタブに残し(draftKey)、宛先を確かめるために画面を読み直しても消さない。
// initialBody(前日と当日の案内のひな形を選んで開いたとき)があれば、残した文の代わりにそれを入れる。
export function MessageDraft({
  heading,
  maxLength,
  draftKey,
  initialBody = '',
}: {
  heading: string
  maxLength: number
  draftKey: string
  initialBody?: string
}) {
  const [body, setBody] = useState(initialBody)
  useEffect(() => {
    try {
      if (initialBody !== '') {
        window.sessionStorage.setItem(draftKey, initialBody)
        return
      }
      const saved = window.sessionStorage.getItem(draftKey)
      if (saved !== null) {
        setBody(saved)
      }
    } catch {
      // 保存が使えないブラウザ(プライベートの窓など)では、文を残さないだけで送れる
    }
  }, [draftKey, initialBody])
  const change = (value: string) => {
    setBody(value)
    try {
      window.sessionStorage.setItem(draftKey, value)
    } catch {
      // 保存が使えないブラウザでは、文を残さないだけで送れる
    }
  }
  const length = [...body].length
  return (
    <div className="adm-message">
      <label className="adm-field">
        <span>送る文</span>
        <textarea
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
      <figure className="adm-bubble-preview">
        <figcaption>届く形</figcaption>
        <p className="adm-bubble">
          {heading}
          {'\n\n'}
          {body.trim() === '' ? 'ここに送る文が入ります' : body.trim()}
        </p>
      </figure>
    </div>
  )
}

/** 送り終えた画面で、送る画面に残した文を消す(次に送る画面を開いたとき、送った文が残らないように) */
export function ForgetDraft({ draftKey }: { draftKey: string }) {
  useEffect(() => {
    try {
      window.sessionStorage.removeItem(draftKey)
    } catch {
      // 保存が使えないブラウザでは、もともと何も残っていない
    }
  }, [draftKey])
  return null
}
