'use server'

import { revalidatePath } from 'next/cache'
import { z } from 'zod'
import { field, idField, inputError, runAction } from '@/admin/action'
import { operatorErrorText } from '@/admin/errorText.utils'
import { readSessionToken } from '@/admin/session'
import { players, teams } from '@/api/admin'
import { ApiError } from '@/api/client'
import { genderSchema } from '@/api/common.types'
import { formats, type ProposalInput } from '@/api/format'
import type { FormatInput, FormatPreview, Proposal } from '@/api/format.types'

// 大会の形(ウィザード)と本部の進行画面の Server Actions。ウィザードは画面を遷移せずに結果を
// 受け取るので、値を返す action にする。進行画面の操作はフォームの送信で、通知つきで元の画面へ戻す。

export type ActionResult<T> = { ok: true; data: T } | { ok: false; error: string }

async function withToken<T>(run: (token: string) => Promise<T>): Promise<ActionResult<T>> {
  const token = await readSessionToken()
  if (token === null) {
    return { ok: false, error: 'ログインが切れました。ログインし直してください' }
  }
  try {
    return { ok: true, data: await run(token) }
  } catch (error) {
    // 入力の誤りと状態の衝突は画面に出す。それ以外(500、通信失敗)はエラー境界へ流す
    if (error instanceof ApiError && (error.status < 500 || error.code === 'unavailable')) {
      return { ok: false, error: operatorErrorText(error.code, error.message) }
    }
    throw error
  }
}

export async function proposeFormatsAction(
  divisionId: number,
  input: ProposalInput,
): Promise<ActionResult<Proposal[]>> {
  return withToken(async (token) => (await formats.propose(token, divisionId, input)).proposals)
}

export async function previewFormatAction(
  divisionId: number,
  input: FormatInput,
): Promise<ActionResult<FormatPreview>> {
  return withToken((token) => formats.preview(token, divisionId, input))
}

export async function applyFormatAction(
  divisionId: number,
  input: FormatInput,
): Promise<ActionResult<FormatPreview>> {
  const result = await withToken((token) => formats.apply(token, divisionId, input))
  revalidatePath('/admin', 'layout')
  return result
}

const teamRowsSchema = z
  .array(z.object({ fullName: z.string().trim().min(1), shortName: z.string().trim().min(1) }))
  .max(512)
const playerRowsSchema = z
  .array(
    z.object({
      teamId: z.number().int().positive(),
      number: z.string().trim(),
      name: z.string().trim().min(1),
      gender: genderSchema.nullable(),
    }),
  )
  .max(4096)

/** 貼り付けのプレビュー(RosterImport)で確かめた行を読む。欄 rows に JSON で入る */
function rosterRows<T>(form: Record<string, string | string[]>, schema: z.ZodType<T>): T {
  let parsed: unknown
  try {
    parsed = JSON.parse(field(form, 'rows') || '[]')
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw inputError('名簿を読めませんでした。貼り付け直してください')
    }
    throw error
  }
  return schema.parse(parsed)
}

/** 名簿の貼り付けからチームをまとめて登録する(行の順がシード順の初期値になる) */
export async function createTeamsFromRosterAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const divisionId = idField(form, 'divisionId')
      const start = Number(field(form, 'startOrder')) || 0
      const rows = rosterRows(form, teamRowsSchema)
      if (rows.length === 0) {
        throw inputError('名簿が空です')
      }
      for (const [index, row] of rows.entries()) {
        await teams.create(token, divisionId, {
          fullName: { ja: row.fullName, en: '' },
          shortName: { ja: row.shortName, en: '' },
          sortOrder: start + index,
        })
      }
      return undefined
    },
    'チームを登録しました',
  )
}

/** 選手名簿の貼り付けから選手をまとめて登録する(チームは貼り付けのチーム名で見つけたもの) */
export async function createPlayersFromRosterAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const rows = rosterRows(form, playerRowsSchema)
      if (rows.length === 0) {
        throw inputError('登録できる選手がいません(チーム名の列を確かめてください)')
      }
      for (const row of rows) {
        await players.create(token, row.teamId, {
          number: row.number,
          name: { ja: row.name, en: '' },
          gender: row.gender,
          role: null,
        })
      }
      return undefined
    },
    '選手を登録しました',
  )
}

/** 試合の無い部門のチームをまとめて削除する(名簿の貼り間違いのやり直し) */
export async function deleteDivisionTeamsAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await teams.removeAll(token, idField(form, 'divisionId'))
      return undefined
    },
    'チームをまとめて削除しました',
  )
}

export async function confirmGroupAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await formats.confirmGroup(token, idField(form, 'groupId'), form.unplayed === 'on')
      return undefined
    },
    '確定しました。次の組み合わせに反映しました',
  )
}

export async function confirmStageAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await formats.confirmStage(token, idField(form, 'stageId'), form.unplayed === 'on')
      return undefined
    },
    '段を確定しました',
  )
}

/** 未決の同順位を、上から選んだ順で決める(欄 order-1, order-2, … にチーム ID) */
export async function decideRankAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const size = Number(field(form, 'size'))
      const order: number[] = []
      for (let position = 1; position <= size; position++) {
        order.push(idField(form, `order-${position}`))
      }
      if (new Set(order).size !== order.length) {
        throw inputError('同じチームを 2 回選んでいます')
      }
      await formats.decideRank(token, idField(form, 'groupId'), order, field(form, 'reason'))
      return undefined
    },
    '順位を決めました',
  )
}

export async function alignTeamsAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await formats.alignTeams(token, idField(form, 'matchId'))
      return undefined
    },
    '対戦チームを順位どおりに差し替えました',
  )
}
