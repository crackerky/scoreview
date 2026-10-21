import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  adminBracketText,
  buildFormatInput,
  buildPreviewDays,
  copyFormat,
  describeSide,
  endText,
  initialConditions,
  parseSport,
  sameInput,
  scheduleOf,
  shiftConditions,
  sportFromContext,
  type WizardForm,
  withDayCount,
  withSport,
} from '../src/admin/formatWizard.utils.ts'
import type { PreviewMatch, SavedFormat } from '../src/api/format.types.ts'

const baseForm: WizardForm = {
  cond: {
    ...initialConditions(null, '2026-11-14'),
    courts: 3,
    days: [{ date: '2026-11-14', start: '09:30', end: '16:00' }],
  },
  chosen: { template: 'pool_bands', params: { teams: 12 }, title: '今回の形' },
  seeds: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
  tiebreak: 'current',
  forfeit: { winner: 15, loser: 0 },
}

const previous: SavedFormat = {
  divisionId: 9,
  tournamentId: 3,
  tournamentName: '前回の大会',
  divisionName: '4on4',
  title: '4 プール(各 3 チーム) → 4 チームのトーナメント × 3(1〜4 位、5〜8 位、9〜12 位)',
  version: 2,
  template: 'pool_bands',
  params: { teams: 12, pools: 4 },
  tiebreak: 'jfda',
  forfeit: { winner: 7, loser: 0 },
  schedule: {
    date: '2025-11-01',
    start: '08:00',
    slotMinutes: 20,
    courts: ['1', '2', '3', '4'],
    teamRestSlots: 0,
    stageGapSlots: 0,
    end: '18:00',
    days: [{ date: '2025-11-01', start: '08:00', end: '18:00' }],
    lunch: null,
    stageDays: false,
    sport: 'four',
  },
  seeds: [],
}

describe('大会の形のウィザードの条件', () => {
  test('前回の形のコピーは形と規則だけを引き継ぎ、今回入れたコート数と時刻を変えない(C3)', () => {
    const { form, notes } = copyFormat(baseForm, previous)
    assert.deepEqual(form.cond, baseForm.cond, 'コート数 3、9:30〜16:00 のまま')
    assert.equal(form.chosen?.title, previous.title)
    assert.equal(form.tiebreak, 'jfda')
    assert.deepEqual(form.forfeit, { winner: 7, loser: 0 })
    assert.deepEqual(notes, [
      `形を前回の「${previous.title}」にしました`,
      '同点の決め方を前回の「JFDA 競技規則」にしました',
      '不戦勝の規定スコアを前回の 7-0 にしました',
      'コート数、日付、時刻、昼休みは今回入れた条件のままです',
    ])
    const input = buildFormatInput(form)
    assert.deepEqual(input?.schedule?.courts, ['1', '2', '3'])
    assert.equal(input?.schedule?.start, '09:30')
  })

  test('プレビューと作成は同じ入力から作る(条件を変えたら同じでなくなる)', () => {
    const previewed = buildFormatInput(baseForm)
    assert.ok(sameInput(previewed, buildFormatInput({ ...baseForm })))
    const changed = { ...baseForm, cond: { ...baseForm.cond, courts: 4 } }
    assert.equal(sameInput(previewed, buildFormatInput(changed)), false)
    assert.equal(buildFormatInput({ ...baseForm, chosen: null }), null)
  })

  test('日数を増やすと翌日を同じ時間で足し、1 日に戻すと段の日分けを外す', () => {
    const two = withDayCount({ ...baseForm.cond, stageDays: true }, 2)
    assert.deepEqual(two.days[1], { date: '2026-11-15', start: '09:30', end: '16:00' })
    assert.equal(two.stageDays, true)
    const one = withDayCount(two, 1)
    assert.equal(one.days.length, 1)
    assert.equal(one.stageDays, false)
    assert.equal(withDayCount(two, 30).days.length, 7)
  })

  test('時間割の設定: 1 日目を date に、昼休みは長さ 0 でなし、段の日分けは 2 日以上だけ', () => {
    const cond = {
      ...withDayCount(baseForm.cond, 2),
      stageDays: true,
      lunchStart: '12:00',
      lunchMinutes: 60,
    }
    const schedule = scheduleOf(cond)
    assert.equal(schedule.date, '2026-11-14')
    assert.equal(schedule.end, '16:00')
    assert.equal(schedule.days.length, 2)
    assert.deepEqual(schedule.lunch, { start: '12:00', minutes: 60 })
    assert.equal(schedule.stageDays, true)
    assert.equal(scheduleOf({ ...cond, lunchMinutes: 0 }).lunch, null)
    assert.equal(scheduleOf(withDayCount(cond, 1)).stageDays, false)
  })

  test('開き直したときは保存した条件を欄に出し、大会のコピーでは日付だけ新しい開催日にする', () => {
    const saved = initialConditions(previous.schedule, '2026-11-14')
    assert.equal(saved.courts, 4)
    assert.equal(saved.sport, 'four')
    assert.deepEqual(saved.days, [{ date: '2025-11-01', start: '08:00', end: '18:00' }])
    const shifted = shiftConditions(withDayCount(saved, 2), '2026-11-14')
    assert.deepEqual(
      shifted.days.map((day) => day.date),
      ['2026-11-14', '2026-11-15'],
    )
    assert.equal(withSport(saved, 'seven').slotMinutes, 60)
    assert.equal(withSport(saved, 'seven').restSlots, 1)
  })
})

describe('管理画面の言い方', () => {
  test('未確定の対戦を運営の言葉にする', () => {
    assert.equal(describeSide('A1'), 'A組1位')
    assert.equal(describeSide('[E2-勝ち]'), 'E2 の勝者')
    assert.equal(describeSide('[E4-負け]'), 'E4 の負けチーム')
    assert.equal(describeSide('シード3'), 'シード3')
  })

  test('プレビューのトーナメント表の見出し', () => {
    assert.equal(adminBracketText('Semifinals'), '準決勝')
    assert.equal(adminBracketText('FINAL'), '決勝')
    assert.equal(adminBracketText('1st Round'), '1 回戦')
    assert.equal(adminBracketText('5th(tied)'), '5 位(同順位)')
    assert.equal(adminBracketText('E2'), 'E2')
    assert.equal(adminBracketText('[E2-負け]'), 'E2 の負けチーム')
  })

  test('終了の言い方', () => {
    assert.equal(endText('16:00', 2), '2 日目の 16:00 終了')
    assert.equal(endText('12:40', 1), '12:40 終了')
    assert.equal(endText(null, null), '時刻未定')
  })

  test('プレビューの時間割を日付ごとに分ける', () => {
    const match = (code: string, date: string, time: string, field: string): PreviewMatch => ({
      code,
      group: 0,
      round: 1,
      left: { label: 'A1', team: null },
      right: { label: 'B2', team: null },
      date,
      time,
      field,
    })
    const days = buildPreviewDays([
      match('E1', '2026-11-15', '09:00', '1'),
      match('A2', '2026-11-14', '10:00', '2'),
      match('A1', '2026-11-14', '09:00', '1'),
    ])
    assert.deepEqual(
      days.map((day) => [day.date, day.times]),
      [
        ['2026-11-14', ['09:00', '10:00']],
        ['2026-11-15', ['09:00']],
      ],
    )
    assert.equal(days[0]?.cells.get('10:00|2')?.code, 'A2')
  })
})

describe('競技の初期値', () => {
  test('部門の名前から推定する', () => {
    assert.equal(sportFromContext('Open 4on4', []), 'four')
    assert.equal(sportFromContext('ビーチミックス Beach Mixed', ['seven']), 'beach')
    assert.equal(sportFromContext('7人制 オープン', ['four']), 'seven')
  })

  test('名前で分からなければ同じ大会の他の部門の形、それも無ければ null', () => {
    assert.equal(sportFromContext('オープン', ['four', 'seven']), 'four')
    assert.equal(sportFromContext('ミックス Mixed', []), null)
  })

  test('推定した競技の枠の長さと休みで始める。端末に保存した値は競技として読めるものだけ', () => {
    const cond = initialConditions(null, '2026-11-14', 'four')
    assert.equal(cond.sport, 'four')
    assert.equal(cond.slotMinutes, 20)
    assert.equal(cond.restSlots, 0)
    assert.equal(initialConditions(null, '2026-11-14').sport, 'seven')
    assert.equal(parseSport('beach'), 'beach')
    assert.equal(parseSport('ten'), null)
    assert.equal(parseSport(null), null)
  })
})
