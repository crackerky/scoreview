import type { Lang } from './lang'

// 参照サイトと同じパスでリンクを作る。全リンクは現在の lang を ?lang= で引き継ぐ(clone-spec A 0)。
// 参照サイトは絶対 URL だが、ここではホストに依存しないようルート相対パスにする。

/** 参照サイトの /pscore/{controller}/open/{id} のコントローラ名。 */
export type PscoreController =
  | 'scv_top'
  | 'scv_team_list'
  | 'scv_search_tn'
  | 'scv_search_div'
  | 'scv_division'
  | 'scv_group'
  | 'scv_tgallery'
  | 'scv_team'
  | 'scv_match'
  | 'scv_player'

function withQuery(path: string, lang: Lang, extra: Record<string, string>): string {
  const params = new URLSearchParams({ lang })
  for (const [key, value] of Object.entries(extra)) {
    params.set(key, value)
  }
  return `${path}?${params.toString()}`
}

/** /pscore/{controller}/open/{id}?lang=..(extra は lang の後ろに付く。スケジュールの絞り込み用) */
export function pscoreHref(
  controller: PscoreController,
  id: number,
  lang: Lang,
  extra: Record<string, string> = {},
): string {
  return withQuery(`/pscore/${controller}/open/${id}`, lang, extra)
}

/** 大会一覧 /sctnmt */
export function tournamentListHref(lang: Lang): string {
  return withQuery('/sctnmt', lang, {})
}

/** 大会一覧から大会トップへのリンク /scview/{id}(clone-spec A 3) */
export function scviewHref(tournamentId: number, lang: Lang): string {
  return withQuery(`/scview/${tournamentId}`, lang, {})
}

export const scvTopHref = (tournamentId: number, lang: Lang) =>
  pscoreHref('scv_top', tournamentId, lang)
export const teamListHref = (tournamentId: number, lang: Lang) =>
  pscoreHref('scv_team_list', tournamentId, lang)
export const tournamentScheduleHref = (tournamentId: number, lang: Lang) =>
  pscoreHref('scv_search_tn', tournamentId, lang)
export const divisionScheduleHref = (divisionId: number, lang: Lang) =>
  pscoreHref('scv_search_div', divisionId, lang)
export const divisionHref = (divisionId: number, lang: Lang) =>
  pscoreHref('scv_division', divisionId, lang)
export const groupHref = (groupId: number, lang: Lang) => pscoreHref('scv_group', groupId, lang)
export const galleryHref = (galleryId: number, lang: Lang) =>
  pscoreHref('scv_tgallery', galleryId, lang)
export const teamHref = (teamId: number, lang: Lang) => pscoreHref('scv_team', teamId, lang)
export const matchHref = (matchId: number, lang: Lang) => pscoreHref('scv_match', matchId, lang)
export const playerHref = (playerId: number, lang: Lang) => pscoreHref('scv_player', playerId, lang)
