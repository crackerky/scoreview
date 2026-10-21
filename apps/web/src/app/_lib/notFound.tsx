import { notFound } from 'next/navigation'
import { cache } from 'react'
import { ApiError } from '@/api/client'
import { NotFoundScreen, RouteNotFoundScreen } from '@/components/ErrorScreen'

// 存在しない ID のエラー画面(clone-spec A 7)を HTTP 404 で出すための仕組み。
//
// not-found.tsx は props を受け取れないので、api のエラー message("TOURNAMENT_ID [999999]")を
// 同じリクエストの描画の中で受け渡す。React の cache() はリクエストごとに 1 つの値を返すので、
// ページが「なぜ 404 か」を settle し、同じセグメントの not-found.tsx がそれを待って読む。
// Next.js は not-found.tsx の要素をページと同じ描画の中で先に作るため、ページは成功・失敗を問わず
// 必ず settle する(しないと not-found.tsx が待ち続けて応答が終わらない)。
// 使う側の規約: not-found.tsx で NotFoundFromRequest を使うセグメントの page.tsx は、
// 必ず loadPageData を通して api を呼ぶ。

export type NotFoundReason = { kind: 'api'; message: string } | { kind: 'route' }

type Slot = {
  promise: Promise<NotFoundReason | null>
  settle: (reason: NotFoundReason | null) => void
}

const requestSlot = cache((): Slot => {
  const { promise, resolve } = Promise.withResolvers<NotFoundReason | null>()
  return { promise, settle: resolve }
})

const numericId = /^[0-9]+$/

/**
 * URL の id を数値として検証し、load で api を 1 回呼ぶ。
 * id が数値でなければルート不一致の 404、api が not_found なら message つきの 404 にする。
 * それ以外のエラーはそのまま投げる(Next.js のエラー境界へ流す)。
 */
export async function loadPageData<T>(rawId: string, load: (id: number) => Promise<T>): Promise<T> {
  const slot = requestSlot()
  if (!numericId.test(rawId)) {
    slot.settle({ kind: 'route' })
    notFound()
  }
  let data: T
  try {
    data = await load(Number(rawId))
  } catch (error) {
    if (error instanceof ApiError && error.code === 'not_found') {
      slot.settle({ kind: 'api', message: error.message })
      notFound()
    }
    slot.settle(null)
    throw error
  }
  slot.settle(null)
  return data
}

/** not-found.tsx から使う。ページが settle した理由に応じた画面を出す。 */
export async function NotFoundFromRequest({ withMenuLink }: { withMenuLink: boolean }) {
  const reason = await requestSlot().promise
  if (reason === null) {
    // ページが正常に描画された(この要素は表示されない)か、404 以外のエラーで終わった
    return null
  }
  if (reason.kind === 'route') {
    return <RouteNotFoundScreen />
  }
  return <NotFoundScreen message={reason.message} withMenuLink={withMenuLink} />
}
