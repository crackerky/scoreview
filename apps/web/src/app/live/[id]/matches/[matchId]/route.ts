import { loadLiveMatch } from '@/live/liveData'

// 観戦アプリで試合を大きく開いたときの得点の流れ(/live/{id}/matches/{matchId})
export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string; matchId: string }> },
): Promise<Response> {
  const { id, matchId } = await params
  if (!/^[0-9]+$/.test(id) || !/^[0-9]+$/.test(matchId)) {
    return Response.json({ error: 'not found' }, { status: 404 })
  }
  const detail = await loadLiveMatch(Number(id), Number(matchId))
  return detail === null
    ? Response.json({ error: 'not found' }, { status: 404 })
    : Response.json(detail)
}
