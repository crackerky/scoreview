'use client'

import { useState } from 'react'

// 送る文の欄と、公式 LINE で届く形の見本。見本の 1 行目の大会名は api が付けるもの(heading)。
export function MessageDraft({ heading, maxLength }: { heading: string; maxLength: number }) {
  const [body, setBody] = useState('')
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
          onChange={(event) => setBody(event.target.value)}
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
