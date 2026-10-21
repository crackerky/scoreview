import type { CSSProperties } from 'react'

// dangerouslySetInnerHTML を使ってよい唯一のファイル(biome.json の override で許可)。
// html は api がサーバー側で許可リスト方式の無害化を済ませた値だけを渡す(大会トップの topHtml など)。
// web 側で無害化をやり直さないので、利用者の入力や外部の文字列をそのまま渡さないこと。

type SanitizedHtmlProps = {
  /** api が無害化済みとして返した HTML */
  html: string
  id?: string
  className?: string
  style?: CSSProperties
}

export function SanitizedHtml({ html, id, className, style }: SanitizedHtmlProps) {
  return (
    <div id={id} className={className} style={style} dangerouslySetInnerHTML={{ __html: html }} />
  )
}
