import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  groupResponseSchema,
  matchResponseSchema,
  scheduleResponseSchema,
} from '../src/api/public.types.ts'
import type { LiveMatch } from '../src/live/live.types.ts'
import {
  buildLiveData,
  courtName,
  elapsedText,
  flowEvents,
  leadPoints,
  placements,
  scoreHits,
  tabMatches,
  teamColor,
  timeGroups,
  winnerOf,
} from '../src/live/live.utils.ts'
import { loadFixture } from './fixtures/load.ts'

// 観戦アプリの、api の応答を画面の形に直す決まり(src/live/live.utils.ts)

const schedule = scheduleResponseSchema.parse(loadFixture('schedule-tournament-1'))
const groups = ['group-1001', 'group-1002', 'group-1003', 'group-1004'].map((name) =>
  groupResponseSchema.parse(loadFixture(name)),
)

test('試合は組の名前、コート、状態、決まっていないチームの決まり方を持つ', () => {
  const data = buildLiveData(schedule, groups, '2026-10-17')
  assert.equal(data.matches.length, schedule.rows.length)
  const first = data.matches[0]
  assert.equal(first?.status, 'done')
  assert.equal(first?.left.teamId, 5001)
  assert.equal(first?.left.color, teamColor(5001))
  const pending = data.matches.find((match) => match.right.pending)
  assert.equal(pending?.right.name, 'A2 の負けチーム')
  assert.equal(pending?.status, 'sched')
  assert.ok(data.pools.length > 0)
  for (const pool of data.pools) {
    const ranks = pool.rows.map((row) => row.rank ?? Number.MAX_SAFE_INTEGER)
    assert.deepEqual(
      ranks,
      [...ranks].sort((a, b) => a - b),
      pool.name,
    )
  }
  assert.equal(data.tournament.date, '2026-10-17')
})

test('決まっていないチームは元の試合の名前で呼び、同じ名前の試合はコートで見分ける', () => {
  const stages: Record<string, string> = {
    A1: '1〜4位 準決勝',
    A2: '1〜4位 準決勝',
    B1: '1位決定戦',
  }
  const named = {
    ...schedule,
    rows: schedule.rows.map((row) => ({ ...row, stage: stages[row.code] ?? null })),
  }
  const final = buildLiveData(named, [], null).matches.find((item) => item.code === 'B1')
  assert.equal(final?.left.name, '準決勝(1コート)の勝者')
  assert.equal(final?.right.name, '準決勝(2コート)の負けチーム')
  assert.equal(final?.stage, '1位決定戦')
  // 別の幅の準決勝があれば幅を残す
  const crossed = {
    ...named,
    rows: named.rows.map((row) => (row.code === 'A2' ? { ...row, stage: '5〜8位 準決勝' } : row)),
  }
  const mixed = buildLiveData(crossed, [], null).matches.find((item) => item.code === 'B1')
  assert.equal(mixed?.left.name, '1〜4位 準決勝の勝者')
  assert.equal(mixed?.right.name, '5〜8位 準決勝の負けチーム')
})

test('コートは「1コート」の形にし、もともと「コート」で始まる名前はそのまま', () => {
  assert.equal(courtName('1'), '1コート')
  assert.equal(courtName('コートA'), 'コートA')
  assert.equal(courtName(null), '')
})

function match(
  id: number,
  status: LiveMatch['status'],
  left: number,
  right: number,
  time = '10:00',
): LiveMatch {
  const side = (teamId: number) => ({
    teamId,
    name: `チーム${teamId}`,
    pending: false,
    color: teamColor(teamId),
    score: 0,
  })
  return {
    id,
    code: `M${id}`,
    date: '2026-10-29',
    time,
    court: '1コート',
    status,
    stage: '予選 A 組',
    groupId: 1,
    pool: true,
    divisionName: 'オープン',
    left: side(left),
    right: side(right),
  }
}

test('タブ: ライブは試合中とこれからの 3 試合、結果は新しい順、マイチームは選んだチームの試合', () => {
  const list = [
    match(1, 'done', 1, 2),
    match(2, 'done', 3, 4),
    match(3, 'live', 1, 3),
    match(4, 'sched', 2, 4),
    match(5, 'sched', 1, 4),
    match(6, 'sched', 2, 3),
    match(7, 'sched', 1, 2),
  ]
  assert.deepEqual(
    tabMatches(list, 'live', []).map((item) => item.id),
    [3, 4, 5, 6],
  )
  assert.deepEqual(
    tabMatches(list, 'done', []).map((item) => item.id),
    [2, 1],
  )
  assert.deepEqual(
    tabMatches(list, 'mine', [1]).map((item) => item.id),
    [3, 5, 7, 1],
  )
})

test('下の帯は続けて同じ時刻の試合をまとめる', () => {
  const list = [
    match(1, 'live', 1, 2, '10:00'),
    match(2, 'live', 3, 4, '10:00'),
    match(3, 'sched', 1, 3, '10:20'),
  ]
  assert.deepEqual(timeGroups(list), [
    { time: '10:00', indexes: [0, 1] },
    { time: '10:20', indexes: [2] },
  ])
})

test('取り直す前より点が増えた側を返す(前に無かった試合は数えない)', () => {
  const first = match(1, 'live', 1, 2)
  const second = match(2, 'live', 3, 4)
  const after: LiveMatch[] = [
    { ...first, left: { ...first.left, score: 1 } },
    { ...second, right: { ...second.right, score: 2 } },
    match(3, 'live', 5, 6),
  ]
  const before = [first, second]
  assert.deepEqual(scoreHits(before, after), { 1: 'left', 2: 'right' })
})

test('得点の流れ: 得点だけを古い順に、経過時間は「分:秒」', () => {
  const detail = matchResponseSchema.parse(loadFixture('match-9001'))
  const events = flowEvents(detail)
  assert.ok(events.length > 0)
  assert.ok(events.every((event) => /^\d+:\d{2}$/.test(event.elapsed)))
  assert.equal(elapsedText(725), '12:05')
  const points = leadPoints(events, 300, 100)
  assert.equal(points.length, events.length)
  assert.equal(points.at(-1)?.x, 300)
  assert.ok(points.every((point) => point.y >= 0 && point.y <= 100))
})

test('勝った側: 終わった試合の点の多い側。終わっていない試合と引き分けは null', () => {
  const finished = match(1, 'done', 1, 2)
  assert.equal(
    winnerOf({
      ...finished,
      left: { ...finished.left, score: 7 },
      right: { ...finished.right, score: 5 },
    }),
    'left',
  )
  assert.equal(
    winnerOf({
      ...finished,
      left: { ...finished.left, score: 5 },
      right: { ...finished.right, score: 5 },
    }),
    null,
  )
  assert.equal(winnerOf(match(2, 'live', 1, 2)), null)
})

test('順位決定戦: 「N位決定戦」だけを順位の順に、終わった試合は勝者を N 位、敗者を N+1 位にする', () => {
  const final = { ...match(1, 'done', 1, 2), stage: '1位決定戦' }
  const third = { ...match(2, 'sched', 3, 4), stage: '3位決定戦' }
  const semi = { ...match(3, 'done', 1, 3), stage: '1〜4位 準決勝' }
  const scored = {
    ...final,
    left: { ...final.left, score: 4 },
    right: { ...final.right, score: 6 },
  }
  const list = placements([third, semi, scored])
  assert.deepEqual(
    list.map((item) => [
      item.place,
      item.result?.map((row) => `${row.place}位 ${row.name}`) ?? null,
    ]),
    [
      [1, ['1位 チーム2', '2位 チーム1']],
      [3, null],
    ],
  )
})
