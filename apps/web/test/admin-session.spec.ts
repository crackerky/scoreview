import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { ZodError, z } from 'zod'
import { canScore, decideAccess } from '../src/admin/access.utils.ts'
import {
  describeZodError,
  failureTarget,
  formToObject,
  safeAdminPath,
  withNotice,
} from '../src/admin/form.utils.ts'
import { formatElapsed, sideChoiceValue, sideName } from '../src/admin/labels.utils.ts'
import {
  SESSION_COOKIE,
  SESSION_COOKIE_PATH,
  sessionCookieOptions,
} from '../src/admin/session.utils.ts'
import type { AdminUser } from '../src/api/admin.types.ts'

const admin: AdminUser = {
  id: 1,
  email: 'a@example.com',
  displayName: '',
  role: 'admin',
  tournamentIds: [],
}
const staff: AdminUser = {
  id: 2,
  email: 's@example.com',
  displayName: '',
  role: 'staff',
  tournamentIds: [7],
}

describe('セッション Cookie', () => {
  test('httpOnly、SameSite=Lax、/admin だけ、api の期限で切れる', () => {
    const options = sessionCookieOptions('2026-10-19T09:00:00Z', true)
    assert.deepEqual(options, {
      httpOnly: true,
      secure: true,
      sameSite: 'lax',
      path: '/admin',
      expires: new Date('2026-10-19T09:00:00Z'),
    })
    assert.equal(SESSION_COOKIE_PATH, '/admin')
    assert.match(SESSION_COOKIE, /^[a-z_]+$/)
  })

  test('Secure は本番だけ(開発の http://localhost では付けない)', () => {
    assert.equal(sessionCookieOptions('2026-10-19T09:00:00Z', false).secure, false)
  })

  test('期限が読めなければ例外', () => {
    assert.throws(() => sessionCookieOptions('tomorrow', true), RangeError)
  })
})

describe('権限によるリダイレクトの判定', () => {
  test('未ログインはログイン画面', () => {
    assert.deepEqual(decideAccess(null, 'any'), { kind: 'login' })
    assert.deepEqual(decideAccess(null, 'admin'), { kind: 'login' })
  })

  test('記録係は管理者用のページに入れない', () => {
    assert.deepEqual(decideAccess(staff, 'admin'), { kind: 'forbidden' })
    assert.deepEqual(decideAccess(staff, 'any'), { kind: 'ok', user: staff })
    assert.deepEqual(decideAccess(admin, 'admin'), { kind: 'ok', user: admin })
  })

  test('スコア入力は担当大会だけ(管理者は全大会)', () => {
    assert.equal(canScore(staff, 7), true)
    assert.equal(canScore(staff, 8), false)
    assert.equal(canScore(admin, 8), true)
  })
})

describe('Server Actions の共通処理', () => {
  test('FormData を平らにし、同じ名前は配列にする', () => {
    const data = new FormData()
    data.append('name', 'x')
    data.append('tournamentIds', '1')
    data.append('tournamentIds', '2')
    data.append('$ACTION_ID_abc', '')
    assert.deepEqual(formToObject(data), { name: 'x', tournamentIds: ['1', '2'] })
  })

  test('戻り先は /admin 以下に限る', () => {
    assert.equal(safeAdminPath('/admin/matches/5/score'), '/admin/matches/5/score')
    assert.equal(safeAdminPath('/admin'), '/admin')
    for (const bad of [
      'https://evil.example/admin',
      '//evil.example',
      '/sctnmt',
      '/admin?x=1',
      '/administrator',
      undefined,
    ]) {
      assert.equal(safeAdminPath(bad), '/admin', String(bad))
    }
  })

  test('通知のクエリは URL エンコードする', () => {
    assert.equal(
      withNotice('/admin', 'ok', '保存しました'),
      '/admin?ok=%E4%BF%9D%E5%AD%98%E3%81%97%E3%81%BE%E3%81%97%E3%81%9F',
    )
  })

  test('失敗の行き先: 401 はログイン、4xx と入力エラーは元の画面、それ以外は投げ直す', () => {
    const apiError = (status: number, code: string) =>
      Object.assign(new Error('message'), { name: 'ApiError', status, code })
    assert.equal(failureTarget('/admin/x', apiError(401, 'unauthorized')), '/admin/login')
    assert.equal(
      failureTarget('/admin/x', apiError(409, 'conflict')),
      withNotice('/admin/x', 'error', 'conflict: message'),
    )
    const zodError = z
      .object({ name: z.string().min(1, '入力してください') })
      .safeParse({ name: '' }).error
    assert.ok(zodError instanceof ZodError)
    assert.equal(describeZodError(zodError), 'name: 入力してください')
    assert.equal(
      failureTarget('/admin/x', zodError),
      withNotice('/admin/x', 'error', 'name: 入力してください'),
    )
    assert.throws(() => failureTarget('/admin/x', apiError(500, 'internal')))
    assert.throws(() => failureTarget('/admin/x', new TypeError('fetch failed')))
  })
})

describe('表示の整形', () => {
  test('経過は 分:秒', () => {
    assert.equal(formatElapsed(725), '12:05')
    assert.equal(formatElapsed(0), '0:00')
  })

  test('未確定チームは [A1-勝ち]、フォームの値は参照を優先する', () => {
    const side = {
      team: { id: 5, name: { ja: '左', en: '' } },
      placeholder: { code: 'A1', outcome: 'win' as const },
      score: null,
    }
    assert.equal(sideName(side), '左')
    assert.equal(sideName({ ...side, team: null }), '[A1-勝ち]')
    assert.equal(sideChoiceValue(side), 'A1-win')
    assert.equal(sideChoiceValue({ ...side, placeholder: null }), '5')
    assert.equal(sideName({ team: null, placeholder: null, score: null }), '未定')
  })
})
