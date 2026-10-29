import { z } from 'zod'

// 大会を作る 3 ステップ(Claude Design「Score View Admin」)の、ブラウザとサーバーで共有する決まり。
// 前回の大会をコピーするときは、出ないチームだけを新しいチーム名に入れ替える。チーム数は変えない
// (大会の形の組分けと試合数がチーム数で決まっているため)。

/** 入れ替え 1 件。部門と部門内のチームは、コピー元とコピー先で同じ並び(並び順、ID の順)の位置で指す */
export const replacementSchema = z.object({
  division: z.number().int().nonnegative(),
  team: z.number().int().nonnegative(),
  name: z.string().trim().min(1).max(100),
})
export type Replacement = z.infer<typeof replacementSchema>

/** フォームの hidden 欄(JSON)から入れ替えを読む。形の違う値は入力の誤りとして投げる */
export function parseReplacements(raw: string): Replacement[] {
  if (raw.trim() === '') {
    return []
  }
  return z.array(replacementSchema).parse(JSON.parse(raw))
}

export type SourceDivision = { name: string; teams: string[] }

/** 確認の画面の「チーム」: 入れ替えたチームは新しい名前で並べる */
export function finalTeamNames(divisions: SourceDivision[], replacements: Replacement[]): string[] {
  return divisions.flatMap((division, divisionIndex) =>
    division.teams.map(
      (team, teamIndex) =>
        replacements.find((item) => item.division === divisionIndex && item.team === teamIndex)
          ?.name ?? team,
    ),
  )
}
