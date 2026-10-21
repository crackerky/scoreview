import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { auditLines } from '../src/admin/audit.utils.ts'
import {
  forfeitScoreText,
  hqTabOf,
  operatorSideName,
  shiftSentence,
  suspendedMinutes,
  updatedAtText,
  withdrawalSentence,
} from '../src/admin/hq.utils.ts'
import { auditListSchema } from '../src/api/admin.types.ts'
import { closingSchema, withdrawalSchema } from '../src/api/format.types.ts'
import { suspensionText } from '../src/components/suspension.utils.ts'

// 本部の進行画面と確認画面の文(src/admin/hq.utils.ts、レビュー H2、H3、H10、M3、M5、M7、L2)。

const team = (id: number, name: string) => ({ id, name: { ja: name, en: '' } })

const withdrawal = withdrawalSchema.parse({
  team: team(8, 'チームH'),
  divisionId: 1,
  withdrawnAt: null,
  targets: [
    {
      id: 30,
      code: 'C3',
      date: '2026-10-18',
      time: '10:20',
      field: '2',
      status: 'scheduled',
      opponent: team(3, 'チームC'),
      result: 'forfeit',
    },
    {
      id: 71,
      code: 'G3',
      date: '2026-10-18',
      time: '13:00',
      field: '1',
      status: 'scheduled',
      opponent: team(9, 'チームI'),
      result: 'forfeit',
    },
  ],
  waiting: [],
  forfeitScore: null,
})

describe('本部の確認の文', () => {
  test('棄権: 残りの試合を並べて相手の不戦勝にすると言う', () => {
    assert.equal(withdrawalSentence(withdrawal), '残り 2 試合(C3、G3)を相手の不戦勝にします')
    assert.equal(
      withdrawalSentence({ ...withdrawal, targets: [] }),
      '今すぐ記録する試合はありません',
    )
  })

  test('規定スコアが無いときは勝敗だけを記録すると言う(スコアで記録するとは言わない)', () => {
    assert.match(forfeitScoreText(null), /勝敗だけを記録/)
    assert.equal(forfeitScoreText({ winner: 15, loser: 0 }), '不戦勝は規定スコア 15-0 で記録します')
  })

  test('遅れの調整の確認', () => {
    assert.equal(
      shiftSentence('3', 8, 15),
      'コート 3 のまだ始まっていない 8 試合を 15 分遅らせます',
    )
    assert.equal(
      shiftSentence(null, 20, -10),
      '全コートのまだ始まっていない 20 試合を 10 分早めます',
    )
  })

  test('中断の帯と再開でずらす分数', () => {
    const suspension = {
      reason: 'thunder' as const,
      note: '',
      resumeAt: '13:30',
      since: '2026-10-18T04:00:00Z',
    }
    assert.equal(suspensionText(suspension, 'jp'), '雷のため中断中(再開予定 13:30)')
    assert.equal(suspensionText(suspension, 'en'), 'Suspended due to lightning (resuming at 13:30)')
    assert.equal(
      suspensionText({ ...suspension, reason: 'other', note: '停電', resumeAt: null }, 'jp'),
      '停電のため中断中(再開時刻は未定)',
    )
    assert.equal(suspendedMinutes(suspension.since, new Date('2026-10-18T04:17:00Z')), 20)
    assert.equal(suspendedMinutes(suspension.since, new Date('2026-10-18T03:59:00Z')), 0)
  })

  test('最終更新の時刻は日本時間、タブは知らない値なら「いまやること」', () => {
    assert.equal(updatedAtText(new Date('2026-10-18T04:42:10Z')), '最終更新 13:42:10')
    assert.equal(hqTabOf('delay'), 'delay')
    assert.equal(hqTabOf('x'), 'todo')
    assert.equal(hqTabOf(undefined), 'todo')
  })

  test('未定のチームは「E4 の負けチーム」と書く', () => {
    const side = (placeholder: { code: string; outcome: 'win' | 'lose' } | null) => ({
      team: null,
      score: null,
      labelEn: '',
      placeholder,
    })
    assert.equal(operatorSideName(side({ code: 'E4', outcome: 'lose' })), 'E4 の負けチーム')
    assert.equal(operatorSideName(side({ code: 'E2', outcome: 'win' })), 'E2 の勝ちチーム')
    assert.equal(operatorSideName(side(null)), '未定')
  })

  test('締めの応答のスキーマ', () => {
    const closing = closingSchema.parse({
      remaining: [],
      ranks: [{ rank: 9, team: team(1, 'A'), tied: true, proposed: true }],
      ties: ['9〜12 位は 9 位タイ(A、B、C、D)'],
      closedAt: '2026-10-18T08:00:00Z',
      published: true,
    })
    assert.equal(closing.ranks[0]?.proposed, true)
  })
})

describe('変更の履歴', () => {
  const state = (status: string, left: number, right: number) => ({
    status,
    resultType: 'normal',
    leftScore: left,
    rightScore: right,
    winnerSide: null,
  })
  const snapshot = (match: ReturnType<typeof state>, note: string | null = null) => ({
    match,
    event: null,
    finalRank: null,
    note,
  })
  const user = { id: 1, name: '本部', role: 'admin' }
  const names = { left: 'A', right: 'B' }

  test('1 回の訂正(終了 → 試合中 → 終了)は 1 行にまとめる', () => {
    const entries = auditListSchema.parse({
      entries: [
        {
          id: 3,
          at: '2026-10-21T10:52:41Z',
          user,
          action: 'match.status',
          matchId: 1,
          teamId: null,
          before: snapshot(state('live', 11, 6)),
          after: snapshot(state('finished', 11, 9)),
        },
        {
          id: 2,
          at: '2026-10-21T10:52:41Z',
          user,
          action: 'match.status',
          matchId: 1,
          teamId: null,
          before: snapshot(state('finished', 11, 6)),
          after: snapshot(state('live', 11, 6)),
        },
        {
          id: 1,
          at: '2026-10-21T10:49:07Z',
          user,
          action: 'match.status',
          matchId: 1,
          teamId: null,
          before: snapshot(state('scheduled', 0, 0)),
          after: snapshot(
            state('finished', 11, 6),
            '得点経過(9-8)と違う最終スコア(11-6)で本部が確定',
          ),
        },
      ],
    }).entries
    const lines = auditLines(entries, names)
    assert.equal(lines.length, 2)
    assert.deepEqual(lines[0]?.line, {
      what: '結果の訂正',
      target: '',
      before: '終了 11-6',
      after: '終了 11-9',
    })
    assert.equal(lines[1]?.line.target, '得点経過(9-8)と違う最終スコア(11-6)で本部が確定')
  })

  test('別の人の操作や時間の離れた操作はまとめない', () => {
    const entries = auditListSchema.parse({
      entries: [
        {
          id: 2,
          at: '2026-10-21T11:00:00Z',
          user: { id: 2, name: '記録係', role: 'staff' },
          action: 'match.status',
          matchId: 1,
          teamId: null,
          before: snapshot(state('live', 11, 6)),
          after: snapshot(state('finished', 11, 9)),
        },
        {
          id: 1,
          at: '2026-10-21T10:59:59Z',
          user,
          action: 'match.status',
          matchId: 1,
          teamId: null,
          before: snapshot(state('finished', 11, 6)),
          after: snapshot(state('live', 11, 6)),
        },
      ],
    }).entries
    assert.equal(auditLines(entries, names).length, 2)
  })
})
