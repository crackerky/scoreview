import { z } from 'zod'
import {
  bracketCellSchema,
  divisionKindSchema,
  entryKindSchema,
  genderSchema,
  groupKindSchema,
  i18nTextSchema,
  matchRowSchema,
  matchStatusSchema,
  playerRefSchema,
  sideSchema,
  statsKindSchema,
  tallySchema,
  teamBarStyleSchema,
  teamRefSchema,
  tournamentFrameSchema,
  tournamentStatusSchema,
} from './common.types'

// docs/api.md「閲覧用エンドポイント」の応答。エンドポイントごとに 1 スキーマ。

/** GET /api/v1/tournaments */
export const tournamentListResponseSchema = z.object({
  sections: z.array(
    z.object({
      status: tournamentStatusSchema,
      tournaments: z.array(z.object({ id: z.number().int(), name: i18nTextSchema })),
    }),
  ),
})
export type TournamentListResponse = z.infer<typeof tournamentListResponseSchema>

/** GET /api/v1/tournaments/{id} */
export const tournamentTopResponseSchema = z.object({
  frame: tournamentFrameSchema,
  logoUrl: z.string().nullable(),
  topHtml: i18nTextSchema,
  divisions: z.array(
    z.object({
      id: z.number().int(),
      name: i18nTextSchema,
      scheduleLinked: z.boolean(),
      standingsLinked: z.boolean(),
    }),
  ),
})
export type TournamentTopResponse = z.infer<typeof tournamentTopResponseSchema>

/** GET /api/v1/tournaments/{id}/teams */
export const teamListResponseSchema = z.object({
  frame: tournamentFrameSchema,
  divisions: z.array(
    z.object({
      id: z.number().int(),
      name: i18nTextSchema,
      kind: divisionKindSchema,
      entryKind: entryKindSchema,
      teams: z.array(
        z.object({
          id: z.number().int(),
          name: i18nTextSchema,
          barStyle: teamBarStyleSchema,
          memberCount: z.number().int().nullable(),
        }),
      ),
    }),
  ),
})
export type TeamListResponse = z.infer<typeof teamListResponseSchema>

export const scheduleRowSchema = matchRowSchema.extend({
  division: z.object({ id: z.number().int(), name: i18nTextSchema, kind: divisionKindSchema }),
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

/** GET /api/v1/divisions/{id}/standings */
export const standingsResponseSchema = z.object({
  frame: tournamentFrameSchema,
  division: z.object({ id: z.number().int(), name: i18nTextSchema }),
  standings: z.array(
    tallySchema.extend({ team: teamRefSchema, rank: z.number().int().nullable() }),
  ),
  leaders: z.array(
    z.object({
      team: teamRefSchema,
      player: playerRefSchema,
      games: z.number().int(),
      assists: z.number().int(),
      goals: z.number().int(),
    }),
  ),
  blockLeaders: z.array(
    z.object({
      team: teamRefSchema,
      player: playerRefSchema,
      games: z.number().int(),
      blocks: z.number().int(),
    }),
  ),
})
export type StandingsResponse = z.infer<typeof standingsResponseSchema>

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

/** GET /api/v1/teams/{id} */
export const teamResponseSchema = z.object({
  frame: tournamentFrameSchema,
  team: z.object({
    id: z.number().int(),
    fullName: i18nTextSchema,
    shortName: i18nTextSchema,
    comment: i18nTextSchema,
    layout: z.enum(['standard', 'roster_only']),
  }),
  record: tallySchema.extend({ rank: z.number().int().nullable() }),
  matches: z.array(matchRowSchema),
  players: z.array(playerLineSchema.extend({ games: z.number().int(), absent: z.boolean() })),
  // SOTG スコア一覧(行は matches、値は空)を出すか
  showSotg: z.boolean(),
  staff: z.array(
    z.object({
      id: z.number().int(),
      role: i18nTextSchema,
      name: i18nTextSchema,
      gender: genderSchema.nullable(),
    }),
  ),
})
export type TeamResponse = z.infer<typeof teamResponseSchema>

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

const playerWithRoleSchema = playerRefSchema.extend({ role: i18nTextSchema.nullable() })

/** GET /api/v1/players/{id}。role が非 null ならスタッフ。 */
export const playerResponseSchema = z.object({
  frame: tournamentFrameSchema,
  player: playerWithRoleSchema,
  team: teamRefSchema,
  prev: playerWithRoleSchema,
  next: playerWithRoleSchema,
  statsKind: statsKindSchema,
  stats: z.object({
    games: z.number().int(),
    assists: z.number().int(),
    goals: z.number().int(),
    callahans: z.number().int(),
    greatests: z.number().int(),
    blocks: z.number().int(),
  }),
  events: z.array(
    z.object({
      code: z.string(),
      date: z.string().nullable(),
      opponent: teamRefSchema.nullable(),
      elapsedSec: z.number().int(),
      callahan: z.boolean(),
      scorer: playerRefSchema.nullable(),
      assist: playerRefSchema.nullable(),
    }),
  ),
})
export type PlayerResponse = z.infer<typeof playerResponseSchema>

/** GET /api/v1/galleries/{id} */
export const galleryResponseSchema = z.object({
  frame: tournamentFrameSchema,
  gallery: z.object({
    id: z.number().int(),
    name: z.string(),
    photos: z.array(z.object({ thumbUrl: z.string(), fullUrl: z.string() })),
  }),
})
export type GalleryResponse = z.infer<typeof galleryResponseSchema>
