import { ZodError } from 'zod'
import { operatorErrorText } from './errorText.utils'

// Server Actions の共通処理(フォームの読み取り、戻り先の決定、エラーの文面)。Next.js に依存しない。

/**
 * FormData を平らなオブジェクトにする。同じ名前が複数あれば配列にする(チェックボックスの一覧)。
 * ファイルは扱わない(管理画面は URL を登録するだけ)。Next.js が足す $ACTION_ で始まる欄は捨てる。
 */
export function formToObject(formData: FormData): Record<string, string | string[]> {
  const out: Record<string, string | string[]> = {}
  for (const [key, value] of formData.entries()) {
    if (typeof value !== 'string' || key.startsWith('$ACTION')) {
      continue
    }
    const current = out[key]
    if (current === undefined) {
      out[key] = value
    } else if (Array.isArray(current)) {
      current.push(value)
    } else {
      out[key] = [current, value]
    }
  }
  return out
}

/**
 * 戻り先のパスを /admin 以下に限る(フォームの hidden 欄を書き換えて外部へ飛ばされないように)。
 * 不正なら /admin。
 */
export function safeAdminPath(path: unknown): string {
  if (typeof path !== 'string' || !/^\/admin(\/[\w\-/]*)?$/.test(path)) {
    return '/admin'
  }
  return path
}

/** パスに通知のクエリ(?ok= か ?error=)を付ける */
export function withNotice(path: string, kind: 'ok' | 'error', message: string): string {
  return `${path}?${new URLSearchParams({ [kind]: message }).toString()}`
}

/** zod の検証エラーを、欄の名前つきの 1 行にする */
export function describeZodError(error: ZodError): string {
  return error.issues
    .map((issue) =>
      issue.path.length > 0 ? `${issue.path.join('.')}: ${issue.message}` : issue.message,
    )
    .join(' / ')
}

/**
 * Server Action の失敗の行き先。セッション切れはログイン画面、入力と api の検証エラーは元の画面に
 * 文面つきで戻す。それ以外(api の 500、通信失敗)は握りつぶさず投げ直す(エラー境界へ)。
 */
export function failureTarget(back: string, error: unknown): string {
  if (isApiError(error)) {
    if (error.status === 401) {
      return '/admin/login'
    }
    // 503(混雑)は待てば通るので、エラー画面ではなく元の画面に文面つきで戻す
    if (error.status < 500 || error.code === 'unavailable') {
      return withNotice(back, 'error', operatorErrorText(error.code, error.message))
    }
  }
  if (error instanceof ZodError) {
    return withNotice(back, 'error', describeZodError(error))
  }
  throw error
}

/**
 * api の ApiError(src/api/client.ts)かどうか。client.ts は server-only を読み込むので、このファイルを
 * テストから使えるよう import せず、ApiError が持つ name と status、code で判定する。
 */
export function isApiError(error: unknown): error is Error & { status: number; code: string } {
  return (
    error instanceof Error &&
    error.name === 'ApiError' &&
    'status' in error &&
    typeof error.status === 'number' &&
    'code' in error &&
    typeof error.code === 'string'
  )
}
