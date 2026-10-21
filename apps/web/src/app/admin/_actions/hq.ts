'use server'

import { ZodError } from 'zod'
import { field, idField, runActionTo, withQuery } from '@/admin/action'
import { withNotice } from '@/admin/form.utils'
import { matches } from '@/api/admin'
import { ApiError } from '@/api/client'
import { type Downstream, formats, type SuspendBody } from '@/api/format'

// 本部の進行画面と結果の入力の Server Actions(レビュー H2、H3、H5、H7、H10、M5、M6)。
// 失敗しても入力を捨てずに元の画面へ戻すものは、戻り先の URL に入力と理由を載せる。

function inputError(message: string): ZodError {
  return new ZodError([{ code: 'custom', message, path: [], input: undefined }])
}

function scoreField(form: Record<string, string | string[]>, name: string): number {
  const value = field(form, name)
  if (!/^\d{1,3}$/.test(value)) {
    throw inputError('スコアは 0〜999 の数で入れてください')
  }
  return Number(value)
}

function sideField(form: Record<string, string | string[]>, name: string): 'left' | 'right' | null {
  const value = field(form, name)
  return value === 'left' || value === 'right' ? value : null
}

function downstreamOf(value: string): Downstream {
  return value === 'keep' || value === 'reset' ? value : null
}

/** 「確かめた値」(画面を開いたときのスコア "11-9")を読む */
function seenScore(
  form: Record<string, string | string[]>,
): { left: number; right: number } | null {
  const found = /^(\d+)-(\d+)$/.exec(field(form, 'seen'))
  return found ? { left: Number(found[1]), right: Number(found[2]) } : null
}

/** 下流の扱いの承認が無くて断られたか(409 conflict の理由に「承認」を含む。format_guard.go) */
function approvalMissing(error: unknown): boolean {
  return error instanceof ApiError && error.code === 'conflict' && error.message.includes('承認')
}

function isScoreMismatch(error: unknown): boolean {
  return error instanceof ApiError && error.code === 'score_mismatch'
}

/** 結果の入力と訂正(結果の入力画面から)。古い画面からの保存と承認の漏れは、入力を残して画面へ戻す */
export async function correctMatchAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const id = idField(form, 'matchId')
    const left = scoreField(form, 'left')
    const right = scoreField(form, 'right')
    const winner = sideField(form, 'winnerSide')
    const keep = field(form, 'keepEvents') === 'on' || field(form, 'keepEvents') === '1'
    const echo = {
      left,
      right,
      winner: winner ?? undefined,
      keep: keep ? '1' : undefined,
      next: field(form, 'next'),
    }
    try {
      await formats.correct(token, id, {
        score: { left, right },
        winnerSide: winner,
        downstream: downstreamOf(field(form, 'downstream')),
        expectedScore: seenScore(form),
        keepEvents: keep,
      })
    } catch (error) {
      if (isScoreMismatch(error)) {
        return withQuery(back, { ...echo, changed: '1' })
      }
      if (approvalMissing(error)) {
        return withQuery(back, { ...echo, seen: field(form, 'seen'), approval: 'missing' })
      }
      throw error
    }
    const next = field(form, 'next')
    return withNotice(next.startsWith('/admin/') ? next : back, 'ok', '結果を保存しました')
  })
}

/** 試合をしなかった結果(不戦勝、両チーム不戦敗、不成立、未消化)を本部が記録する(H3) */
export async function setResultAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const id = idField(form, 'matchId')
    const resultType = field(form, 'resultType')
    if (!['forfeit', 'double_forfeit', 'no_contest', 'unplayed'].includes(resultType)) {
      throw inputError('結果の種類を選んでください')
    }
    const needsWinner = resultType === 'forfeit' || resultType === 'no_contest'
    const winner = sideField(form, 'resultWinner')
    if (needsWinner && winner === null) {
      throw inputError('勝ったチーム(不戦勝なら来ていたチーム)を選んでください')
    }
    try {
      await matches.setResult(token, id, {
        resultType,
        winnerSide: needsWinner ? winner : null,
        expectedScore: seenScore(form),
      })
    } catch (error) {
      if (isScoreMismatch(error)) {
        return withQuery(back, { changed: '1', resultType })
      }
      throw error
    }
    const next = field(form, 'next')
    return withNotice(next.startsWith('/admin/') ? next : back, 'ok', '結果を記録しました')
  })
}

/** 得点経過のある終了した試合を試合中へ戻す。下流の扱いを選ばずに断られたら、欄を赤で囲んで戻す(M6) */
export async function reopenMatchAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const id = idField(form, 'matchId')
    try {
      await matches.setStatus(token, id, {
        status: 'live',
        downstream: downstreamOf(field(form, 'downstream')),
      })
    } catch (error) {
      if (approvalMissing(error)) {
        return withQuery(back, { approval: 'reopen' })
      }
      throw error
    }
    return withNotice(
      `/admin/matches/${id}/score`,
      'ok',
      '試合中に戻しました。得点経過を直して試合を終えてください',
    )
  })
}

/** 遅れの調整(確認の後)。直後に「元に戻す」を出すため、ずらした分を戻り先に載せる(M5) */
export async function shiftScheduleAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const court = field(form, 'court')
    const minutes = Number(field(form, 'minutes'))
    if (!Number.isInteger(minutes) || minutes === 0) {
      throw inputError('ずらす分数を入れてください')
    }
    const result = await formats.shift(
      token,
      idField(form, 'divisionId'),
      court === '' ? null : court,
      minutes,
    )
    const undo = field(form, 'undo') === '1'
    const where = court === '' ? '全コート' : `コート ${court} `
    const message = undo
      ? `${where}の ${result.moved} 試合を元の時刻に戻しました`
      : `${where}の ${result.moved} 試合を ${Math.abs(minutes)} 分${minutes > 0 ? '遅らせ' : '早め'}ました`
    return withQuery(back, {
      ok: message,
      tab: 'delay',
      undoCourt: undo ? undefined : court === '' ? '*' : court,
      undoMinutes: undo ? undefined : -minutes,
    })
  })
}

/** チームの棄権の一括処理(H3)。確認画面で見た試合だけを記録する */
export async function withdrawTeamAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form }) => {
    const teamId = idField(form, 'teamId')
    const raw = form.matchIds
    const ids = (Array.isArray(raw) ? raw : raw === undefined || raw === '' ? [] : [raw]).map(
      Number,
    )
    const done = await formats.withdraw(token, teamId, ids)
    const next = field(form, 'next')
    return withNotice(
      next.startsWith('/admin/') ? next : `/admin/divisions/${done.divisionId}/progress`,
      'ok',
      ids.length === 0
        ? `${done.team.name.ja} を棄権にしました`
        : `${done.team.name.ja} を棄権にし、${ids.length} 試合を記録しました`,
    )
  })
}

/** 大会を締める(H10)。確認画面で見た残りの試合を未消化にし、順位を書いて公開する */
export async function closeDivisionAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form }) => {
    const divisionId = idField(form, 'divisionId')
    const raw = form.matchIds
    const ids = (Array.isArray(raw) ? raw : raw === undefined || raw === '' ? [] : [raw]).map(
      Number,
    )
    const publish = field(form, 'publish') === 'on'
    await formats.close(token, divisionId, ids, publish)
    return withNotice(
      `/admin/divisions/${divisionId}/progress`,
      'ok',
      publish
        ? '大会を締め、最終順位を公開しました'
        : '大会を締めました(最終順位はまだ公開していません)',
    )
  })
}

const reasons = ['thunder', 'rain', 'heat', 'other'] as const

/** 大会を中断する(H2) */
export async function suspendAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const reason = reasons.find((value) => value === field(form, 'reason'))
    if (reason === undefined) {
      throw inputError('中断の理由を選んでください')
    }
    const resumeAt = field(form, 'resumeAt')
    const body: SuspendBody = {
      reason,
      note: field(form, 'note'),
      resumeAt: resumeAt === '' ? null : resumeAt,
    }
    await formats.suspend(token, idField(form, 'tournamentId'), body)
    return withNotice(back, 'ok', '中断しました。閲覧ページの上部に中断の帯を出しています')
  })
}

/** 中断を終える(H2)。中断した分だけ未開始の試合をずらし、終了時刻を過ぎる試合を知らせる */
export async function resumeAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const raw = field(form, 'minutes')
    const minutes = raw === '' ? null : Number(raw)
    if (minutes !== null && (!Number.isInteger(minutes) || minutes < 0)) {
      throw inputError('ずらす分数は 0 以上の数で入れてください')
    }
    const result = await formats.resume(token, idField(form, 'tournamentId'), minutes)
    let message = `再開しました。まだ始まっていない ${result.shifted} 試合を ${result.minutes} 分遅らせました`
    if (result.overEnd.length > 0) {
      const list = result.overEnd
        .slice(0, 6)
        .map((match) => `${match.code} ${match.time}(${match.overMinutes} 分超過)`)
        .join('、')
      const more = result.overEnd.length > 6 ? ` ほか ${result.overEnd.length - 6} 試合` : ''
      message += `。会場の終了時刻を過ぎる試合が ${result.overEnd.length} 試合あります: ${list}${more}`
    }
    return withNotice(back, result.overEnd.length > 0 ? 'error' : 'ok', message)
  })
}
