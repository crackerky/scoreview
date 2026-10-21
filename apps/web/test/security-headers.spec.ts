import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import nextConfig from '../next.config.ts'
import { adminCsp, browseCsp, IMAGE_HOSTS, isAdminPath } from '../src/security/csp.utils.ts'

// CSP、Permissions-Policy、HSTS(セキュリティ点検 L-4)。

const prod = { dev: false, referenceImages: false }

/** CSP を指令ごとの値の一覧にする */
function directives(policy: string): Map<string, string[]> {
  return new Map(
    policy.split(';').map((part) => {
      const [name = '', ...values] = part.trim().split(/\s+/)
      return [name, values]
    }),
  )
}

describe('Content-Security-Policy', () => {
  test('管理画面は nonce のスクリプトだけを実行し、埋め込み、<base>、他サイトへの送信を禁じる', () => {
    const csp = directives(adminCsp('abc123', prod))
    assert.deepEqual(csp.get('script-src'), ["'self'", "'nonce-abc123'", "'strict-dynamic'"])
    assert.deepEqual(csp.get('frame-ancestors'), ["'none'"])
    assert.deepEqual(csp.get('base-uri'), ["'none'"])
    assert.deepEqual(csp.get('form-action'), ["'self'"])
    assert.deepEqual(csp.get('object-src'), ["'none'"])
    assert.ok(csp.has('upgrade-insecure-requests'))
    // next dev だけ eval を許し、https への引き上げをしない
    const dev = directives(adminCsp('abc123', { dev: true, referenceImages: false }))
    assert.ok(dev.get('script-src')?.includes("'unsafe-eval'"))
    assert.ok(!dev.has('upgrade-insecure-requests'))
  })

  test('閲覧ページの画像は自サイトだけ。参照サイトの画像を出す設定のときだけその置き場も許す', () => {
    const csp = directives(browseCsp(prod))
    assert.deepEqual(csp.get('img-src'), ["'self'", 'data:', 'blob:'])
    assert.deepEqual(directives(browseCsp({ dev: false, referenceImages: true })).get('img-src'), [
      "'self'",
      'data:',
      'blob:',
      ...IMAGE_HOSTS,
    ])
    assert.deepEqual(csp.get('frame-ancestors'), ["'none'"])
    assert.deepEqual(csp.get('base-uri'), ["'none'"])
    assert.deepEqual(csp.get('form-action'), ["'self'"])
    assert.ok(!csp.get('img-src')?.includes('https:'), '任意の https の画像を許さない')
    assert.ok(
      directives(browseCsp({ dev: true, referenceImages: false }))
        .get('img-src')
        ?.includes('https://placehold.co'),
    )
  })

  test('管理画面のパスの判定', () => {
    assert.equal(isAdminPath('/admin'), true)
    assert.equal(isAdminPath('/admin/matches/1/score'), true)
    assert.equal(isAdminPath('/administrator'), false)
    assert.equal(isAdminPath('/pscore/scv_top/open/1'), false)
  })
})

test('全ページに Permissions-Policy と HSTS を付ける', async () => {
  const rules = (await nextConfig.headers?.()) ?? []
  const all = rules.find((rule) => rule.source === '/(.*)')
  const headers = new Map(all?.headers.map((header) => [header.key, header.value]))
  assert.equal(headers.get('Strict-Transport-Security'), 'max-age=63072000')
  assert.match(headers.get('Permissions-Policy') ?? '', /camera=\(\), microphone=\(\)/)
  assert.equal(headers.get('X-Frame-Options'), 'DENY')
})
