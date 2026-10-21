import type { I18nText } from '@/api/common.types'

/** 表示言語。参照サイトの ?lang= の値をそのまま使う(clone-spec A 0)。 */
export type Lang = 'jp' | 'en'

/**
 * URL の lang クエリを読む。参照サイトと同じく en 以外(省略、不正値、jp)はすべて jp。
 * Next.js の searchParams は同じキーが複数あると配列になるので、先頭の値を使う。
 */
export function parseLang(value: string | string[] | undefined): Lang {
  const first = Array.isArray(value) ? value[0] : value
  return first === 'en' ? 'en' : 'jp'
}

/** I18nText から言語に応じた文字列を選ぶ。英語が空でも日本語で補わない(requirements「言語」)。 */
export function pickText(lang: Lang, text: I18nText): string {
  return lang === 'en' ? text.en : text.ja
}
