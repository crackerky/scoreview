'use server'

import { idField, runAction } from '@/admin/action'
import { divisions, groups, matches, teams } from '@/api/admin'
import {
  bracketFormSchema,
  divisionFormSchema,
  finalRanksFromForm,
  groupFormSchema,
  groupTeamsFromForm,
  matchFormSchema,
  teamFormSchema,
} from '@/api/admin.types'

// ディビジョン、最終順位、グループ(トーナメント表の生成を含む)、チームと試合の作成。

export async function updateDivisionAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await divisions.update(token, idField(form, 'id'), divisionFormSchema.parse(form))
    return undefined
  })
}

export async function deleteDivisionAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await divisions.remove(token, idField(form, 'id'))
      return `/admin/tournaments/${idField(form, 'tournamentId')}`
    },
    '削除しました',
  )
}

/** 最終順位を書き、[最終順位] の公開も切り替える */
export async function putFinalRanksAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    const id = idField(form, 'id')
    await divisions.putFinalRanks(token, id, finalRanksFromForm(form))
    await divisions.update(token, id, { standingsPublished: form.standingsPublished === 'on' })
    return undefined
  })
}

export async function createGroupAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    const created = await groups.create(
      token,
      idField(form, 'divisionId'),
      groupFormSchema.parse(form),
    )
    return `/admin/groups/${created.id}`
  })
}

export async function updateGroupAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await groups.update(token, idField(form, 'id'), groupFormSchema.parse(form))
    return undefined
  })
}

export async function deleteGroupAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await groups.remove(token, idField(form, 'id'))
      return `/admin/divisions/${idField(form, 'divisionId')}`
    },
    '削除しました',
  )
}

export async function putGroupTeamsAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await groups.putTeams(token, idField(form, 'id'), groupTeamsFromForm(form))
    return undefined
  })
}

export async function generateBracketAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await groups.generateBracket(token, idField(form, 'id'), bracketFormSchema.parse(form))
      return undefined
    },
    'トーナメント表と試合を作りました',
  )
}

export async function createTeamAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await teams.create(token, idField(form, 'divisionId'), teamFormSchema.parse(form))
    return undefined
  })
}

export async function createMatchAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await matches.create(token, idField(form, 'divisionId'), matchFormSchema.parse(form))
    return undefined
  })
}
