import assert from 'node:assert/strict'
import { test } from 'node:test'
import { isActionResponse, submitDecision } from '../src/admin/submitFeedback.utils.ts'

// 管理画面の送信中の表示と二重送信の防止(src/components/admin/SubmitFeedback.tsx)の判断

test('Server Action のフォームは処理中にし、送信中の同じボタンの押し直しは止める', () => {
  assert.equal(
    submitDecision({ submitterBusy: false, methodAttribute: 'POST', skip: false }),
    'mark',
  )
  // 画面の切り替え後に React が作ったフォームは method 属性を持たない
  assert.equal(submitDecision({ submitterBusy: false, methodAttribute: null, skip: false }), 'mark')
  assert.equal(
    submitDecision({ submitterBusy: true, methodAttribute: 'POST', skip: false }),
    'block',
  )
})

test('GET の検索フォームと対象外のフォームは何もしない', () => {
  assert.equal(
    submitDecision({ submitterBusy: false, methodAttribute: 'get', skip: false }),
    'ignore',
  )
  assert.equal(
    submitDecision({ submitterBusy: true, methodAttribute: 'GET', skip: false }),
    'ignore',
  )
  assert.equal(submitDecision({ submitterBusy: true, methodAttribute: null, skip: true }), 'ignore')
})

const origin = 'https://scoreview.example'
const submission = { pageUrl: `${origin}/admin/tournaments/7?ok=x#top`, submittedAt: 100 }

test('送信した画面の URL への、送信より後の fetch を Server Action の応答とみなす', () => {
  const entry = {
    name: `${origin}/admin/tournaments/7?ok=x`,
    initiatorType: 'fetch',
    startTime: 120,
  }
  assert.equal(isActionResponse(entry, submission, origin), true)
})

test('画面の取り直し、他の URL、送信前の fetch、画像などは応答とみなさない', () => {
  const base = {
    name: `${origin}/admin/tournaments/7?ok=x`,
    initiatorType: 'fetch',
    startTime: 120,
  }
  const cases = [
    { ...base, name: `${origin}/admin/tournaments/7?ok=x&_rsc=abc` },
    { ...base, name: `${origin}/admin/divisions/10` },
    { ...base, name: 'https://other.example/admin/tournaments/7?ok=x' },
    { ...base, startTime: 50 },
    { ...base, initiatorType: 'img' },
  ]
  for (const entry of cases) {
    assert.equal(isActionResponse(entry, submission, origin), false, JSON.stringify(entry))
  }
})
