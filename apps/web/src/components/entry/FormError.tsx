'use client'

import { useEffect, useRef } from 'react'

// 公開側のフォームのエラーの文。送信ボタンはフォームの下にあり、文はフォームの先頭に出るので、
// スマートフォンでは押した人の画面の外になる。文が出るたびに、文まで画面を動かしてフォーカスを移す
// (読み上げでも、押した後に何が起きたかが伝わるように)。

type Props = {
  /** api の文。null なら何も出さない */
  error: string | null
  /** 送信の結果ごとに変わる値(同じ文が続けて出たときも、もう一度動かすため) */
  attempt: object
}

export function FormError({ error, attempt }: Props) {
  const ref = useRef<HTMLParagraphElement>(null)
  useEffect(() => {
    // attempt は送信の結果が変わったことを知らせるだけで、中身は使わない
    void attempt
    if (error !== null) {
      ref.current?.scrollIntoView({ block: 'center' })
      ref.current?.focus()
    }
  }, [error, attempt])
  return error === null ? null : (
    <p ref={ref} className="entry-notice error" role="alert" tabIndex={-1}>
      {error}
    </p>
  )
}
