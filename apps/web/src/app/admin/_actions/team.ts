'use server'

import { idField, runAction } from '@/admin/action'
import { players, teams } from '@/api/admin'
import { playerFormSchema, teamFormSchema } from '@/api/admin.types'

// チームと選手(スタッフを含む)。

export async function updateTeamAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await teams.update(token, idField(form, 'id'), teamFormSchema.parse(form))
    return undefined
  })
}

export async function deleteTeamAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await teams.remove(token, idField(form, 'id'))
      return `/admin/divisions/${idField(form, 'divisionId')}`
    },
    '削除しました',
  )
}

export async function createPlayerAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await players.create(token, idField(form, 'teamId'), playerFormSchema.parse(form))
    return undefined
  })
}

export async function updatePlayerAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await players.update(token, idField(form, 'id'), playerFormSchema.parse(form))
    return undefined
  })
}

export async function deletePlayerAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await players.remove(token, idField(form, 'id'))
      return undefined
    },
    '削除しました',
  )
}
