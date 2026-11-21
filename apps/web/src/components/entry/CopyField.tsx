'use client'

import { useRef, useState } from 'react'

// 文(招待リンクなど)を見せて、ボタンでコピーする欄。LINE のアプリの中のブラウザのように、ページから
// クリップボードに書けないところでは、欄の文を選んだ状態にして、長押しでコピーしてもらう。

type Props = {
  label: string
  value: string
}

type Result = 'idle' | 'copied' | 'failed'

export function CopyField({ label, value }: Props) {
  const ref = useRef<HTMLInputElement>(null)
  const [result, setResult] = useState<Result>('idle')
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setResult('copied')
    } catch {
      // 書けないブラウザでは、選んだ状態にして手でコピーしてもらう(失敗は画面の案内で伝える)
      ref.current?.focus()
      ref.current?.select()
      setResult('failed')
    }
  }
  return (
    <div className="entry-field entry-copy">
      <span>{label}</span>
      <div className="entry-copy-row">
        <input ref={ref} type="text" value={value} readOnly aria-label={label} />
        <button type="button" className="entry-copy-button" onClick={copy}>
          {result === 'copied' ? 'コピーしました' : 'コピー'}
        </button>
      </div>
      {result === 'failed' ? (
        <p className="entry-muted" role="status">
          ここではコピーできませんでした。選んだ文を長押しして、コピーしてください。
        </p>
      ) : null}
    </div>
  )
}
