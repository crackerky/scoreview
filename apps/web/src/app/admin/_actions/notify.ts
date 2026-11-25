'use server'

import { field, idField, runAction, runActionTo, withQuery } from '@/admin/action'
import { failureTarget, isApiError, withNotice } from '@/admin/form.utils'
import { notifications } from '@/api/notify'
import { notifyPaths } from '@/notify/notify.paths'
import {
  audienceFromQuery,
  audienceQuery,
  checkedAudience,
  checkedTiming,
  keepPlan,
  localTimeToIso,
  sameAudience,
  sameTiming,
  scheduleText,
  sentNotice,
  timingFromQuery,
  timingQuery,
} from '@/notify/notify.utils'
import { uncertainFailure } from '@/notify/notifyError.utils'

// 管理画面から公式 LINE で送る。送るフォームは、画面で選んでいる宛先の条件と送る時刻(kind、status、
// division、timing、at)と、人数を数えた条件と時刻(checked で始まる hidden 欄)の両方を持つ。

const changedPlan =
  '宛先か送る時刻を変えたので、数え直しました。確かめてから、もう一度押してください'

const stoppedSend =
  '送信が途中で止まったかもしれません。同じ条件で送り直す前に、送った記録を確かめてください(途中で止まった送信は、記録の画面から送り直せます)'

/** 「宛先と時刻を確かめる」。選んでいる条件と時刻で、送る画面を開き直す(書いた文は画面が残す) */
export async function checkAudienceAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ form }) =>
    withQuery(notifyPaths.create(idField(form, 'tournamentId')), {
      ...audienceQuery(audienceFromQuery(form)),
      ...timingQuery(timingFromQuery(form)),
    }),
  )
}

export async function sendNotificationAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const tournamentId = idField(form, 'tournamentId')
    const audience = audienceFromQuery(form)
    const timing = timingFromQuery(form)
    const plan = { ...audienceQuery(audience), ...timingQuery(timing) }
    // 宛先か時刻を変えたまま押された(JavaScript が動かないとき、ブラウザがフォームの状態を戻したときも)。
    // 数えた人数や確かめた時刻と違う送信にしないよう、送らずに選んでいる条件で数え直した画面に戻す
    if (
      !sameAudience(audience, checkedAudience(form)) ||
      !sameTiming(timing, checkedTiming(form))
    ) {
      return withQuery(notifyPaths.create(tournamentId), { ...plan, error: changedPlan })
    }
    // すぐ送るのに送る日時も入っている(JavaScript が動かないとき)。どちらのつもりか分からないので送らない
    if (timing.kind === 'now' && field(form, 'at') !== '') {
      return withQuery(notifyPaths.create(tournamentId), {
        ...plan,
        error:
          '「すぐ送る」と送る日時の両方が入っています。予約するなら「日時を決めて送る」を選んでください',
      })
    }
    const sendAt = timing.kind === 'later' ? localTimeToIso(timing.at) : null
    if (timing.kind === 'later' && sendAt === null) {
      return withQuery(notifyPaths.create(tournamentId), {
        ...plan,
        error: '送る日時を入れてください',
      })
    }
    try {
      const sent = await notifications.send(token, tournamentId, {
        audience,
        body: field(form, 'body'),
        requestKey: field(form, 'requestKey'),
        sendAt,
      })
      if (sent.state === 'scheduled' && sent.sendAt !== null) {
        return withNotice(
          notifyPaths.detail(sent.id),
          'ok',
          `${scheduleText(sent.sendAt)} に送る予約をしました`,
        )
      }
      const [kind, message] = sentNotice(sent.counts)
      return withNotice(notifyPaths.detail(sent.id), kind, message)
    } catch (error) {
      if (uncertainFailure(error)) {
        logUnknown('send notification', error)
        return withNotice(notifyPaths.list(tournamentId), 'error', stoppedSend)
      }
      // 宛先の条件と時刻を選び直さずに済むよう、付けて送る画面へ戻す
      return keepPlan(failureTarget(back, error), back, plan)
    }
  })
}

export async function cancelNotificationAction(formData: FormData): Promise<never> {
  return runAction(
    formData,
    async ({ token, form }) => {
      await notifications.cancel(token, idField(form, 'id'))
      return undefined
    },
    '予約を取りやめました',
  )
}

export async function resendNotificationAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    try {
      const sent = await notifications.resend(token, idField(form, 'id'))
      const [kind, message] = sentNotice(sent.counts)
      return withNotice(back, kind, kind === 'ok' ? '送れなかった人に送り直しました' : message)
    } catch (error) {
      if (uncertainFailure(error)) {
        logUnknown('resend notification', error)
        return withNotice(
          back,
          'error',
          '送り直しが途中で止まったかもしれません。結果を確かめてください',
        )
      }
      return failureTarget(back, error)
    }
  })
}

/** api の答えではない失敗(通信、本文が JSON でない、応答の形)は api のログに残らないので、ここで残す */
function logUnknown(what: string, error: unknown) {
  if (!isApiError(error)) {
    console.error(`${what}: the result is unknown`, error)
  }
}
