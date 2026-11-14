import { loadAdmin, requireUser } from '@/admin/session'
import { entries } from '@/api/entry'
import { entriesCsv, entriesCsvName } from '@/entry/entryCsv.utils'

// 申込の書き出し(CSV)。管理者だけ(代表の連絡先が入るため)。ブラウザに保存させ、使い回させない。
export async function GET(_request: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id: raw } = await ctx.params
  if (!/^[0-9]+$/.test(raw)) {
    return new Response('not found', { status: 404 })
  }
  const id = Number(raw)
  const { token } = await requireUser('admin')
  const data = await loadAdmin(() => entries.export(token, id))
  return new Response(entriesCsv(data), {
    headers: {
      'Content-Type': 'text/csv; charset=utf-8',
      'Content-Disposition': `attachment; filename="${entriesCsvName(id, new Date())}"`,
      'Cache-Control': 'private, no-store',
    },
  })
}
