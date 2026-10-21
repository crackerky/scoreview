import { z } from 'zod'
import {
  bracketCellSchema,
  groupKindSchema,
  matchStatusSchema,
  suspensionSchema,
  tallySchema,
  teamRefSchema,
} from './common.types'

// docs/api.md「大会の形と進行」の応答の zod スキーマ。

const id = z.number().int()

export const sportSchema = z.enum(['seven', 'four', 'beach'])
export type Sport = z.infer<typeof sportSchema>

export const tiebreakSchema = z.enum(['current', 'current_no_h2h', 'jfda'])
export type Tiebreak = z.infer<typeof tiebreakSchema>

/** 形のパラメータ(テンプレートごとに形が違う JSON)。中身は api が検査するので、ここでは素通しにする */
export const formatParamsSchema = z.record(z.string(), z.unknown())
export type FormatParams = z.infer<typeof formatParamsSchema>

export const proposalSchema = z.object({
  id: z.string(),
  title: z.string(),
  template: z.string(),
  params: formatParamsSchema,
  recommended: z.boolean(),
  note: z.string(),
  matches: z.number().int(),
  end: z.string(),
  ranked: z.number().int(),
  unique: z.number().int(),
  violations: z.number().int(),
})
export type Proposal = z.infer<typeof proposalSchema>

/** POST /admin/divisions/{id}/format/proposals */
export const proposalListSchema = z.object({ proposals: z.array(proposalSchema) })

export const scheduleSettingSchema = z.object({
  date: z.string(),
  start: z.string(),
  slotMinutes: z.number().int(),
  courts: z.array(z.string()),
  teamRestSlots: z.number().int(),
  stageGapSlots: z.number().int(),
  end: z.string().nullable(),
})
export type ScheduleSetting = z.infer<typeof scheduleSettingSchema>

export const forfeitSchema = z.object({ winner: z.number().int(), loser: z.number().int() })

/** PUT /admin/divisions/{id}/format と preview のボディ */
export type FormatInput = {
  template: string
  params: FormatParams
  seeds: number[]
  tiebreak: Tiebreak
  forfeit: { winner: number; loser: number }
  schedule: ScheduleSetting | null
}

const previewSideSchema = z.object({ label: z.string(), team: teamRefSchema.nullable() })

export const violationSchema = z.object({
  kind: z.string(),
  codes: z.array(z.string()),
  shortMinutes: z.number().int(),
  detail: z.string(),
})
export type Violation = z.infer<typeof violationSchema>

export const violationListSchema = z.object({ violations: z.array(violationSchema) })

export const previewBracketSchema = z.object({
  pattern: z.string(),
  ptnName: z.string(),
  columns: z.number().int(),
  rows: z.array(z.array(bracketCellSchema)),
})

export const previewGroupSchema = z.object({
  index: z.number().int(),
  letter: z.string(),
  name: z.string(),
  kind: groupKindSchema,
  stage: z.number().int(),
  slots: z.array(
    z.object({
      position: z.number().int(),
      label: z.string(),
      seed: z.number().int().nullable(),
      team: teamRefSchema.nullable(),
    }),
  ),
  bracket: previewBracketSchema.nullable(),
})
export type PreviewGroup = z.infer<typeof previewGroupSchema>

export const previewMatchSchema = z.object({
  code: z.string(),
  group: z.number().int(),
  round: z.number().int(),
  left: previewSideSchema,
  right: previewSideSchema,
  date: z.string().nullable(),
  time: z.string().nullable(),
  field: z.string().nullable(),
})
export type PreviewMatch = z.infer<typeof previewMatchSchema>

/** POST /admin/divisions/{id}/format/preview と PUT /admin/divisions/{id}/format の応答 */
export const formatPreviewSchema = z.object({
  title: z.string(),
  startLetter: z.string(),
  teams: z.number().int(),
  stages: z.array(z.string()),
  groups: z.array(previewGroupSchema),
  matches: z.array(previewMatchSchema),
  courts: z.array(z.string()),
  violations: z.array(violationSchema),
  end: z.string().nullable(),
  ruleText: z.array(z.string()),
  advancement: z.array(z.string()),
  ranked: z.number().int(),
  unique: z.number().int(),
  replacesGroups: z.number().int(),
  replacesMatches: z.number().int(),
})
export type FormatPreview = z.infer<typeof formatPreviewSchema>

export const savedFormatSchema = z.object({
  divisionId: id,
  tournamentId: id,
  tournamentName: z.string(),
  divisionName: z.string(),
  title: z.string(),
  version: z.number().int(),
  template: z.string(),
  params: formatParamsSchema,
  tiebreak: tiebreakSchema,
  forfeit: forfeitSchema,
  schedule: scheduleSettingSchema.nullable(),
  seeds: z.array(id),
})
export type SavedFormat = z.infer<typeof savedFormatSchema>

/** GET /admin/divisions/{id}/format */
export const savedFormatOneSchema = z.object({ format: savedFormatSchema.nullable() })
/** GET /admin/formats */
export const savedFormatListSchema = z.object({ formats: z.array(savedFormatSchema) })

// ===== 進行状況 =====

export const groupStateSchema = z.enum([
  'waiting',
  'playing',
  'tied',
  'ready',
  'confirmed',
  'finished',
])
export type GroupState = z.infer<typeof groupStateSchema>

const progressSideSchema = z.object({
  label: z.string(),
  team: teamRefSchema.nullable(),
  score: z.number().int().nullable(),
})
export type ProgressSide = z.infer<typeof progressSideSchema>

export const progressMatchSchema = z.object({
  id,
  code: z.string(),
  groupName: z.string(),
  /** 管理画面に出すグループ名(「予選 A 組」「1〜4 位を決めるトーナメント」) */
  groupLabel: z.string(),
  date: z.string().nullable(),
  time: z.string().nullable(),
  field: z.string().nullable(),
  status: matchStatusSchema,
  left: progressSideSchema,
  right: progressSideSchema,
})
export type ProgressMatch = z.infer<typeof progressMatchSchema>

export const todoSchema = z.object({
  kind: z.enum([
    'tie',
    'confirm_group',
    'confirm_stage',
    'overdue',
    'court_delay',
    'mismatch',
    'overlap',
    'withdrawn',
  ]),
  text: z.string(),
  groupId: id.nullable(),
  stageId: id.nullable(),
  matchId: id.nullable(),
  /** withdrawn の棄権したチーム */
  teamId: id.nullable(),
  court: z.string().nullable(),
  minutes: z.number().int(),
})
export type Todo = z.infer<typeof todoSchema>

export const groupProgressSchema = z.object({
  id,
  letter: z.string(),
  name: z.string(),
  /** 管理画面に出すグループ名 */
  label: z.string(),
  kind: groupKindSchema,
  stage: z.number().int(),
  state: groupStateSchema,
  confirmedUnplayed: z.boolean(),
  remaining: z.number().int(),
  slots: z.array(progressSideSchema),
  rows: z.array(
    tallySchema.extend({
      team: teamRefSchema,
      rank: z.number().int(),
      points: z.number().int(),
      reason: z.string(),
      unresolved: z.boolean(),
    }),
  ),
  ties: z.array(z.array(teamRefSchema)),
  conditions: z.array(z.object({ team: teamRefSchema, text: z.string() })),
})
export type GroupProgress = z.infer<typeof groupProgressSchema>

/** GET /admin/divisions/{id}/progress */
export const progressSchema = z.object({
  divisionId: id,
  title: z.string(),
  version: z.number().int(),
  tiebreak: tiebreakSchema,
  ruleText: z.array(z.string()),
  advancement: z.array(z.string()),
  slotMinutes: z.number().int(),
  courts: z.array(z.string()),
  now: z.string(),
  todo: z.array(todoSchema),
  stages: z.array(
    z.object({
      id,
      index: z.number().int(),
      name: z.string(),
      confirmed: z.boolean(),
      confirmedUnplayed: z.boolean(),
      needsConfirm: z.boolean(),
    }),
  ),
  groups: z.array(groupProgressSchema),
  live: z.array(progressMatchSchema),
  next: z.array(progressMatchSchema),
  final: z.array(z.object({ rank: z.number().int(), team: teamRefSchema, tied: z.boolean() })),
  /** 棄権したチームの ID */
  withdrawn: z.array(id),
  /** 大会を締めた時刻。まだなら null */
  closedAt: z.iso.datetime().nullable(),
  /** 大会の中断(全部門に共通) */
  suspension: suspensionSchema.nullable(),
})
export type Progress = z.infer<typeof progressSchema>

/** POST /admin/divisions/{id}/schedule-shift */
export const shiftResultSchema = z.object({
  violations: z.array(violationSchema),
  /** 動いた(dryRun なら動く)試合の数 */
  moved: z.number().int(),
})

const withdrawalMatchSchema = z.object({
  id,
  code: z.string(),
  date: z.string().nullable(),
  time: z.string().nullable(),
  field: z.string().nullable(),
  status: matchStatusSchema,
  opponent: teamRefSchema.nullable(),
  result: z.enum(['forfeit', 'double_forfeit']),
})
export type WithdrawalMatch = z.infer<typeof withdrawalMatchSchema>

/** GET / POST /admin/teams/{id}/withdrawal(チームの棄権) */
export const withdrawalSchema = z.object({
  team: teamRefSchema,
  divisionId: id,
  withdrawnAt: z.iso.datetime().nullable(),
  targets: z.array(withdrawalMatchSchema),
  waiting: z.array(withdrawalMatchSchema),
  forfeitScore: z.object({ winner: z.number().int(), loser: z.number().int() }).nullable(),
})
export type Withdrawal = z.infer<typeof withdrawalSchema>

/** GET /admin/divisions/{id}/closing と POST close(大会の締め) */
export const closingSchema = z.object({
  remaining: z.array(progressMatchSchema),
  ranks: z.array(
    z.object({
      rank: z.number().int(),
      team: teamRefSchema,
      tied: z.boolean(),
      proposed: z.boolean(),
    }),
  ),
  ties: z.array(z.string()),
  closedAt: z.iso.datetime().nullable(),
  published: z.boolean(),
})
export type Closing = z.infer<typeof closingSchema>

/** POST /admin/tournaments/{id}/resume(中断の再開) */
export const resumeSchema = z.object({
  minutes: z.number().int(),
  shifted: z.number().int(),
  overEnd: z.array(
    z.object({
      matchId: id,
      code: z.string(),
      division: z.string(),
      time: z.string(),
      overMinutes: z.number().int(),
    }),
  ),
})
export type ResumeResult = z.infer<typeof resumeSchema>

/** GET /admin/divisions/{id}/format/matches */
export const progressMatchListSchema = z.object({ matches: z.array(progressMatchSchema) })

/** POST /admin/matches/{id}/correction-impact と correction の応答 */
export const impactSchema = z.object({
  lines: z.array(z.string()),
  matches: z.array(
    z.object({
      matchId: id,
      code: z.string(),
      groupName: z.string(),
      beforeLeft: teamRefSchema.nullable(),
      beforeRight: teamRefSchema.nullable(),
      afterLeft: teamRefSchema.nullable(),
      afterRight: teamRefSchema.nullable(),
      hasResult: z.boolean(),
    }),
  ),
  needsApproval: z.boolean(),
})
export type Impact = z.infer<typeof impactSchema>
