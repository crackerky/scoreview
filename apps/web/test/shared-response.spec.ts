import assert from 'node:assert/strict'
import { test } from 'node:test'
import { sharedJson } from '../src/app/_lib/sharedResponse.ts'

// 観客が取り直す公開データの応答(src/app/_lib/sharedResponse.ts)

test('配信には短く使い回させ、ブラウザ向けの Cache-Control は付けない', async () => {
  const response = sharedJson({ ok: true })
  assert.equal(
    response.headers.get('Vercel-CDN-Cache-Control'),
    'max-age=5, stale-while-revalidate=10',
  )
  assert.equal(response.headers.get('Cache-Control'), null)
  assert.deepEqual(await response.json(), { ok: true })
})
