import type { TournamentSummary } from '@/api/public.types'

/** 「大会終了」にそのまま並べる最近の大会の数。これより前の大会は年ごとにたたむ(クライアントの要望) */
export const recentFinishedCount = 5

export type YearGroup = {
  /** 開催年。開催日が未定の大会は null */
  year: number | null
  tournaments: TournamentSummary[]
}

/**
 * 終了した大会を、最近の recentFinishedCount 件と、それより前の年ごとのまとめに分ける。
 * 入力は api の並び(開催日の新しい順、未定は先頭)のままとし、並びを変えない。
 */
export function splitFinished(tournaments: TournamentSummary[]): {
  recent: TournamentSummary[]
  older: YearGroup[]
} {
  const recent = tournaments.slice(0, recentFinishedCount)
  const older: YearGroup[] = []
  for (const tournament of tournaments.slice(recentFinishedCount)) {
    const year = tournament.startDate === null ? null : Number(tournament.startDate.slice(0, 4))
    const last = older.at(-1)
    if (last !== undefined && last.year === year) {
      last.tournaments.push(tournament)
    } else {
      older.push({ year, tournaments: [tournament] })
    }
  }
  return { recent, older }
}

/** messages の "{n}" などを値で埋める。 */
export function fillTemplate(template: string, values: Record<string, string | number>): string {
  return template.replace(/\{(\w+)\}/g, (whole, key: string) =>
    key in values ? String(values[key]) : whole,
  )
}
