'use server'

import { idField, runAction } from '@/admin/action'
import { assignments } from '@/api/admin'
import { assignmentFormSchema } from '@/api/admin.types'

// 記録係の割り当て(コートと時間帯、または試合)の登録と削除。

export async function createAssignmentAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await assignments.create(
        token,
        idField(form, 'tournamentId'),
        assignmentFormSchema.parse(form),
      )
      return undefined
    },
    '割り当てました',
  )
}

export async function deleteAssignmentAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await assignments.remove(token, idField(form, 'id'))
      return undefined
    },
    '割り当てを外しました',
  )
}
