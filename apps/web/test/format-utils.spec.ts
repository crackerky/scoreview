import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  moveTo,
  parseRoster,
  proposalSentence,
  roundUpToFive,
  swapAt,
} from '../src/admin/format.utils.ts'
import type { Proposal } from '../src/api/format.types.ts'

const proposal: Proposal = {
  id: 'pb4',
  title: '4 プール(各 3 チーム) → 4 チームのトーナメント × 3(1〜4 位、5〜8 位、9〜12 位)',
  template: 'pool_bands',
  params: { teams: 12 },
  recommended: true,
  note: '',
  matches: 24,
  end: '12:40',
  ranked: 12,
  unique: 12,
  violations: 0,
  endDay: 1,
  overflow: 0,
  fitNote: null,
}

describe('大会の形のウィザードの整形', () => {
  test('名簿の貼り付け: タブ、カンマ、読点で正式名と短縮名を分け、空行を飛ばす', () => {
    assert.deepEqual(
      parseRoster('架空クラブ東京\tトウキョウ\n\n 架空クラブ大阪 , オオサカ\n架空A'),
      [
        { fullName: '架空クラブ東京', shortName: 'トウキョウ' },
        { fullName: '架空クラブ大阪', shortName: 'オオサカ' },
        { fullName: '架空A', shortName: '架空A' },
      ],
    )
  })

  test('候補の要約', () => {
    assert.equal(
      proposalSentence(proposal, 12),
      'この形で 24 試合、12:40 終了、全 12 位まで決まります',
    )
    assert.equal(
      proposalSentence({ ...proposal, unique: 8 }, 12),
      'この形で 24 試合、12:40 終了、全チームの順位が決まります(4 チームは同順位)',
    )
    assert.equal(
      proposalSentence({ ...proposal, ranked: 8, unique: 8 }, 16),
      'この形で 24 試合、12:40 終了、16 チーム中 8 チームの順位が決まります',
    )
  })

  test('シードの入れ替えと移動は元の配列を変えない', () => {
    const seeds = [1, 2, 3, 4]
    assert.deepEqual(swapAt(seeds, 0, 3), [4, 2, 3, 1])
    assert.deepEqual(moveTo(seeds, 3, 0), [4, 1, 2, 3])
    assert.deepEqual(seeds, [1, 2, 3, 4])
    assert.deepEqual(swapAt(seeds, 0, 9), [1, 2, 3, 4])
  })

  test('遅れは 5 分単位に切り上げ、最小 5 分', () => {
    assert.equal(roundUpToFive(1), 5)
    assert.equal(roundUpToFive(12), 15)
    assert.equal(roundUpToFive(15), 15)
  })
})
