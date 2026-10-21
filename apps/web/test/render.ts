import type { ReactElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

// 読み上げ用の補足(ScreenReaderNote。画面には出ない本サイト独自の追記)。参照サイトの再現を確かめる
// テストでは、比較ツール(apps/api/internal/parity)と同じく取り除いてから比べる
const screenReaderNote = /<span class="sv_sr" data-scoreview-addition="">[^<]*<\/span>/g

/**
 * 要素を静的 HTML にする(読み上げ用の補足を含む)。React 19 が先頭に足す画像の <link rel="preload"> は
 * Next.js では <head> に移るものなので、本文の構造を検証しやすいよう取り除く。
 */
export function renderHtmlWithNotes(element: ReactElement): string {
  return renderToStaticMarkup(element).replace(/<link rel="preload"[^>]*\/>/g, '')
}

/** renderHtmlWithNotes から読み上げ用の補足を除いたもの(参照サイトの再現の検証に使う)。 */
export function renderHtml(element: ReactElement): string {
  return renderHtmlWithNotes(element).replace(screenReaderNote, '')
}
