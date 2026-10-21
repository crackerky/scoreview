import { relayDetail } from '@/admin/scoreRelay'

// 記録係の画面が最新の試合の状態を取り直す(Cookie のセッションで api へ中継する)。
export async function GET(_request: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params
  return relayDetail(id)
}
