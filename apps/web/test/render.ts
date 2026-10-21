import type { ReactElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

/**
 * 要素を静的 HTML にする。React 19 が先頭に足す画像の <link rel="preload"> は
 * Next.js では <head> に移るものなので、本文の構造を検証しやすいよう取り除く。
 */
export function renderHtml(element: ReactElement): string {
  return renderToStaticMarkup(element).replace(/<link rel="preload"[^>]*\/>/g, '')
}
