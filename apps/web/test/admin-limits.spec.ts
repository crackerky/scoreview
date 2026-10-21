import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { elapsedSeconds } from '../src/admin/scoreOps.utils.ts'
import {
  MAX_ELAPSED_SEC,
  MAX_PLAYER_STAT,
  MAX_SCORE,
  playerStatsFromForm,
  statusFormSchema,
} from '../src/api/admin.types.ts'

// 記録の数値の上限(api の internal/domain/limits.go と同じ。セキュリティ点検 L-2)。

describe('記録の数値の上限', () => {
  test('スコアは 0 以上 MAX_SCORE 以下', () => {
    const status = (left: string) =>
      statusFormSchema.safeParse({ status: 'finished', leftScore: left, rightScore: '0' }).success
    assert.equal(status(String(MAX_SCORE)), true)
    assert.equal(status(String(MAX_SCORE + 1)), false)
    assert.equal(status('3000000000'), false)
    assert.equal(status('-1'), false)
  })

  test('選手記録の数は MAX_PLAYER_STAT 以下', () => {
    assert.deepEqual(playerStatsFromForm({ 'blocks-left-7': String(MAX_PLAYER_STAT) }).stats, [
      { playerId: 7, side: 'left', blocks: MAX_PLAYER_STAT, greatests: 0 },
    ])
    assert.throws(() => playerStatsFromForm({ 'blocks-left-7': '3000000000' }))
  })

  test('端末の時計が飛んでも経過秒は MAX_ELAPSED_SEC で止まる', () => {
    assert.equal(elapsedSeconds(0, 10 * MAX_ELAPSED_SEC * 1000), MAX_ELAPSED_SEC)
  })
})
