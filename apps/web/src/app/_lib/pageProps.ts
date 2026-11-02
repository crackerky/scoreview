/** /…/[id] のページが受け取る props(Next.js 16 では params と searchParams は Promise)。 */
export type IdPageProps = {
  params: Promise<{ id: string }>
  searchParams: Promise<Record<string, string | string[] | undefined>>
}
