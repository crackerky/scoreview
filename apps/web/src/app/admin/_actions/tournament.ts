'use server'

import { idField, runAction } from '@/admin/action'
import { divisions, galleries, tournaments } from '@/api/admin'
import {
  bannersFormSchema,
  divisionFormSchema,
  galleryFormSchema,
  tournamentFormSchema,
} from '@/api/admin.types'

// 大会、バナー、大会に属するディビジョンとギャラリーの作成。

export async function createTournamentAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    const created = await tournaments.create(token, tournamentFormSchema.parse(form))
    return `/admin/tournaments/${created.id}`
  })
}

export async function updateTournamentAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await tournaments.update(token, idField(form, 'id'), tournamentFormSchema.parse(form))
    return undefined
  })
}

export async function deleteTournamentAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await tournaments.remove(token, idField(form, 'id'))
      return '/admin'
    },
    '削除しました',
  )
}

export async function putBannersAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await tournaments.putBanners(token, idField(form, 'id'), bannersFormSchema.parse(form))
    return undefined
  })
}

export async function createDivisionAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    const created = await divisions.create(
      token,
      idField(form, 'tournamentId'),
      divisionFormSchema.parse(form),
    )
    return `/admin/divisions/${created.id}`
  })
}

export async function createGalleryAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    const created = await galleries.create(
      token,
      idField(form, 'tournamentId'),
      galleryFormSchema.parse(form),
    )
    return `/admin/galleries/${created.id}`
  })
}
