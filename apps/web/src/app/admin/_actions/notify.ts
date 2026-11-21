'use server'

import { field, idField, runActionTo } from '@/admin/action'
import { failureTarget, withNotice } from '@/admin/form.utils'
import { notifications } from '@/api/notify'
import { notifyPaths } from '@/notify/notify.paths'
import { audienceFromQuery, keepAudience, sentNotice } from '@/notify/notify.utils'

// 管理画面から公式 LINE で送る。

export async function sendNotificationAction(formData: FormData): Promise<never> {
  return runActionTo(formData, async ({ token, form, back }) => {
    const tournamentId = idField(form, 'tournamentId')
    const audience = audienceFromQuery(form)
    try {
      const sent = await notifications.send(token, tournamentId, {
        audience,
        body: field(form, 'body'),
        requestKey: field(form, 'requestKey'),
      })
      const [kind, message] = sentNotice(sent.counts)
      return withNotice(notifyPaths.detail(sent.id), kind, message)
    } catch (error) {
      // 宛先の条件を選び直さずに済むよう、条件を付けて送る画面へ戻す(api の 500 は投げ直される)
      return keepAudience(failureTarget(back, error), back, audience)
    }
  })
}
