import { divisions } from '@/api/admin'
import type { AdminDivision } from '@/api/admin.types'
import { formats } from '@/api/format'
import type { DivisionBoard } from './hqBoard.utils'
import { loadAdmin } from './session'

// 大会単位の本部と大会ホームが読む、部門ごとの進行状況。形を決めた部門だけが本部に出る
// (形の無い部門は進行状況を持たない)。

export type TournamentBoards = {
  divisions: AdminDivision[]
  boards: DivisionBoard[]
  /** 形をまだ決めていない部門 */
  unformatted: AdminDivision[]
}

export async function loadBoards(token: string, tournamentId: number): Promise<TournamentBoards> {
  const list = await loadAdmin(() => divisions.list(token, tournamentId))
  const loaded = await loadAdmin(() =>
    Promise.all(
      list.divisions.map(async (division) => {
        const progress = await formats.progressIfFormatted(token, division.id)
        if (progress === null) {
          return { division, board: null }
        }
        const matches = await formats.matches(token, division.id)
        const board: DivisionBoard = {
          divisionId: division.id,
          divisionName: division.name.ja,
          progress,
          matches: matches.matches,
        }
        return { division, board }
      }),
    ),
  )
  return {
    divisions: list.divisions,
    boards: loaded.flatMap((item) => (item.board ? [item.board] : [])),
    unformatted: loaded.filter((item) => item.board === null).map((item) => item.division),
  }
}
