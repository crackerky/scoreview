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
  scoreHits,
  tabMatches,
  teamColor,
  timeGroups,
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
    stage: '予選',
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
