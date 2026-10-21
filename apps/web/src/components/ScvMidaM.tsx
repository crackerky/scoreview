import type { ReactNode } from 'react'

/**
 * 橙のバッジ見出し span.SCV_MIDA_M(clone-spec A 2.2、C 0)。
 * "< 文言 >" の山括弧は呼び出し側で angleHeading() を使って付ける(jp と en で空白が違うため)。
 */
export function ScvMidaM({ children }: { children: ReactNode }) {
  return <span className="SCV_MIDA_M">{children}</span>
}
