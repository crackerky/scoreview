import type { LiveTablesConfig } from '@/api/public.types'
import type { LiveBracket, LiveMatch, LivePool, LiveSide } from './live.types'

// 観戦アプリの「表で見る」(Claude Design「Score View Admin」の 2)の、星取表とトーナメント表の中身を
// 組み立てる決まり。見せ方(列、記号、並び、試合中のマス、進出圏の色)は管理者が決めた config に従う。

/** 管理画面で押して選ぶ、表の部分 */
export type TablePart = 'cols' | 'mark' | 'live' | 'order' | 'hl'

export type ColumnKey = keyof LiveTablesConfig['columns']

/** 星取表の右に並べる列と、その見出し(並びは左から) */
export const columnDefs: ReadonlyArray<{ key: ColumnKey; label: string; heads: string[] }> = [
  { key: 'wdl', label: '勝・分・敗', heads: ['勝', '分', '敗'] },
  { key: 'goals', label: '得点・失点', heads: ['得点', '失点'] },
  { key: 'diff', label: '得失点差', heads: ['差'] },
  { key: 'points', label: '勝点', heads: ['勝点'] },
  { key: 'rank', label: '順位', heads: ['順位'] },
]

/** 勝ち・引き分け・負けの記号と、表の下の説明 */
export const markSets: Record<
  LiveTablesConfig['marks'],
  { marks: [string, string, string]; legend: string }
> = {
  maru: { marks: ['○', '△', '×'], legend: '○ 勝ち　△ 引き分け　× 負け' },
  wdl: { marks: ['W', 'D', 'L'], legend: 'W 勝ち　D 引き分け　L 負け' },
  kanji: { marks: ['勝', '分', '敗'], legend: '勝・分・敗' },
}

/** スマホで横にずらさずに見られる、右の列の数 */
export const fitColumns = 4

export type MatrixCell =
  | { kind: 'self' }
  | { kind: 'none' }
  | { kind: 'done'; mark: string; score: string; lost: boolean }
  | { kind: 'live'; score: string }
  | { kind: 'sched'; court: string; time: string }

export type MatrixRow = {
  teamId: number
  name: string
  color: string
  cells: MatrixCell[]
  /** 右の列の値(列ごと。出さない列も入れる。管理画面は薄く出す) */
  stats: Record<ColumnKey, string[]>
  /** 進出圏の色をつける */
  advancing: boolean
}

export type PoolMatrix = {
  id: number
  name: string
  status: string
  /** 横の見出しのチーム(行と同じ並び) */
  heads: string[]
  rows: MatrixRow[]
}

function signed(value: number): string {
  return value > 0 ? `+${value}` : String(value)
}

/**
 * 組の星取表。行と列は登録順か順位順。マスは行のチームから見た結果で、終わった試合は記号と点、
 * 試合中はいまの点(空けるなら空)、これからの試合はコートと時刻。
 */
export function poolMatrix(
  pool: LivePool,
  matches: LiveMatch[],
  config: LiveTablesConfig,
): PoolMatrix {
  const rows =
    config.order === 'rank' ? pool.rows : [...pool.rows].sort((a, b) => a.entry - b.entry)
  const games = matches.filter((match) => match.groupId === pool.id)
  const left = games.filter((match) => match.status !== 'done').length
  const marks = markSets[config.marks].marks
  const cell = (teamId: number, otherId: number): MatrixCell => {
    if (teamId === otherId) {
      return { kind: 'self' }
    }
    const match = games.find(
      (game) =>
        (game.left.teamId === teamId && game.right.teamId === otherId) ||
        (game.left.teamId === otherId && game.right.teamId === teamId),
    )
    if (!match) {
      return { kind: 'none' }
    }
    const [mine, theirs] =
      match.left.teamId === teamId ? [match.left, match.right] : [match.right, match.left]
    const score = `${mine.score ?? 0}-${theirs.score ?? 0}`
    if (match.status === 'done') {
      const my = mine.score ?? 0
      const their = theirs.score ?? 0
      return {
        kind: 'done',
        mark: my > their ? marks[0] : my < their ? marks[2] : marks[1],
        score,
        lost: my < their,
      }
    }
    if (match.status === 'live') {
      return { kind: 'live', score: config.live === 'score' ? score : '' }
    }
    return { kind: 'sched', court: match.court, time: match.time }
  }
  return {
    id: pool.id,
    name: pool.name,
    status: pool.done ? '全試合終了' : `途中経過 ・ 残り${left}試合`,
    heads: rows.map((row) => row.name),
    rows: rows.map((row) => ({
      teamId: row.teamId,
      name: row.name,
      color: row.color,
      cells: rows.map((other) => cell(row.teamId, other.teamId)),
      stats: {
        wdl: [String(row.wins), String(row.draws), String(row.losses)],
        goals: [String(row.goalsFor), String(row.goalsAgainst)],
        diff: [signed(row.goalsFor - row.goalsAgainst)],
        points: [String(row.points)],
        rank: [row.rank === null ? '' : String(row.rank)],
      },
      advancing: config.highlight && row.rank !== null && row.rank <= pool.advance,
    })),
  }
}

/** 星取表の表の下の説明 */
export function matrixLegend(config: LiveTablesConfig, columns: number): string {
  return [
    '左の列のチームから見た結果です。',
    `${markSets[config.marks].legend}。`,
    config.highlight && config.columns.rank ? '色のついた順位は決勝トーナメント進出圏。' : '',
    columns > fitColumns ? '表は横にずらせます。' : '',
  ].join('')
}

/** 右に並べる列の数(見出しの数) */
export function columnCount(config: LiveTablesConfig): number {
  return columnDefs
    .filter((def) => config.columns[def.key])
    .reduce((sum, def) => sum + def.heads.length, 0)
}

export type BracketLeaf = { kind: 'team'; label: string; side: LiveSide }
export type BracketNode = {
  kind: 'match'
  match: LiveMatch
  left: BracketEntry
  right: BracketEntry
}
export type BracketEntry = BracketLeaf | BracketNode

export type BracketTree = {
  /** 何を決める試合か(「1位決定戦」「3位決定戦」) */
  title: string
  /** 勝ったチームの呼び名(「優勝」「3位」) */
  place: string
  root: BracketNode
}

export type BracketView = { name: string; trees: BracketTree[] }

const rangePrefix = /^\d+〜\d+位 /

/**
 * 部門のトーナメントを、勝ち上がりの木にする。木の根は、勝者がこのトーナメントの別の試合へ進まない
 * 試合(決勝、3位決定戦など)で、順位の順。勝者が進んできた側は前の試合をたどり、それ以外(組の順位、
 * 前の試合の敗者)は葉にする。
 */
export function bracketViews(brackets: LiveBracket[], matches: LiveMatch[]): BracketView[] {
  return brackets.flatMap((bracket) => {
    const byCode = new Map(
      matches
        .filter((match) => match.divisionId === bracket.divisionId)
        .map((match) => [match.code, match]),
    )
    const shapes = new Map(bracket.matches.map((shape) => [shape.code, shape]))
    const advanced = new Set(
      bracket.matches.flatMap((shape) =>
        [shape.left, shape.right]
          .filter((side) => side.from?.outcome === 'win')
          .map((side) => side.from?.code ?? ''),
      ),
    )
    // 前の試合の敗者の呼び名。同じ名前の試合が複数あればコートで見分ける
    const loserLabel = (code: string): string => {
      const source = byCode.get(code)
      if (!source) {
        return `${code} の敗者`
      }
      const name = source.stage.replace(rangePrefix, '')
      const same = bracket.matches.filter((shape) => byCode.get(shape.code)?.stage === source.stage)
      return same.length > 1 && source.court !== ''
        ? `${name}(${source.court})の敗者`
        : `${name}の敗者`
    }
    const entry = (side: LiveBracket['matches'][number]['left'], team: LiveSide): BracketEntry => {
      const from = side.from
      if (from?.outcome === 'win' && shapes.has(from.code)) {
        const node = build(from.code)
        if (node) {
          return node
        }
      }
      const label = side.seed ?? (from ? loserLabel(from.code) : '')
      return { kind: 'team', label, side: team }
    }
    const build = (code: string): BracketNode | null => {
      const shape = shapes.get(code)
      const match = byCode.get(code)
      if (!shape || !match) {
        return null
      }
      return {
        kind: 'match',
        match,
        left: entry(shape.left, match.left),
        right: entry(shape.right, match.right),
      }
    }
    const trees = bracket.matches
      .filter((shape) => !advanced.has(shape.code))
      .sort((a, b) => (a.winnerPlace || 999) - (b.winnerPlace || 999))
      .flatMap((shape) => {
        const root = build(shape.code)
        if (!root) {
          return []
        }
        const place = /^(\d+)位決定戦$/.exec(root.match.stage)?.[1]
        return [
          {
            title: root.match.stage,
            place: place === undefined ? '勝者' : place === '1' ? '優勝' : `${place}位`,
            root,
          },
        ]
      })
    return trees.length > 0 ? [{ name: bracket.name, trees }] : []
  })
}

/** 木の葉の数(縦の段数) */
export function leafCount(entry: BracketEntry): number {
  return entry.kind === 'team' ? 1 : leafCount(entry.left) + leafCount(entry.right)
}

/** 木の深さ(葉は 0、葉だけを持つ試合は 1) */
export function depthOf(entry: BracketEntry): number {
  return entry.kind === 'team' ? 0 : 1 + Math.max(depthOf(entry.left), depthOf(entry.right))
}
