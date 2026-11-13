import { PoolCard } from '@/components/score/PoolCard'
import { columnCount, matrixLegend } from '@/live/live.tables'
import { requireScoreBoard } from '@/score/score.load'
import { scorePaths } from '@/score/score.paths'
import { scoreDivisions } from '@/score/score.utils'

// スコアサイトの入口。全部の組の星取表を部門ごとに並べ、どの組に誰がいて、どこが勝ち上がるかを
// 1 画面で見せる。組を押すと組の画面、マスを押すと試合の画面へ移る。

export default async function ScoreTopPage({ params }: { params: Promise<{ id: string }> }) {
  const { board, data } = await requireScoreBoard((await params).id)
  const id = board.tournament.id
  const divisions = scoreDivisions(board, data)
  const config = data.tables.config
  return (
    <>
      <title>{`${board.tournament.name.ja} - スコア`}</title>
      {divisions.length === 0 ? (
        <p className="sc-empty">
          組み合わせはまだ決まっていません。
          {board.matches.length > 0 ? (
            <>
              {' '}
              <a href={scorePaths.schedule(id)}>タイムスケジュールを見る</a>
            </>
          ) : null}
        </p>
      ) : null}
      {divisions.length > 1 ? (
        <nav className="sc-chips" aria-label="部門">
          {divisions.map((division) => (
            <a key={division.id} href={`#division-${division.id}`}>
              {division.name}
            </a>
          ))}
        </nav>
      ) : null}
      {divisions.map((division) => (
        <section key={division.id} id={`division-${division.id}`} className="sc-section">
          <h2 className="sc-heading">{division.name}</h2>
          <div className="sc-pools">
            {division.pools.map((table) => (
              <PoolCard key={table.id} tournamentId={id} table={table} config={config} />
            ))}
          </div>
          {division.brackets.length > 0 ? (
            <a className="sc-next" href={`${scorePaths.bracket(id)}#division-${division.id}`}>
              {division.pools.length > 0 ? '決勝トーナメントを見る' : 'トーナメントを見る'}
              <span aria-hidden="true"> ›</span>
            </a>
          ) : null}
        </section>
      ))}
      {divisions.some((division) => division.pools.length > 0) ? (
        <p className="sc-legend">{matrixLegend(config, columnCount(config))}</p>
      ) : null}
    </>
  )
}
