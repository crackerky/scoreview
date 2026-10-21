import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { z } from 'zod'
import { matchRowSchema, tournamentFrameSchema } from '../src/api/common.types.ts'
import {
  galleryResponseSchema,
  groupResponseSchema,
  matchResponseSchema,
  playerResponseSchema,
  scheduleResponseSchema,
  standingsResponseSchema,
  teamListResponseSchema,
  teamResponseSchema,
  tournamentListResponseSchema,
  tournamentTopResponseSchema,
} from '../src/api/public.types.ts'
import { loadFixture } from './fixtures/load.ts'

// docs/api.md の例に沿った fixture がすべての閲覧用応答スキーマで parse できることを確かめる
const cases: Array<[string, z.ZodType]> = [
  ['tournaments', tournamentListResponseSchema],
  ['tournament-1', tournamentTopResponseSchema],
  ['tournament-2', tournamentTopResponseSchema],
  ['tournament-3', tournamentTopResponseSchema],
  ['teams-1', teamListResponseSchema],
  ['teams-2', teamListResponseSchema],
  ['schedule-tournament-1', scheduleResponseSchema],
  ['schedule-division-101', scheduleResponseSchema],
  ['standings-101', standingsResponseSchema],
  ['group-1001', groupResponseSchema],
  ['group-1002', groupResponseSchema],
  ['team-5001', teamResponseSchema],
  ['team-5003', teamResponseSchema],
  ['match-9001', matchResponseSchema],
  ['match-9002', matchResponseSchema],
  ['player-70001', playerResponseSchema],
  ['player-70009', playerResponseSchema],
  ['player-70101', playerResponseSchema],
  ['gallery-201', galleryResponseSchema],
]

describe('閲覧用応答のスキーマ', () => {
  for (const [name, schema] of cases) {
    test(`${name} を parse できる`, () => {
      const result = schema.safeParse(loadFixture(name))
      assert.equal(result.success, true, result.error?.message)
    })
  }

  test('frame が欠けた応答は失敗する', () => {
    const { frame: _frame, ...rest } = galleryResponseSchema.parse(loadFixture('gallery-201'))
    assert.equal(galleryResponseSchema.safeParse(rest).success, false)
  })

  test('null で返すべきキーの省略は失敗する(nullable であって optional ではない)', () => {
    const top = tournamentTopResponseSchema.parse(loadFixture('tournament-1'))
    const { logoUrl: _logoUrl, ...rest } = top
    assert.equal(tournamentTopResponseSchema.safeParse(rest).success, false)
    assert.equal(tournamentTopResponseSchema.safeParse({ ...top, logoUrl: null }).success, true)
  })

  test('MatchRow の未確定チームは placeholder で表す', () => {
    const row = {
      matchId: 1,
      hasDetail: false,
      code: 'B1',
      date: null,
      time: null,
      field: null,
      status: 'scheduled',
      resultType: 'normal',
      winnerSide: null,
      left: { team: null, placeholder: { code: 'A1', outcome: 'win' }, score: null, labelEn: '' },
      right: {
        team: null,
        placeholder: { code: 'A2', outcome: 'draw' },
        score: null,
        labelEn: '',
      },
    }
    assert.equal(matchRowSchema.safeParse(row).success, false)
    const fixed = { ...row, right: { ...row.right, placeholder: { code: 'A2', outcome: 'lose' } } }
    assert.equal(matchRowSchema.safeParse(fixed).success, true)
  })

  test('frame のラベル種別は列挙値以外を拒否する', () => {
    const frame = tournamentFrameSchema.parse(loadFixture('frame-1'))
    const bad = { ...frame, labels: { teamList: 'players', groups: 'groups' } }
    assert.equal(tournamentFrameSchema.safeParse(bad).success, false)
  })

  test('グループのマトリクスは kind で形が決まる', () => {
    const group = groupResponseSchema.parse(loadFixture('group-1001'))
    const cell = group.pool?.matrix[0]?.[1]
    assert.equal(cell?.kind, 'match')
    const bad = structuredClone(loadFixture('group-1001'))
    assert.ok(typeof bad === 'object' && bad !== null)
    const broken = { ...bad, pool: { ...group.pool, matrix: [[{ kind: 'match' }]] } }
    assert.equal(groupResponseSchema.safeParse(broken).success, false)
  })
})
