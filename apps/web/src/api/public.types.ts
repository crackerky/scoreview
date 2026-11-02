import { z } from 'zod'
import {
  bracketCellSchema,
  divisionKindSchema,
  groupKindSchema,
  i18nTextSchema,
  matchRowSchema,
  matchStatusSchema,
  playerRefSchema,
  sideSchema,
  tallySchema,
  teamRefSchema,
  tournamentFrameSchema,
} from './common.types'

// docs/api.md「閲覧用エンドポイント」の応答。エンドポイントごとに 1 スキーマ。

/** GET /api/v1/tournaments の区分。開催中は api が試合と日付から決める(docs/api.md) */
export const listSectionSchema = z.enum(['in_progress', 'entry_open', 'entry_closed', 'finished'])
export type ListSection = z.infer<typeof listSectionSchema>

export const tournamentSummarySchema = z.object({
  id: z.number().int(),
  name: i18nTextSchema,
  startDate: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/)
    .nullable(),
  teamCount: z.number().int().nonnegative(),
})
export type TournamentSummary = z.infer<typeof tournamentSummarySchema>

/** GET /api/v1/tournaments */
export const tournamentListResponseSchema = z.object({
  sections: z.array(
    z.object({ section: listSectionSchema, tournaments: z.array(tournamentSummarySchema) }),
  ),
})
export type TournamentListResponse = z.infer<typeof tournamentListResponseSchema>

export const scheduleRowSchema = matchRowSchema.extend({
  division: z.object({ id: z.number().int(), name: i18nTextSchema, kind: divisionKindSchema }),
  /** 何を決める試合か(「1〜4位 準決勝」「3位決定戦」)。形を決めていない部門は null */
  stage: z.string().nullable(),
})
export type ScheduleRow = z.infer<typeof scheduleRowSchema>

/** GET /api/v1/tournaments/{id}/schedule と GET /api/v1/divisions/{id}/schedule */
export const scheduleResponseSchema = z.object({
  frame: tournamentFrameSchema,
  scope: z.enum(['tournament', 'division']),
  division: z.object({ id: z.number().int(), name: i18nTextSchema }).nullable(),
  filters: z.object({
    dates: z.array(z.string()),
    teams: z.array(z.object({ value: z.string(), name: i18nTextSchema })),
    // その範囲の試合で使われたコートの異なり値(自然順)。参照サイトの select と同じ
    fields: z.array(z.string()),
    selected: z.object({ ymd: z.string(), team: z.string(), field: z.string() }),
  }),
  sort: z.object({ key: z.string(), dir: z.enum(['asc', 'desc']) }).nullable(),
  rows: z.array(scheduleRowSchema),
})
export type ScheduleResponse = z.infer<typeof scheduleResponseSchema>

export const poolMatrixCellSchema = z.discriminatedUnion('kind', [
  z.object({ kind: z.enum(['self', 'lower', 'empty']) }),
  z.object({
    kind: z.literal('match'),
    matchId: z.number().int(),
    status: matchStatusSchema,
    text: z.string(),
  }),
])
export type PoolMatrixCell = z.infer<typeof poolMatrixCellSchema>

/** GET /api/v1/groups/{id}。kind に応じて pool / bracket / image のどれか 1 つが非 null。 */
export const groupResponseSchema = z.object({
  frame: tournamentFrameSchema,
  group: z.object({
    id: z.number().int(),
    name: z.string(),
    kind: groupKindSchema,
    division: z.object({ id: z.number().int(), name: i18nTextSchema }),
  }),
  pool: z
    .object({
      teams: z.array(teamRefSchema),
      matrix: z.array(z.array(poolMatrixCellSchema)),
      results: z.array(
        tallySchema.extend({
          team: teamRefSchema,
          points: z.number().int(),
          rank: z.number().int().nullable(),
        }),
      ),
      // 本サイト独自の補足(docs/requirements.md「意図的な差分」)。順位の根拠と、進行中の勝ち上がり条件
      notes: z.object({
        reasons: z.array(z.object({ team: teamRefSchema, text: z.string() })),
        conditions: z.array(z.object({ team: teamRefSchema, text: z.string() })),
      }),
    })
    .nullable(),
  bracket: z
    .object({
      pattern: z.string(),
      ptnName: z.string(),
      columns: z.number().int(),
      rows: z.array(z.array(bracketCellSchema)),
    })
    .nullable(),
  image: z
    .object({
      url: z.string(),
      width: z.number().int().nullable(),
      height: z.number().int().nullable(),
    })
    .nullable(),
  matches: z.array(matchRowSchema),
})
export type GroupResponse = z.infer<typeof groupResponseSchema>

const playerLineSchema = z.object({
  player: playerRefSchema,
  assists: z.number().int(),
  goals: z.number().int(),
  blocks: z.number().int(),
})

const matchTeamStatsSchema = z.object({
  goals: z.number().int(),
  fromOffense: z.number().int(),
  fromDefense: z.number().int(),
  fromTurnover: z.number().int(),
  timeouts: z.number().int(),
})

// fullName はチームの正式名(<title> に使う)。team が null なら null
const matchPageSideSchema = z.object({
  team: teamRefSchema.nullable(),
  fullName: i18nTextSchema.nullable(),
  score: z.number().int().nullable(),
})

/** GET /api/v1/matches/{id}。events は古い順(画面では web が新しい順に並べ替える)。 */
export const matchResponseSchema = z.object({
  frame: tournamentFrameSchema,
  match: z.object({
    id: z.number().int(),
    code: z.string(),
    groupName: z.string().nullable(),
    date: z.string().nullable(),
    time: z.string().nullable(),
    field: z.string().nullable(),
    status: matchStatusSchema,
    left: matchPageSideSchema,
    right: matchPageSideSchema,
    // スコアボード下の行の中央の備考と、Block 列・統計表を出さない試合(clone-spec D 2.2、2.4、2.5)
    note: z.string(),
    hideBlocks: z.boolean(),
    hideStats: z.boolean(),
    // 通常以外の結果(不戦勝など)をスコアボードに出す(本サイト独自)
    resultType: z.enum([
      'normal',
      'forfeit',
      'double_forfeit',
      'no_contest',
      'abandoned',
      'unplayed',
    ]),
    winnerSide: sideSchema.nullable(),
    breadcrumb: z.object({
      divisionId: z.number().int(),
      divisionName: i18nTextSchema,
      groupId: z.number().int().nullable(),
    }),
  }),
  events: z.array(
    z.object({
      elapsedSec: z.number().int(),
      kind: z.enum(['goal', 'timeout', 'pull', 'end']),
      // 参照サイトの「キャラハンゴール」。kind が goal のときだけ true になる
      callahan: z.boolean(),
      side: sideSchema.nullable(),
      scorer: playerRefSchema.nullable(),
      assist: playerRefSchema.nullable(),
      leftScore: z.number().int(),
      rightScore: z.number().int(),
    }),
  ),
  lineups: z.object({ left: z.array(playerLineSchema), right: z.array(playerLineSchema) }),
  stats: z.object({ left: matchTeamStatsSchema, right: matchTeamStatsSchema }),
})
export type MatchResponse = z.infer<typeof matchResponseSchema>

/** 観戦アプリの「表で見る」の見せ方(docs/api.md「GET /api/v1/tournaments/{id}/live-tables」) */
export const liveTablesConfigSchema = z.object({
  shown: z.boolean(),
  poolShown: z.boolean(),
  bracketShown: z.boolean(),
  columns: z.object({
    wdl: z.boolean(),
    goals: z.boolean(),
    diff: z.boolean(),
    points: z.boolean(),
    rank: z.boolean(),
  }),
  marks: z.enum(['maru', 'wdl', 'kanji']),
  order: z.enum(['entry', 'rank']),
  live: z.enum(['score', 'blank']),
  highlight: z.boolean(),
})
export type LiveTablesConfig = z.infer<typeof liveTablesConfigSchema>

const liveBracketSideSchema = z.object({
  from: z.object({ code: z.string(), outcome: z.enum(['win', 'lose']) }).nullable(),
  seed: z.string().nullable(),
})

/** GET /api/v1/tournaments/{id}/live-tables */
export const liveTablesResponseSchema = z.object({
  config: liveTablesConfigSchema,
  divisions: z.array(
    z.object({
      id: z.number().int(),
      pools: z.array(z.object({ groupId: z.number().int(), advance: z.number().int() })),
      brackets: z.array(
        z.object({
          groupId: z.number().int(),
          name: z.string(),
          matches: z.array(
            z.object({
              code: z.string(),
              round: z.number().int(),
              winnerPlace: z.number().int(),
              loserPlace: z.number().int(),
              left: liveBracketSideSchema,
              right: liveBracketSideSchema,
            }),
          ),
        }),
      ),
    }),
  ),
})
export type LiveTablesResponse = z.infer<typeof liveTablesResponseSchema>
