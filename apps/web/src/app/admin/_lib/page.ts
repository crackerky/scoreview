import { notFound } from 'next/navigation'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { type Notice, readNotice } from '@/components/admin/AdminParts'

export type SearchProps = {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}

/** /admin/…/[id] のページの id(数字でなければ 404)と通知を読む */
export async function readAdminPage(props: IdPageProps): Promise<{ id: number; notice: Notice }> {
  const [params, query] = await Promise.all([props.params, props.searchParams])
  if (!/^[0-9]+$/.test(params.id)) {
    notFound()
  }
  return { id: Number(params.id), notice: readNotice(query) }
}
