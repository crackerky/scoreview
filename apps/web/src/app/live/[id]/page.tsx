import { notFound } from 'next/navigation'
import { ApiError } from '@/api/client'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { LiveApp } from '@/components/live/LiveApp'
import { loadLiveData, loadSwitchTournaments } from '@/live/liveData'

// 観戦アプリ(Claude Design「StageCam」)。参照サイトの模倣ページとは別の、本サイト独自のスマートフォン
// 向けの観戦の画面。今日の試合をカードの輪で見せ、試合を開くと大きなスコアと得点の流れを出す。
// 初回はサーバーで描き、そのあとはブラウザが /live/{id}/data を一定の間隔で取り直す。
export default async function LivePage({ params, searchParams }: IdPageProps) {
  const [{ id: raw }, query] = await Promise.all([params, searchParams])
  if (!/^[0-9]+$/.test(raw)) {
    notFound()
  }
  const id = Number(raw)
  const { list, dates } = await loadSwitchTournaments()
  let data: Awaited<ReturnType<typeof loadLiveData>>
  try {
    data = await loadLiveData(id, dates.get(id) ?? null)
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      notFound()
    }
    throw error
  }
  return (
    <>
      <title>{`${data.tournament.name} - 観戦アプリ`}</title>
      <LiveApp
        initial={data}
        tournaments={list}
        initialMatch={
          typeof query.match === 'string' && /^[0-9]+$/.test(query.match)
            ? Number(query.match)
            : null
        }
      />
    </>
  )
}
