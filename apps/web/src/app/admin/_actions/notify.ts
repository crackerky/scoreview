'use server'

import { field, idField, runActionTo, withQuery } from '@/admin/action'
import { failureTarget, isApiError, withNotice } from '@/admin/form.utils'
import { notifications } from '@/api/notify'
import { notifyPaths } from '@/notify/notify.paths'
import {
  audienceFromQuery,
  audienceQuery,
  checkedAudience,
  keepAudience,
  sameAudience,
  sentNotice,
} from '@/notify/notify.utils'
import { uncertainFailure } from '@/notify/notifyError.utils'

// 管理画面から公式 LINE で送る。送るフォームは、画面で選んでいる宛先の条件(kind、status、division)と、
// 人数を数えた宛先の条件(checked で始まる hidden 欄)の両方を持つ。

const changedAudience =
  '宛先を変えたので、人数を数え直しました。宛先を確かめてから、もう一度送ってください'

const stoppedSend =
  '送信が途中で止まったかもしれません。同じ条件で送り直す前に、送った記録を確かめてください(途中で止まった送信は、記録の画面から送り直せます)'

/** 「宛先を確かめる」。選んでいる宛先の条件で、送る画面を開き直す(書いた文は画面がブラウザに残す) */
export async function checkAudienceAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ form }) =>
    withQuery(
      notifyPaths.create(idField(form, 'tournamentId')),
      audienceQuery(audienceFromQuery(form)),
    ),
  )
}

export async function sendNotificationAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const tournamentId = idField(form, 'tournamentId')
    const audience = audienceFromQuery(form)
    // 宛先の選択を変えたまま押された(JavaScript が動かないとき、ブラウザがフォームの状態を戻したときも)。
    // 数えた人数と違う宛先に送らないよう、送らずに選んでいる宛先で数え直した画面に戻す
    if (!sameAudience(audience, checkedAudience(form))) {
      return withQuery(notifyPaths.create(tournamentId), {
        ...audienceQuery(audience),
        error: changedAudience,
      })
    }
    try {
      const sent = await notifications.send(token, tournamentId, {
        audience,
        body: field(form, 'body'),
        requestKey: field(form, 'requestKey'),
      })
      const [kind, message] = sentNotice(sent.counts)
      return withNotice(notifyPaths.detail(sent.id), kind, message)
    } catch (error) {
      if (uncertainFailure(error)) {
        logUnknown('send notification', error)
        return withNotice(notifyPaths.list(tournamentId), 'error', stoppedSend)
      }
      // 宛先の条件を選び直さずに済むよう、条件を付けて送る画面へ戻す
      return keepAudience(failureTarget(back, error), back, audience)
    }
  })
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
