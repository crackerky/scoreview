import 'server-only'
import type { z } from 'zod'
import { noContentSchema } from './admin.types'
import { ApiError, apiFetch } from './client'
import { suspensionSchema } from './common.types'
import {
  closingSchema,
  type FormatInput,
  formatPreviewSchema,
  impactSchema,
  progressMatchListSchema,
  progressSchema,
  proposalListSchema,
  resumeSchema,
  type ScheduleSetting,
  savedFormatListSchema,
  savedFormatOneSchema,
  shiftResultSchema,
  violationListSchema,
  withdrawalSchema,
} from './format.types'

// docs/api.md「大会の形と進行」を呼ぶ関数。応答は zod で parse して返す。

const admin = '/api/v1/admin'

type Method = 'GET' | 'POST' | 'PUT'

function call<T>(
  token: string,
  method: Method,
  path: string,
  schema: z.ZodType<T>,
  body?: unknown,
) {
  return apiFetch(path, schema, { method, body, sessionToken: token })
}

/** 形の候補の条件。schedule があれば courts 以下の代わりにそれを使う(形の適用と同じ条件) */
export type ProposalInput = {
  teams: number
  courts: number
  slotMinutes: number
  date: string
  start: string
  sport: string
  schedule: ScheduleSetting | null
}

/** 結果のある下流の試合の対戦が変わるときの承認と下流の扱い。null は承認しない */
export type Downstream = 'keep' | 'reset' | null

export type CorrectionBody = {
  score: { left: number; right: number }
  winnerSide: 'left' | 'right' | null
  downstream: Downstream
  /** 画面を開いたときのスコア(楽観的ロック。違えば 409 score_mismatch) */
  expectedScore?: { left: number; right: number } | null
  /** 得点経過を残したまま score で確定する */
  keepEvents?: boolean
}

/** 大会の中断の入力 */
export type SuspendBody = {
  reason: 'thunder' | 'rain' | 'heat' | 'other'
  note: string
  resumeAt: string | null
}

export const formats = {
  propose: (token: string, divisionId: number, body: ProposalInput) =>
    call(
      token,
      'POST',
      `${admin}/divisions/${divisionId}/format/proposals`,
      proposalListSchema,
      body,
    ),
  preview: (token: string, divisionId: number, body: FormatInput) =>
    call(
      token,
      'POST',
      `${admin}/divisions/${divisionId}/format/preview`,
      formatPreviewSchema,
      body,
    ),
  apply: (token: string, divisionId: number, body: FormatInput) =>
    call(token, 'PUT', `${admin}/divisions/${divisionId}/format`, formatPreviewSchema, body),
  get: (token: string, divisionId: number) =>
    call(token, 'GET', `${admin}/divisions/${divisionId}/format`, savedFormatOneSchema),
  list: (token: string, exclude: number) =>
    call(token, 'GET', `${admin}/formats?exclude=${exclude}`, savedFormatListSchema),
  progress: (token: string, divisionId: number) =>
    call(token, 'GET', `${admin}/divisions/${divisionId}/progress`, progressSchema),
  matches: (token: string, divisionId: number) =>
    call(token, 'GET', `${admin}/divisions/${divisionId}/format/matches`, progressMatchListSchema),
  /** 形の無い部門(409)なら null。印刷物で未確定の対戦の決まり方を出すため(記録係も呼べる) */
  matchesIfFormatted: async (token: string, divisionId: number) => {
    try {
      return await call(
        token,
        'GET',
        `${admin}/divisions/${divisionId}/format/matches`,
        progressMatchListSchema,
      )
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        return null
      }
      throw error
    }
  },
  confirmGroup: (token: string, groupId: number, unplayed: boolean) =>
    call(token, 'POST', `${admin}/groups/${groupId}/confirm`, noContentSchema, { unplayed }),
  confirmStage: (token: string, stageId: number, unplayed: boolean) =>
    call(token, 'POST', `${admin}/stages/${stageId}/confirm`, noContentSchema, { unplayed }),
  decideRank: (token: string, groupId: number, teamIds: number[], reason: string) =>
    call(token, 'POST', `${admin}/groups/${groupId}/rank-decisions`, noContentSchema, {
      teamIds,
      reason,
    }),
  correctionImpact: (token: string, matchId: number, body: CorrectionBody) =>
    call(token, 'POST', `${admin}/matches/${matchId}/correction-impact`, impactSchema, body),
  correct: (token: string, matchId: number, body: CorrectionBody) =>
    call(token, 'POST', `${admin}/matches/${matchId}/correction`, impactSchema, body),
  alignTeams: (token: string, matchId: number) =>
    call(token, 'POST', `${admin}/matches/${matchId}/align-teams`, noContentSchema),
  checkSchedule: (token: string, divisionId: number) =>
    call(token, 'GET', `${admin}/divisions/${divisionId}/schedule-check`, violationListSchema),
  shift: (
    token: string,
    divisionId: number,
    court: string | null,
    minutes: number,
    dryRun = false,
  ) =>
    call(token, 'POST', `${admin}/divisions/${divisionId}/schedule-shift`, shiftResultSchema, {
      court,
      minutes,
      dryRun,
    }),
  withdrawal: (token: string, teamId: number) =>
    call(token, 'GET', `${admin}/teams/${teamId}/withdrawal`, withdrawalSchema),
  withdraw: (token: string, teamId: number, matchIds: number[]) =>
    call(token, 'POST', `${admin}/teams/${teamId}/withdrawal`, withdrawalSchema, { matchIds }),
  closing: (token: string, divisionId: number) =>
    call(token, 'GET', `${admin}/divisions/${divisionId}/closing`, closingSchema),
  close: (token: string, divisionId: number, matchIds: number[], publish: boolean) =>
    call(token, 'POST', `${admin}/divisions/${divisionId}/close`, closingSchema, {
      matchIds,
      publish,
    }),
  suspend: (token: string, tournamentId: number, body: SuspendBody) =>
    call(token, 'POST', `${admin}/tournaments/${tournamentId}/suspension`, suspensionSchema, body),
  resume: (token: string, tournamentId: number, minutes: number | null) =>
    call(token, 'POST', `${admin}/tournaments/${tournamentId}/resume`, resumeSchema, { minutes }),
  swap: (token: string, matchId: number, otherMatchId: number) =>
    call(token, 'POST', `${admin}/matches/${matchId}/swap-schedule`, violationListSchema, {
      otherMatchId,
    }),
}
