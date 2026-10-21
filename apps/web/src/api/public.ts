import 'server-only'
import { apiFetch } from './client'
import {
  type GalleryResponse,
  type GroupResponse,
  galleryResponseSchema,
  groupResponseSchema,
  type MatchResponse,
  matchResponseSchema,
  type PlayerResponse,
  playerResponseSchema,
  type ScheduleResponse,
  type StandingsResponse,
  scheduleResponseSchema,
  standingsResponseSchema,
  type TeamListResponse,
  type TeamResponse,
  type TournamentListResponse,
  type TournamentTopResponse,
  teamListResponseSchema,
  teamResponseSchema,
  tournamentListResponseSchema,
  tournamentTopResponseSchema,
} from './public.types'

// docs/api.md「閲覧用エンドポイント」を呼ぶ関数。応答は必ず zod で parse して返す。
const base = '/api/v1'

export function getTournaments(): Promise<TournamentListResponse> {
  return apiFetch(`${base}/tournaments`, tournamentListResponseSchema)
}

export function getTournamentTop(id: number): Promise<TournamentTopResponse> {
  return apiFetch(`${base}/tournaments/${id}`, tournamentTopResponseSchema)
}

export function getTeamList(tournamentId: number): Promise<TeamListResponse> {
  return apiFetch(`${base}/tournaments/${tournamentId}/teams`, teamListResponseSchema)
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

export function getDivisionSchedule(
  divisionId: number,
  query: ScheduleQuery = {},
): Promise<ScheduleResponse> {
  return apiFetch(`${base}/divisions/${divisionId}/schedule`, scheduleResponseSchema, { query })
}

export function getDivisionStandings(divisionId: number): Promise<StandingsResponse> {
  return apiFetch(`${base}/divisions/${divisionId}/standings`, standingsResponseSchema)
}

export function getGroup(groupId: number): Promise<GroupResponse> {
  return apiFetch(`${base}/groups/${groupId}`, groupResponseSchema)
}

export function getTeam(teamId: number): Promise<TeamResponse> {
  return apiFetch(`${base}/teams/${teamId}`, teamResponseSchema)
}

export function getMatch(matchId: number): Promise<MatchResponse> {
  return apiFetch(`${base}/matches/${matchId}`, matchResponseSchema)
}

export function getPlayer(playerId: number): Promise<PlayerResponse> {
  return apiFetch(`${base}/players/${playerId}`, playerResponseSchema)
}

export function getGallery(galleryId: number): Promise<GalleryResponse> {
  return apiFetch(`${base}/galleries/${galleryId}`, galleryResponseSchema)
}
