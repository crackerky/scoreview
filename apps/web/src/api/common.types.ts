import { z } from 'zod'

// docs/api.md「共通の型」の zod 表現。値がないことは null で来るので、キーの省略は許さない。

export const i18nTextSchema = z.object({ ja: z.string(), en: z.string() })
export type I18nText = z.infer<typeof i18nTextSchema>

export const tournamentStatusSchema = z.enum(['entry_open', 'entry_closed', 'finished'])
export type TournamentStatus = z.infer<typeof tournamentStatusSchema>

export const divisionKindSchema = z.enum(['men', 'women', 'mixed', 'none'])
export type DivisionKind = z.infer<typeof divisionKindSchema>

export const teamBarStyleSchema = z.enum(['default', 'mixed', 'women', 'univ', 'waku1', 'waku2'])
export type TeamBarStyle = z.infer<typeof teamBarStyleSchema>

export const entryKindSchema = z.enum(['team', 'individual'])
export type EntryKind = z.infer<typeof entryKindSchema>

export const statsKindSchema = z.enum(['ultimate', 'discgolf'])
export type StatsKind = z.infer<typeof statsKindSchema>

export const matchStatusSchema = z.enum(['scheduled', 'live', 'finished'])
export type MatchStatus = z.infer<typeof matchStatusSchema>

/** 試合の結果の種類(docs/api.md 共通の型 ResultType) */
export const resultTypeSchema = z.enum([
  'normal',
  'forfeit',
  'double_forfeit',
  'no_contest',
  'abandoned',
  'unplayed',
])
export type ResultType = z.infer<typeof resultTypeSchema>

export const groupKindSchema = z.enum(['pool', 'bracket', 'image'])
export type GroupKind = z.infer<typeof groupKindSchema>

export const sideSchema = z.enum(['left', 'right'])
export type Side = z.infer<typeof sideSchema>

export const genderSchema = z.enum(['M', 'F'])
export type Gender = z.infer<typeof genderSchema>

/** 大会ナビとサイドバーを描くための骨格。閲覧用の応答はすべてこれを含む。 */
export const tournamentFrameSchema = z.object({
  id: z.number().int(),
  name: i18nTextSchema,
  shortName: z.string(),
  labels: z.object({
    teamList: z.enum(['teams', 'entries']),
    groups: z.enum(['groups', 'results']),
  }),
  showSchedule: z.boolean(),
  divisions: z.array(
    z.object({
      id: z.number().int(),
      name: i18nTextSchema,
      // サイドバーの SCORE メニューに出すか(スケジュールのないディビジョンは出ない)
      scheduleEnabled: z.boolean(),
      groups: z.array(z.object({ id: z.number().int(), name: z.string(), linked: z.boolean() })),
    }),
  ),
  galleries: z.array(z.object({ id: z.number().int(), name: z.string() })),
  banners: z.array(z.object({ imageUrl: z.string(), linkUrl: z.string().nullable() })),
})
export type TournamentFrame = z.infer<typeof tournamentFrameSchema>

/** name は短縮名(画面の表で使う名前)。 */
export const teamRefSchema = z.object({ id: z.number().int(), name: i18nTextSchema })
export type TeamRef = z.infer<typeof teamRefSchema>

export const playerRefSchema = z.object({
  id: z.number().int(),
  number: z.string(),
  name: i18nTextSchema,
  gender: genderSchema.nullable(),
})
export type PlayerRef = z.infer<typeof playerRefSchema>

/** 日程や結果一覧の 1 行の片側。team が null のときは placeholder に未確定チームの参照が入る。 */
export const matchSideSchema = z.object({
  team: teamRefSchema.nullable(),
  placeholder: z.object({ code: z.string(), outcome: z.enum(['win', 'lose']) }).nullable(),
  score: z.number().int().nullable(),
})
export type MatchSide = z.infer<typeof matchSideSchema>

export const matchRowSchema = z.object({
  matchId: z.number().int(),
  hasDetail: z.boolean(),
  code: z.string(),
  date: z.string().nullable(),
  time: z.string().nullable(),
  // コートは数字とは限らない(参照サイトのディビジョン 467 は "A" "B")
  field: z.string().nullable(),
  status: matchStatusSchema,
  left: matchSideSchema,
  right: matchSideSchema,
})
export type MatchRow = z.infer<typeof matchRowSchema>

/** 成績表の 1 行(チームページ、ディビジョン、プール)の集計値。 */
export const tallySchema = z.object({
  games: z.number().int(),
  wins: z.number().int(),
  draws: z.number().int(),
  losses: z.number().int(),
  goalsFor: z.number().int(),
  goalsAgainst: z.number().int(),
})
export type Tally = z.infer<typeof tallySchema>

export const bracketCellSchema = z.object({
  borders: z.object({
    top: z.boolean(),
    bottom: z.boolean(),
    left: z.boolean(),
    right: z.boolean(),
  }),
  kind: z.enum([
    '',
    'team',
    'win',
    'lose',
    'game',
    'txt_yen',
    'txt_sharp',
    'txt_and',
    'rank',
    'txt_left',
    'txt_right',
  ]),
  text: z.string(),
  team: teamRefSchema.nullable(),
  matchId: z.number().int().nullable(),
  status: matchStatusSchema.nullable(),
})
export type BracketCell = z.infer<typeof bracketCellSchema>
