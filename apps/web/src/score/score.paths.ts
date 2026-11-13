// スコアサイトの URL。ページの置き場所はここだけで決め、画面の部品はこの関数で行き先を作る。

export const scorePaths = {
  /** 入口(全部の組の星取表) */
  top: (tournamentId: number) => `/tournaments/${tournamentId}`,
  schedule: (tournamentId: number) => `/tournaments/${tournamentId}/schedule`,
  bracket: (tournamentId: number) => `/tournaments/${tournamentId}/bracket`,
  pool: (tournamentId: number, groupId: number) => `/tournaments/${tournamentId}/pools/${groupId}`,
  match: (tournamentId: number, matchId: number) =>
    `/tournaments/${tournamentId}/matches/${matchId}`,
  /** 同じ大会の観戦アプリ */
  live: (tournamentId: number) => `/live/${tournamentId}`,
} as const

/** スコアサイトの 3 つの入口(並びは画面の順) */
export const scoreTabs = [
  { key: 'score', label: 'スコア', path: scorePaths.top },
  { key: 'schedule', label: 'タイムスケジュール', path: scorePaths.schedule },
  { key: 'bracket', label: 'トーナメント', path: scorePaths.bracket },
] as const

export type ScoreTab = (typeof scoreTabs)[number]['key']

/** パスがどの入口の中か。組と試合の画面は、そこへ来た入口が分からないのでスコアに含める */
export function tabOf(pathname: string, tournamentId: number): ScoreTab {
  if (pathname.startsWith(scorePaths.schedule(tournamentId))) {
    return 'schedule'
  }
  if (pathname.startsWith(scorePaths.bracket(tournamentId))) {
    return 'bracket'
  }
  return 'score'
}
