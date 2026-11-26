import { z } from 'zod'
import { i18nTextSchema } from './common.types'

// docs/api.md「申込」の応答と入力の形。

/** 上限(api の internal/entry/rules.go の同名の定数と揃える) */
export const entryLimits = {
  maxQuestions: 30,
  maxOptions: 30,
  maxMembers: 60,
  maxShortText: 100,
  maxNumberText: 10,
  maxLongText: 2000,
} as const

export const entryScopes = ['team', 'member'] as const
export const entryKinds = ['short_text', 'long_text', 'choice', 'number', 'yes_no'] as const
export const entryStatuses = ['pending', 'returned', 'approved', 'cancelled'] as const
export const entryActions = ['approve', 'return', 'cancel', 'reopen'] as const

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

const entryMemberSchema = z.object({
  id: z.number().int(),
  name: z.string(),
  number: z.string(),
  registered: z.boolean(),
  playerId: z.number().int().nullable(),
})
export type EntryMember = z.infer<typeof entryMemberSchema>

const entryAnswerSchema = z.object({ questionId: z.number().int(), value: z.string() })

export const entryDetailSchema = entrySummarySchema.extend({
  tournamentId: z.number().int(),
  repPhone: z.string(),
  repEmail: z.string(),
  returnNote: z.string(),
  inviteCode: z.string(),
  inviteUrl: z.string(),
  decidedAt: z.string().nullable(),
  memberList: z.array(entryMemberSchema),
  answers: z.array(entryAnswerSchema),
})
export type EntryDetail = z.infer<typeof entryDetailSchema>

/**
 * 運営の画面の申込。メンバーに、本人が答えたメンバー向けの質問を足す(本人用のリンクの応答には無い。
 * ほかのメンバーや代表に見せないため)。
 */
export const entryAdminDetailSchema = entryDetailSchema.extend({
  memberList: z.array(entryMemberSchema.extend({ answers: z.array(entryAnswerSchema) })),
})
export type EntryAdminDetail = z.infer<typeof entryAdminDetailSchema>

/** 参加者の前回の入力(代表の名前と連絡先) */
const entryProfileSchema = z.object({ name: z.string(), phone: z.string(), email: z.string() })
export type EntryProfile = z.infer<typeof entryProfileSchema>

/** 参加者が前にほかの大会に申し込んだチーム */
const entryPastTeamSchema = z.object({
  fullName: z.string(),
  shortName: z.string(),
  members: z.array(z.object({ name: z.string(), number: z.string() })),
})
export type EntryPastTeam = z.infer<typeof entryPastTeamSchema>

export const entryLinkMemberSchema = z.object({
  tournamentName: z.string(),
  team: entrySummarySchema,
  questions: z.array(entryQuestionSchema),
  unclaimed: z.array(entryMemberSchema),
  mine: entryMemberSchema.nullable(),
  mineAnswers: z.array(entryAnswerSchema),
  profileName: z.string(),
  expiresAt: z.string(),
})
export type EntryLinkMember = z.infer<typeof entryLinkMemberSchema>

export const entryLinkSchema = z.object({
  purpose: z.enum(['team', 'member']),
  team: z
    .object({
      tournament: z.object({ id: z.number().int(), name: z.string() }),
      open: z.boolean(),
      divisions: z.array(z.object({ id: z.number().int(), name: i18nTextSchema })),
      questions: z.array(entryQuestionSchema),
      mine: z.array(entryDetailSchema),
      profile: entryProfileSchema.nullable(),
      pastTeams: z.array(entryPastTeamSchema),
      expiresAt: z.string(),
    })
    .nullable(),
  member: entryLinkMemberSchema.nullable(),
})
export type EntryLink = z.infer<typeof entryLinkSchema>

/** 本人用のリンクからのメンバー登録 */
export type EntryJoinInput = {
  memberId: number | null
  name: string
  number: string
  answers: Array<{ questionId: number; value: string }>
}

export const entryExportSchema = z.object({
  questions: z.array(entryQuestionSchema),
  entries: z.array(entryAdminDetailSchema),
})
export type EntryExport = z.infer<typeof entryExportSchema>

/** 保存する質問。id が null なら新しく作る */
export type EntryQuestionInput = Omit<EntryQuestion, 'id' | 'answered'> & { id: number | null }

/** 差し戻した申込の送り直し。registeredIds は画面を開いたときに本人が登録していたメンバーの行 */
export type EntryResubmitInput = EntryDraftInput & { registeredIds: number[] }

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
