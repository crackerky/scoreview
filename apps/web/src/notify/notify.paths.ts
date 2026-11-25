import type { Audience } from '@/api/notify.types'
import { audienceQuery, type ReminderPreset, type Timing, timingQuery } from './notify.utils'

// 管理画面の「公式 LINE で送る」の URL。

export const notifyPaths = {
  /** 大会の送った記録 */
  list: (tournamentId: number) => `/admin/tournaments/${tournamentId}/notifications`,
  /** 送る画面(パスだけ。フォームの戻り先に使う) */
  create: (tournamentId: number) => `/admin/tournaments/${tournamentId}/notifications/new`,
  /**
   * 送る画面を、宛先の条件と送る時刻を選んだ状態で開く。preset は前日と当日の案内のひな形で、
   * 付けたときだけ文の欄にひな形を入れる(数え直しの行き先には付けない。書いた文を戻さないため)。
   */
  createFor: (
    tournamentId: number,
    audience: Audience,
    timing: Timing = { kind: 'now', at: '' },
    preset?: ReminderPreset,
  ) => {
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries({
      ...audienceQuery(audience),
      ...timingQuery(timing),
    })) {
      if (value !== undefined) {
        params.set(key, value)
      }
    }
    if (preset) {
      params.set('preset', preset)
    }
    return `${notifyPaths.create(tournamentId)}?${params.toString()}`
  },
  detail: (id: number) => `/admin/notifications/${id}`,
}
