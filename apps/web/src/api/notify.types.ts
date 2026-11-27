import { z } from 'zod'

// docs/api.md「公式 LINE で送る」の応答と入力の形。

/** 送る文の文字数の上限(api の internal/notify/rules.go の MaxBody と揃える) */
export const maxNotificationBody = 2000

export const audienceKinds = [
  'reps',
  'incomplete',
  'participants',
  'past_reps',
  'no_scorer',
  'friends',
] as const
/** reps だけで選べる申込の状態('' は取り消した申込を除く全部) */
export const audienceStatuses = ['', 'pending', 'returned', 'approved'] as const
export const jobStatuses = ['pending', 'sent', 'skipped', 'failed'] as const
export const batchStates = ['scheduled', 'sent', 'cancelled', 'skipped'] as const
/** 送る時刻に送れなかった予約の理由('' は理由なし) */
export const skipReasons = ['', 'no_recipients', 'quota', 'line_not_configured'] as const

export type AudienceKind = (typeof audienceKinds)[number]
export type AudienceStatus = (typeof audienceStatuses)[number]
export type JobStatus = (typeof jobStatuses)[number]
export type BatchState = (typeof batchStates)[number]
export type SkipReason = (typeof skipReasons)[number]

export const audienceSchema = z.object({
  kind: z.enum(audienceKinds),
  status: z.enum(audienceStatuses),
  divisionId: z.number().int().nullable(),
})
export type Audience = z.infer<typeof audienceSchema>

export const notificationPreviewSchema = z.object({
  heading: z.string(),
  recipients: z.array(
    z.object({ name: z.string(), teams: z.array(z.string()), reachable: z.boolean() }),
  ),
  reachable: z.number().int(),
  unreachable: z.number().int(),
  quota: z.object({ limit: z.number().int().nullable(), used: z.number().int() }).nullable(),
  canSend: z.boolean(),
  /** この宛先の条件で文に書ける差し込み({name})と、sampleName の人の値 */
  variables: z.array(z.object({ name: z.string(), example: z.string() })),
  sampleName: z.string(),
  /** 友だち全員の条件か(recipients は空で、reachable は LINE の数からの見込み。読めなければ 0) */
  everyone: z.boolean(),
})
export type NotificationPreview = z.infer<typeof notificationPreviewSchema>

const countsSchema = z.object({
  pending: z.number().int(),
  sent: z.number().int(),
  skipped: z.number().int(),
  failed: z.number().int(),
})
export type NotificationCounts = z.infer<typeof countsSchema>

/** 届く文の下のボタン。entry は「エントリー」を送る(押すと本人用の申込のリンクが返る)、site はスコアサイト、url は運営が書いた URL */
export const buttonKinds = ['entry', 'site', 'url'] as const
const buttonSchema = z.object({ kind: z.enum(buttonKinds), label: z.string(), url: z.string() })
export type NotificationButton = z.infer<typeof buttonSchema>

/** 送る画面で選ぶボタン(label が空なら api が決まった文字にする。url は url のときだけ) */
export type NotificationButtonInput = {
  kind: (typeof buttonKinds)[number]
  label: string
  url: string
}

export const notificationSummarySchema = z.object({
  id: z.number().int(),
  tournamentId: z.number().int(),
  audience: audienceSchema,
  body: z.string(),
  automatic: z.boolean(),
  sentBy: z.string(),
  createdAt: z.string(),
  sendAt: z.string().nullable(),
  state: z.enum(batchStates),
  note: z.enum(skipReasons),
  counts: countsSchema,
  canResend: z.boolean(),
  buttons: z.array(buttonSchema),
})
export type NotificationSummary = z.infer<typeof notificationSummarySchema>

export const notificationDetailSchema = notificationSummarySchema.extend({
  jobs: z.array(
    z.object({
      name: z.string(),
      teams: z.string(),
      status: z.enum(jobStatuses),
      error: z.string(),
      sentAt: z.string().nullable(),
    }),
  ),
})
export type NotificationDetail = z.infer<typeof notificationDetailSchema>

export const notificationListSchema = z.object({
  notifications: z.array(notificationSummarySchema),
  canSend: z.boolean(),
})

/** sendAt は予約の送る時刻(RFC 3339)。null ならすぐ送る */
export type NotificationInput = {
  audience: Audience
  body: string
  buttons: NotificationButtonInput[]
  /** 友だち全員に送ることを運営が確かめたか(友だち全員のときは必須) */
  confirmAll: boolean
  requestKey: string
  sendAt: string | null
}

export const dispatchResultSchema = z.object({ sent: z.number().int(), skipped: z.number().int() })
