import { ZodError } from 'zod'
import { isApiError } from './form.utils'

// ログイン画面のエラーの決め事。Next.js に依存しない純粋な関数にしてテストする。
//
// エラーはクエリ ?error= にコードで渡し、文面は画面側で決める。文面をクエリで渡すと、他人が
// 作ったリンク(/admin/login?error=...)で任意の文を正規の画面に出させられる(セキュリティ点検 I-5)。

/** ログイン画面に出すエラーのコードと文面 */
const loginErrorMessages = {
  // メールアドレスの有無を区別しない(api と同じ)
  invalid: 'メールアドレスかパスワードが違います',
  limited: 'ログインの失敗が続いたため、しばらく受け付けません。少し待ってから試してください',
  busy: 'ただいま混み合っています。少し待ってから試してください',
} as const

export type LoginErrorCode = keyof typeof loginErrorMessages

function isLoginErrorCode(value: string): value is LoginErrorCode {
  return Object.hasOwn(loginErrorMessages, value)
}

/** ?error= の値から画面に出す文面を決める。知らないコード(書き換えたリンク)なら何も出さない */
export function loginErrorMessage(code: string | undefined): string | undefined {
  return code !== undefined && isLoginErrorCode(code) ? loginErrorMessages[code] : undefined
}

/** エラーのコードつきのログイン画面のパス */
export function loginErrorPath(code: LoginErrorCode): string {
  return `/admin/login?${new URLSearchParams({ error: code }).toString()}`
}

/**
 * ログインの失敗をログイン画面のエラーのコードにする。入力の形の誤りは「違います」と同じに扱う。
 * それ以外(api の 500、通信失敗)は null(呼び出し側が投げ直してエラー境界へ流す)。
 */
export function loginErrorCode(error: unknown): LoginErrorCode | null {
  if (error instanceof ZodError) {
    return 'invalid'
  }
  if (isApiError(error)) {
    switch (error.status) {
      case 400:
      case 401:
        return 'invalid'
      case 429:
        return 'limited'
      case 503:
        return 'busy'
    }
  }
  return null
}
