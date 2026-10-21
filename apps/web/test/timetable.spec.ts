import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  buildTimetable,
  cellKey,
  compareFields,
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
