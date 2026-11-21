import { randomUUID } from 'node:crypto'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { notifications } from '@/api/notify'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, Stat } from '@/components/admin/AdminParts'
import { NotificationComposer } from '@/components/admin/NotificationComposer'
import { notifyPaths } from '@/notify/notify.paths'
import { audienceFromQuery, draftStorageKey, quotaState } from '@/notify/notify.utils'

// 公式 LINE で送る。上で宛先の条件を選んで人数を確かめ(GET のフォーム)、下で文を書いて送る。
// 送る画面を開くたびに作る requestKey で、押し直しや再送でも 1 回しか送らない(api が弾く)。
export default async function NewNotificationPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const audience = audienceFromQuery(await props.searchParams)
  const { user, token } = await requireUser('admin')
  const [tournament, divisionList, preview] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      divisions.list(token, id),
      notifications.preview(token, id, audience),
    ]),
  )
  const quota = quotaState(preview.quota, preview.reachable)
  const ready = preview.canSend && preview.reachable > 0 && quota.enough
  const formId = 'notification-form'
  return (
    <AdminShell
      user={user}
      title="公式 LINE で送る"
      notice={notice}
      crumbs={[
        { href: `/admin/tournaments/${id}`, label: tournament.name.ja },
        { href: notifyPaths.list(id), label: '送った記録' },
      ]}
      primary={
        ready
          ? { label: `${preview.reachable} 人に送る`, form: formId }
          : { label: '送る', form: formId, disabled: true }
      }
    >
      <NotificationComposer
        checked={audience}
        divisions={divisionList.divisions.map((division) => ({
          value: String(division.id),
          label: division.name.ja,
        }))}
        heading={preview.heading}
        formId={formId}
        hidden={{ back: notifyPaths.create(id), tournamentId: id, requestKey: randomUUID() }}
        draftKey={draftStorageKey(id)}
      >
        <div className="adm-stats">
          <Stat label="送る人" value={String(preview.reachable)} />
          <Stat
            label="ブロック中で送らない人"
            value={String(preview.unreachable)}
            tone={preview.unreachable > 0 ? 'warn' : undefined}
          />
          <Stat label="今回使う配信数" value={`${preview.reachable} 通`} />
        </div>
        <p className={quota.enough ? 'adm-muted' : 'adm-notice error'}>
          {quota.text}
          {quota.enough ? '' : '。今回の分が足りないので送れません。宛先を絞ってください'}
        </p>
        {preview.canSend ? null : (
          <p className="adm-notice error">
            公式 LINE の設定が無いので、宛先の確認だけできます。送るには api に LINE
            の設定が要ります。
          </p>
        )}
        {preview.recipients.length === 0 ? (
          <p className="adm-muted">この条件に当てはまる人はいません。</p>
        ) : (
          <details className="adm-recipients">
            <summary>宛先 {preview.recipients.length} 人を見る</summary>
            <ul className="adm-rows">
              {preview.recipients.map((recipient, index) => (
                // 同じ名前の人がいるので、並び(api が返す順)を key に使う
                // biome-ignore lint/suspicious/noArrayIndexKey: 宛先の並びはこの描画の間は変わらない
                <li key={index} className="adm-row">
                  <span className="adm-row-main">
                    <span className="adm-row-title">{recipient.name || '名前なし'}</span>
                    <span className="adm-row-meta">{recipient.teams.join('、')}</span>
                  </span>
                  {recipient.reachable ? null : <Chip tone="done">ブロック中</Chip>}
                </li>
              ))}
            </ul>
          </details>
        )}
      </NotificationComposer>
    </AdminShell>
  )
}
