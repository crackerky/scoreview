// 申込の管理画面の URL。

export const entryPaths = {
  /** 大会の申込一覧。status で状態を絞る */
  list: (tournamentId: number, status?: string) =>
    `/admin/tournaments/${tournamentId}/entries${status ? `?status=${status}` : ''}`,
  /** 電話や紙で受けた申込を代わりに入れる */
  create: (tournamentId: number) => `/admin/tournaments/${tournamentId}/entries/new`,
  /** 大会ごとの質問 */
  questions: (tournamentId: number) => `/admin/tournaments/${tournamentId}/entry-questions`,
  /** 申込の書き出し(CSV) */
  csv: (tournamentId: number) => `/admin/tournaments/${tournamentId}/entries/csv`,
  detail: (id: number) => `/admin/entries/${id}`,
}
