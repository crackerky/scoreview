'use server'

import { idField, runAction } from '@/admin/action'
import { galleries } from '@/api/admin'
import { galleryFormSchema, photosFormSchema } from '@/api/admin.types'

export async function updateGalleryAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    const id = idField(form, 'id')
    await galleries.update(token, id, galleryFormSchema.parse(form))
    await galleries.putPhotos(token, id, photosFormSchema.parse(form))
    return undefined
  })
}

export async function deleteGalleryAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await galleries.remove(token, idField(form, 'id'))
      return `/admin/tournaments/${idField(form, 'tournamentId')}`
    },
    '削除しました',
  )
}
