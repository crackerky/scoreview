import 'server-only'
import { apiFetch } from './client'
import {
  type Audience,
  dispatchResultSchema,
  type NotificationInput,
  notificationDetailSchema,
  notificationListSchema,
  notificationPreviewSchema,
} from './notify.types'

// docs/api.md「公式 LINE で送る」を呼ぶ関数。管理者だけ。応答は zod で parse して返す。

const admin = '/api/v1/admin'

export const notifications = {
  list: (token: string, tournamentId: number) =>
    apiFetch(`${admin}/tournaments/${tournamentId}/notifications`, notificationListSchema, {
      sessionToken: token,
    }),
  preview: (token: string, tournamentId: number, audience: Audience) =>
    apiFetch(
      `${admin}/tournaments/${tournamentId}/notifications/preview`,
      notificationPreviewSchema,
      {
        sessionToken: token,
        query: {
          kind: audience.kind,
          status: audience.status || undefined,
          divisionId: audience.divisionId === null ? undefined : String(audience.divisionId),
        },
      },
    ),
  send: (token: string, tournamentId: number, input: NotificationInput) =>
    apiFetch(`${admin}/tournaments/${tournamentId}/notifications`, notificationDetailSchema, {
      method: 'POST',
      sessionToken: token,
      body: input,
    }),
  get: (token: string, id: number) =>
    apiFetch(`${admin}/notifications/${id}`, notificationDetailSchema, { sessionToken: token }),
  /** 予約を取りやめる */
  cancel: (token: string, id: number) =>
    apiFetch(`${admin}/notifications/${id}/cancel`, notificationDetailSchema, {
      method: 'POST',
      sessionToken: token,
    }),
  /** 送る時刻の来た予約を送る(定期の呼び出し /cron/notify から。利用者のログインは無い) */
  dispatch: () =>
    apiFetch('/api/v1/internal/notifications/dispatch', dispatchResultSchema, { method: 'POST' }),
  /** 送れなかった宛先に、同じまとまりと送り直しのキーで送り直す */
  resend: (token: string, id: number) =>
    apiFetch(`${admin}/notifications/${id}/resend`, notificationDetailSchema, {
      method: 'POST',
      sessionToken: token,
    }),
}
