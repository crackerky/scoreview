'use server'

import { headers } from 'next/headers'
import { redirect } from 'next/navigation'
import { formToObject } from '@/admin/form.utils'
import { loginErrorCode, loginErrorPath } from '@/admin/login.utils'
import { clientIpFrom, LoginFailureLimiter } from '@/admin/loginLimit.utils'
import { clearSession, readSessionToken, saveSession } from '@/admin/session'
import { login, logout } from '@/api/admin'
import { loginFormSchema } from '@/api/admin.types'

// IP ごとのログインの失敗。サーバーのプロセス(Vercel では関数のインスタンス)の中で共有する
const loginFailures = new LoginFailureLimiter()

export async function loginAction(formData: FormData): Promise<never> {
  const ip = clientIpFrom(await headers())
  if (!loginFailures.allows(ip, Date.now())) {
    redirect(loginErrorPath('limited'))
  }
  let target = '/admin'
  try {
    const input = loginFormSchema.parse(formToObject(formData))
    const session = await login(input.email, input.password, ip)
    await saveSession(session.token, session.expiresAt)
  } catch (error) {
    const code = loginErrorCode(error)
    if (code === null) {
      throw error
    }
    if (code === 'invalid') {
      loginFailures.recordFailure(ip, Date.now())
    }
    target = loginErrorPath(code)
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
