import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { ScoreOp } from '../src/admin/scoreOps.utils.ts'
import {
  deviceQueueKey,
  enqueueOp,
  type KeyValueStore,
  matchQueueOf,
  pendingCounts,
  readDeviceQueue,
} from '../src/admin/scoreQueue.utils.ts'
import { type SendFailure, sendPending } from '../src/admin/scoreSend.utils.ts'
import { matchDetailSchema } from '../src/api/admin.types.ts'
import { loadFixture } from './fixtures/load.ts'

// 記録係の未送信の操作を端末全体で持ち、どの画面からでも全試合ぶんを送る
// (src/admin/scoreQueue.utils.ts と scoreSend.utils.ts)。

const detail = matchDetailSchema.parse(loadFixture('admin-match-detail'))
const uuid = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`

function memory(): KeyValueStore & { values: Map<string, string> } {
  const values = new Map<string, string>()
  return {
    values,
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

function goal(n: number): ScoreOp {
  return {
    type: 'event.add',
    opId: uuid(n),
    kind: 'goal',
    side: 'left',
    elapsedSec: n,
    scorerId: null,
    assistId: null,
    callahan: false,
    winnerSide: null,
    expectedScore: null,
  }
}

describe('端末全体の未送信の操作', () => {
  test('試合ごとに記録した順で持ち、試合ごとの件数を数える', () => {
    const store = memory()
    enqueueOp(store, 10, goal(1), 1000)
    enqueueOp(store, 20, goal(2), null)
    enqueueOp(store, 10, goal(3), 2000)
    const queue = readDeviceQueue(store)
    assert.deepEqual(
      matchQueueOf(queue, 10).ops.map((op) => op.opId),
      [uuid(1), uuid(3)],
    )
    assert.equal(matchQueueOf(queue, 10).startedAt, 1000, '経過 0 秒の時刻は最初の記録のまま')
    assert.deepEqual(
      [...pendingCounts(queue)],
      [
        [10, 2],
        [20, 1],
      ],
    )
  })

  test('古い画面が試合ごとのキーに残した操作も読み、書くときに 1 つのキーへ移す', () => {
    const store = memory()
    store.setItem(
      `${deviceQueueKey}:30`,
      JSON.stringify({ version: 1, ops: [goal(5)], startedAt: 500 }),
    )
    enqueueOp(store, 30, goal(6), null)
    assert.deepEqual(
      matchQueueOf(readDeviceQueue(store), 30).ops.map((op) => op.opId),
      [uuid(5), uuid(6)],
    )
    assert.equal(store.getItem(`${deviceQueueKey}:30`), null)
  })

  test('壊れた値は空として読む', () => {
    const store = memory()
    store.setItem(deviceQueueKey, '{not json')
    assert.equal(pendingCounts(readDeviceQueue(store)).size, 0)
  })
})

describe('全試合ぶんの送信', () => {
  const ok = { status: 200, body: { detail } }

  test('どの試合の操作も、試合の中は記録した順に送り、送れたものを消す', async () => {
    const store = memory()
    enqueueOp(store, 10, goal(1), null)
    enqueueOp(store, 20, goal(2), null)
    enqueueOp(store, 10, goal(3), null)
    const sent: string[] = []
    const applied: number[] = []
    const result = await sendPending(
      {
        store,
        post: async (matchId, op) => {
          sent.push(`${matchId}:${op.opId}`)
          return ok
        },
      },
      { applied: (matchId) => applied.push(matchId), dropped: () => assert.fail('not dropped') },
    )
    assert.equal(result, 'done')
    assert.deepEqual(sent, [`10:${uuid(1)}`, `10:${uuid(3)}`, `20:${uuid(2)}`])
    assert.deepEqual(applied, [10, 10, 20])
    assert.equal(pendingCounts(readDeviceQueue(store)).size, 0)
  })

  test('圏外なら止めて、送れなかった操作を残す', async () => {
    const store = memory()
    enqueueOp(store, 10, goal(1), null)
    enqueueOp(store, 10, goal(2), null)
    let calls = 0
    const result = await sendPending(
      {
        store,
        post: async () => {
          calls++
          if (calls === 2) {
            throw new TypeError('Failed to fetch')
          }
          return ok
        },
      },
      { applied: () => {}, dropped: () => {} },
    )
    assert.equal(result, 'offline')
    assert.deepEqual(
      matchQueueOf(readDeviceQueue(store), 10).ops.map((op) => op.opId),
      [uuid(2)],
    )
  })

  test('送っている間に別の画面が足した操作も続けて送る', async () => {
    const store = memory()
    enqueueOp(store, 10, goal(1), null)
    const sent: string[] = []
    await sendPending(
      {
        store,
        post: async (_matchId, op) => {
          sent.push(op.opId)
          if (op.opId === uuid(1)) {
            enqueueOp(store, 40, goal(9), null)
          }
          return ok
        },
      },
      { applied: () => {}, dropped: () => {} },
    )
    assert.deepEqual(sent, [uuid(1), uuid(9)])
  })

  test('api が断った操作は捨てて理由を知らせ、次の操作を送る。ログイン切れは止めて残す', async () => {
    const store = memory()
    enqueueOp(store, 10, goal(1), null)
    enqueueOp(store, 10, goal(2), null)
    const dropped: SendFailure[] = []
    const result = await sendPending(
      {
        store,
        post: async (_matchId, op) =>
          op.opId === uuid(1)
            ? {
                status: 409,
                body: { error: { code: 'score_mismatch', message: 'the score is now 2-3' } },
              }
            : { status: 401, body: { error: { code: 'unauthorized', message: 'login' } } },
      },
      { applied: () => {}, dropped: (_matchId, _op, failure) => dropped.push(failure) },
    )
    assert.equal(result, 'login')
    assert.deepEqual(dropped, [
      { status: 409, code: 'score_mismatch', message: 'the score is now 2-3' },
    ])
    assert.deepEqual(
      matchQueueOf(readDeviceQueue(store), 10).ops.map((op) => op.opId),
      [uuid(2)],
    )
  })
})
