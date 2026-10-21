import 'server-only'
import type { z } from 'zod'
import { noContentSchema } from './admin.types'
import { apiFetch } from './client'
import {
  type FormatInput,
  formatPreviewSchema,
  impactSchema,
  progressMatchListSchema,
  progressSchema,
  proposalListSchema,
  savedFormatListSchema,
  savedFormatOneSchema,
  violationListSchema,
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

export type ProposalInput = {
  teams: number
  courts: number
  slotMinutes: number
  date: string
  start: string
  sport: string
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
  shift: (token: string, divisionId: number, court: string | null, minutes: number) =>
    call(token, 'POST', `${admin}/divisions/${divisionId}/schedule-shift`, violationListSchema, {
      court,
      minutes,
    }),
  swap: (token: string, matchId: number, otherMatchId: number) =>
    call(token, 'POST', `${admin}/matches/${matchId}/swap-schedule`, violationListSchema, {
      otherMatchId,
    }),
}
