import { ApiError } from '@/api/client'
import { loadLiveData } from '@/live/liveData'

// 観戦アプリが一定の間隔で取り直すデータ(/live/{id}/data)。開催日は初回の描画で渡しているので
// ここでは読まない(大会一覧の api を毎回呼ばないため)
export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string }> },
): Promise<Response> {
  const { id } = await params
  if (!/^[0-9]+$/.test(id)) {
    return Response.json({ error: 'not found' }, { status: 404 })
  }
  try {
    return Response.json(await loadLiveData(Number(id), null))
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return Response.json({ error: 'not found' }, { status: 404 })
    }
    throw error
  }
}
