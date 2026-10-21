import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  clearDeviceQueue,
  deviceQueueKey,
  type KeyValueStore,
  pendingTotal,
} from '../src/admin/scoreQueue.utils.ts'

// 記録係の端末に溜めた未送信の操作は利用者に結び付かないので、ログアウトで数えて消す
// (セキュリティ点検 L-9。src/components/admin/LogoutButton.tsx)。

function mapStore(entries: Record<string, string>): KeyValueStore {
  const values = new Map(Object.entries(entries))
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => {
      values.set(key, value)
    },
    removeItem: (key) => {
      values.delete(key)
    },
    key: (index) => [...values.keys()][index] ?? null,
    get length() {
      return values.size
    },
  }
}

const op = (n: number) => ({
  type: 'status',
  opId: `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`,
  status: 'live',
  winnerSide: null,
})

test('ログアウトで端末の未送信の操作を数えて消す(版 1 の試合ごとのキーも、他のアプリの値は残す)', () => {
  const store = mapStore({
    [deviceQueueKey]: JSON.stringify({
      version: 2,
      matches: { '15': { ops: [op(1), op(2)], startedAt: null } },
    }),
    'sv-score-queue:16': JSON.stringify({ version: 1, ops: [op(3)], startedAt: null }),
    'other-app': 'x',
  })
  assert.equal(pendingTotal(store), 3)
  clearDeviceQueue(store)
  assert.equal(pendingTotal(store), 0)
  assert.equal(store.getItem('sv-score-queue:16'), null)
  assert.equal(store.getItem('other-app'), 'x')
})
