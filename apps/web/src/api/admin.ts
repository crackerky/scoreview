import 'server-only'
import type { z } from 'zod'
import {
  adminDivisionListSchema,
  adminDivisionSchema,
  adminGalleryListSchema,
  adminGallerySchema,
  adminGroupListSchema,
  adminGroupSchema,
  adminMatchListSchema,
  adminMatchSchema,
  adminPlayerListSchema,
  adminPlayerSchema,
  adminTeamListSchema,
  adminTeamSchema,
  adminTournamentListSchema,
  adminTournamentSchema,
  adminUserSchema,
  assignedMatchListSchema,
  assignmentListSchema,
  assignmentSchema,
  auditListSchema,
  bannersSchema,
  bracketTemplateListSchema,
  deletedTeamsSchema,
  loginResponseSchema,
  matchDetailSchema,
  meResponseSchema,
  noContentSchema,
  type PasswordChangeInput,
  scorerResultSchema,
  type TournamentCopyInput,
  userListSchema,
} from './admin.types'
import { apiFetch } from './client'

// docs/api.md「管理用エンドポイント」を呼ぶ関数。セッショントークンを Bearer で渡す。
// 応答は必ず zod で parse して返す。

const base = '/api/v1'
const admin = `${base}/admin`

type Method = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'

function call<T>(
  token: string,
  method: Method,
  path: string,
  schema: z.ZodType<T>,
  body?: unknown,
) {
  return apiFetch(path, schema, { method, body, sessionToken: token })
}

/** clientIp は利用者の IP(api の IP 単位の試行制限に使う。分からなければ空文字) */
export function login(email: string, password: string, clientIp: string) {
  return apiFetch(`${base}/auth/login`, loginResponseSchema, {
    method: 'POST',
    body: { email, password },
    clientIp,
  })
}

/** 本人がパスワードを変える。このセッションは残り、他の端末のセッションは消える */
export function changePassword(token: string, body: PasswordChangeInput, clientIp: string) {
  return apiFetch(`${base}/auth/password`, noContentSchema, {
    method: 'POST',
    body,
    sessionToken: token,
    clientIp,
  })
}

export function logout(token: string) {
  return call(token, 'POST', `${base}/auth/logout`, noContentSchema)
}

export function getMe(token: string) {
  return call(token, 'GET', `${base}/auth/me`, meResponseSchema)
}

// 大会とバナー
export const tournaments = {
  list: (token: string) => call(token, 'GET', `${admin}/tournaments`, adminTournamentListSchema),
  get: (token: string, id: number) =>
    call(token, 'GET', `${admin}/tournaments/${id}`, adminTournamentSchema),
  create: (token: string, body: unknown) =>
    call(token, 'POST', `${admin}/tournaments`, adminTournamentSchema, body),
  update: (token: string, id: number, body: unknown) =>
    call(token, 'PATCH', `${admin}/tournaments/${id}`, adminTournamentSchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/tournaments/${id}`, noContentSchema),
  banners: (token: string, id: number) =>
    call(token, 'GET', `${admin}/tournaments/${id}/banners`, bannersSchema),
  putBanners: (token: string, id: number, body: unknown) =>
    call(token, 'PUT', `${admin}/tournaments/${id}/banners`, bannersSchema, body),
  /** 大会をコピーして新しい大会を作る(チームと結果以外。withTeams ならチームと選手も) */
  copy: (token: string, id: number, body: TournamentCopyInput) =>
    call(token, 'POST', `${admin}/tournaments/${id}/copy`, adminTournamentSchema, body),
}

export const divisions = {
  list: (token: string, tournamentId: number) =>
    call(token, 'GET', `${admin}/tournaments/${tournamentId}/divisions`, adminDivisionListSchema),
  get: (token: string, id: number) =>
    call(token, 'GET', `${admin}/divisions/${id}`, adminDivisionSchema),
  create: (token: string, tournamentId: number, body: unknown) =>
    call(
      token,
      'POST',
      `${admin}/tournaments/${tournamentId}/divisions`,
      adminDivisionSchema,
      body,
    ),
  update: (token: string, id: number, body: unknown) =>
    call(token, 'PATCH', `${admin}/divisions/${id}`, adminDivisionSchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/divisions/${id}`, noContentSchema),
  putFinalRanks: (token: string, id: number, body: unknown) =>
    call(token, 'PUT', `${admin}/divisions/${id}/final-ranks`, adminTeamListSchema, body),
}

export const groups = {
  list: (token: string, divisionId: number) =>
    call(token, 'GET', `${admin}/divisions/${divisionId}/groups`, adminGroupListSchema),
  get: (token: string, id: number) => call(token, 'GET', `${admin}/groups/${id}`, adminGroupSchema),
  create: (token: string, divisionId: number, body: unknown) =>
    call(token, 'POST', `${admin}/divisions/${divisionId}/groups`, adminGroupSchema, body),
  update: (token: string, id: number, body: unknown) =>
    call(token, 'PATCH', `${admin}/groups/${id}`, adminGroupSchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/groups/${id}`, noContentSchema),
  putTeams: (token: string, id: number, body: unknown) =>
    call(token, 'PUT', `${admin}/groups/${id}/teams`, adminGroupSchema, body),
  generateBracket: (token: string, id: number, body: unknown) =>
    call(token, 'POST', `${admin}/groups/${id}/bracket`, adminGroupSchema, body),
  templates: (token: string) =>
    call(token, 'GET', `${admin}/bracket-templates`, bracketTemplateListSchema),
}

export const teams = {
  list: (token: string, divisionId: number) =>
    call(token, 'GET', `${admin}/divisions/${divisionId}/teams`, adminTeamListSchema),
  get: (token: string, id: number) => call(token, 'GET', `${admin}/teams/${id}`, adminTeamSchema),
  create: (token: string, divisionId: number, body: unknown) =>
    call(token, 'POST', `${admin}/divisions/${divisionId}/teams`, adminTeamSchema, body),
  update: (token: string, id: number, body: unknown) =>
    call(token, 'PATCH', `${admin}/teams/${id}`, adminTeamSchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/teams/${id}`, noContentSchema),
  /** 試合の無い部門のチームをまとめて消す(試合があれば 409) */
  removeAll: (token: string, divisionId: number) =>
    call(token, 'DELETE', `${admin}/divisions/${divisionId}/teams`, deletedTeamsSchema),
}

export const players = {
  list: (token: string, teamId: number) =>
    call(token, 'GET', `${admin}/teams/${teamId}/players`, adminPlayerListSchema),
  create: (token: string, teamId: number, body: unknown) =>
    call(token, 'POST', `${admin}/teams/${teamId}/players`, adminPlayerSchema, body),
  update: (token: string, id: number, body: unknown) =>
    call(token, 'PATCH', `${admin}/players/${id}`, adminPlayerSchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/players/${id}`, noContentSchema),
}

export const matches = {
  list: (token: string, divisionId: number) =>
    call(token, 'GET', `${admin}/divisions/${divisionId}/matches`, adminMatchListSchema),
  detail: (token: string, id: number) =>
    call(token, 'GET', `${admin}/matches/${id}`, matchDetailSchema),
  create: (token: string, divisionId: number, body: unknown) =>
    call(token, 'POST', `${admin}/divisions/${divisionId}/matches`, adminMatchSchema, body),
  update: (token: string, id: number, body: unknown) =>
    call(token, 'PATCH', `${admin}/matches/${id}`, adminMatchSchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/matches/${id}`, noContentSchema),
  addEvent: (token: string, id: number, body: unknown) =>
    call(token, 'POST', `${admin}/matches/${id}/events`, matchDetailSchema, body),
  removeEvent: (token: string, id: number, eventId: number) =>
    call(token, 'DELETE', `${admin}/matches/${id}/events/${eventId}`, matchDetailSchema),
  putPlayerStats: (token: string, id: number, body: unknown) =>
    call(token, 'PUT', `${admin}/matches/${id}/player-stats`, matchDetailSchema, body),
  setStatus: (token: string, id: number, body: unknown) =>
    call(token, 'POST', `${admin}/matches/${id}/status`, matchDetailSchema, body),
  setResult: (token: string, id: number, body: unknown) =>
    call(token, 'POST', `${admin}/matches/${id}/result`, matchDetailSchema, body),
  /** eventRef は得点経過の id か clientId(UUID)。opId は操作 ID(送り直しの冪等性) */
  removeEventByRef: (token: string, id: number, eventRef: string, opId: string) =>
    call(
      token,
      'DELETE',
      `${admin}/matches/${id}/events/${encodeURIComponent(eventRef)}?clientId=${encodeURIComponent(opId)}`,
      matchDetailSchema,
    ),
  updateEvent: (token: string, id: number, eventRef: string, body: unknown) =>
    call(
      token,
      'PATCH',
      `${admin}/matches/${id}/events/${encodeURIComponent(eventRef)}`,
      matchDetailSchema,
      body,
    ),
  /** 得点を相手チームの得点に付け替える(eventRef は得点経過の id か clientId) */
  moveGoal: (token: string, id: number, eventRef: string, body: unknown) =>
    call(
      token,
      'PUT',
      `${admin}/matches/${id}/events/${encodeURIComponent(eventRef)}/side`,
      matchDetailSchema,
      body,
    ),
  /** 試合の記録担当を引き受ける・延長する。takeover なら他の端末から引き継ぐ */
  claimScorer: (token: string, id: number, body: { deviceId: string; takeover: boolean }) =>
    call(token, 'POST', `${admin}/matches/${id}/scorer`, scorerResultSchema, body),
  releaseScorer: (token: string, id: number, deviceId: string) =>
    call(
      token,
      'DELETE',
      `${admin}/matches/${id}/scorer?deviceId=${encodeURIComponent(deviceId)}`,
      noContentSchema,
    ),
  audit: (token: string, id: number) =>
    call(token, 'GET', `${admin}/matches/${id}/audit`, auditListSchema),
  mine: (token: string) => call(token, 'GET', `${admin}/me/matches`, assignedMatchListSchema),
}

// 記録係の割り当て
export const assignments = {
  list: (token: string, tournamentId: number) =>
    call(token, 'GET', `${admin}/tournaments/${tournamentId}/assignments`, assignmentListSchema),
  create: (token: string, tournamentId: number, body: unknown) =>
    call(token, 'POST', `${admin}/tournaments/${tournamentId}/assignments`, assignmentSchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/assignments/${id}`, noContentSchema),
}

export const galleries = {
  list: (token: string, tournamentId: number) =>
    call(token, 'GET', `${admin}/tournaments/${tournamentId}/galleries`, adminGalleryListSchema),
  get: (token: string, id: number) =>
    call(token, 'GET', `${admin}/galleries/${id}`, adminGallerySchema),
  create: (token: string, tournamentId: number, body: unknown) =>
    call(token, 'POST', `${admin}/tournaments/${tournamentId}/galleries`, adminGallerySchema, body),
  update: (token: string, id: number, body: unknown) =>
    call(token, 'PATCH', `${admin}/galleries/${id}`, adminGallerySchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/galleries/${id}`, noContentSchema),
  putPhotos: (token: string, id: number, body: unknown) =>
    call(token, 'PUT', `${admin}/galleries/${id}/photos`, adminGallerySchema, body),
}

export const users = {
  list: (token: string) => call(token, 'GET', `${admin}/users`, userListSchema),
  get: (token: string, id: number) => call(token, 'GET', `${admin}/users/${id}`, adminUserSchema),
  create: (token: string, body: unknown) =>
    call(token, 'POST', `${admin}/users`, adminUserSchema, body),
  update: (token: string, id: number, body: unknown) =>
    call(token, 'PATCH', `${admin}/users/${id}`, adminUserSchema, body),
  remove: (token: string, id: number) =>
    call(token, 'DELETE', `${admin}/users/${id}`, noContentSchema),
  putTournaments: (token: string, id: number, body: unknown) =>
    call(token, 'PUT', `${admin}/users/${id}/tournaments`, adminUserSchema, body),
}
