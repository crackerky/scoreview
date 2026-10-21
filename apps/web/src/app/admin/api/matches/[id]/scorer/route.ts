import { relayScorer } from '@/admin/scoreRelay'

// 記録画面を開いた端末を、試合の記録担当にする・延長する・引き継ぐ・外す(Cookie のセッションで
// api へ中継する)。
export async function POST(request: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params
  return relayScorer(request, id)
}
