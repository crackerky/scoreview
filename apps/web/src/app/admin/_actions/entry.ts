'use server'

import { field, idField, inputError, runAction } from '@/admin/action'
import { entries } from '@/api/entry'
import { entryActions } from '@/api/entry.types'
import { entryPaths } from '@/entry/entry.paths'
import { draftFromForm, questionsFromForm } from '@/entry/entry.utils'

// 大会への申込(質問の設定、代わりに入れる申込、承認と差し戻しと取消)。

/** 質問の設定の行の数(今の質問の数と足す行の数。画面が hidden 欄の rows で渡す) */
function rowCount(form: Record<string, string | string[]>): number {
  const rows = Number(field(form, 'rows'))
  if (!Number.isInteger(rows) || rows < 0 || rows > 100) {
    throw new RangeError('form field rows is out of range')
  }
  return rows
}

export async function saveEntryQuestionsAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await entries.saveQuestions(
      token,
      idField(form, 'tournamentId'),
      questionsFromForm(form, rowCount(form)),
    )
    return undefined
  })
}

export async function createEntryAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const tournamentId = idField(form, 'tournamentId')
      const { questions } = await entries.questions(token, tournamentId)
      const draft = draftFromForm(form, questions)
      if (draft.members.length === 0) {
        throw inputError('メンバーを 1 人以上書いてください(1 行に 1 人)')
      }
      const created = await entries.create(token, tournamentId, draft)
      return entryPaths.detail(created.id)
    },
    '申込を入れました',
  )
}

const decisionMessages = {
  approve: '参加を決めました。部門にチームと選手を作りました',
  return: '直し待ちにしました',
  cancel: '取り消しました',
} as const

export async function decideEntryAction(formData: FormData): Promise<never> {
  const action = entryActions.find((value) => value === formData.get('action'))
  return runAction(
    formData,
    async ({ token, form }) => {
      if (action === undefined) {
        throw inputError('操作を選んでください')
      }
      await entries.decide(token, idField(form, 'id'), action, field(form, 'note'))
      return undefined
    },
    action === undefined ? '保存しました' : decisionMessages[action],
  )
}
