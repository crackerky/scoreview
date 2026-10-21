'use server'

import { field, idField, inputError, runAction } from '@/admin/action'
import { divisions, galleries, tournaments } from '@/api/admin'
import {
  bannersFormSchema,
  divisionFormSchema,
  galleryFormSchema,
  tournamentFormSchema,
} from '@/api/admin.types'

// 大会、バナー、大会に属する部門とギャラリーの作成。大会のコピーと公開の切り替え。

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

/** 大会をコピーして新しい大会を作る(運営者レビュー H9)。開催日は必須、名前は空ならコピー元と同じ */
export async function copyTournamentAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      const startDate = field(form, 'startDate')
      if (!/^\d{4}-\d{2}-\d{2}$/.test(startDate)) {
        throw inputError('新しい大会の開催日を入れてください')
      }
      const created = await tournaments.copy(token, idField(form, 'sourceId'), {
        name: { ja: field(form, 'nameJa').trim(), en: '' },
        startDate,
        withTeams: form.withTeams === 'on',
      })
      return `/admin/tournaments/${created.id}`
    },
    '大会をコピーしました。準備ができたら「公開する」を押してください',
  )
}

/** 大会を公開の一覧に載せる、または外す(運営者レビュー M2: 作った大会は準備中で非公開) */
export async function setListedAction(formData: FormData): Promise<never> {
  const listed = formData.get('listed') === 'on'
  return runAction(
    formData,
    async ({ token, form }) => {
      await tournaments.update(token, idField(form, 'id'), { listed })
      return undefined
    },
    listed ? '公開しました。大会一覧に載ります' : '非公開にしました',
  )
}
