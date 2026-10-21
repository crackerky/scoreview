'use server'

import { headers } from 'next/headers'
import { redirect } from 'next/navigation'
import { runAction } from '@/admin/action'
import { formToObject } from '@/admin/form.utils'
import { loginErrorCode, loginErrorPath } from '@/admin/login.utils'
import { clientIpFrom, LoginFailureLimiter } from '@/admin/loginLimit.utils'
import { clearSession, readSessionToken, saveSession } from '@/admin/session'
import { safeNext } from '@/admin/session.utils'
import { changePassword, login, logout } from '@/api/admin'
import { loginFormSchema, passwordChangeFormSchema } from '@/api/admin.types'

// IP ごとのログインの失敗。サーバーのプロセス(Vercel では関数のインスタンス)の中で共有する
const loginFailures = new LoginFailureLimiter()

export async function loginAction(formData: FormData): Promise<never> {
  // ログインの前に開いていた画面(記録画面など)へ戻す
  const next = safeNext(formData.get('next'))
  const ip = clientIpFrom(await headers())
  if (!loginFailures.allows(ip, Date.now())) {
    redirect(loginErrorPath('limited', next))
  }
  let target = next
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
    target = loginErrorPath(code, next)
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

/** 本人のパスワードの変更。このセッションは残り、他の端末はログインし直しになる */
export async function changePasswordAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const ip = clientIpFrom(await headers())
      await changePassword(token, passwordChangeFormSchema.parse(form), ip)
      return undefined
    },
    'パスワードを変更しました。他の端末はログインし直しになります',
  )
}
