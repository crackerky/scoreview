// 観客が数秒おきに取り直す公開データの応答。Vercel の配信(CDN)で短く使い回し、同じ大会を見ている
// 人のあいだで関数と api の呼び出しを 1 回にまとめる。ブラウザには next.config.ts の no-cache のまま
// 毎回確かめさせる(Vercel-CDN-Cache-Control は Vercel の配信だけが読み、ブラウザへは渡らない)。

/** 配信が使い回す秒数。観客の画面の取り直し(15 秒)より十分短く、点が見えるまでの遅れが気にならない長さ */
const sharedSeconds = 5

/** 使い回しの期限が切れたあと、裏で取り直すあいだ古い応答を返してよい秒数 */
const staleSeconds = 10

/** 観客のあいだで使い回してよい公開データの JSON 応答 */
export function sharedJson(body: unknown): Response {
  return Response.json(body, {
    headers: {
      'Vercel-CDN-Cache-Control': `max-age=${sharedSeconds}, stale-while-revalidate=${staleSeconds}`,
    },
  })
}
