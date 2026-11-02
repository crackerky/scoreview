'use client'

import type { MouseEvent, ReactNode } from 'react'
import type { LiveTablesConfig } from '@/api/public.types'
import {
  type BracketEntry,
  type BracketTree,
  bracketViews,
  type ColumnKey,
  columnCount,
  columnDefs,
  depthOf,
  leafCount,
  type MatrixCell,
  matrixLegend,
  poolMatrix,
  type TablePart,
} from '@/live/live.tables'
import type { LiveData } from '@/live/live.types'
import { winnerOf } from '@/live/live.utils'

// 観戦アプリの「表で見る」の中身(星取表とトーナメント表。Claude Design「Score View Admin」の 2b)。
// 観戦アプリの全画面の表と、管理画面「観戦アプリの表」の見本の両方で使う。見本(edit)では、押した
// 部分を選び、出さない列も薄く残して見出しで出し入れできるようにする。

export type TableKind = 'pool' | 'ko'

export type TableEdit = {
  selected: TablePart | null
  onPick: (part: TablePart) => void
  onToggleColumn: (key: ColumnKey) => void
}

/** 出せる表の種類(見せ方で出すもの、かつ中身のあるもの)。並びは星取表、トーナメント表 */
export function tableKinds(data: LiveData, config: LiveTablesConfig): TableKind[] {
  const kinds: TableKind[] = []
  if (config.poolShown && data.pools.length > 0) {
    kinds.push('pool')
  }
  if (config.bracketShown && bracketViews(data.tables.brackets, data.matches).length > 0) {
    kinds.push('ko')
  }
  return kinds
}

export const tableNames: Record<TableKind, string> = { pool: '星取表', ko: 'トーナメント表' }

export function TablesPanel({
  data,
  config,
  kind,
  edit,
}: {
  data: LiveData
  config: LiveTablesConfig
  kind: TableKind
  edit?: TableEdit
}) {
  // 見本では、押した場所の data-part / data-col から選ぶ部分を決める(右の欄のボタンでも選べる)
  const pick = (event: MouseEvent<HTMLDivElement>) => {
    if (!edit || !(event.target instanceof Element)) {
      return
    }
    const column = event.target.closest('[data-col]')?.getAttribute('data-col')
    const known = columnDefs.find((def) => def.key === column)
    if (known) {
      edit.onToggleColumn(known.key)
      edit.onPick('cols')
      return
    }
    const part = event.target.closest('[data-part]')?.getAttribute('data-part')
    if (
      part === 'cols' ||
      part === 'mark' ||
      part === 'live' ||
      part === 'order' ||
      part === 'hl'
    ) {
      edit.onPick(part)
    }
  }
  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: 見本の表を押して選ぶ近道(右の欄のボタンでも同じことができる)
    // biome-ignore lint/a11y/useKeyWithClickEvents: 同上
    <div className={edit ? 'lt-panel lt-editing' : 'lt-panel'} onClick={pick}>
      <div className="lt-title">
        <strong>{data.tournament.name}</strong>
        <span>{kind === 'pool' ? '予選リーグ 星取表' : '決勝トーナメント表'}</span>
      </div>
      {kind === 'pool' ? (
        <PoolTables data={data} config={config} edit={edit} />
      ) : (
        <BracketTables data={data} />
      )}
    </div>
  )
}

function ring(edit: TableEdit | undefined, part: TablePart): string {
  return edit?.selected === part ? ' lt-ring' : ''
}

function PoolTables({
  data,
  config,
  edit,
}: {
  data: LiveData
  config: LiveTablesConfig
  edit?: TableEdit
}) {
  // 見本では出さない列も薄く残す
  const columns = columnDefs.filter((def) => edit || config.columns[def.key])
  const count = columnCount(config)
  return (
    <div className="lt-stack">
      {data.pools.map((pool) => {
        const table = poolMatrix(pool, data.matches, config)
        const template = `92px repeat(${table.heads.length}, 56px)${
          columns.length > 0
            ? ` repeat(${columns.reduce((sum, def) => sum + def.heads.length, 0)}, 34px)`
            : ''
        }`
        return (
          <div key={table.id} className="lt-pool">
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
      })}
      <span className="lt-legend">{matrixLegend(config, count)}</span>
    </div>
  )
}

function Cell({
  cell,
  className,
  edit,
}: {
  cell: MatrixCell
  className: string
  edit?: TableEdit
}) {
  switch (cell.kind) {
    case 'self':
      return <div className={`lt-cell lt-self${className}`} />
    case 'done':
      return (
        <div className={`lt-cell${className}${ring(edit, 'mark')}`} data-part="mark">
          <span className="lt-mark">{cell.mark}</span>
          <span className={cell.lost ? 'lt-score lt-lost' : 'lt-score'}>{cell.score}</span>
        </div>
      )
    case 'live':
      return (
        <div className={`lt-cell lt-live${className}${ring(edit, 'live')}`} data-part="live">
          <span className="lt-info">● 試合中</span>
          <span className="lt-score">{cell.score}</span>
        </div>
      )
    case 'sched':
      return (
        <div className={`lt-cell${className}`}>
          <span className="lt-info lt-muted">{cell.court}</span>
          <span className="lt-score lt-muted">{cell.time}</span>
        </div>
      )
    default:
      return <div className={`lt-cell${className}`} />
  }
}

function BracketTables({ data }: { data: LiveData }) {
  const views = bracketViews(data.tables.brackets, data.matches)
  return (
    <div className="lt-stack">
      {views.map((view) => (
        <section key={view.name} className="lt-bracket">
          <h3>{view.name}</h3>
          {view.trees.map((tree) => (
            <TreeView key={tree.root.match.id} tree={tree} />
          ))}
        </section>
      ))}
      <span className="lt-legend">
        赤い線が勝ち上がり。線の上の数字はその試合の得点です。まだ決まっていない枠は点線で出しています。
      </span>
    </div>
  )
}

/** 木の寸法(px)。チームの枠の幅と高さ、1 段の縦の間隔、回戦ごとの横の間隔、結果の枠の幅 */
const tree = { boxW: 136, boxH: 48, row: 60, step: 40, tail: 24, resultW: 110 }

const lineRed = '3px solid #d32f2f'
const lineGray = '1.5px solid #8b929a'

function TreeView({ tree: shape }: { tree: BracketTree }) {
  const parts: ReactNode[] = []
  let leaf = 0
  let key = 0
  const hLine = (x: number, y: number, width: number, on: boolean) =>
    parts.push(
      <span
        key={key++}
        className="lt-line"
        style={{
          left: x,
          top: y,
          width: width + 1.5,
          borderTop: on ? lineRed : lineGray,
          transform: 'translateY(-50%)',
          zIndex: on ? 2 : 1,
        }}
      />,
    )
  const vLine = (x: number, y1: number, y2: number, on: boolean) =>
    parts.push(
      <span
        key={key++}
        className="lt-line"
        style={{
          left: x,
          top: Math.min(y1, y2) - 1.5,
          height: Math.abs(y2 - y1) + 3,
          borderLeft: on ? lineRed : lineGray,
          transform: 'translateX(-50%)',
          zIndex: on ? 2 : 1,
        }}
      />,
    )
  const label = (x: number, y: number, text: string, color: string, small = false) =>
    parts.push(
      <span
        key={key++}
        className={small ? 'lt-tree-label lt-small' : 'lt-tree-label'}
        style={{ left: x, top: y, color }}
      >
        {text}
      </span>,
    )
  /** 枠を置き、右へ出る線の端の位置を返す */
  const place = (entry: BracketEntry): { x: number; y: number } => {
    if (entry.kind === 'team') {
      const y = tree.row / 2 + leaf * tree.row
      leaf += 1
      const pending = entry.side.pending
      parts.push(
        <div
          key={key++}
          className={pending ? 'lt-team lt-pending' : 'lt-team'}
          style={{ top: y - tree.boxH / 2, width: tree.boxW, height: tree.boxH }}
        >
          <span className="lt-bar" style={{ background: pending ? '#d8d4cb' : entry.side.color }} />
          <span className="lt-team-text">
            <span className="lt-team-label">{entry.label}</span>
            <span className="lt-team-name">{pending ? '未定' : entry.side.name}</span>
          </span>
        </div>,
      )
      return { x: tree.boxW, y }
    }
    const left = place(entry.left)
    const right = place(entry.right)
    const x = tree.boxW + tree.step * depthOf(entry)
    const y = (left.y + right.y) / 2
    const winner = winnerOf(entry.match)
    for (const [side, at] of [
      ['left', left],
      ['right', right],
    ] as const) {
      const won = winner === side
      hLine(at.x, at.y, x - at.x, won)
      vLine(x, at.y, y, won)
      const score = entry.match[side].score
      if (entry.match.status !== 'sched' && score !== null) {
        label(
          at.x + 6,
          at.y - 21,
          String(score),
          won ? '#d32f2f' : entry.match.status === 'live' ? '#e65c00' : '#59636e',
        )
      }
    }
    if (entry.match.status === 'sched' && entry !== shape.root) {
      label(x + 5, y + 3, entry.match.time, '#59636e', true)
    }
    return { x, y }
  }
  const out = place(shape.root)
  const match = shape.root.match
  const winner = winnerOf(match)
  hLine(out.x, out.y, tree.tail, winner !== null)
  const result =
    winner !== null
      ? { label: shape.place, name: match[winner].name, className: 'lt-result lt-won' }
      : match.status === 'live'
        ? {
            label: '試合中',
            name: `${match.left.score ?? 0} - ${match.right.score ?? 0}`,
            className: 'lt-result lt-playing',
          }
        : {
            label: [match.time, match.court].filter(Boolean).join(' ・ '),
            name: '未定',
            className: 'lt-result',
          }
  const resultX = out.x + tree.tail
  const height = leafCount(shape.root) * tree.row
  return (
    <div className="lt-tree-scroll">
      <span className="lt-tree-title">{shape.title}</span>
      <div className="lt-tree" style={{ width: resultX + tree.resultW, height }}>
        {parts}
        <div
          className={result.className}
          style={{ left: resultX, top: out.y - 30, width: tree.resultW }}
        >
          <span className="lt-result-label">{result.label}</span>
          <span className="lt-result-name">{result.name}</span>
        </div>
      </div>
    </div>
  )
}
