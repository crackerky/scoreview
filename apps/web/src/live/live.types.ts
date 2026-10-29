import { z } from 'zod'

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
  /** 組の名前(「Qualifying pool A」)。組に入っていない試合は部門名 */
  stage: z.string(),
  pool: z.boolean(),
  divisionName: z.string(),
  left: liveSideSchema,
  right: liveSideSchema,
})
export type LiveMatch = z.infer<typeof liveMatchSchema>

export const livePoolSchema = z.object({
  id: z.number().int(),
  name: z.string(),
  done: z.boolean(),
  rows: z.array(
    z.object({
      rank: z.number().int().nullable(),
      teamId: z.number().int(),
      name: z.string(),
      color: z.string(),
      points: z.number().int(),
    }),
  ),
})
export type LivePool = z.infer<typeof livePoolSchema>

export const liveDataSchema = z.object({
  tournament: z.object({ id: z.number().int(), name: z.string(), date: z.string().nullable() }),
  sponsor: z.object({ imageUrl: z.string(), linkUrl: z.string().nullable() }).nullable(),
  matches: z.array(liveMatchSchema),
  pools: z.array(livePoolSchema),
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
