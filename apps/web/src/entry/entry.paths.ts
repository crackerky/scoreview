// 申込の画面の URL(管理画面と、公式 LINE が返す本人用のリンクの画面)。

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
  /** 本人用のリンクの画面。done で送った結果を伝える */
  link: (token: string, query?: Record<string, string>) =>
    `/entry/${token}${query ? `?${new URLSearchParams(query).toString()}` : ''}`,
}
