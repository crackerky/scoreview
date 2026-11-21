import { apiForward } from '@/api/client'

// 公式 LINE の Webhook の受け口(docs/line-integration.md「構成」)。api はインターネットに公開しないので、
// web が受けて本文と署名をそのまま api に渡す。署名の確かめと処理は api が行う。
export async function POST(request: Request) {
  const body = await request.arrayBuffer()
  const status = await apiForward('/api/v1/line/webhook', body, {
    'Content-Type': 'application/json',
    'X-Line-Signature': request.headers.get('x-line-signature') ?? '',
  })
  // LINE には 200 だけを成功として返す。api が失敗したら同じ状態を返し、LINE の送り直しに任せる
  return new Response(status === 200 ? 'ok' : 'error', { status })
}
