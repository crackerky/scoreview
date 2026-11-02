import 'server-only'
import { apiFetch } from './client'
import {
  type GroupResponse,
  groupResponseSchema,
  type LiveTablesResponse,
  liveTablesResponseSchema,
  type MatchResponse,
  matchResponseSchema,
  type ScheduleResponse,
  scheduleResponseSchema,
  type TournamentListResponse,
  tournamentListResponseSchema,
} from './public.types'

// docs/api.md「閲覧用エンドポイント」を呼ぶ関数(大会の一覧と観戦アプリが使う)。応答は必ず zod で
// parse して返す。
const base = '/api/v1'

export function getTournaments(): Promise<TournamentListResponse> {
  return apiFetch(`${base}/tournaments`, tournamentListResponseSchema)
}

/**
 * スケジュールの絞り込み。undefined のキーは送らない(ymd はキーの有無で既定値が変わる。
 * 空文字は「全日程」を意味するので送る)。
 */
export type ScheduleQuery = {
  ymd?: string
  team?: string
  field?: string
  sort?: string
  dir?: 'asc' | 'desc'
}

export function getTournamentSchedule(
  tournamentId: number,
  query: ScheduleQuery = {},
): Promise<ScheduleResponse> {
  return apiFetch(`${base}/tournaments/${tournamentId}/schedule`, scheduleResponseSchema, {
    query,
  })
}

export function getGroup(groupId: number): Promise<GroupResponse> {
  return apiFetch(`${base}/groups/${groupId}`, groupResponseSchema)
}

export function getMatch(matchId: number): Promise<MatchResponse> {
  return apiFetch(`${base}/matches/${matchId}`, matchResponseSchema)
}

/** 観戦アプリの「表で見る」の見せ方とトーナメントの形 */
export function getLiveTables(tournamentId: number): Promise<LiveTablesResponse> {
  return apiFetch(`${base}/tournaments/${tournamentId}/live-tables`, liveTablesResponseSchema)
}
