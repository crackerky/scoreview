import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { LiveTablesConfig } from '../src/api/public.types.ts'
import {
  bracketViews,
  columnCount,
  depthOf,
  leafCount,
  matrixLegend,
  poolMatrix,
} from '../src/live/live.tables.ts'
import type { LiveBracket, LiveMatch, LivePool } from '../src/live/live.types.ts'
import { teamColor } from '../src/live/live.utils.ts'

// 観戦アプリの「表で見る」の中身(src/live/live.tables.ts)

const config: LiveTablesConfig = {
  shown: true,
  poolShown: true,
  bracketShown: true,
  columns: { wdl: true, goals: true, diff: true, points: true, rank: true },
  marks: 'maru',
  order: 'entry',
  live: 'score',
  highlight: true,
}

function side(teamId: number | null, score: number | null, name = `チーム${teamId}`) {
  return { teamId, name, pending: teamId === null, color: teamColor(teamId), score }
}

function match(
  id: number,
  code: string,
  status: LiveMatch['status'],
  left: ReturnType<typeof side>,
  right: ReturnType<typeof side>,
  extra: Partial<LiveMatch> = {},
): LiveMatch {
  return {
    id,
    code,
    date: '2026-10-29',
    time: '10:00',
    court: '1コート',
    status,
    stage: '予選 A 組',
    groupId: 1,
    pool: true,
    divisionId: 101,
    divisionName: 'オープン',
    left,
    right,
    ...extra,
  }
}

function row(teamId: number, rank: number, entry: number, points: number) {
  return {
    rank,
    entry,
    teamId,
    name: `チーム${teamId}`,
    color: teamColor(teamId),
    points,
    wins: points / 3,
    draws: 0,
    losses: 0,
    goalsFor: 7,
    goalsAgainst: 4,
  }
}

const pool: LivePool = {
  id: 1,
  name: '予選 A 組',
  done: false,
  advance: 1,
  // 順位の順(1 位はチーム 2)
  rows: [row(2, 1, 1, 3), row(1, 2, 0, 0), row(3, 3, 2, 0)],
}

const games = [
  match(10, 'A1', 'done', side(1, 5), side(2, 7)),
  match(11, 'A2', 'live', side(2, 3), side(3, 1)),
  match(12, 'A3', 'sched', side(1, null), side(3, null), { time: '11:20', court: '2コート' }),
]

test('星取表: 登録順に並べ、マスは行のチームから見た結果、試合中はいまの点、これからはコートと時刻', () => {
  const table = poolMatrix(pool, games, config)
  assert.deepEqual(table.heads, ['チーム1', 'チーム2', 'チーム3'])
  const [one, two] = table.rows
  assert.deepEqual(one?.cells[0], { kind: 'self' })
  assert.deepEqual(one?.cells[1], { kind: 'done', mark: '×', score: '5-7', lost: true })
  assert.deepEqual(two?.cells[0], { kind: 'done', mark: '○', score: '7-5', lost: false })
  assert.deepEqual(two?.cells[2], { kind: 'live', score: '3-1' })
  assert.deepEqual(one?.cells[2], { kind: 'sched', court: '2コート', time: '11:20' })
  assert.deepEqual(two?.stats.diff, ['+3'])
  assert.equal(two?.advancing, true)
  assert.equal(one?.advancing, false)
  assert.equal(table.status, '途中経過 ・ 残り2試合')
})

test('星取表: 順位順、漢字の記号、試合中は空ける、色はつけない', () => {
  const table = poolMatrix(pool, games, {
    ...config,
    order: 'rank',
    marks: 'kanji',
    live: 'blank',
    highlight: false,
  })
  assert.deepEqual(table.heads, ['チーム2', 'チーム1', 'チーム3'])
  assert.deepEqual(table.rows[0]?.cells[1], { kind: 'done', mark: '勝', score: '7-5', lost: false })
  assert.deepEqual(table.rows[0]?.cells[2], { kind: 'live', score: '' })
  assert.equal(table.rows[0]?.advancing, false)
})

test('列の数と表の下の説明', () => {
  assert.equal(columnCount(config), 8)
  const few = { ...config, columns: { ...config.columns, wdl: false, goals: false } }
  assert.equal(columnCount(few), 3)
  assert.match(matrixLegend(config, 8), /表は横にずらせます/)
  assert.doesNotMatch(matrixLegend(few, 3), /横にずらせます/)
})

test('トーナメント: 決勝と 3 位決定戦を根にし、勝ち上がりは前の試合をたどる。負けチームは葉にする', () => {
  const bracket: LiveBracket = {
    divisionId: 101,
    name: '1〜4 位を決めるトーナメント',
    matches: [
      {
        code: 'E1',
        winnerPlace: 1,
        left: { from: { code: 'E2', outcome: 'win' }, seed: null },
        right: { from: { code: 'E3', outcome: 'win' }, seed: null },
      },
      {
        code: 'E2',
        winnerPlace: 0,
        left: { from: null, seed: '予選 A 組 1位' },
        right: { from: null, seed: '予選 D 組 1位' },
      },
      {
        code: 'E3',
        winnerPlace: 0,
        left: { from: null, seed: '予選 B 組 1位' },
        right: { from: null, seed: '予選 C 組 1位' },
      },
      {
        code: 'E4',
        winnerPlace: 3,
        left: { from: { code: 'E2', outcome: 'lose' }, seed: null },
        right: { from: { code: 'E3', outcome: 'lose' }, seed: null },
      },
    ],
  }
  const knockout = { pool: false, groupId: 5 }
  const list = [
    match(20, 'E2', 'done', side(1, 7), side(4, 3), { ...knockout, stage: '1〜4位 準決勝' }),
    match(21, 'E3', 'live', side(2, 2), side(3, 4), {
      ...knockout,
      stage: '1〜4位 準決勝',
      court: '2コート',
    }),
    match(22, 'E1', 'sched', side(1, null), side(null, null, '準決勝(2コート)の勝者'), {
      ...knockout,
      stage: '1位決定戦',
    }),
    match(23, 'E4', 'sched', side(4, null), side(null, null), { ...knockout, stage: '3位決定戦' }),
  ]
  const [view] = bracketViews([bracket], list)
  assert.equal(view?.name, '1〜4 位を決めるトーナメント')
  assert.deepEqual(
    view?.trees.map((tree) => [tree.title, tree.place]),
    [
      ['1位決定戦', '優勝'],
      ['3位決定戦', '3位'],
    ],
  )
  const final = view?.trees[0]?.root
  assert.ok(final)
  assert.equal(leafCount(final), 4)
  assert.equal(depthOf(final), 2)
  assert.equal(
    final.left.kind === 'match' && final.left.left.kind === 'team' && final.left.left.label,
    '予選 A 組 1位',
  )
  const third = view?.trees[1]?.root
  assert.equal(third?.left.kind === 'team' && third.left.label, '準決勝(1コート)の敗者')
  assert.equal(third?.right.kind === 'team' && third.right.label, '準決勝(2コート)の敗者')
})
