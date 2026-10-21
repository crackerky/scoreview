'use server'

import { revalidatePath } from 'next/cache'
import { ZodError } from 'zod'
import { field, idField, runAction } from '@/admin/action'
import { parseRoster } from '@/admin/format.utils'
import { readSessionToken } from '@/admin/session'
import { teams } from '@/api/admin'
import { ApiError } from '@/api/client'
import { type CorrectionBody, formats, type ProposalInput } from '@/api/format'
import type { FormatInput, FormatPreview, Proposal } from '@/api/format.types'

// 大会の形(ウィザード)と本部の進行画面の Server Actions。ウィザードは画面を遷移せずに結果を
// 受け取るので、値を返す action にする。進行画面の操作はフォームの送信で、通知つきで元の画面へ戻す。

/** 入力の誤り。runAction が文面つきで元の画面へ戻す(ZodError として扱われる) */
function inputError(message: string): ZodError {
  return new ZodError([{ code: 'custom', message, path: [], input: undefined }])
}

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
    if (error instanceof ApiError && error.status < 500) {
      return { ok: false, error: error.message.replace(/^(conflict|bad request): /, '') }
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

/** 名簿の貼り付けからチームをまとめて登録する(行の順がシード順の初期値になる) */
export async function createTeamsFromRosterAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const divisionId = idField(form, 'divisionId')
      const start = Number(field(form, 'startOrder')) || 0
      const lines = parseRoster(field(form, 'roster'))
      if (lines.length === 0) {
        throw inputError('名簿が空です')
      }
      for (const [index, line] of lines.entries()) {
        await teams.create(token, divisionId, {
          fullName: { ja: line.fullName, en: '' },
          shortName: { ja: line.shortName, en: '' },
          sortOrder: start + index,
        })
      }
      return undefined
    },
    'チームを登録しました',
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

export async function shiftScheduleAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const court = field(form, 'court')
      const minutes = Number(field(form, 'minutes'))
      if (!Number.isInteger(minutes) || minutes === 0) {
        throw inputError('分数を入れてください')
      }
      await formats.shift(token, idField(form, 'divisionId'), court === '' ? null : court, minutes)
      return undefined
    },
    '時刻をずらしました',
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

/** 結果の入力と訂正。影響の確認画面から送る */
export async function correctMatchAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const winner = field(form, 'winnerSide')
      const body: CorrectionBody = {
        score: { left: Number(field(form, 'left')), right: Number(field(form, 'right')) },
        winnerSide: winner === 'left' || winner === 'right' ? winner : null,
        approved: form.approved === 'on',
      }
      await formats.correct(token, idField(form, 'matchId'), body)
      const next = field(form, 'next')
      return next.startsWith('/admin/') ? next : undefined
    },
    '結果を保存しました',
  )
}
