import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  divisionSteps,
  draftState,
  parsePoolTable,
  poolLetter,
  seedsFromPools,
  snakePools,
} from '../src/admin/build.utils.ts'

const teams = [
  { id: 1, fullName: '熱海ディスクス', shortName: '熱海' },
  { id: 2, fullName: '伊東フライヤーズ', shortName: '伊東' },
  { id: 3, fullName: '沼津ウェーブ', shortName: '沼津' },
  { id: 4, fullName: '三島スカイ', shortName: '三島' },
  { id: 5, fullName: '下田サンズ', shortName: '下田' },
]

describe('組分け', () => {
  test('組からシード順と組ごとのシード番号を作る(各組の 1 番目から順に)', () => {
    assert.deepEqual(
      seedsFromPools([
        [11, 14, 15],
        [12, 13],
      ]),
      {
        seeds: [11, 12, 14, 13, 15],
        poolSeeds: [
          [1, 3, 5],
          [2, 4],
        ],
      },
    )
    assert.equal(poolLetter(2), 'C')
  })
  test('おまかせは蛇行で配る', () => {
    assert.deepEqual(snakePools([1, 2, 3, 4, 5, 6, 7], 3), [
      [1, 6, 7],
      [2, 5],
      [3, 4],
    ])
  })
  test('進み具合(全チームが 1 回ずつ入り、どの組も 2 チーム以上で決められる)', () => {
    assert.deepEqual(draftState([[1, 2], [3]], [1, 2, 3]), { placed: 3, total: 3, complete: false })
    assert.deepEqual(
      draftState(
        [
          [1, 2],
          [3, 4],
        ],
        [1, 2, 3, 4],
      ),
      {
        placed: 4,
        total: 4,
        complete: true,
      },
    )
    assert.equal(draftState([[1, 2], []], [1, 2, 3]).complete, false)
  })
})

describe('組分けの貼り付け', () => {
  test('1 行目が組の名前で、その下にチーム名(Excel のタブ区切り)', () => {
    const got = parsePoolTable('A組\tB組\n熱海\t伊東\nぬまづウェーブ\t三島スカイ\n下田サン', teams)
    assert.deepEqual(got.names, ['A', 'B'])
    assert.deepEqual(got.pools, [[1], [2, 4]])
    assert.deepEqual(
      got.unknown.map((item) => item.name),
      ['ぬまづウェーブ', '下田サン'],
    )
    assert.equal(got.unknown[1]?.near, '下田サンズ')
    assert.deepEqual(got.missing, [3, 5])
  })
  test('「組, チーム名」の 2 列(CSV。見出しの行と全角は気にしない)', () => {
    const got = parsePoolTable(
      '組,チーム\nプールA,熱海ディスクス\nプールA,ｓｈｉｍｏｄａ\nB,伊東\nB,沼津ウェーブ\nA,三島 スカイ\nB,熱海',
      [...teams.slice(0, 4), { id: 5, fullName: '下田', shortName: 'SHIMODA' }],
    )
    assert.deepEqual(got.names, ['A', 'B'])
    assert.deepEqual(got.pools, [
      [1, 5, 4],
      [2, 3],
    ])
    assert.deepEqual(got.duplicates, ['熱海'])
    assert.deepEqual(got.unknown, [])
  })
})

describe('部門の段', () => {
  const base = {
    id: 1,
    name: 'オープン',
    teams: 16,
    pendingEntries: 0,
    formatted: false,
    pools: 0,
    draft: null,
    matches: 0,
    scheduled: 0,
  }
  test('組分けがまだなら、次は組分け', () => {
    const steps = divisionSteps({ ...base, pendingEntries: 2 })
    assert.equal(steps.teams.tone, 'warn')
    assert.equal(steps.pools.label, 'まだ')
    assert.deepEqual(steps.next, { label: '組分けをする', step: 'pools' })
  })
  test('組を決めたら次は時間割、形を作ったら済み', () => {
    const decided = divisionSteps({ ...base, draft: { placed: 16, complete: true, pools: 4 } })
    assert.equal(decided.pools.tone, 'ok')
    assert.deepEqual(decided.next, { label: '時間割を作る', step: 'times' })
    assert.equal(decided.times.sub, '作れます')
    const done = divisionSteps({ ...base, formatted: true, pools: 4, matches: 40, scheduled: 40 })
    assert.equal(done.times.label, '済み')
  })
})

test('組の数の無い形(ブロック)は、組の数の代わりに形があることを書く', () => {
  const steps = divisionSteps({
    id: 1,
    name: 'ミックス',
    teams: 95,
    pendingEntries: 0,
    formatted: true,
    pools: 0,
    draft: null,
    matches: 284,
    scheduled: 284,
  })
  assert.equal(steps.pools.sub, '組と順位決定戦あり')
})
