import assert from 'node:assert/strict'
import { test } from 'node:test'
import { savedQueueKey, savedQueues } from '../src/admin/scoreOps.utils.ts'

// 記録係の端末に溜める未送信の操作は利用者ごとに分け、ログアウトで数えて消す(セキュリティ点検 L-9)。

/** localStorage と同じ読み方のできる Map */
function memoryStorage(entries: Record<string, string>) {
  const keys = Object.keys(entries)
  return {
    length: keys.length,
    key: (index: number) => keys[index] ?? null,
    getItem: (key: string) => entries[key] ?? null,
  }
}

const op = (n: number) => ({
  type: 'status',
  opId: `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`,
  status: 'live',
  winnerSide: null,
})

test('未送信の操作のキーは利用者と試合ごと', () => {
  assert.equal(savedQueueKey(3, 15), 'sv-score-queue:3:15')
  assert.notEqual(savedQueueKey(3, 15), savedQueueKey(4, 15))
})

test('ログアウトで消すキーと件数を数える(古い形式のキーも含め、他のアプリのキーは除く)', () => {
  const storage = memoryStorage({
    [savedQueueKey(3, 15)]: JSON.stringify({ version: 1, ops: [op(1), op(2)], startedAt: null }),
    'sv-score-queue:15': JSON.stringify({ version: 1, ops: [op(3)], startedAt: null }),
    'sv-score-queue:3:16': 'broken',
    'other-app': 'x',
  })
  assert.deepEqual(savedQueues(storage), [
    { key: 'sv-score-queue:3:15', ops: 2 },
    { key: 'sv-score-queue:15', ops: 1 },
    { key: 'sv-score-queue:3:16', ops: 0 },
  ])
})
