import 'server-only'
import { revalidatePath } from 'next/cache'
import { redirect } from 'next/navigation'
import { failureTarget, formToObject, safeAdminPath, withNotice } from './form.utils'
import { readSessionToken } from './session'

export type ActionContext = {
  token: string
  form: Record<string, string | string[]>
  /** フォームの hidden 欄 back(/admin 以下に限る)。成功と失敗の通知はここへ戻す */
  back: string
}

/** フォームの欄を文字列で読む(無ければ空文字) */
export function field(form: ActionContext['form'], name: string): string {
  const value = form[name]
  return typeof value === 'string' ? value : ''
}

/** フォームの欄を正の整数の ID として読む */
export function idField(form: ActionContext['form'], name: string): number {
  const value = Number(field(form, name))
  if (!Number.isInteger(value) || value <= 0) {
    throw new RangeError(`form field ${name} is not an id`)
  }
  return value
}

/**
 * Server Action の共通の流れ。セッションのトークンを読み、run を実行して、成功なら run が返したパス
 * (無ければ back)へ「保存しました」つきで、入力や api の検証エラーなら back へ文面つきで戻す。
 * redirect は例外で抜けるので try の外で呼ぶ。
 */
export async function runAction(
  formData: FormData,
  run: (ctx: ActionContext) => Promise<string | undefined>,
  okMessage = '保存しました',
): Promise<never> {
  const form = formToObject(formData)
  const back = safeAdminPath(form.back)
  const token = await readSessionToken()
  if (token === null) {
    redirect('/admin/login')
  }
  let target: string
  try {
    const next = await run({ token, form, back })
    target = withNotice(next ?? back, 'ok', okMessage)
  } catch (error) {
    target = failureTarget(back, error)
  }
  // 閲覧ページは api を no-store で読むが、管理画面の同じパスの描画結果を捨てて読み直させる
  revalidatePath('/admin', 'layout')
  redirect(target)
}
