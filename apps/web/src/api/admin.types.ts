import { z } from 'zod'
import {
  divisionKindSchema,
  entryKindSchema,
  genderSchema,
  groupKindSchema,
  i18nTextSchema,
  matchSideSchema,
  matchStatusSchema,
  playerRefSchema,
  resultTypeSchema,
  sideSchema,
  statsKindSchema,
  teamBarStyleSchema,
  tournamentStatusSchema,
} from './common.types'

// docs/api.md「管理用エンドポイント」の応答と、管理画面のフォーム入力の zod スキーマ。
// 閲覧用(public.types.ts)とは分ける。フォームのスキーマは FormData を formToObject で平らにした
// オブジェクト(値は文字列か文字列の配列)を受け取り、api のボディの形にして返す。

const id = z.number().int()

// ===== 応答 =====

export const roleSchema = z.enum(['admin', 'staff'])
export type Role = z.infer<typeof roleSchema>

export const adminUserSchema = z.object({
  id,
  email: z.string(),
  displayName: z.string(),
  role: roleSchema,
  tournamentIds: z.array(id),
})
export type AdminUser = z.infer<typeof adminUserSchema>

/** POST /api/v1/auth/login */
export const loginResponseSchema = z.object({
  token: z.string().min(1),
  expiresAt: z.iso.datetime(),
  user: adminUserSchema,
})
export type LoginResponse = z.infer<typeof loginResponseSchema>

/** GET /api/v1/auth/me */
export const meResponseSchema = z.object({ user: adminUserSchema })

/** 204 の応答(本文なし) */
export const noContentSchema = z.undefined()

const tournamentInputShape = {
  name: i18nTextSchema,
  shortName: z.string(),
  status: tournamentStatusSchema,
  listed: z.boolean(),
  startDate: z.string().nullable(),
  logoUrl: z.string().nullable(),
  topHtml: i18nTextSchema,
  labels: z.object({
    teamList: z.enum(['teams', 'entries']),
    groups: z.enum(['groups', 'results']),
  }),
  showSchedule: z.boolean(),
  forfeitScore: z.object({ winner: z.number().int(), loser: z.number().int() }).nullable(),
}
export const adminTournamentSchema = z.object({ id, ...tournamentInputShape })
export type AdminTournament = z.infer<typeof adminTournamentSchema>
export const adminTournamentListSchema = z.object({ tournaments: z.array(adminTournamentSchema) })

export const bannerSchema = z.object({ imageUrl: z.string(), linkUrl: z.string().nullable() })
export const bannersSchema = z.object({ banners: z.array(bannerSchema) })
export type Banner = z.infer<typeof bannerSchema>

export const adminDivisionSchema = z.object({
  id,
  tournamentId: id,
  name: i18nTextSchema,
  kind: divisionKindSchema,
  sortOrder: z.number().int(),
  scheduleEnabled: z.boolean(),
  standingsPublished: z.boolean(),
  entryKind: entryKindSchema,
  statsKind: statsKindSchema,
})
export type AdminDivision = z.infer<typeof adminDivisionSchema>
export const adminDivisionListSchema = z.object({ divisions: z.array(adminDivisionSchema) })

export const placeholderSchema = z.object({ code: z.string(), outcome: z.enum(['win', 'lose']) })

export const adminGroupSchema = z.object({
  id,
  divisionId: id,
  name: z.string(),
  kind: groupKindSchema,
  sortOrder: z.number().int(),
  linked: z.boolean(),
  published: z.boolean(),
  image: z
    .object({
      url: z.string(),
      width: z.number().int().nullable(),
      height: z.number().int().nullable(),
    })
    .nullable(),
  bracket: z.object({ pattern: z.string(), ptnName: z.string() }).nullable(),
  teams: z.array(z.object({ teamId: id, rankOverride: z.number().int().nullable() })),
})
export type AdminGroup = z.infer<typeof adminGroupSchema>
export const adminGroupListSchema = z.object({ groups: z.array(adminGroupSchema) })

export const bracketTemplateListSchema = z.object({
  templates: z.array(z.object({ id: z.string(), name: i18nTextSchema, teams: z.number().int() })),
})
export type BracketTemplate = z.infer<typeof bracketTemplateListSchema>['templates'][number]

export const adminTeamSchema = z.object({
  id,
  tournamentId: id,
  divisionId: id,
  fullName: i18nTextSchema,
  shortName: i18nTextSchema,
  comment: i18nTextSchema,
  barStyle: teamBarStyleSchema,
  sortOrder: z.number().int(),
  finalRank: z.number().int().nullable(),
})
export type AdminTeam = z.infer<typeof adminTeamSchema>
export const adminTeamListSchema = z.object({ teams: z.array(adminTeamSchema) })

export const adminPlayerSchema = z.object({
  id,
  teamId: id,
  number: z.string(),
  name: i18nTextSchema,
  gender: genderSchema.nullable(),
  role: i18nTextSchema.nullable(),
})
export type AdminPlayer = z.infer<typeof adminPlayerSchema>
export const adminPlayerListSchema = z.object({ players: z.array(adminPlayerSchema) })

export const adminMatchSchema = z.object({
  id,
  divisionId: id,
  groupId: id.nullable(),
  code: z.string(),
  date: z.string().nullable(),
  time: z.string().nullable(),
  field: z.string().nullable(),
  hasDetail: z.boolean(),
  status: matchStatusSchema,
  left: matchSideSchema,
  right: matchSideSchema,
  winnerSide: sideSchema.nullable(),
  resultType: resultTypeSchema,
})
export type AdminMatch = z.infer<typeof adminMatchSchema>
export const adminMatchListSchema = z.object({ matches: z.array(adminMatchSchema) })

export const adminEventSchema = z.object({
  id,
  clientId: z.string(),
  seq: z.number().int(),
  elapsedSec: z.number().int(),
  kind: z.enum(['goal', 'timeout', 'pull', 'end']),
  side: sideSchema.nullable(),
  scorer: playerRefSchema.nullable(),
  assist: playerRefSchema.nullable(),
  callahan: z.boolean(),
  leftScore: z.number().int(),
  rightScore: z.number().int(),
})
export type AdminEvent = z.infer<typeof adminEventSchema>

export const playerStatSchema = z.object({
  playerId: id,
  side: sideSchema,
  blocks: z.number().int().min(0),
  greatests: z.number().int().min(0),
})
export type PlayerStat = z.infer<typeof playerStatSchema>

/** GET /api/v1/admin/matches/{id} とスコア入力の各操作の応答 */
export const forfeitScoreSchema = z.object({ winner: z.number().int(), loser: z.number().int() })
export type ForfeitScore = z.infer<typeof forfeitScoreSchema>

export const scorerClaimSchema = z.object({
  userId: id,
  name: z.string(),
  deviceId: z.string(),
  expiresAt: z.iso.datetime(),
})
export type ScorerClaim = z.infer<typeof scorerClaimSchema>

export const concurrentRecordSchema = z.object({
  kind: z.enum(['goal', 'pull']),
  firstClientId: z.string(),
  secondClientId: z.string(),
  firstUser: z.string(),
  secondUser: z.string(),
  secondsApart: z.number().int().min(0),
})
export type ConcurrentRecord = z.infer<typeof concurrentRecordSchema>

export const matchDetailSchema = z.object({
  match: adminMatchSchema,
  context: z.object({
    tournamentId: id,
    tournamentName: z.string(),
    divisionName: z.string(),
    groupName: z.string().nullable(),
  }),
  inBracket: z.boolean(),
  forfeitScore: forfeitScoreSchema.nullable(),
  startedAt: z.iso.datetime().nullable(),
  events: z.array(adminEventSchema),
  playerStats: z.array(playerStatSchema),
  rosters: z.object({ left: z.array(adminPlayerSchema), right: z.array(adminPlayerSchema) }),
  /** 試合の記録担当(いま記録している端末。期限内のもの) */
  scorer: scorerClaimSchema.nullable(),
  /** 別の記録係がほぼ同時に入れた同じ種類の記録の組(二重記録の疑い) */
  concurrentRecords: z.array(concurrentRecordSchema),
  /** ディビジョンの時間割の 1 試合の枠の長さ(分)。設定が無ければ null */
  matchMinutes: z.number().int().positive().nullable(),
})
export type MatchDetail = z.infer<typeof matchDetailSchema>

/** POST /api/v1/admin/matches/{id}/scorer の応答 */
export const scorerResultSchema = z.object({ scorer: scorerClaimSchema, mine: z.boolean() })
export type ScorerResult = z.infer<typeof scorerResultSchema>

const auditMatchStateSchema = z.object({
  status: matchStatusSchema,
  resultType: resultTypeSchema,
  leftScore: z.number().int().nullable(),
  rightScore: z.number().int().nullable(),
  winnerSide: sideSchema.nullable(),
})
const auditPlayerSchema = z.object({ id, number: z.string(), name: z.string() })
const auditSnapshotSchema = z.object({
  match: auditMatchStateSchema.nullable(),
  event: z
    .object({
      clientId: z.string(),
      kind: z.enum(['goal', 'timeout', 'pull', 'end']),
      side: sideSchema.nullable(),
      elapsedSec: z.number().int(),
      scorer: auditPlayerSchema.nullable(),
      assist: auditPlayerSchema.nullable(),
      callahan: z.boolean(),
    })
    .nullable(),
  finalRank: z.number().int().nullable(),
  /** 操作の補足(「得点経過(9-8)と違う最終スコア(11-9)で本部が確定」) */
  note: z.string().nullable(),
})
export type AuditSnapshot = z.infer<typeof auditSnapshotSchema>

export const auditEntrySchema = z.object({
  id,
  at: z.iso.datetime(),
  user: z.object({ id, name: z.string(), role: roleSchema.nullable() }).nullable(),
  action: z.enum([
    'event.add',
    'event.remove',
    'event.update',
    'event.side',
    'match.status',
    'match.result',
    'team.final_rank',
    'match.align_teams',
    'match.withdraw',
  ]),
  matchId: id.nullable(),
  teamId: id.nullable(),
  before: auditSnapshotSchema,
  after: auditSnapshotSchema,
})
export type AuditEntry = z.infer<typeof auditEntrySchema>
/** GET /api/v1/admin/matches/{id}/audit */
export const auditListSchema = z.object({ entries: z.array(auditEntrySchema) })

export const assignmentSchema = z.object({
  id,
  userId: id,
  userName: z.string(),
  matchId: id.nullable(),
  matchCode: z.string().nullable(),
  field: z.string().nullable(),
  date: z.string().nullable(),
  from: z.string().nullable(),
  to: z.string().nullable(),
})
export type Assignment = z.infer<typeof assignmentSchema>
export const assignmentListSchema = z.object({ assignments: z.array(assignmentSchema) })

/** GET /api/v1/admin/me/matches */
export const assignedMatchListSchema = z.object({
  matches: z.array(
    adminMatchSchema.extend({
      tournamentId: id,
      tournamentName: z.string(),
      divisionName: z.string(),
    }),
  ),
})
export type AssignedMatch = z.infer<typeof assignedMatchListSchema>['matches'][number]

export const adminGallerySchema = z.object({
  id,
  tournamentId: id,
  name: z.string(),
  sortOrder: z.number().int(),
  photos: z.array(z.object({ thumbUrl: z.string(), fullUrl: z.string() })),
})
export type AdminGallery = z.infer<typeof adminGallerySchema>
export const adminGalleryListSchema = z.object({ galleries: z.array(adminGallerySchema) })

export const userListSchema = z.object({ users: z.array(adminUserSchema) })

// ===== フォーム入力 =====

/** 空欄(またはキーなし)を null にする文字列 */
const optionalText = z.preprocess(
  (value) => (typeof value === 'string' && value.trim() !== '' ? value.trim() : null),
  z.string().nullable(),
)
/** 前後の空白を除いた必須の文字列 */
const requiredText = z.string().trim().min(1, '入力してください')
/** 加工せずに保存する必須の文字列(チームの正式名。docs/line-integration.md) */
const rawRequiredText = z.string().refine((value) => value.trim() !== '', '入力してください')
/** 空欄なら空文字(英語名など) */
const text = z.preprocess((value) => (typeof value === 'string' ? value.trim() : ''), z.string())
/** チェックボックス(チェックなしはキーごと送られない) */
const checkbox = z.preprocess((value) => value === 'on' || value === 'true', z.boolean())
/** 空欄なら null の整数 */
const optionalInt = z.preprocess(
  (value) => (typeof value === 'string' && value.trim() !== '' ? Number(value) : null),
  z.number().int().nullable(),
)
// 記録の数値の上限(api の internal/domain/limits.go と同じ値。DB の integer を超えさせない)
/** 得点経過の経過秒の上限(1 日) */
export const MAX_ELAPSED_SEC = 24 * 60 * 60
/** 試合のスコアと不戦勝の規定スコアの上限 */
export const MAX_SCORE = 999
/** 1 試合の選手のブロック数とグレイテスト数の上限 */
export const MAX_PLAYER_STAT = 999

/** 空欄なら null のスコア(0 以上 MAX_SCORE 以下) */
const optionalScore = optionalInt.pipe(
  z.number().int().min(0).max(MAX_SCORE, `${MAX_SCORE} 以下で入力してください`).nullable(),
)
/** 空欄なら 0 の整数 */
const intOrZero = z.preprocess(
  (value) => (typeof value === 'string' && value.trim() !== '' ? Number(value) : 0),
  z.number().int(),
)
const formId = z.coerce.number().int().positive()
const urlText = z
  .string()
  .trim()
  .refine(
    (value) => /^https?:\/\/[^/]/.test(value) || (value.startsWith('/') && !value.startsWith('//')),
    'http(s):// で始まる URL か / で始まるパスにしてください',
  )
const optionalUrl = z.preprocess(
  (value) => (typeof value === 'string' && value.trim() !== '' ? value.trim() : null),
  urlText.nullable(),
)

export const loginFormSchema = z.object({
  email: requiredText,
  password: z.string().min(1, '入力してください'),
})

export const tournamentFormSchema = z
  .object({
    nameJa: requiredText,
    nameEn: text,
    shortName: text,
    status: tournamentStatusSchema,
    listed: checkbox,
    startDate: z.preprocess(
      (value) => (value === '' || value === undefined ? null : value),
      z.iso.date('YYYY-MM-DD で入力してください').nullable(),
    ),
    logoUrl: optionalUrl,
    topHtmlJa: z.string().default(''),
    topHtmlEn: z.string().default(''),
    teamListLabel: z.enum(['teams', 'entries']),
    groupLabel: z.enum(['groups', 'results']),
    showSchedule: checkbox,
    forfeitWinner: optionalScore,
    forfeitLoser: optionalScore,
  })
  .refine((form) => (form.forfeitWinner === null) === (form.forfeitLoser === null), {
    path: ['forfeitLoser'],
    message: '不戦勝のスコアは勝ちと負けの両方を入れるか、両方空欄にしてください',
  })
  .refine(
    (form) =>
      form.forfeitWinner === null ||
      form.forfeitLoser === null ||
      (form.forfeitLoser >= 0 && form.forfeitWinner > form.forfeitLoser),
    { path: ['forfeitWinner'], message: '勝ちのスコアを負けのスコアより大きくしてください' },
  )
  .transform((form) => ({
    name: { ja: form.nameJa, en: form.nameEn },
    shortName: form.shortName,
    status: form.status,
    listed: form.listed,
    startDate: form.startDate,
    logoUrl: form.logoUrl,
    topHtml: { ja: form.topHtmlJa, en: form.topHtmlEn },
    labels: { teamList: form.teamListLabel, groups: form.groupLabel },
    showSchedule: form.showSchedule,
    forfeitScore:
      form.forfeitWinner === null || form.forfeitLoser === null
        ? null
        : { winner: form.forfeitWinner, loser: form.forfeitLoser },
  }))
export type TournamentInput = z.output<typeof tournamentFormSchema>

/** 1 行に「画像 URL [リンク URL]」を空白区切りで書くバナーの一覧 */
export const bannersFormSchema = z
  .object({ banners: z.string().default('') })
  .transform((form) => linesOf(form.banners))
  .pipe(
    z.array(
      z
        .array(z.string())
        .max(2, '1 行に URL は 2 つまでです')
        .transform(([imageUrl = '', linkUrl]) => ({ imageUrl, linkUrl: linkUrl ?? null }))
        .pipe(z.object({ imageUrl: urlText, linkUrl: urlText.nullable() })),
    ),
  )
  .transform((banners) => ({ banners }))

/** 1 行に「サムネイル URL 拡大 URL」を書く写真の一覧 */
export const photosFormSchema = z
  .object({ photos: z.string().default('') })
  .transform((form) => linesOf(form.photos))
  .pipe(
    z.array(
      z
        .array(z.string())
        .length(2, '1 行にサムネイルと拡大画像の URL を書いてください')
        .transform(([thumbUrl = '', fullUrl = '']) => ({ thumbUrl, fullUrl }))
        .pipe(z.object({ thumbUrl: urlText, fullUrl: urlText })),
    ),
  )
  .transform((photos) => ({ photos }))

/** 空行を除いた各行を空白で分ける */
export function linesOf(value: string): string[][] {
  return value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line !== '')
    .map((line) => line.split(/\s+/))
}

export const divisionFormSchema = z
  .object({
    nameJa: requiredText,
    nameEn: text,
    kind: divisionKindSchema,
    sortOrder: intOrZero,
    scheduleEnabled: checkbox,
    standingsPublished: checkbox,
    entryKind: entryKindSchema,
    statsKind: statsKindSchema,
  })
  .transform(({ nameJa, nameEn, ...rest }) => ({ name: { ja: nameJa, en: nameEn }, ...rest }))

export const groupFormSchema = z
  .object({
    name: requiredText,
    kind: groupKindSchema,
    sortOrder: intOrZero,
    linked: checkbox,
    published: checkbox,
    imageUrl: optionalUrl,
    imageWidth: optionalInt,
    imageHeight: optionalInt,
  })
  .refine((form) => form.kind !== 'image' || form.imageUrl !== null, {
    path: ['imageUrl'],
    message: '画像のグループには画像 URL が必要です',
  })
  .transform(({ imageUrl, imageWidth, imageHeight, ...rest }) => ({
    ...rest,
    image: imageUrl === null ? null : { url: imageUrl, width: imageWidth, height: imageHeight },
  }))

/**
 * グループのチーム。各チームの「順」欄(team-{id})に数字があれば、その順でグループに入れる。
 * 空欄のチームは入れない。順位の上書きは rank-{id}。
 */
export function groupTeamsFromForm(form: Record<string, unknown>) {
  const entries: Array<{ order: number; teamId: number; rankOverride: number | null }> = []
  for (const [key, value] of Object.entries(form)) {
    const match = /^team-(\d+)$/.exec(key)
    if (!match?.[1] || typeof value !== 'string' || value.trim() === '') {
      continue
    }
    const teamId = Number(match[1])
    const rank = form[`rank-${teamId}`]
    entries.push({
      order: z.coerce.number().int().parse(value),
      teamId,
      rankOverride: optionalInt.parse(typeof rank === 'string' ? rank : ''),
    })
  }
  entries.sort((a, b) => a.order - b.order || a.teamId - b.teamId)
  return { teams: entries.map(({ teamId, rankOverride }) => ({ teamId, rankOverride })) }
}

/**
 * 「チーム ID」か「A1-win」の形の未確定チームの参照(トーナメント表の枠、試合の対戦チーム)。
 * 空欄は null(未定)。
 */
export const sideChoiceSchema = z.preprocess(
  (value) => (typeof value === 'string' ? value.trim() : ''),
  z.string().transform((value, ctx) => {
    if (value === '') {
      return { teamId: null, placeholder: null }
    }
    if (/^\d+$/.test(value)) {
      return { teamId: Number(value), placeholder: null }
    }
    const ref = /^(.+)-(win|lose)$/.exec(value)
    if (!ref?.[1] || !ref[2]) {
      ctx.addIssue({ code: 'custom', message: 'チームか「A1-win」の形で指定してください' })
      return z.NEVER
    }
    return { teamId: null, placeholder: placeholderSchema.parse({ code: ref[1], outcome: ref[2] }) }
  }),
)

export const bracketFormSchema = z
  .object({ template: requiredText, codePrefix: requiredText, seedCount: formId })
  .loose()
  .transform((form, ctx) => {
    const seeds = []
    for (let index = 0; index < form.seedCount; index++) {
      const seed = sideChoiceSchema.parse(form[`seed-${index}`])
      if (seed.teamId === null && seed.placeholder === null) {
        ctx.addIssue({ code: 'custom', message: `${index + 1} 番目の枠を選んでください` })
        return z.NEVER
      }
      seeds.push(seed)
    }
    return { template: form.template, codePrefix: form.codePrefix, seeds }
  })

export const teamFormSchema = z
  .object({
    fullNameJa: rawRequiredText,
    fullNameEn: z.string().default(''),
    shortNameJa: requiredText,
    shortNameEn: text,
    commentJa: z.string().default(''),
    commentEn: z.string().default(''),
    barStyle: teamBarStyleSchema,
    sortOrder: intOrZero,
    finalRank: optionalInt.pipe(z.number().int().positive().nullable()),
  })
  .transform((form) => ({
    fullName: { ja: form.fullNameJa, en: form.fullNameEn },
    shortName: { ja: form.shortNameJa, en: form.shortNameEn },
    comment: { ja: form.commentJa, en: form.commentEn },
    barStyle: form.barStyle,
    sortOrder: form.sortOrder,
    finalRank: form.finalRank,
  }))

export const playerFormSchema = z
  .object({
    number: text,
    nameJa: requiredText,
    nameEn: text,
    gender: z.preprocess(
      (value) => (value === '' || value === undefined ? null : value),
      genderSchema.nullable(),
    ),
    roleJa: optionalText,
    roleEn: text,
  })
  .transform((form) => ({
    number: form.number,
    name: { ja: form.nameJa, en: form.nameEn },
    gender: form.gender,
    role: form.roleJa === null ? null : { ja: form.roleJa, en: form.roleEn },
  }))

/** 最終順位。各チームの rank-{id} 欄(空欄は順位なし) */
export function finalRanksFromForm(form: Record<string, unknown>) {
  const ranks: Array<{ teamId: number; rank: number | null }> = []
  for (const [key, value] of Object.entries(form)) {
    const match = /^rank-(\d+)$/.exec(key)
    if (match?.[1]) {
      const rank = optionalInt.pipe(z.number().int().positive().nullable()).parse(value)
      ranks.push({ teamId: Number(match[1]), rank })
    }
  }
  return ranks
}

export const matchFormSchema = z
  .object({
    groupId: optionalInt,
    code: text,
    date: z.preprocess(
      (value) => (value === '' || value === undefined ? null : value),
      z.iso.date('YYYY-MM-DD で入力してください').nullable(),
    ),
    time: z.preprocess(
      (value) => (value === '' || value === undefined ? null : value),
      z
        .string()
        .regex(/^([01]\d|2[0-3]):[0-5]\d$/, 'HH:MM で入力してください')
        .nullable(),
    ),
    field: optionalText,
    hasDetail: checkbox,
    left: sideChoiceSchema,
    right: sideChoiceSchema,
  })
  .refine((form) => form.left.teamId === null || form.left.teamId !== form.right.teamId, {
    path: ['right'],
    message: '左右に同じチームは選べません',
  })

/** "mm:ss" か分だけの数字を秒にする。空欄は null(api が最初のイベントからの経過で決める) */
export const elapsedSchema = z.preprocess(
  (value) => (typeof value === 'string' ? value.trim() : ''),
  z.string().transform((value, ctx) => {
    if (value === '') {
      return null
    }
    const match = /^(\d{1,3})(?::([0-5]\d))?$/.exec(value)
    if (!match?.[1]) {
      ctx.addIssue({ code: 'custom', message: '経過は「分:秒」か分で入力してください' })
      return z.NEVER
    }
    return Number(match[1]) * 60 + Number(match[2] ?? 0)
  }),
)

const optionalId = z.preprocess(
  (value) => (typeof value === 'string' && value !== '' ? Number(value) : null),
  z.number().int().positive().nullable(),
)
const optionalSide = z.preprocess(
  (value) => (value === '' || value === undefined ? null : value),
  sideSchema.nullable(),
)

export const eventFormSchema = z
  .object({
    kind: z.enum(['goal', 'timeout', 'pull', 'end']),
    side: optionalSide,
    elapsed: elapsedSchema,
    scorerId: optionalId,
    assistId: optionalId,
    callahan: checkbox,
    winnerSide: optionalSide,
  })
  .refine((form) => form.kind === 'end' || form.side !== null, {
    path: ['side'],
    message: '左右を選んでください',
  })
  .refine((form) => !form.callahan || form.assistId === null, {
    path: ['assistId'],
    message: 'キャラハンゴールにアシストは付けられません',
  })
  .transform((form) => ({
    kind: form.kind,
    side: form.kind === 'end' ? null : form.side,
    elapsedSec: form.elapsed,
    scorerId: form.kind === 'goal' ? form.scorerId : null,
    assistId: form.kind === 'goal' ? form.assistId : null,
    callahan: form.kind === 'goal' && form.callahan,
    winnerSide: form.kind === 'end' ? form.winnerSide : null,
  }))
export type EventInput = z.output<typeof eventFormSchema>

export const statusFormSchema = z
  .object({
    status: matchStatusSchema,
    winnerSide: optionalSide,
    leftScore: optionalScore,
    rightScore: optionalScore,
  })
  .refine((form) => (form.leftScore === null) === (form.rightScore === null), {
    path: ['rightScore'],
    message: 'スコアは左右とも入れるか、両方空欄にしてください',
  })
  .transform((form) => ({
    status: form.status,
    winnerSide: form.status === 'finished' ? form.winnerSide : null,
    score:
      form.leftScore === null || form.rightScore === null
        ? null
        : { left: form.leftScore, right: form.rightScore },
  }))

/**
 * 試合ごとの選手記録。各選手の欄は played-{side}-{id}(出場)、blocks-{side}-{id}、
 * greatests-{side}-{id}。出場にチェックがあるか、数が 1 以上の選手だけを送る(行があることが出場)。
 */
export function playerStatsFromForm(form: Record<string, unknown>) {
  const count = z.preprocess(
    (value) => (typeof value === 'string' && value.trim() !== '' ? Number(value) : 0),
    z
      .number()
      .int()
      .min(0, '0 以上で入力してください')
      .max(MAX_PLAYER_STAT, `${MAX_PLAYER_STAT} 以下で入力してください`),
  )
  const stats: PlayerStat[] = []
  const seen = new Set<string>()
  for (const key of Object.keys(form)) {
    const match = /^(?:played|blocks|greatests)-(left|right)-(\d+)$/.exec(key)
    if (!match?.[1] || !match[2] || seen.has(`${match[1]}-${match[2]}`)) {
      continue
    }
    const side = sideSchema.parse(match[1])
    const suffix = `${side}-${match[2]}`
    seen.add(suffix)
    const blocks = count.parse(form[`blocks-${suffix}`])
    const greatests = count.parse(form[`greatests-${suffix}`])
    if (checkbox.parse(form[`played-${suffix}`]) || blocks > 0 || greatests > 0) {
      stats.push({ playerId: Number(match[2]), side, blocks, greatests })
    }
  }
  return { stats }
}

export const galleryFormSchema = z.object({ name: requiredText, sortOrder: intOrZero })

export const userFormSchema = z.object({
  email: requiredText.pipe(z.email('メールアドレスの形にしてください')),
  displayName: text,
  role: roleSchema,
  password: z.preprocess(
    (value) => (value === '' ? undefined : value),
    z.string().min(12, 'パスワードは 12 文字以上にしてください').max(256).optional(),
  ),
})

/** 本人によるパスワードの変更。新しいパスワードは確認の欄と一致すること(打ち間違いで締め出されない) */
export const passwordChangeFormSchema = z
  .object({
    currentPassword: z.string().min(1, '今のパスワードを入力してください'),
    newPassword: z.string().min(12, 'パスワードは 12 文字以上にしてください').max(256),
    newPasswordConfirm: z.string(),
  })
  .refine((form) => form.newPassword === form.newPasswordConfirm, {
    path: ['newPasswordConfirm'],
    message: '新しいパスワードと確認が一致しません',
  })
  .transform((form) => ({ currentPassword: form.currentPassword, newPassword: form.newPassword }))
export type PasswordChangeInput = z.output<typeof passwordChangeFormSchema>

/** ユーザーの作成(パスワードは必須) */
export const userCreateFormSchema = userFormSchema.extend({
  password: z.string().min(12, 'パスワードは 12 文字以上にしてください').max(256),
})

export const userTournamentsFormSchema = z
  .object({ tournamentIds: z.union([z.string(), z.array(z.string())]).default([]) })
  .transform((form) => ({
    tournamentIds: (Array.isArray(form.tournamentIds) ? form.tournamentIds : [form.tournamentIds])
      .filter((value) => value !== '')
      .map((value) => formId.parse(value)),
  }))

const optionalClock = z.preprocess(
  (value) => (value === '' || value === undefined ? null : value),
  z
    .string()
    .regex(/^([01]\d|2[0-3]):[0-5]\d$/, 'HH:MM で入力してください')
    .nullable(),
)

/**
 * 記録係の割り当て(docs/api.md「記録係の割り当て」)。試合を選べば試合だけ、選ばなければコート、日付、
 * 時間帯の範囲(空欄はその条件なし)。
 */
export const assignmentFormSchema = z
  .object({
    userId: formId,
    matchId: optionalInt,
    field: optionalText,
    date: z.preprocess(
      (value) => (value === '' || value === undefined ? null : value),
      z.iso.date('YYYY-MM-DD で入力してください').nullable(),
    ),
    from: optionalClock,
    to: optionalClock,
  })
  .refine((form) => form.from === null || form.to === null || form.from < form.to, {
    path: ['to'],
    message: '終わりの時刻を始まりより後にしてください',
  })
  .transform((form) =>
    form.matchId === null
      ? { ...form, matchId: null }
      : {
          userId: form.userId,
          matchId: form.matchId,
          field: null,
          date: null,
          from: null,
          to: null,
        },
  )
