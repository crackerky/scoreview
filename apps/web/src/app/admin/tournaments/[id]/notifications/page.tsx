import { loadAdmin, requireUser } from '@/admin/session'
import { divisions, tournaments } from '@/api/admin'
import { notifications } from '@/api/notify'
import type { NotificationSummary } from '@/api/notify.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip } from '@/components/admin/AdminParts'
import { dateTimeText } from '@/entry/entry.utils'
import { notifyPaths } from '@/notify/notify.paths'
import {
  audienceText,
  batchStateLabels,
  bodyExcerpt,
  countsText,
  scheduleText,
} from '@/notify/notify.utils'

// 大会で公式 LINE から送った記録(新しい順)。押すと宛先ごとの結果を開く。管理者だけ(宛先は申込の
// 代表とメンバーのため)。
export default async function NotificationsPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const { user, token } = await requireUser('admin')
  const [tournament, divisionList, history] = await loadAdmin(() =>
    Promise.all([
      tournaments.get(token, id),
      divisions.list(token, id),
      notifications.list(token, id),
    ]),
  )
  const divisionNames = new Map(
    divisionList.divisions.map((division) => [division.id, division.name.ja]),
  )
  return (
    <AdminShell
      user={user}
      title="公式 LINE で送った記録"
      notice={notice}
      crumbs={[{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]}
      primary={{ label: '新しく送る', href: notifyPaths.create(id) }}
    >
      {history.canSend ? (
        <p className="adm-muted">
          申込の代表やメンバーに、条件を選んで公式 LINE から送ります。送った人数が公式 LINE
          の月の配信数に数えられます。
        </p>
      ) : (
        <p className="adm-notice error">
          公式 LINE の設定が無いので、今は送れません(宛先の確認だけできます)。
        </p>
      )}
      {history.notifications.length === 0 ? (
        <p className="adm-muted">まだ送っていません。</p>
      ) : (
        <ul className="adm-rows">
          {history.notifications.map((item) => (
            <li key={item.id}>
              <a className="adm-row" href={notifyPaths.detail(item.id)}>
                <span className="adm-row-main">
                  <span className="adm-row-title">{bodyExcerpt(item.body)}</span>
                  <span className="adm-row-meta">
                    {[
                      item.sendAt
                        ? `${scheduleText(item.sendAt)} に予約`
                        : dateTimeText(item.createdAt),
                      audienceText(item.audience, (division) => divisionNames.get(division)),
                      item.state === 'sent' ? countsText(item.counts) : '',
                      item.sentBy,
                    ]
                      .filter((part) => part !== '')
                      .join(' ・ ')}
                  </span>
                </span>
                <BatchChip item={item} />
                <span className="adm-chevron" aria-hidden="true">
                  ›
                </span>
              </a>
            </li>
          ))}
        </ul>
      )}
    </AdminShell>
  )
}

/** 一覧の印。予約、取りやめ、送れなかった予約は段階を、送った送信は結果の困りごとだけを出す */
function BatchChip({ item }: { item: NotificationSummary }) {
  switch (item.state) {
    case 'scheduled':
      return <Chip tone="sched">{batchStateLabels.scheduled}</Chip>
    case 'cancelled':
      return <Chip tone="done">{batchStateLabels.cancelled}</Chip>
    case 'skipped':
      return <Chip tone="live">{batchStateLabels.skipped}</Chip>
    case 'sent':
      if (item.counts.failed > 0) {
        return <Chip tone="live">送れなかった人あり</Chip>
      }
      return item.counts.pending > 0 ? <Chip tone="warn">送っている途中</Chip> : null
  }
}
