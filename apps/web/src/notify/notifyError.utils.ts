import { isApiError } from '@/admin/form.utils'

// 送る操作の失敗の見分け(Server Action が使う。送る画面の部品が zod を読み込まないよう notify.utils と分ける)。

/**
 * 送ったかどうかが分からない失敗か(api の 500、api に届かなかった)。記録の途中で止まった送信が
 * ありうるので、同じ条件で送り直させず、送った記録を確かめてもらう。
 */
export function uncertainFailure(error: unknown): boolean {
  return (isApiError(error) && error.status >= 500) || error instanceof TypeError
}
