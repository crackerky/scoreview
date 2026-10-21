import 'server-only'
import { cookies } from 'next/headers'
import { notFound, redirect } from 'next/navigation'
import { getMe } from '@/api/admin'
import type { AdminUser } from '@/api/admin.types'
import { ApiError } from '@/api/client'
import { decideAccess, type Requirement } from './access.utils'
import { withNotice } from './form.utils'
import { SESSION_COOKIE, SESSION_COOKIE_PATH, sessionCookieOptions } from './session.utils'

// 管理画面のページと Server Actions が使う、セッション Cookie の読み書きと権限の確認。

export async function readSessionToken(): Promise<string | null> {
  return (await cookies()).get(SESSION_COOKIE)?.value ?? null
}

export async function saveSession(token: string, expiresAt: string): Promise<void> {
  const production = process.env.NODE_ENV === 'production'
  ;(await cookies()).set(SESSION_COOKIE, token, sessionCookieOptions(expiresAt, production))
}

export async function clearSession(): Promise<void> {
  ;(await cookies()).delete({ name: SESSION_COOKIE, path: SESSION_COOKIE_PATH })
}

/** auth/me のユーザー。トークンが無いか無効(401)なら null */
async function currentUser(token: string | null): Promise<AdminUser | null> {
  if (token === null) {
    return null
  }
  try {
    return (await getMe(token)).user
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      return null
    }
    throw error
  }
}

/**
 * ページを出す前に auth/me で権限を確かめる。未ログインならログイン画面へ、権限が足りなければ
 * 管理画面のトップへリダイレクトする。web 側の確認は表示のためで、最終的な認可は api が行う。
 */
export async function requireUser(
  requirement: Requirement,
): Promise<{ user: AdminUser; token: string }> {
  const token = await readSessionToken()
  const decision = decideAccess(await currentUser(token), requirement)
  if (decision.kind === 'login' || token === null) {
    redirect('/admin/login')
  }
  if (decision.kind === 'forbidden') {
    redirect(withNotice('/admin', 'error', '権限がありません'))
  }
  return { user: decision.user, token }
}

/**
 * ページの読み込みで api を呼ぶ。存在しない ID は 404、権限が無ければ管理画面のトップ、
 * セッション切れはログイン画面へ。それ以外は投げ直す(エラー境界へ)。
 */
export async function loadAdmin<T>(load: () => Promise<T>): Promise<T> {
  try {
    return await load()
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.status === 401) {
        redirect('/admin/login')
      }
      if (error.status === 403) {
        redirect(withNotice('/admin', 'error', '権限がありません'))
      }
      if (error.status === 404) {
        notFound()
      }
    }
    throw error
  }
}
