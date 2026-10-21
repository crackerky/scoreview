'use server'

import { idField, runAction } from '@/admin/action'
import { matches } from '@/api/admin'
import {
  eventFormSchema,
  matchFormSchema,
  playerStatsFromForm,
  statusFormSchema,
} from '@/api/admin.types'

// 試合の登録とスコア入力。スコアと状態は api の試合更新関数だけが変える(得点経過と状態の変更)。

export async function updateMatchAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    const input = matchFormSchema.parse(form)
    // 対戦チームの欄を変えていなければ送らない。参照([A1-勝ち])が解決して入ったチームを、
    // 参照だけを送り直して消さないため
    const changed = (side: 'left' | 'right') => form[side] !== form[`${side}Original`]
    await matches.update(token, idField(form, 'id'), {
      ...input,
      left: changed('left') ? input.left : undefined,
      right: changed('right') ? input.right : undefined,
    })
    return undefined
  })
}

export async function deleteMatchAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await matches.remove(token, idField(form, 'id'))
      return `/admin/divisions/${idField(form, 'divisionId')}`
    },
    '削除しました',
  )
}

export async function addEventAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await matches.addEvent(token, idField(form, 'id'), eventFormSchema.parse(form))
      return undefined
    },
    '記録しました',
  )
}

export async function removeEventAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await matches.removeEvent(token, idField(form, 'id'), idField(form, 'eventId'))
      return undefined
    },
    '取り消しました',
  )
}

export async function setStatusAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await matches.setStatus(token, idField(form, 'id'), statusFormSchema.parse(form))
    return undefined
  })
}

export async function putPlayerStatsAction(formData: FormData): Promise<never> {
  return runAction(formData, async ({ token, form }) => {
    await matches.putPlayerStats(token, idField(form, 'id'), playerStatsFromForm(form))
    return undefined
  })
}
