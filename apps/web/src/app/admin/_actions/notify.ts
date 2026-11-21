'use server'

import { field, idField, runActionTo, withQuery } from '@/admin/action'
import { failureTarget, withNotice } from '@/admin/form.utils'
import { notifications } from '@/api/notify'
import { notifyPaths } from '@/notify/notify.paths'
import { audienceFromQuery, audienceQuery, keepAudience, sentNotice } from '@/notify/notify.utils'
import { uncertainFailure } from '@/notify/notifyError.utils'

// 管理画面から公式 LINE で送る。

const changedAudience =
  '宛先を変えたので、人数を数え直しました。宛先を確かめてから、もう一度送ってください'

const stoppedSend =
  '送信が途中で止まったかもしれません。同じ条件で送り直す前に、送った記録を確かめてください(途中で止まった送信は、記録の画面から送り直せます)'

export async function sendNotificationAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const tournamentId = idField(form, 'tournamentId')
    const audience = audienceFromQuery(form)
    // 送る画面で宛先の選択を変えたまま押された。数えた人数と違う宛先に送らないよう、選び直した宛先で
    // 数え直した画面に戻す(書いた文は画面がブラウザに残している)
    if (field(form, 'changed') === '1') {
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
