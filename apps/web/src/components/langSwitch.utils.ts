import type { Lang } from '@/i18n/lang'

/**
 * 今のページ(pathname と、? を除いたクエリ文字列)の lang だけを target に替えた URL。
 * クエリの他の値と並びは保つ(lang が無ければ末尾に足す)。
 */
export function langSwitchHref(pathname: string, query: string, target: Lang): string {
  const params = new URLSearchParams(query)
  params.set('lang', target)
  return `${pathname}?${params.toString()}`
}
