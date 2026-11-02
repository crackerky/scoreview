import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { z } from 'zod'
import { matchRowSchema, tournamentFrameSchema } from '../src/api/common.types.ts'
import {
  groupResponseSchema,
  matchResponseSchema,
  scheduleResponseSchema,
  tournamentListResponseSchema,
  tournamentSummarySchema,
} from '../src/api/public.types.ts'
import { loadFixture } from './fixtures/load.ts'

// docs/api.md の例に沿った fixture がすべての閲覧用応答スキーマで parse できることを確かめる
const cases: Array<[string, z.ZodType]> = [
  ['tournaments', tournamentListResponseSchema],
  ['schedule-tournament-1', scheduleResponseSchema],
  ['group-1001', groupResponseSchema],
  ['group-1002', groupResponseSchema],
  ['match-9001', matchResponseSchema],
  ['match-9002', matchResponseSchema],
]

describe('閲覧用応答のスキーマ', () => {
  for (const [name, schema] of cases) {
    test(`${name} を parse できる`, () => {
      const result = schema.safeParse(loadFixture(name))
      assert.equal(result.success, true, result.error?.message)
    })
  }

  test('frame が欠けた応答は失敗する', () => {
    const { frame: _frame, ...rest } = groupResponseSchema.parse(loadFixture('group-1001'))
    assert.equal(groupResponseSchema.safeParse(rest).success, false)
  })

  test('null で返すべきキーの省略は失敗する(nullable であって optional ではない)', () => {
    const summary = { id: 1, name: { ja: '大会', en: '' }, startDate: null, teamCount: 0 }
    const { startDate: _startDate, ...rest } = summary
    assert.equal(tournamentSummarySchema.safeParse(rest).success, false)
    assert.equal(tournamentSummarySchema.safeParse(summary).success, true)
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
