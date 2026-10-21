import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { z } from 'zod'
import {
  adminDivisionListSchema,
  adminGroupListSchema,
  adminMatchListSchema,
  adminPlayerListSchema,
  adminTeamListSchema,
  adminTournamentSchema,
  bannersSchema,
  bracketTemplateListSchema,
  loginResponseSchema,
  matchDetailSchema,
  meResponseSchema,
  noContentSchema,
  userListSchema,
} from '../src/api/admin.types.ts'
import { loadFixture } from './fixtures/load.ts'

// fixture は開発用の api(Go)の実際の応答を保存したもの(架空データ)。Go の DTO と zod の食い違いを見つける
const cases: Array<[string, z.ZodType]> = [
  ['admin-me', meResponseSchema],
  ['admin-tournament', adminTournamentSchema],
  ['admin-divisions', adminDivisionListSchema],
  ['admin-groups', adminGroupListSchema],
  ['admin-teams', adminTeamListSchema],
  ['admin-players', adminPlayerListSchema],
  ['admin-matches', adminMatchListSchema],
  ['admin-match-detail', matchDetailSchema],
  ['admin-bracket-templates', bracketTemplateListSchema],
  ['admin-banners', bannersSchema],
  ['admin-users', userListSchema],
]

describe('管理用応答のスキーマ', () => {
  for (const [name, schema] of cases) {
    test(`${name} を parse できる`, () => {
      const result = schema.safeParse(loadFixture(name))
      assert.equal(result.success, true, result.error?.message)
    })
  }

  test('生成したトーナメントの後続試合は未確定チームの参照を持つ', () => {
    const list = adminMatchListSchema.parse(loadFixture('admin-matches'))
    const b7 = list.matches.find((match) => match.code === 'B7')
    assert.deepEqual(b7?.left.placeholder, { code: 'B5', outcome: 'win' })
  })

  test('ログインの応答は RFC 3339 の期限を持つ', () => {
    const body = {
      token: 'x'.repeat(43),
      expiresAt: '2026-10-22T09:00:00Z',
      user: { id: 1, email: 'a@example.com', displayName: '', role: 'admin', tournamentIds: [] },
    }
    assert.equal(loginResponseSchema.safeParse(body).success, true)
    assert.equal(loginResponseSchema.safeParse({ ...body, expiresAt: 'tomorrow' }).success, false)
    assert.equal(loginResponseSchema.safeParse({ ...body, token: '' }).success, false)
  })

  test('204 は本文なし', () => {
    assert.equal(noContentSchema.safeParse(undefined).success, true)
  })
})
