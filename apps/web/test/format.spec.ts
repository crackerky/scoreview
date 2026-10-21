import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  displayWidth,
  FormatError,
  formatAverage,
  formatElapsed,
  formatMonthDay,
  formatPercent,
  formatTimeCell,
  formatWinRate,
  formatYmdCompact,
  formatYmdSlash,
  scheduleDivisionWidth,
  scheduleTeamWidth,
  trimToWidth,
} from '../src/format/format.utils.ts'

describe('trimToWidth(clone-spec B 7 の実例)', () => {
  const division: Array<[string, string]> = [
    ['ビーチミックス', 'ビーチミッ'],
    ['ミックス STUDENT', 'ミックス S'],
    ['４人制部門', '４人制部門'],
    ['Beach Mixed', 'Beach Mixe'],
    ['Mix Student', 'Mix Studen'],
    ["Women's", "Women's"],
  ]
  for (const [input, expected] of division) {
    test(`ディビジョン名 ${input} → ${expected}`, () => {
      assert.equal(trimToWidth(input, scheduleDivisionWidth), expected)
    })
  }

  const team: Array<[string, string]> = [
    ["Aichi Syukutoku University Garnet's", 'Aichi Syukutoku '],
    ['Komazawa University Milagro', 'Komazawa Univers'],
    ['シロコロホルモンズ', 'シロコロホルモン'],
    ['バトルロイヤル', 'バトルロイヤル'],
  ]
  for (const [input, expected] of team) {
    test(`チーム名 ${input} → ${expected}`, () => {
      assert.equal(trimToWidth(input, scheduleTeamWidth), expected)
    })
  }

  test('全角文字が境界をまたぐときは手前で切る', () => {
    assert.equal(trimToWidth('aあい', 4), 'aあ')
  })

  test('半角カナは幅 1、全角スペースは幅 2', () => {
    assert.equal(displayWidth('ｽﾛｰｵﾌ'), 5)
    assert.equal(displayWidth('居酒屋　司'), 10)
  })
})

describe('formatWinRate(clone-spec C 1.3 の実例)', () => {
  const cases: Array<[number, number, string]> = [
    [5, 5, '100%'],
    [5, 6, '83.33%'],
    [4, 6, '66.67%'],
    [4, 5, '80%'],
    [3, 4, '75%'],
    [1, 6, '16.67%'],
    [0, 3, '0%'],
    [0, 0, '--%'],
  ]
  for (const [wins, games, expected] of cases) {
    test(`${wins}/${games} → ${expected}`, () => {
      assert.equal(formatWinRate(wins, games), expected)
    })
  }
})

describe('formatPercent(試合統計の内訳、clone-spec D 2.5)', () => {
  test('71.43% と 28.57%', () => {
    assert.equal(formatPercent(5, 7), '71.43%')
    assert.equal(formatPercent(2, 7), '28.57%')
  })
  test('8.33% と 50%', () => {
    assert.equal(formatPercent(1, 12), '8.33%')
    assert.equal(formatPercent(1, 2), '50%')
  })
  test('小数 1 桁目の 0 は残す', () => {
    assert.equal(formatPercent(1, 20), '5%')
    assert.equal(formatPercent(1, 1600), '0.06%')
  })
})

describe('formatAverage(clone-spec D 3.1 の実例)', () => {
  const cases: Array<[number, number, string]> = [
    [5, 4, '1.25'],
    [0, 4, '0'],
    [4, 4, '1'],
    [1, 3, '0.33'],
    [4, 3, '1.33'],
    [9, 4, '2.25'],
    [1, 6, '0.17'],
    [1, 2, '0.5'],
    [3, 0, '--'],
  ]
  for (const [total, games, expected] of cases) {
    test(`${total}/${games} → ${expected}`, () => {
      assert.equal(formatAverage(total, games), expected)
    })
  }
})

describe('日付と時刻', () => {
  test('MM/DD', () => {
    assert.equal(formatMonthDay('2025-10-18'), '10/18')
    assert.equal(formatMonthDay(null), '')
  })
  test('YYYY/MM/DD と YYYYMMDD', () => {
    assert.equal(formatYmdSlash('2025-03-16'), '2025/03/16')
    assert.equal(formatYmdCompact('2025-03-16'), '20250316')
  })
  test('時間列は HH:MM の後ろに半角スペース 1 つ', () => {
    assert.equal(formatTimeCell('09:00'), '09:00 ')
    assert.equal(formatTimeCell(null), '')
  })
  test('契約外の形は FormatError', () => {
    assert.throws(() => formatMonthDay('2025/10/18'), FormatError)
    assert.throws(() => formatTimeCell('9:00'), FormatError)
  })
  test('経過時間は MM:SS', () => {
    assert.equal(formatElapsed(0), '00:00')
    assert.equal(formatElapsed(1020), '17:00')
    assert.equal(formatElapsed(3000), '50:00')
    assert.equal(formatElapsed(65), '01:05')
  })
})
