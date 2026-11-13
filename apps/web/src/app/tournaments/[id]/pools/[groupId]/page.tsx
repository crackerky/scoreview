import { notFound } from 'next/navigation'
import { parseMatchId as parsePathId } from '@/admin/relay.utils'
import { MatchList } from '@/components/score/MatchList'
import { PoolCard } from '@/components/score/PoolCard'
import { requireScoreBoard } from '@/score/score.load'
import { scorePaths } from '@/score/score.paths'
import { poolPage } from '@/score/score.utils'

// 組の画面。星取表と、組の試合の一覧(時刻、コート、点)を出す。

/** ビルドでは描かない(大会の外枠と同じ。最初に開かれたときに描いて使い回す) */
export function generateStaticParams(): Array<{ groupId: string }> {
  return []
}

export default async function PoolPage({
  params,
}: {
  params: Promise<{ id: string; groupId: string }>
}) {
  const { id: rawId, groupId: rawGroup } = await params
  const { board, data } = await requireScoreBoard(rawId)
  const groupId = parsePathId(rawGroup)
  const page = groupId === null ? null : poolPage(data, groupId)
  if (!page) {
    notFound()
  }
  const id = board.tournament.id
  return (
    <>
      <title>{`${page.pool.name} - ${board.tournament.name.ja}`}</title>
      <a className="sc-back" href={scorePaths.top(id)}>
        <span aria-hidden="true">‹ </span>スコア
      </a>
      <h2 className="sc-heading">{page.pool.name}</h2>
      <PoolCard tournamentId={id} table={page.table} config={data.tables.config} linked={false} />
      <section className="sc-section">
        <h3 className="sc-subheading">この組の試合</h3>
        <MatchList tournamentId={id} matches={page.matches} />
      </section>
    </>
  )
}
