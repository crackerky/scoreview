import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { isSameOriginJson, parseMatchId } from '../src/admin/relay.utils.ts'
import {
  applyOp,
  classifyFailure,
  elapsedSeconds,
  newOpId,
  parseSavedQueue,
  project,
  type ScoreOp,
  undoTarget,
} from '../src/admin/scoreOps.utils.ts'
import { type MatchDetail, matchDetailSchema } from '../src/api/admin.types.ts'
import { loadFixture } from './fixtures/load.ts'

// 記録係の画面の、端末に溜める操作と見込みの状態(src/admin/scoreOps.utils.ts)。

const finished = matchDetailSchema.parse(loadFixture('admin-match-detail'))
const uuid = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/

/** fixture の試合を、得点経過の無い試合前の状態にする */
function scheduled(): MatchDetail {
  return {
    ...finished,
    events: [],
    startedAt: null,
    match: {
      ...finished.match,
      status: 'scheduled',
      winnerSide: null,
      left: { ...finished.match.left, score: null },
      right: { ...finished.match.right, score: null },
    },
  }
}

function add(
  n: number,
  kind: 'goal' | 'pull' | 'end',
  side: 'left' | 'right' | null,
  winnerSide: 'left' | 'right' | null = null,
): ScoreOp {
  return {
    type: 'event.add',
    opId: uuid(n),
    kind,
    side,
    elapsedSec: n * 60,
    scorerId: null,
    assistId: null,
    callahan: false,
    winnerSide,
    expectedScore: null,
  }
}

describe('見込みの状態', () => {
  test('プルで試合中になり、得点ごとに累計スコアが付く', () => {
    const view = project(scheduled(), [
      add(1, 'pull', 'left'),
      add(2, 'goal', 'right'),
      add(3, 'goal', 'right'),
    ])
    assert.equal(view.match.status, 'live')
    assert.equal(view.match.left.score, 0)
    assert.equal(view.match.right.score, 2)
    assert.deepEqual(
      view.events.map((event) => [event.leftScore, event.rightScore]),
      [
        [0, 0],
        [0, 1],
        [0, 2],
      ],
    )
  })

  test('まだ送っていない得点も clientId で取り消せ、得点者を後から付けられる', () => {
    const view = project(scheduled(), [
      add(1, 'goal', 'left'),
      add(2, 'goal', 'left'),
      { type: 'event.remove', opId: uuid(10), eventRef: uuid(2) },
      {
        type: 'event.update',
        opId: uuid(11),
        eventRef: uuid(1),
        scorerId: 100000001,
        assistId: 100000000,
        callahan: false,
      },
    ])
    assert.equal(view.match.left.score, 1)
    assert.equal(view.events.length, 1)
    assert.equal(view.events[0]?.scorer?.number, '10')
    assert.equal(view.events[0]?.assist?.number, '7')
  })

  test('同点の終了は勝者を持ち、終了の取り消し(試合中へ戻す)で end が消える', () => {
    const tied = project(scheduled(), [
      add(1, 'goal', 'left'),
      add(2, 'goal', 'right'),
      add(3, 'end', null, 'right'),
    ])
    assert.equal(tied.match.status, 'finished')
    assert.equal(tied.match.winnerSide, 'right')
    const reopened = applyOp(tied, {
      type: 'status',
      opId: uuid(9),
      status: 'live',
      winnerSide: null,
      expectedScore: null,
    })
    assert.equal(reopened.match.status, 'live')
    assert.equal(
      reopened.events.some((event) => event.kind === 'end'),
      false,
    )
  })

  test('結果の種類: 不戦勝は規定スコア、両チーム不戦敗はスコアなし、途中打ち切りはその時点のスコア', () => {
    const forfeit = applyOp(scheduled(), {
      type: 'result',
      opId: uuid(1),
      resultType: 'forfeit',
      winnerSide: 'right',
      forfeitScore: { winner: 15, loser: 0 },
      expectedScore: null,
    })
    assert.equal(forfeit.match.status, 'finished')
    assert.equal(forfeit.match.resultType, 'forfeit')
    assert.deepEqual([forfeit.match.left.score, forfeit.match.right.score], [0, 15])

    const both = applyOp(scheduled(), {
      type: 'result',
      opId: uuid(2),
      resultType: 'double_forfeit',
      winnerSide: null,
      forfeitScore: null,
      expectedScore: null,
    })
    assert.deepEqual([both.match.left.score, both.match.right.score], [null, null])

    const abandoned = applyOp(project(scheduled(), [add(1, 'goal', 'left')]), {
      type: 'result',
      opId: uuid(3),
      resultType: 'abandoned',
      winnerSide: null,
      forfeitScore: null,
      expectedScore: null,
    })
    assert.deepEqual([abandoned.match.left.score, abandoned.match.right.score], [1, 0])
  })
})

describe('取り消しの対象', () => {
  test('試合中は最後の記録、記録が無ければ無し', () => {
    assert.equal(undoTarget(scheduled(), false), null)
    const live = project(scheduled(), [add(1, 'pull', 'left'), add(2, 'goal', 'left')])
    const target = undoTarget(live, false)
    assert.equal(target?.kind, 'event')
    assert.equal(target?.kind === 'event' ? target.event.clientId : null, uuid(2))
  })

  test('終了した試合は管理者だけが試合中へ戻せる', () => {
    assert.equal(undoTarget(finished, false), null)
    assert.deepEqual(undoTarget(finished, true), { kind: 'reopen' })
  })
})

describe('端末への保存と送信', () => {
  test('保存した値が壊れていたら空にする', () => {
    assert.deepEqual(parseSavedQueue(null), { version: 1, ops: [], startedAt: null })
    assert.deepEqual(parseSavedQueue('{not json'), { version: 1, ops: [], startedAt: null })
    assert.deepEqual(parseSavedQueue('{"version":2}'), { version: 1, ops: [], startedAt: null })
    const saved = { version: 1, ops: [add(1, 'goal', 'left')], startedAt: 1000 }
    assert.deepEqual(parseSavedQueue(JSON.stringify(saved)), saved)
  })

  test('randomUUID が無い端末でも UUID v4 の形の ID を作る', () => {
    let counter = 0
    const fake = {
      getRandomValues: <T extends ArrayBufferView | null>(array: T): T => {
        if (array instanceof Uint8Array) {
          array.forEach((_, index) => {
            array[index] = (counter++ * 37) % 256
          })
        }
        return array
      },
    }
    assert.match(newOpId(fake), uuidPattern)
  })

  test('通信の失敗と 5xx は送り直し、401 はログイン、それ以外の 4xx はあきらめる', () => {
    assert.equal(classifyFailure(null), 'retry')
    assert.equal(classifyFailure(502), 'retry')
    assert.equal(classifyFailure(401), 'login')
    assert.equal(classifyFailure(409), 'drop')
    assert.equal(classifyFailure(403), 'drop')
  })

  test('経過秒は開始時刻からの秒数(開始前は 0)', () => {
    assert.equal(elapsedSeconds(null, 5000), 0)
    assert.equal(elapsedSeconds(1000, 62_500), 61)
  })

  test('Route Handler は同じサイトからの JSON だけを受ける', () => {
    const request = (headers: Record<string, string>) =>
      new Request('http://localhost:3003/admin/api/matches/1/ops', { method: 'POST', headers })
    const json = { 'content-type': 'application/json', host: 'localhost:3003' }
    const site = ['http://localhost:3003']
    assert.equal(
      isSameOriginJson(request({ ...json, origin: 'http://localhost:3003' }), site),
      true,
    )
    assert.equal(
      isSameOriginJson(request({ ...json, origin: 'https://evil.example' }), site),
      false,
    )
    assert.equal(isSameOriginJson(request(json), site), false)
    assert.equal(
      isSameOriginJson(
        request({
          'content-type': 'text/plain',
          host: 'localhost:3003',
          origin: 'http://localhost:3003',
        }),
        site,
      ),
      false,
    )
    assert.equal(parseMatchId('12'), 12)
    assert.equal(parseMatchId('1x'), null)
  })
})
