import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  nextSort,
  parseSortkey1,
  type ScheduleSortKey,
  scheduleHref,
  scheduleQueryFromSearchParams,
  toSortkey1,
} from '../src/components/schedule.utils.ts'

describe('sortkey1 と api の sort / dir(clone-spec B 2)', () => {
  const cases: Array<[ScheduleSortKey, string]> = [
    ['division', 'mdv.DIVISION_SORTKEY'],
    ['date', 'mat.MATCH_DATE'],
    ['time', 'mat.START_TIME'],
    ['field', 'mat.FIELD_NO'],
    ['code', 'mat.MATCH_CD'],
    ['leftTeam', 'lt_name'],
    ['rightTeam', 'rt_name'],
  ]
  for (const [key, column] of cases) {
    test(`${key} ⇔ ASC_${column} / DSC_${column}`, () => {
      assert.equal(toSortkey1({ key, dir: 'asc' }), `ASC_${column}`)
      assert.equal(toSortkey1({ key, dir: 'desc' }), `DSC_${column}`)
      assert.deepEqual(parseSortkey1(`ASC_${column}`), { key, dir: 'asc' })
      assert.deepEqual(parseSortkey1(`DSC_${column}`), { key, dir: 'desc' })
    })
  }

  test('知らない値は null(DESC_ 接頭辞は参照サイトに無い)', () => {
    assert.equal(parseSortkey1('DESC_mat.FIELD_NO'), null)
    assert.equal(parseSortkey1('ASC_mat.UNKNOWN'), null)
    assert.equal(parseSortkey1(''), null)
  })

  test('並べ替え中の列は反転、他の列は昇順から', () => {
    assert.deepEqual(nextSort('field', null), { key: 'field', dir: 'asc' })
    assert.deepEqual(nextSort('field', { key: 'field', dir: 'asc' }), {
      key: 'field',
      dir: 'desc',
    })
    assert.deepEqual(nextSort('field', { key: 'field', dir: 'desc' }), {
      key: 'field',
      dir: 'asc',
    })
    assert.deepEqual(nextSort('date', { key: 'field', dir: 'asc' }), { key: 'date', dir: 'asc' })
  })
})

describe('URL のクエリから api のクエリへ', () => {
  test('ymd のキーなしは undefined、空はそのまま空(全日程)', () => {
    assert.equal(scheduleQueryFromSearchParams({}).ymd, undefined)
    assert.equal(scheduleQueryFromSearchParams({ ymd: '' }).ymd, '')
    assert.equal(scheduleQueryFromSearchParams({ ymd: ['20261017', 'x'] }).ymd, '20261017')
  })

  test('sortkey1 は sort / dir に変換し、知らない値は送らない', () => {
    const query = scheduleQueryFromSearchParams({
      lang: 'jp',
      ymd: '',
      team: '架空チームA',
      field: '2',
      sortkey1: 'DSC_mat.FIELD_NO',
    })
    assert.deepEqual(query, {
      ymd: '',
      team: '架空チームA',
      field: '2',
      sort: 'field',
      dir: 'desc',
    })
    const unknown = scheduleQueryFromSearchParams({ sortkey1: 'DESC_lt_name' })
    assert.equal(unknown.sort, undefined)
    assert.equal(unknown.dir, undefined)
  })

  test('リンクは lang の後ろに絞り込みの現在値をすべて付ける', () => {
    const selected = { ymd: '', team: '5001', field: '1' }
    assert.equal(
      scheduleHref('scv_search_div', 101, 'en', selected, { key: 'leftTeam', dir: 'asc' }),
      '/pscore/scv_search_div/open/101?lang=en&ymd=&team=5001&field=1&sortkey1=ASC_lt_name',
    )
    assert.equal(
      scheduleHref('scv_search_tn', 1, 'jp', selected),
      '/pscore/scv_search_tn/open/1?lang=jp&ymd=&team=5001&field=1',
    )
  })
})
