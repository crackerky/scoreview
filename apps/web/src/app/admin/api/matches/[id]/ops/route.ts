import { relayOp } from '@/admin/scoreRelay'

// 記録係の画面が端末に溜めた操作を 1 つずつ送る(Cookie のセッションで api へ中継する)。
export async function POST(request: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params
  return relayOp(request, id)
}
