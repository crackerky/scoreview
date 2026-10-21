import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { ZodError } from 'zod'
import { loginErrorCode, loginErrorMessage, loginErrorPath } from '../src/admin/login.utils.ts'
import {
  clientIpFrom,
  LOGIN_FAILURE_WINDOW_MS,
  LoginFailureLimiter,
  MAX_LOGIN_FAILURES_PER_IP,
} from '../src/admin/loginLimit.utils.ts'

/** src/api/client.ts の ApiError と同じ形(client.ts は server-only を読むのでテストから import しない) */
function apiError(status: number, code: string): Error {
  return Object.assign(new Error(code), { name: 'ApiError', status, code })
}

describe('ログイン画面のエラー', () => {
  test('?error= はコードだけを受け取り、文面は画面側で決める', () => {
    assert.equal(loginErrorMessage('invalid'), 'メールアドレスかパスワードが違います')
    assert.match(loginErrorMessage('limited') ?? '', /しばらく受け付けません/)
    // 書き換えたリンクの任意の文は出さない
    assert.equal(loginErrorMessage('アカウントが凍結されました。0120-xxx に電話'), undefined)
    assert.equal(loginErrorMessage('toString'), undefined)
    assert.equal(loginErrorMessage(undefined), undefined)
    assert.equal(loginErrorPath('busy'), '/admin/login?error=busy')
  })

  test('api の失敗をコードにする。500 と通信失敗は投げ直す(null)', () => {
    assert.equal(loginErrorCode(apiError(401, 'unauthorized')), 'invalid')
    assert.equal(loginErrorCode(apiError(400, 'bad_request')), 'invalid')
    assert.equal(loginErrorCode(apiError(429, 'too_many_requests')), 'limited')
    assert.equal(loginErrorCode(apiError(503, 'unavailable')), 'busy')
    assert.equal(loginErrorCode(new ZodError([])), 'invalid')
    assert.equal(loginErrorCode(apiError(500, 'internal')), null)
    assert.equal(loginErrorCode(new TypeError('fetch failed')), null)
  })
})

describe('ログインの IP 単位の制限', () => {
  test('Vercel が上書きした x-forwarded-for の先頭を使い、IP でなければ空文字', () => {
    const ip = (value: string | null) =>
      clientIpFrom(new Headers(value === null ? {} : { 'x-forwarded-for': value }))
    assert.equal(ip('203.0.113.7, 10.0.0.1'), '203.0.113.7')
    assert.equal(ip('2001:db8::1'), '2001:db8::1')
    assert.equal(ip('not-an-ip'), '')
    assert.equal(ip(null), '')
  })

  test('窓の中の失敗が上限に達した IP だけを断り、窓が過ぎれば許す', () => {
    const limiter = new LoginFailureLimiter()
    const start = 1_000_000
    for (let i = 0; i < MAX_LOGIN_FAILURES_PER_IP; i++) {
      assert.equal(limiter.allows('203.0.113.7', start), true)
      limiter.recordFailure('203.0.113.7', start)
    }
    assert.equal(limiter.allows('203.0.113.7', start + 1), false)
    assert.equal(limiter.allows('198.51.100.1', start + 1), true)
    assert.equal(limiter.allows('203.0.113.7', start + LOGIN_FAILURE_WINDOW_MS), true)
  })

  test('IP が分からない試行は数えない', () => {
    const limiter = new LoginFailureLimiter(1)
    limiter.recordFailure('', 0)
    limiter.recordFailure('', 0)
    assert.equal(limiter.allows('', 0), true)
  })

  test('覚える IP の数に上限があり、古く使った IP から忘れる', () => {
    const limiter = new LoginFailureLimiter(1, LOGIN_FAILURE_WINDOW_MS, 2)
    limiter.recordFailure('192.0.2.1', 0)
    limiter.recordFailure('192.0.2.2', 0)
    limiter.recordFailure('192.0.2.3', 0)
    assert.equal(limiter.allows('192.0.2.1', 1), true)
    assert.equal(limiter.allows('192.0.2.2', 1), false)
    assert.equal(limiter.allows('192.0.2.3', 1), false)
  })
})
