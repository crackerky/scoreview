import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  courtLabel,
  isLongElapsed,
  restoreOp,
  scorerStateOf,
  UNDO_COOLDOWN_MS,
  undoAllowed,
  winnerLine,
} from '../src/admin/scoreKeeper.utils.ts'
import { applyOp, project, type ScoreOp } from '../src/admin/scoreOps.utils.ts'
import { loginPath, safeNext } from '../src/admin/session.utils.ts'
import { type MatchDetail, matchDetailSchema } from '../src/api/admin.types.ts'
import { loadFixture } from './fixtures/load.ts'

// 記録係のスコア入力画面の判断(src/admin/scoreKeeper.utils.ts)と、画面に重ねる見込みの状態。

const finished = matchDetailSchema.parse(loadFixture('admin-match-detail'))
const uuid = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`

function live(): MatchDetail {
  return {
    ...finished,
    events: [],
    startedAt: null,
    match: {
      ...finished.match,
      status: 'live',
      winnerSide: null,
      left: { ...finished.match.left, score: 0 },
      right: { ...finished.match.right, score: 0 },
    },
  }
}

function goal(n: number, side: 'left' | 'right'): ScoreOp {
  return {
    type: 'event.add',
    opId: uuid(n),
    kind: 'goal',
    side,
    elapsedSec: n * 60,
    scorerId: null,
    assistId: null,
    callahan: false,
    winnerSide: null,
    expectedScore: null,
  }
}

describe('取り消しの連打', () => {
  test('前の取り消しから 1 秒たつまでは次の取り消しを受け付けない', () => {
    assert.equal(undoAllowed(null, 5000), true)
    assert.equal(undoAllowed(5000, 5000 + UNDO_COOLDOWN_MS - 1), false)
    assert.equal(undoAllowed(5000, 5000 + UNDO_COOLDOWN_MS), true)
  })

  test('「元に戻す」は取り消した得点を同じ内容で足し直す', () => {
    const scored = project(live(), [goal(1, 'left'), goal(2, 'right')])
    const last = scored.events.at(-1)
    assert.ok(last)
    const removed = applyOp(scored, {
      type: 'event.remove',
      opId: uuid(3),
      eventRef: last.clientId,
    })
    assert.deepEqual([removed.match.left.score, removed.match.right.score], [1, 0])
    const restored = applyOp(removed, restoreOp(last, uuid(4)))
    assert.deepEqual([restored.match.left.score, restored.match.right.score], [1, 1])
    assert.equal(restored.events.at(-1)?.elapsedSec, last.elapsedSec)
  })
})

describe('得点の訂正', () => {
  test('キャラハンは得点者だけを付けてアシストなしで保存する', () => {
    const scorer = finished.rosters.left.find((player) => player.role === null)
    assert.ok(scorer)
    const view = project(live(), [
      goal(1, 'left'),
      {
        type: 'event.update',
        opId: uuid(2),
        eventRef: uuid(1),
        scorerId: scorer.id,
        assistId: null,
        callahan: true,
      },
    ])
    const event = view.events[0]
    assert.equal(event?.scorer?.id, scorer.id)
    assert.equal(event?.assist, null)
    assert.equal(event?.callahan, true)
  })

  test('数点前の得点を相手チームの得点にすると、その後の累計スコアも直る', () => {
    const view = project(live(), [
      goal(1, 'left'),
      goal(2, 'left'),
      goal(3, 'right'),
      { type: 'event.side', opId: uuid(4), eventRef: uuid(1), side: 'right' },
    ])
    assert.deepEqual(
      view.events.map((event) => `${event.leftScore}-${event.rightScore}`),
      ['0-1', '1-1', '1-2'],
    )
    assert.deepEqual([view.match.left.score, view.match.right.score], [1, 2])
  })

  test('api に届いた後の取り直しに既にある得点は、もう一度重ねない', () => {
    const sent = project(live(), [goal(1, 'left')])
    const again = project(sent, [goal(1, 'left')])
    assert.equal(again.events.length, 1)
    assert.equal(again.match.left.score, 1)
  })
})

describe('終了後の勝者', () => {
  test('同点で勝者を決めた試合は「勝者: X(同点)」', () => {
    const tied: MatchDetail = {
      ...finished,
      match: {
        ...finished.match,
        winnerSide: 'right',
        left: { ...finished.match.left, score: 5 },
        right: { ...finished.match.right, score: 5 },
      },
    }
    assert.equal(winnerLine(tied), '勝者: 架空2(同点)')
    assert.equal(winnerLine({ ...tied, match: { ...tied.match, winnerSide: null } }), '引き分け')
  })

  test('スコアに差があれば勝者はスコアから', () => {
    assert.match(winnerLine(finished) ?? '', /^勝者: /)
  })
})

describe('試合の記録担当', () => {
  const now = Date.parse('2026-10-21T00:00:10Z')
  const claim = (deviceId: string, userId = 7, expiresAt = '2026-10-21T00:00:30Z') => ({
    userId,
    name: '記録係 山田',
    deviceId,
    expiresAt,
  })

  test('他の端末が期限内の記録担当なら、引き継ぐまで記録させない', () => {
    assert.deepEqual(scorerStateOf(claim(uuid(2)), uuid(1), 9, now), {
      kind: 'other',
      name: '記録係 山田',
      sameUser: false,
    })
    assert.deepEqual(scorerStateOf(claim(uuid(2), 9), uuid(1), 9, now), {
      kind: 'other',
      name: '記録係 山田',
      sameUser: true,
    })
  })

  test('この端末が担当なら mine、期限切れや担当なしなら free', () => {
    assert.deepEqual(scorerStateOf(claim(uuid(1)), uuid(1), 9, now), { kind: 'mine' })
    assert.deepEqual(scorerStateOf(claim(uuid(2), 7, '2026-10-21T00:00:05Z'), uuid(1), 9, now), {
      kind: 'free',
    })
    assert.deepEqual(scorerStateOf(null, uuid(1), 9, now), { kind: 'free' })
  })
})

describe('画面の細部', () => {
  test('経過時間が試合時間の 1.5 倍を超えたら確かめる(試合時間が無ければ 90 分)', () => {
    assert.equal(isLongElapsed(null, null), false)
    assert.equal(isLongElapsed(135 * 60, null), false)
    assert.equal(isLongElapsed(135 * 60 + 1, null), true)
    assert.equal(isLongElapsed(46 * 60, 30), true)
  })

  test('コートの値に「コート」が付いていれば重ねない', () => {
    assert.equal(courtLabel('コート1'), 'コート1')
    assert.equal(courtLabel('3'), 'コート 3')
    assert.equal(courtLabel(null), '')
  })

  test('ログインの後は元の管理画面へ戻す。他のサイトへは飛ばさない', () => {
    assert.equal(safeNext('/admin/matches/12/score'), '/admin/matches/12/score')
    assert.equal(safeNext('https://evil.example/admin'), '/admin')
    assert.equal(safeNext('//evil.example'), '/admin')
    assert.equal(safeNext('/admin//evil'), '/admin')
    assert.equal(safeNext('/admin/login'), '/admin')
    assert.equal(
      loginPath('/admin/matches/12/score'),
      '/admin/login?next=%2Fadmin%2Fmatches%2F12%2Fscore',
    )
    assert.equal(loginPath(), '/admin/login')
  })
})
