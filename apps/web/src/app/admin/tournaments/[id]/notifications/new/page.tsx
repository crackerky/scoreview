import { randomUUID } from 'node:crypto'
import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { notifications } from '@/api/notify'
import { audienceKinds, audienceStatuses, maxNotificationBody } from '@/api/notify.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { sendNotificationAction } from '@/app/admin/_actions/notify'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, Hidden, Select, Stat } from '@/components/admin/AdminParts'
import { MessageDraft } from '@/components/admin/MessageDraft'
import { notifyPaths } from '@/notify/notify.paths'
import {
  audienceFromQuery,
  audienceKindLabels,
  audienceQuery,
  audienceStatusLabels,
  quotaState,
} from '@/notify/notify.utils'

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
  const query = audienceQuery(audience)
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
      <h2 className="adm-section-title">宛先</h2>
      <form method="get" className="adm-stack">
        <fieldset>
          <legend>誰に送るか</legend>
          {audienceKinds.map((kind) => (
            <label key={kind} className="adm-check adm-audience">
              <input
                type="radio"
                name="kind"
                value={kind}
                defaultChecked={audience.kind === kind}
              />
              <span>
                {audienceKindLabels[kind].label}
                <small className="adm-muted">{audienceKindLabels[kind].hint}</small>
              </span>
            </label>
          ))}
        </fieldset>
        <div className="adm-grid">
          <Select
            label="申込の状態(申し込んだ代表のときだけ)"
            name="status"
            defaultValue={audience.status}
            options={audienceStatuses.map((status) => ({
              value: status,
              label: audienceStatusLabels[status],
            }))}
          />
          <Select
            label="部門(前の大会の代表には使いません)"
            name="division"
            defaultValue={audience.divisionId === null ? '' : String(audience.divisionId)}
            options={[
              { value: '', label: 'すべての部門' },
              ...divisionList.divisions.map((division) => ({
                value: String(division.id),
                label: division.name.ja,
              })),
            ]}
          />
        </div>
        <div className="adm-actions">
          <button type="submit">宛先を確かめる</button>
        </div>
      </form>

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

      <h2 className="adm-section-title">文</h2>
      <form id={formId} action={sendNotificationAction} className="adm-stack">
        <Hidden
          values={{
            back: notifyPaths.create(id),
            tournamentId: id,
            requestKey: randomUUID(),
            kind: query.kind ?? audience.kind,
            status: query.status ?? '',
            division: query.division ?? '',
          }}
        />
        <MessageDraft heading={preview.heading} maxLength={maxNotificationBody} />
      </form>
    </AdminShell>
  )
}
