import assert from 'node:assert/strict'
import { test } from 'node:test'
import { boardResponseSchema } from '../src/api/public.types.ts'
import { buildLiveData } from '../src/live/live.utils.ts'
import { scorePaths, tabOf } from '../src/score/score.paths.ts'
import {
  dayHasLive,
  dayLabel,
  initialDayIndex,
  poolPage,
  scoreDays,
  scoreDivisions,
  slotText,
} from '../src/score/score.utils.ts'
import { loadFixture } from './fixtures/load.ts'

// スコアサイトの、掲示板を画面の形に直す決まり(src/score/score.utils.ts、score.paths.ts)

const board = boardResponseSchema.parse(loadFixture('board-tournament-1'))
const data = buildLiveData(board, null)

test('部門ごとに組の星取表を出し、組もトーナメントも無い部門は出さない', () => {
  const divisions = scoreDivisions(board, data)
  assert.deepEqual(
    divisions.map((division) => [division.id, division.name, division.pools.length]),
    [[101, 'ビーチミックス', 1]],
  )
  const [pool] = divisions[0]?.pools ?? []
  assert.equal(pool?.id, 1001)
  assert.equal(pool?.rows.length, 2)
})

test('時間割は日付ごとに、元の時刻とコートで枠を決め、決まっていない試合は表の下に回す', () => {
  const days = scoreDays(board, data)
  assert.deepEqual(
    days.map((day) => day.date),
    [...new Set(board.matches.map((row) => row.date))].sort((a, b) =>
      a === null ? 1 : b === null ? -1 : a.localeCompare(b),
    ),
  )
  const placed = days.flatMap((day) => [...day.cells.values()].flat())
  const unplaced = days.flatMap((day) => day.unplaced)
  assert.equal(placed.length + unplaced.length, board.matches.length)
  for (const slot of unplaced) {
    assert.ok(slot.time === null || slot.field === null)
  }
})

test('最初に開く日は今日、無ければ試合中の日、それも無ければ最初の日', () => {
  const days = [
    { date: '2026-11-01', live: false },
    { date: '2026-11-02', live: true },
    { date: null, live: false },
  ]
  assert.equal(initialDayIndex(days, '2026-11-01'), 0)
  assert.equal(initialDayIndex(days, '2026-11-09'), 1)
  assert.equal(initialDayIndex([{ date: '2026-11-01', live: false }], ''), 0)
  assert.equal(initialDayIndex([], '2026-11-01'), 0)
})

test('試合中の試合がある日を見分ける', () => {
  const live = {
    ...board,
    matches: board.matches.map((row, index) =>
      index === 0 ? { ...row, status: 'live' as const } : row,
    ),
  }
  const days = scoreDays(live, buildLiveData(live, null))
  assert.equal(days.some(dayHasLive), true)
  assert.equal(scoreDays(board, data).some(dayHasLive), false)
})

test('組の画面は組の星取表と組の試合だけを出し、知らない組は null', () => {
  const page = poolPage(data, 1001)
  assert.equal(page?.pool.id, 1001)
  assert.ok(page?.matches.every((match) => match.groupId === 1001))
  assert.equal(page?.matches.length, 1)
  assert.equal(poolPage(data, 999), null)
})

test('日付の見出しと試合の時刻、コート', () => {
  assert.equal(dayLabel('2026-11-03'), '11月3日（火）')
  assert.equal(dayLabel(null), '日付未定')
  const match = data.matches[0]
  assert.ok(match)
  assert.equal(slotText({ ...match, time: '09:00', court: '1コート' }), '09:00 ・ 1コート')
  assert.equal(slotText({ ...match, time: '', court: '' }), '')
})

test('URL と、いまの画面がどの入口の中か', () => {
  assert.equal(scorePaths.pool(7, 10), '/tournaments/7/pools/10')
  assert.equal(scorePaths.match(7, 35), '/tournaments/7/matches/35')
  assert.equal(tabOf('/tournaments/7', 7), 'score')
  assert.equal(tabOf('/tournaments/7/pools/10', 7), 'score')
  assert.equal(tabOf('/tournaments/7/schedule', 7), 'schedule')
  assert.equal(tabOf('/tournaments/7/bracket', 7), 'bracket')
})
