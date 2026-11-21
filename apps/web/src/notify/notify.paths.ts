import type { Audience } from '@/api/notify.types'
import { audienceQuery } from './notify.utils'

// 管理画面の「公式 LINE で送る」の URL。

export const notifyPaths = {
  /** 大会の送った記録 */
  list: (tournamentId: number) => `/admin/tournaments/${tournamentId}/notifications`,
  /** 送る画面(パスだけ。フォームの戻り先に使う) */
  create: (tournamentId: number) => `/admin/tournaments/${tournamentId}/notifications/new`,
  /** 送る画面を、宛先の条件を選んだ状態で開く */
  createFor: (tournamentId: number, audience: Audience) => {
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries(audienceQuery(audience))) {
      if (value !== undefined) {
        params.set(key, value)
      }
    }
    return `${notifyPaths.create(tournamentId)}?${params.toString()}`
  },
  detail: (id: number) => `/admin/notifications/${id}`,
}
