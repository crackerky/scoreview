import type { LiveTablesConfig } from '@/api/public.types'
import { PoolMatrixTable } from '@/components/tables/PoolMatrixTable'
import type { PoolMatrix } from '@/live/live.tables'
import { scorePaths } from '@/score/score.paths'

// 組 1 つの星取表。見出しを押すと組の画面へ、試合のマスを押すと試合の画面へ移る。組の画面では
// 見出しのリンクを付けない(linked が偽)。

export function PoolCard({
  tournamentId,
  table,
  config,
  linked = true,
}: {
  tournamentId: number
  table: PoolMatrix
  config: LiveTablesConfig
  linked?: boolean
}) {
  return (
    <article className="sc-pool">
      {linked ? (
        <a className="sc-pool-head" href={scorePaths.pool(tournamentId, table.id)}>
          <span className="sc-pool-name">{table.name}</span>
          <span className="sc-more">
            詳しく<span aria-hidden="true"> ›</span>
          </span>
        </a>
      ) : null}
      <div className="sc-paper">
        <PoolMatrixTable
          table={table}
          config={config}
          cellHref={(matchId) => scorePaths.match(tournamentId, matchId)}
        />
      </div>
    </article>
  )
}
