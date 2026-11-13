import 'server-only'
import { apiFetch } from './client'
import {
  type BoardResponse,
  boardResponseSchema,
  type MatchResponse,
  matchResponseSchema,
  type TournamentListResponse,
  tournamentListResponseSchema,
} from './public.types'

// docs/api.md「閲覧用エンドポイント」を呼ぶ関数(大会の一覧、スコアサイト、観戦アプリが使う)。応答は必ず zod で
// parse して返す。
const base = '/api/v1'

export function getTournaments(): Promise<TournamentListResponse> {
  return apiFetch(`${base}/tournaments`, tournamentListResponseSchema)
}

/** 試合。revalidateSeconds は getBoard と同じ */
export function getMatch(
  matchId: number,
  { revalidateSeconds }: { revalidateSeconds?: number } = {},
): Promise<MatchResponse> {
  return apiFetch(`${base}/matches/${matchId}`, matchResponseSchema, { revalidateSeconds })
}

/**
 * 大会の掲示板(星取表、順位、日程、トーナメントの形を 1 回で)。revalidateSeconds があれば、その秒数
 * のあいだ観客のあいだで応答を使い回す(公開データなので誰に返しても同じ)
 */
export function getBoard(
  tournamentId: number,
  { revalidateSeconds }: { revalidateSeconds?: number } = {},
): Promise<BoardResponse> {
  return apiFetch(`${base}/tournaments/${tournamentId}/board`, boardResponseSchema, {
    revalidateSeconds,
  })
}
