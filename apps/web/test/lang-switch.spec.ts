import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { langSwitchHref } from '../src/components/langSwitch.utils.ts'

describe('言語切替のリンク先', () => {
  test('lang だけを替え、クエリの他の値と並びを保つ', () => {
    assert.equal(
      langSwitchHref('/pscore/scv_search_div/open/949', 'lang=jp&ymd=20260316&team=17322', 'en'),
      '/pscore/scv_search_div/open/949?lang=en&ymd=20260316&team=17322',
    )
  })

  test('lang が無ければ末尾に足す', () => {
    assert.equal(langSwitchHref('/sctnmt', '', 'en'), '/sctnmt?lang=en')
    assert.equal(langSwitchHref('/scview/525', 'refpos=1', 'jp'), '/scview/525?refpos=1&lang=jp')
  })
})
