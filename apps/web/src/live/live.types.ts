import { z } from 'zod'
import { liveTablesConfigSchema } from '@/api/public.types'

// 観戦アプリ(/live/{大会ID})が自分のデータの口(/live/{id}/data、/live/{id}/matches/{matchId})から
// 受け取る形。ブラウザは取り直した応答をこのスキーマで確かめてから使う。

export const liveStatusSchema = z.enum(['live', 'sched', 'done'])
export type LiveStatus = z.infer<typeof liveStatusSchema>

export const liveSideSchema = z.object({
  teamId: z.number().int().nullable(),
  name: z.string(),
  /** チームが決まっていない(「A1 の勝者」のような決まり方を出している) */
  pending: z.boolean(),
  color: z.string(),
  score: z.number().int().nullable(),
})
export type LiveSide = z.infer<typeof liveSideSchema>

export const liveMatchSchema = z.object({
  id: z.number().int(),
  code: z.string(),
  date: z.string().nullable(),
  time: z.string(),
  court: z.string(),
  status: liveStatusSchema,
  /** 何を決める試合か(「予選 A 組」「1〜4位 準決勝」「3位決定戦」)。形の無い部門は組の名前 */
  stage: z.string(),
  /** 組の ID(順位表の組とつなぐ)。組に入っていない試合は null */
  groupId: z.number().int().nullable(),
  pool: z.boolean(),
  divisionId: z.number().int(),
  divisionName: z.string(),
  left: liveSideSchema,
  right: liveSideSchema,
})
export type LiveMatch = z.infer<typeof liveMatchSchema>

export const livePoolSchema = z.object({
  id: z.number().int(),
  /** 組の名前。形を決めた部門は運営の言葉(「予選 A 組」) */
  name: z.string(),
  done: z.boolean(),
  /** 優勝を争うグループへ進む上位の数(星取表の進出圏の色。0 は色なし) */
  advance: z.number().int(),
  /** 順位の順(順位の無いチームは最後)。entry は組の登録順の位置(0 から) */
  rows: z.array(
    z.object({
      rank: z.number().int().nullable(),
      entry: z.number().int(),
      teamId: z.number().int(),
      name: z.string(),
      color: z.string(),
      points: z.number().int(),
      wins: z.number().int(),
      draws: z.number().int(),
      losses: z.number().int(),
      goalsFor: z.number().int(),
      goalsAgainst: z.number().int(),
    }),
  ),
})
export type LivePool = z.infer<typeof livePoolSchema>

const bracketSideSchema = z.object({
  from: z.object({ code: z.string(), outcome: z.enum(['win', 'lose']) }).nullable(),
  seed: z.string().nullable(),
})

/** トーナメント表の形(docs/api.md「GET /api/v1/tournaments/{id}/live-tables」)。試合は部門と試合番号で引く */
export const liveBracketSchema = z.object({
  divisionId: z.number().int(),
  name: z.string(),
  matches: z.array(
    z.object({
      code: z.string(),
      winnerPlace: z.number().int(),
      left: bracketSideSchema,
      right: bracketSideSchema,
    }),
  ),
})
export type LiveBracket = z.infer<typeof liveBracketSchema>

export const liveDataSchema = z.object({
  tournament: z.object({ id: z.number().int(), name: z.string(), date: z.string().nullable() }),
  sponsor: z.object({ imageUrl: z.string(), linkUrl: z.string().nullable() }).nullable(),
  matches: z.array(liveMatchSchema),
  pools: z.array(livePoolSchema),
  /** 「表で見る」の見せ方とトーナメント表の形 */
  tables: z.object({ config: liveTablesConfigSchema, brackets: z.array(liveBracketSchema) }),
})
export type LiveData = z.infer<typeof liveDataSchema>

const flowPlayerSchema = z.object({ name: z.string(), number: z.string() }).nullable()

export const flowEventSchema = z.object({
  elapsed: z.string(),
  side: z.enum(['left', 'right']),
  scorer: flowPlayerSchema,
  assist: flowPlayerSchema,
  callahan: z.boolean(),
  leftScore: z.number().int(),
  rightScore: z.number().int(),
})
export type FlowEvent = z.infer<typeof flowEventSchema>

/** GET /live/{id}/matches/{matchId} */
export const liveMatchDetailSchema = z.object({ events: z.array(flowEventSchema) })
export type LiveMatchDetail = z.infer<typeof liveMatchDetailSchema>
