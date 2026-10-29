import assert from 'node:assert/strict'
import { test } from 'node:test'
import { finalTeamNames, parseReplacements } from '../src/admin/newTournament.utils.ts'

// 大会を作る 3 ステップの入れ替え(src/admin/newTournament.utils.ts)

test('入れ替えの JSON を読む(空なら入れ替えなし、名前の前後の空白は落とす)', () => {
  assert.deepEqual(parseReplacements(''), [])
  assert.deepEqual(parseReplacements('[{"division":0,"team":2,"name":" 新チーム "}]'), [
    { division: 0, team: 2, name: '新チーム' },
  ])
})

test('名前が空、位置が負の入れ替えは入力の誤り', () => {
  assert.throws(() => parseReplacements('[{"division":0,"team":1,"name":"  "}]'))
  assert.throws(() => parseReplacements('[{"division":-1,"team":1,"name":"a"}]'))
})

test('確認の画面のチームは、入れ替えたチームだけ新しい名前で並ぶ', () => {
  const divisions = [
    { name: 'オープン', teams: ['湘南', '港北'] },
    { name: 'ウィメン', teams: ['浜風'] },
  ]
  assert.deepEqual(finalTeamNames(divisions, [{ division: 1, team: 0, name: '三浦' }]), [
    '湘南',
    '港北',
    '三浦',
  ])
})
