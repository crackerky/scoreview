import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { groupShown, nextGroupSortOrder } from '../src/admin/group.utils.ts'
import { groupFormSchema } from '../src/api/admin.types.ts'

// 管理画面の組の欄(組を手で足す、組の設定)

describe('組の欄', () => {
  test('手で足す組の並び順は、今ある組の一番後ろにする', () => {
    assert.equal(nextGroupSortOrder([]), 0)
    assert.equal(nextGroupSortOrder([{ sortOrder: 0 }, { sortOrder: 5 }, { sortOrder: 2 }]), 6)
  })

  test('公開していてリンクにする組だけを、スコアサイトに出している組とみなす', () => {
    assert.equal(groupShown({ published: true, linked: true }), true)
    assert.equal(groupShown({ published: true, linked: false }), false)
    assert.equal(groupShown({ published: false, linked: true }), false)
  })

  test('組の設定で出すの欄を外して保存すると、リンクにしたまま公開しない組になる', () => {
    const body = groupFormSchema.parse({
      name: '予選 A 組',
      kind: 'pool',
      sortOrder: '3',
      linked: 'on',
    })
    assert.equal(body.published, false)
    assert.equal(body.linked, true)
    assert.equal(body.sortOrder, 3)
  })
})
