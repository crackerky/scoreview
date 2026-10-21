import { type Lang, parseLang } from '@/i18n/lang'

/** /…/[id] のページが受け取る props(Next.js 16 では params と searchParams は Promise)。 */
export type IdPageProps = {
  params: Promise<{ id: string }>
  searchParams: Promise<Record<string, string | string[] | undefined>>
}

/** URL の id(文字列のまま。数値の検証は loadPageData が行う)と lang を取り出す。 */
export async function readIdPage(props: IdPageProps): Promise<{ id: string; lang: Lang }> {
  const [params, query] = await Promise.all([props.params, props.searchParams])
  return { id: params.id, lang: parseLang(query.lang) }
}
