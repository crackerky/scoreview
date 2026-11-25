import { notifications } from '@/api/notify'
import { isCronRequest } from '@/notify/cron.utils'

// 予約した送信を送る定期の呼び出しの受け口(vercel.json の schedules が 15 分ごとに POST する。
// docs/line-integration.md「管理画面から送る仕組み」)。api はインターネットに公開しないので、web が
// CRON_SECRET を確かめてから api に渡す。CRON_SECRET が無ければ誰の呼び出しも受けない。
export async function POST(request: Request) {
  if (!isCronRequest(request.headers.get('authorization'), process.env.CRON_SECRET)) {
    return new Response('unauthorized', { status: 401 })
  }
  const result = await notifications.dispatch()
  return Response.json(result)
}
