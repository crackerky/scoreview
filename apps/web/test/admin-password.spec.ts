import assert from 'node:assert/strict'
import { test } from 'node:test'
import { passwordChangeFormSchema } from '../src/api/admin.types.ts'

// 本人のパスワードの変更のフォーム(セキュリティ点検 L-8)。

test('新しいパスワードは 12 文字以上で、確認の欄と一致すること', () => {
  const form = (newPassword: string, newPasswordConfirm: string) =>
    passwordChangeFormSchema.safeParse({
      currentPassword: 'old password 0123',
      newPassword,
      newPasswordConfirm,
    })
  assert.deepEqual(form('a brand new password', 'a brand new password').data, {
    currentPassword: 'old password 0123',
    newPassword: 'a brand new password',
  })
  assert.equal(form('a brand new password', 'a brand new passw0rd').success, false)
  assert.equal(form('short', 'short').success, false)
  assert.equal(
    passwordChangeFormSchema.safeParse({
      currentPassword: '',
      newPassword: 'a brand new password',
      newPasswordConfirm: 'a brand new password',
    }).success,
    false,
  )
})
