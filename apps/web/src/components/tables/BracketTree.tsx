import type { ReactNode } from 'react'
import { type BracketEntry, depthOf, leafCount, type BracketTree as Tree } from '@/live/live.tables'
import { winnerOf } from '@/live/live.utils'

// トーナメントの木 1 本(勝ち上がりは赤い線)。観戦アプリの「表で見る」、管理画面の見本、スコアサイトで共有する。

/** 木の寸法(px)。チームの枠の幅と高さ、1 段の縦の間隔、回戦ごとの横の間隔、結果の枠の幅 */
const tree = { boxW: 136, boxH: 48, row: 60, step: 40, tail: 24, resultW: 110 }

const lineRed = '3px solid #d32f2f'
const lineGray = '1.5px solid #8b929a'

export function BracketTree({ tree: shape }: { tree: Tree }) {
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
