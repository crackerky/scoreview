import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { auditUser, describeAudit, formatAuditTime } from '../src/admin/audit.utils.ts'
import { type AuditEntry, auditListSchema } from '../src/api/admin.types.ts'

// 監査ログの 1 行を本部が読める文にする(src/admin/audit.utils.ts)。

const names = { left: '架空A', right: '架空B' }
const state = (status: string, left: number | null, right: number | null, extra = {}) => ({
  status,
  resultType: 'normal',
  leftScore: left,
  rightScore: right,
  winnerSide: null,
  ...extra,
})
const goal = (scorer: { id: number; number: string; name: string } | null) => ({
  clientId: '00000000-0000-4000-8000-000000000001',
  kind: 'goal',
  side: 'left',
  elapsedSec: 125,
  scorer,
  assist: null,
  callahan: false,
})

const entries: AuditEntry[] = auditListSchema.parse({
  entries: [
    {
      id: 3,
      at: '2026-10-25T01:02:03Z',
      user: { id: 2, name: '記録係 山田', role: 'staff' },
      action: 'event.update',
      matchId: 1,
      teamId: null,
      before: { match: state('live', 1, 0), event: goal(null), finalRank: null, note: null },
      after: {
        match: state('live', 1, 0),
        event: goal({ id: 7, number: '7', name: '選手1' }),
        finalRank: null,
        note: null,
      },
    },
    {
      id: 2,
      at: '2026-10-25T01:00:00Z',
      user: { id: 9, name: '', role: null },
      action: 'match.result',
      matchId: 1,
      teamId: null,
      before: { match: state('scheduled', null, null), event: null, finalRank: null, note: null },
      after: {
        match: state('finished', 15, 0, { resultType: 'forfeit', winnerSide: 'left' }),
        event: null,
        finalRank: null,
        note: null,
      },
    },
    {
      id: 1,
      at: '2026-10-25T00:59:00Z',
      user: { id: 1, name: '本部', role: 'admin' },
      action: 'event.add',
      matchId: 1,
      teamId: null,
      before: { match: state('scheduled', 0, 0), event: null, finalRank: null, note: null },
      after: { match: state('live', 1, 0), event: goal(null), finalRank: null, note: null },
    },
  ],
}).entries

describe('監査ログの文', () => {
  test('得点の記録は、何点目かと、変更前後のスコア', () => {
    const [, , added] = entries
    assert.ok(added)
    assert.deepEqual(describeAudit(added, names), {
      what: '記録',
      target: '2:05 架空A 得点',
      before: '試合前 0-0',
      after: '試合中 1-0',
    })
  })

  test('得点者の後付けは、選手の変化を出す', () => {
    const [updated] = entries
    assert.ok(updated)
    const line = describeAudit(updated, names)
    assert.equal(line.before, 'なし')
    assert.equal(line.after, '#7 選手1')
  })

  test('不戦勝は結果の種類と勝者を言葉で出す', () => {
    const [, result] = entries
    assert.ok(result)
    assert.equal(describeAudit(result, names).after, '終了 不戦勝 15-0 架空Aの勝ち')
    assert.equal(auditUser(result), '削除された利用者 #9')
  })

  test('得点のチームの訂正は、1 行でチームとスコアの変化を出す', () => {
    const [moved] = auditListSchema.parse({
      entries: [
        {
          id: 4,
          at: '2026-10-25T01:03:00Z',
          user: { id: 2, name: '記録係 山田', role: 'staff' },
          action: 'event.side',
          matchId: 1,
          teamId: null,
          before: { match: state('live', 2, 1), event: goal(null), finalRank: null, note: null },
          after: {
            match: state('live', 1, 2),
            event: { ...goal(null), side: 'right' },
            finalRank: null,
            note: null,
          },
        },
      ],
    }).entries
    assert.ok(moved)
    assert.deepEqual(describeAudit(moved, names), {
      what: '得点のチームの訂正',
      target: '2:05 架空A 得点',
      before: '架空Aの得点 試合中 2-1',
      after: '架空Bの得点 試合中 1-2',
    })
  })

  test('時刻は日本時間、操作した人は役割つき', () => {
    const [updated] = entries
    assert.ok(updated)
    assert.equal(formatAuditTime(updated.at), '10/25 10:02:03')
    assert.equal(auditUser(updated), '記録係 山田(記録係)')
  })
})
