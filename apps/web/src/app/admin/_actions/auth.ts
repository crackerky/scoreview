'use server'

import { redirect } from 'next/navigation'
import { failureTarget, formToObject, withNotice } from '@/admin/form.utils'
import { clearSession, readSessionToken, saveSession } from '@/admin/session'
import { login, logout } from '@/api/admin'
import { loginFormSchema } from '@/api/admin.types'

export async function loginAction(formData: FormData): Promise<never> {
  let target = '/admin'
  try {
    const input = loginFormSchema.parse(formToObject(formData))
    const session = await login(input.email, input.password)
    await saveSession(session.token, session.expiresAt)
  } catch (error) {
    // 401 は failureTarget ではログイン画面へ戻すだけなので、ここで文面を付ける。
    // 文面はメールアドレスの有無を区別しない(api と同じ)
    target = failureTarget('/admin/login', error)
    if (target === '/admin/login') {
      target = withNotice('/admin/login', 'error', 'メールアドレスかパスワードが違います')
    }
  }
  redirect(target)
}

export async function logoutAction(): Promise<never> {
  const token = await readSessionToken()
  if (token !== null) {
    // api 側のセッションを消せなくても(期限切れなど)、Cookie は必ず消してログイン画面へ戻す
    await logout(token).catch((error: unknown) => {
      console.warn('logout failed', error)
    })
  }
  await clearSession()
  redirect('/admin/login')
}
