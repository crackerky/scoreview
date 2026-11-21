import { isApiError } from '@/admin/form.utils'

// 送る操作の失敗の見分け(Server Action が使う。送る画面の部品が zod を読み込まないよう notify.utils と分ける)。

/**
 * 送ったかどうかが分からない失敗か。api が送らなかったと答えた失敗(4xx の ApiError)のほかは、すべて
 * 分からないとみなす(api の 500、Vercel の 502 や 504 で本文が JSON でない、api に届かなかった、応答の
 * 形が違った)。記録の途中で止まった送信がありうるので、同じ条件で送り直させず、送った記録を確かめて
 * もらう。
 */
export function uncertainFailure(error: unknown): boolean {
  return !(isApiError(error) && error.status < 500)
}
