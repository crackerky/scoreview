import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { clockOf, courtOf, dayOf, parseSchedule } from '../src/admin/schedulePaste.utils.ts'

const teams = [
  { id: 1, fullName: '熱海ディスクス', shortName: '熱海' },
  { id: 2, fullName: '伊東フライヤーズ', shortName: '伊東' },
  { id: 3, fullName: '沼津ウェーブ', shortName: '沼津' },
  { id: 4, fullName: '三島スカイ', shortName: '三島' },
]
const matches = [
  { id: 10, code: 'A1', status: 'scheduled', leftTeamId: 1, rightTeamId: 2 },
  { id: 11, code: 'A2', status: 'scheduled', leftTeamId: 3, rightTeamId: 4 },
  { id: 12, code: 'A3', status: 'scheduled', leftTeamId: 1, rightTeamId: 3 },
  { id: 13, code: 'E1', status: 'scheduled', leftTeamId: null, rightTeamId: null },
  { id: 14, code: 'A4', status: 'finished', leftTeamId: 2, rightTeamId: 4 },
]

describe('時間割の貼り付けの値', () => {
  test('時刻、日付、コート', () => {
    assert.equal(clockOf('8:30'), '08:30')
    assert.equal(clockOf('８時３０分'), '08:30')
    assert.equal(clockOf('9時'), '09:00')
    assert.equal(clockOf('08:30:00'), '08:30')
    assert.equal(clockOf('25:00'), null)
    assert.equal(dayOf('2/9', '2027-02-08'), '2027-02-09')
    assert.equal(dayOf('2027/2/10', '2026-01-01'), '2027-02-10')
    assert.equal(dayOf('2月9日', '2027-01-01'), '2027-02-09')
    assert.equal(dayOf('あした', '2027-01-01'), null)
    assert.equal(courtOf('コート3'), '3')
    assert.equal(courtOf('Court 12'), '12')
    assert.equal(courtOf('第2コート'), '2')
    assert.equal(courtOf('a'), 'A')
    assert.equal(courtOf('メイン'), 'メイン')
  })
})

describe('時間割の貼り付け', () => {
  test('一覧の形(見出し、日付、試合番号かチームの組)', () => {
    const got = parseSchedule(
      '日付\t時刻\tコート\t試合番号\tチーム\tチーム\n2/9\t8:30\tコート1\t\t熱海\t伊東\n2/9\t8:30\tコート2\tA2\t\t\n2/9\t9:20\t1\t\t沼津ウェーブ\t熱海',
      '2027-02-09',
      matches,
      teams,
    )
    assert.deepEqual(got.problems, [])
    assert.deepEqual(got.rows, [
      { matchId: 10, code: 'A1', date: '2027-02-09', time: '08:30', field: '1' },
      { matchId: 11, code: 'A2', date: '2027-02-09', time: '08:30', field: '2' },
      { matchId: 12, code: 'A3', date: '2027-02-09', time: '09:20', field: '1' },
    ])
    assert.deepEqual(got.untouched, ['E1'], 'matches not in the table keep their times')
  })
  test('格子の形(1 行目がコート、枠に試合番号か「A - B」。日付は画面で選んだ日)', () => {
    const got = parseSchedule(
      '時刻\tコート1\tコート2\n8:30\t熱海 - 伊東\t沼津 vs 三島\n9:20\tA3\tE1',
      '2027-02-10',
      matches,
      teams,
    )
    assert.deepEqual(got.problems, [])
    assert.deepEqual(
      got.rows.map((row) => `${row.code} ${row.date} ${row.time} ${row.field}`),
      [
        'A1 2027-02-10 08:30 1',
        'A2 2027-02-10 08:30 2',
        'A3 2027-02-10 09:20 1',
        'E1 2027-02-10 09:20 2',
      ],
    )
  })
  test('当てはまらない行、始まった試合、2 回出てくる試合、重なりを知らせる', () => {
    const got = parseSchedule(
      '時刻\tコート1\tコート2\n8:30\tA1\tA3\n9:20\tA4\t下田 - 伊東\n10:10\tA1\t',
      '2027-02-09',
      matches,
      teams,
    )
    assert.equal(got.problems.length, 4)
    assert.match(got.problems.join('\n'), /A4 はもう始まっている/)
    assert.match(got.problems.join('\n'), /下田 - 伊東/)
    assert.match(got.problems.join('\n'), /A1 が 2 回/)
    assert.match(got.problems.join('\n'), /同じチームが A1 と A3/)
  })
  test('読めない表', () => {
    assert.equal(parseSchedule('A1\nA2', '2027-02-09', matches, teams).problems.length, 1)
  })
})
