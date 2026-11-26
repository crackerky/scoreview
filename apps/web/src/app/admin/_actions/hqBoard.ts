'use server'

import { field, idField, inputError, runActionTo, withQuery } from '@/admin/action'
import { withNotice } from '@/admin/form.utils'
import { hqPath } from '@/admin/hqBoard.utils'
import { courtLabel } from '@/admin/scoreKeeper.utils'
import { formats } from '@/api/format'

// 大会単位の本部から進む作業(時間をずらす、組の確定)の Server Actions。終わったら本部の「いま」へ戻る。

/** 選べるずらす時間(分)。本部の画面の選択肢と同じ */
const shiftChoices = [5, 10, 15, 20]

/**
 * コートのまだ始まっていない試合を、部門をまたいでまとめてずらす。api のずらし方は部門ごとなので、
 * そのコートに試合のある部門(divisionIds)ごとに呼ぶ。ずらしたあとの本部の画面には「元に戻す」を出す
 * (同じコートを同じ分だけ逆にずらす。minutes が負なら元に戻す操作)。
 */
export async function shiftCourtAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form }) => {
    const tournamentId = idField(form, 'tournamentId')
    const court = field(form, 'court')
    const minutes = Number(field(form, 'minutes'))
    if (court === '') {
      throw inputError('ずらすコートを選んでください')
    }
    if (!shiftChoices.includes(Math.abs(minutes))) {
      throw inputError('ずらす時間を選んでください')
    }
    const divisionIds = field(form, 'divisionIds')
      .split(',')
      .filter((value) => /^\d+$/.test(value))
      .map(Number)
    let moved = 0
    for (const divisionId of divisionIds) {
      const result = await formats.shift(token, divisionId, court, minutes)
      moved += result.moved
    }
    if (minutes < 0) {
      return withNotice(
        hqPath(tournamentId),
        'ok',
        `${courtLabel(court)}の ${moved} 試合を元の時刻に戻しました`,
      )
    }
    return withQuery(hqPath(tournamentId), {
      ok: `${courtLabel(court)}の ${moved} 試合を ${minutes} 分ずらしました`,
      undoCourt: court,
      undoMinutes: minutes,
      undoDivisions: divisionIds.join(','),
    })
  })
}

/** 全試合が終わった組の順位を確定し、次の段の組み合わせに入れる */
export async function confirmGroupFromHqAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form }) => {
    const tournamentId = idField(form, 'tournamentId')
    await formats.confirmGroup(token, idField(form, 'groupId'), false)
    return withNotice(
      hqPath(tournamentId),
      'ok',
      `${field(form, 'groupLabel')}を確定しました。次の組み合わせに入りました`,
    )
  })
}
