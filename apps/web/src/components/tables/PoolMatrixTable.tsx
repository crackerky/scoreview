import type { LiveTablesConfig } from '@/api/public.types'
import {
  type ColumnKey,
  columnDefs,
  type MatrixCell,
  type PoolMatrix,
  type TablePart,
} from '@/live/live.tables'

// 星取表 1 組分。観戦アプリの「表で見る」、管理画面「観戦アプリの表」の見本、スコアサイトで共有する。
// 見本(edit)では押した部分を枠で示し、出さない列も薄く残して見出しで出し入れできるようにする。

export type TableEdit = {
  selected: TablePart | null
  onPick: (part: TablePart) => void
  onToggleColumn: (key: ColumnKey) => void
}

function ring(edit: TableEdit | undefined, part: TablePart): string {
  return edit?.selected === part ? ' lt-ring' : ''
}

export function PoolMatrixTable({
  table,
  config,
  edit,
  cellHref,
}: {
  table: PoolMatrix
  config: LiveTablesConfig
  edit?: TableEdit
  /** 試合のマスの行き先。無ければマスは押せない */
  cellHref?: (matchId: number) => string
}) {
  // 見本では出さない列も薄く残す
  const columns = columnDefs.filter((def) => edit || config.columns[def.key])
  const template = `92px repeat(${table.heads.length}, 56px)${
    columns.length > 0
      ? ` repeat(${columns.reduce((sum, def) => sum + def.heads.length, 0)}, 34px)`
      : ''
  }`
  return (
    <div className="lt-pool">
      <div className="lt-scroll" data-matrix="">
        <div className="lt-grid" style={{ gridTemplateColumns: template }}>
          <div className="lt-corner">{table.name.replace(/^予選 /, '')}</div>
          {table.heads.map((head, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: 同じ名前のチームがありうるので位置で区別する
            <div key={index} className="lt-head">
              <span>{head}</span>
            </div>
          ))}
          {columns.flatMap((def) =>
            def.heads.map((head) => (
              <div
                key={`${def.key}-${head}`}
                className={`lt-stat-head${ring(edit, def.key === 'rank' ? 'hl' : 'cols')}`}
                data-col={def.key}
                style={{ opacity: config.columns[def.key] ? 1 : 0.3 }}
              >
                {edit ? (
                  <span className="lt-toggle">{config.columns[def.key] ? '✓' : '＋'}</span>
                ) : null}
                <span>{head}</span>
              </div>
            )),
          )}
          {table.rows.map((row, rowIndex) => {
            const last = rowIndex === table.rows.length - 1 ? ' lt-last' : ''
            return [
              <div
                key={`name-${row.teamId}`}
                className={`lt-name${last}${ring(edit, 'order')}`}
                data-part="order"
              >
                <span className="lt-bar" style={{ background: row.color }} />
                <span>{row.name}</span>
              </div>,
              ...row.cells.map((cell, index) => (
                <Cell
                  // biome-ignore lint/suspicious/noArrayIndexKey: マスの位置(相手の列)が識別子
                  key={`cell-${row.teamId}-${index}`}
                  cell={cell}
                  className={last}
                  edit={edit}
                  href={cellHref}
                />
              )),
              ...columns.flatMap((def) =>
                row.stats[def.key].map((value, index) => (
                  <div
                    // biome-ignore lint/suspicious/noArrayIndexKey: 列の中の位置(勝、分、敗)が識別子
                    key={`stat-${row.teamId}-${def.key}-${index}`}
                    className={`lt-stat lt-stat-${def.key}${last}${ring(
                      edit,
                      def.key === 'rank' ? 'hl' : 'cols',
                    )}`}
                    data-part={def.key === 'rank' ? 'hl' : 'cols'}
                    style={{
                      opacity: config.columns[def.key] ? 1 : 0.3,
                      background: def.key === 'rank' && row.advancing ? '#ffe3c2' : undefined,
                    }}
                  >
                    {value}
                  </div>
                )),
              ),
            ]
          })}
        </div>
      </div>
      <span className="lt-status">{table.status}</span>
    </div>
  )
}

function Cell({
  cell,
  className,
  edit,
  href,
}: {
  cell: MatrixCell
  className: string
  edit?: TableEdit
  href?: (matchId: number) => string
}) {
  if (cell.kind === 'self' || cell.kind === 'none') {
    return <div className={`lt-cell${cell.kind === 'self' ? ' lt-self' : ''}${className}`} />
  }
  const part = cell.kind === 'done' ? 'mark' : cell.kind === 'live' ? 'live' : undefined
  const classes = `lt-cell${cell.kind === 'live' ? ' lt-live' : ''}${className}${
    part ? ring(edit, part) : ''
  }`
  const content =
    cell.kind === 'done' ? (
      <>
        <span className="lt-mark">{cell.mark}</span>
        <span className={cell.lost ? 'lt-score lt-lost' : 'lt-score'}>{cell.score}</span>
      </>
    ) : cell.kind === 'live' ? (
      <>
        <span className="lt-info">● 試合中</span>
        <span className="lt-score">{cell.score}</span>
      </>
    ) : (
      <>
        <span className="lt-info lt-muted">{cell.court}</span>
        <span className="lt-score lt-muted">{cell.time}</span>
      </>
    )
  // 見本(edit)では押すと見せ方を選ぶので、試合へは移らない
  if (href && !edit) {
    return (
      <a className={`${classes} lt-link`} href={href(cell.matchId)}>
        {content}
      </a>
    )
  }
  return (
    <div className={classes} data-part={part}>
      {content}
    </div>
  )
}
