import { z } from 'zod'
import { i18nTextSchema } from './common.types'

// docs/api.md「申込」の応答と入力の形。

/** 上限(api の internal/entry/rules.go の同名の定数と揃える) */
export const entryLimits = {
  maxQuestions: 30,
  maxOptions: 30,
  maxMembers: 60,
  maxShortText: 100,
  maxLongText: 2000,
} as const

export const entryScopes = ['team', 'member'] as const
export const entryKinds = ['short_text', 'long_text', 'choice', 'number', 'yes_no'] as const
export const entryStatuses = ['pending', 'returned', 'approved', 'cancelled'] as const
export const entryActions = ['approve', 'return', 'cancel'] as const

export type EntryScope = (typeof entryScopes)[number]
export type EntryKind = (typeof entryKinds)[number]
export type EntryStatus = (typeof entryStatuses)[number]
export type EntryAction = (typeof entryActions)[number]

export const entryQuestionSchema = z.object({
  id: z.number().int(),
  scope: z.enum(entryScopes),
  label: z.string(),
  kind: z.enum(entryKinds),
  required: z.boolean(),
  options: z.array(z.string()),
  answered: z.number().int(),
})
export type EntryQuestion = z.infer<typeof entryQuestionSchema>

export const entryQuestionListSchema = z.object({ questions: z.array(entryQuestionSchema) })

export const entrySummarySchema = z.object({
  id: z.number().int(),
  division: z.object({ id: z.number().int(), name: i18nTextSchema }),
  fullName: z.string(),
  shortName: z.string(),
  repName: z.string(),
  channel: z.enum(['line', 'desk']),
  status: z.enum(entryStatuses),
  members: z.number().int(),
  registered: z.number().int(),
  submittedAt: z.string(),
  teamId: z.number().int().nullable(),
})
export type EntrySummary = z.infer<typeof entrySummarySchema>

export const entryListSchema = z.object({ entries: z.array(entrySummarySchema) })

export const entryDetailSchema = entrySummarySchema.extend({
  tournamentId: z.number().int(),
  repPhone: z.string(),
  repEmail: z.string(),
  returnNote: z.string(),
  inviteCode: z.string(),
  decidedAt: z.string().nullable(),
  memberList: z.array(
    z.object({
      id: z.number().int(),
      name: z.string(),
      number: z.string(),
      registered: z.boolean(),
      playerId: z.number().int().nullable(),
    }),
  ),
  answers: z.array(z.object({ questionId: z.number().int(), value: z.string() })),
})
export type EntryDetail = z.infer<typeof entryDetailSchema>

export const entryExportSchema = z.object({
  questions: z.array(entryQuestionSchema),
  entries: z.array(entryDetailSchema),
})
export type EntryExport = z.infer<typeof entryExportSchema>

/** 保存する質問。id が null なら新しく作る */
export type EntryQuestionInput = Omit<EntryQuestion, 'id' | 'answered'> & { id: number | null }

/** 運営が代わりに入れる申込 */
export type EntryDraftInput = {
  divisionId: number
  fullName: string
  shortName: string
  repName: string
  repPhone: string
  repEmail: string
  members: Array<{ name: string; number: string }>
  answers: Array<{ questionId: number; value: string }>
}
