'use server'

import { idField, runAction } from '@/admin/action'
import { users } from '@/api/admin'
import { userCreateFormSchema, userFormSchema, userTournamentsFormSchema } from '@/api/admin.types'

export async function createUserAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    const created = await users.create(token, userCreateFormSchema.parse(form))
    return `/admin/users/${created.id}`
  })
}

export async function updateUserAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    // password が空欄なら送らない(変えない)
    await users.update(token, idField(form, 'id'), userFormSchema.parse(form))
    return undefined
  })
}

export async function putUserTournamentsAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await users.putTournaments(token, idField(form, 'id'), userTournamentsFormSchema.parse(form))
    return undefined
  })
}

export async function deleteUserAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await users.remove(token, idField(form, 'id'))
      return '/admin/users'
    },
    '削除しました',
  )
}
