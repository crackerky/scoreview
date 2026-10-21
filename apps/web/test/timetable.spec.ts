import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  buildTimetable,
  cellKey,
  compareFields,
  sideLabelsOf,
  sideText,
  type TimetableMatch,
} from '../src/admin/timetable.utils.ts'

// 時間割(コート × 時刻)の組み立て(src/admin/timetable.utils.ts)。

function match(
  id: number,
  date: string | null,
  time: string | null,
  field: string | null,
): TimetableMatch {
  const side = { team: null, placeholder: null, score: null, labelEn: '' }
  return {
    id,
    divisionId: 1,
    groupId: null,
    code: `A${id}`,
    date,
    time,
    field,
    hasDetail: true,
    status: 'scheduled',
    left: side,
    right: side,
    winnerSide: null,
    resultType: 'normal',
    divisionName: 'オープン',
  }
}

describe('時間割', () => {
  test('コートは数字を数値順で先に、それ以外を後に並べる', () => {
    assert.deepEqual(['B', '10', '2', 'A'].sort(compareFields), ['2', '10', 'A', 'B'])
  })

  test('日付ごとに時刻 × コートの枠へ入れ、未定の試合は別にする', () => {
    const days = buildTimetable([
      match(1, '2026-10-26', '09:00', '1'),
      match(2, '2026-10-25', '10:00', '10'),
      match(3, '2026-10-25', '09:00', '2'),
      match(4, '2026-10-25', null, '2'),
      match(5, null, '09:00', '1'),
    ])
    assert.deepEqual(
      days.map((day) => day.date),
      ['2026-10-25', '2026-10-26', null],
    )
    const [first] = days
    assert.ok(first)
    assert.deepEqual(first.fields, ['2', '10'])
    assert.deepEqual(first.times, ['09:00', '10:00'])
    assert.equal(first.cells.get(cellKey('09:00', '2'))?.[0]?.id, 3)
    assert.deepEqual(
      first.unplaced.map((item) => item.id),
      [4],
    )
  })
})

describe('印刷物の未確定の対戦(H11)', () => {
  const empty = { team: null, placeholder: null, score: null, labelEn: '' }
  test('形の決まり方があれば運営の言葉で出す', () => {
    assert.equal(sideText(empty, 'A1'), 'A組1位')
    assert.equal(sideText(empty, '[E2-勝ち]'), 'E2 の勝者')
  })
  test('決まり方が無ければ参照サイトの表記を言い換え、何も無ければ未定', () => {
    const placeholder = { ...empty, placeholder: { code: 'E4', outcome: 'lose' as const } }
    assert.equal(sideText(placeholder, undefined), 'E4 の負けチーム')
    assert.equal(sideText(empty, undefined), '未定')
  })
  test('形の試合一覧から試合 ID ごとの決まり方を作る', () => {
    const side = (label: string) => ({ label, team: null, score: null })
    const labels = sideLabelsOf([
      {
        id: 7,
        code: 'E4',
        groupName: '1st placement tournament',
        date: null,
        time: null,
        field: null,
        status: 'scheduled',
        left: side('A1'),
        right: side('C2'),
      },
    ])
    assert.deepEqual(labels.get(7), { left: 'A1', right: 'C2' })
  })
})
