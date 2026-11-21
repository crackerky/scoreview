import 'server-only'
import { apiFetch } from './client'
import {
  type Audience,
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
}
