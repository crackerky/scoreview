import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { DEV_SITE_URL, isSameOriginJson, siteOrigins } from '../src/admin/relay.utils.ts'

// 記録係の Route Handler の同一オリジン判定は、Host ヘッダーではなく設定の公開 URL と比べる
// (セキュリティ点検 I-4)。

const post = (headers: Record<string, string>) =>
  new Request('https://score.example.com/admin/api/matches/1/ops', { method: 'POST', headers })
const json = { 'content-type': 'application/json' }

describe('Route Handler のオリジンの判定', () => {
  test('Host ヘッダーを Origin に合わせても、設定の URL でなければ断る', () => {
    const site = siteOrigins({ PUBLIC_SITE_URL: 'https://score.example.com' })
    assert.equal(
      isSameOriginJson(post({ ...json, origin: 'https://score.example.com' }), site),
      true,
    )
    assert.equal(
      isSameOriginJson(
        post({ ...json, host: 'evil.example', origin: 'https://evil.example' }),
        site,
      ),
      false,
    )
    // 同じホストでも scheme かポートが違えば別のオリジン
    assert.equal(
      isSameOriginJson(post({ ...json, origin: 'http://score.example.com' }), site),
      false,
    )
    assert.equal(isSameOriginJson(post({ ...json, origin: 'null' }), site), false)
  })

  test('PUBLIC_SITE_URL が無ければ手元の開発用の URL、Vercel ではデプロイの URL も許す', () => {
    assert.deepEqual(siteOrigins({}), [DEV_SITE_URL])
    assert.deepEqual(siteOrigins({ PUBLIC_SITE_URL: 'https://score.example.com/' }), [
      'https://score.example.com',
    ])
    assert.deepEqual(
      siteOrigins({
        PUBLIC_SITE_URL: 'https://score.example.com',
        VERCEL_URL: 'sv-abc.vercel.app',
      }),
      ['https://score.example.com', 'https://sv-abc.vercel.app'],
    )
    // 壊れた設定はどのオリジンとも一致させない
    assert.deepEqual(siteOrigins({ PUBLIC_SITE_URL: 'not a url' }), [])
  })
})
