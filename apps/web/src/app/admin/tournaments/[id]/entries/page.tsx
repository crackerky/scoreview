import { loadAdmin, requireUser } from '@/admin/session'
import { tournaments } from '@/api/admin'
import { entries } from '@/api/entry'
import { entryStatuses } from '@/api/entry.types'
import type { IdPageProps } from '@/app/_lib/pageProps'
import { readAdminPage } from '@/app/admin/_lib/page'
import { AdminShell, Chip, Stat } from '@/components/admin/AdminParts'
import { entryPaths } from '@/entry/entry.paths'
import {
  channelLabels,
  countByStatus,
  dateTimeText,
  parseStatus,
  registeredText,
  statusLabels,
  statusTones,
} from '@/entry/entry.utils'
import { notifyPaths } from '@/notify/notify.paths'

// 申込一覧(Lイベントの「顧客」にあたる)。大会に届いた申込を受けた順に並べ、状態で絞る。押すと申込の
// 中身と承認の画面を開く。管理者だけ(代表の連絡先が入るため)。
export default async function EntriesPage(props: IdPageProps) {
  const { id, notice } = await readAdminPage(props)
  const status = parseStatus((await props.searchParams).status)
  const { user, token } = await requireUser('admin')
  const [tournament, list] = await loadAdmin(() =>
    Promise.all([tournaments.get(token, id), entries.list(token, id)]),
  )
  const counts = countByStatus(list.entries)
  const shown = status ? list.entries.filter((item) => item.status === status) : list.entries
  const tabs = [
    {
      href: entryPaths.list(id),
      label: 'すべて',
      current: status === null,
      count: list.entries.length,
    },
    ...entryStatuses.map((value) => ({
      href: entryPaths.list(id, value),
      label: statusLabels[value],
      current: status === value,
      count: counts[value],
    })),
  ]
  return (
    <AdminShell
      user={user}
      title="申込"
      kicker={tournament.status === 'entry_open' ? '受付中' : '受付していません'}
      notice={notice}
      crumbs={[{ href: `/admin/tournaments/${id}`, label: tournament.name.ja }]}
      primary={{ label: '申込を代わりに入れる', href: entryPaths.create(id) }}
      tabs={tabs}
    >
      <div className="adm-stats">
        <Stat
          label="確認を待つ申込"
          value={String(counts.pending)}
          tone={counts.pending > 0 ? 'warn' : undefined}
        />
        <Stat label="承認した申込" value={String(counts.approved)} />
        <Stat label="差し戻し中" value={String(counts.returned)} />
      </div>
      <p className="adm-links">
        <a href={entryPaths.questions(id)}>申込で聞く質問を決める</a>
        {' ・ '}
        <a href={notifyPaths.createFor(id, { kind: 'reps', status: '', divisionId: null })}>
          公式 LINE で代表に送る
        </a>
        {list.entries.length > 0 ? (
          <>
            {' ・ '}
            <a href={entryPaths.csv(id)} download>
              CSV で書き出す(メンバー 1 人 1 行)
            </a>
          </>
        ) : null}
      </p>
      {shown.length === 0 ? (
        <p className="adm-muted">
          {list.entries.length === 0
            ? 'まだ申込がありません。電話や紙で受けた申込は「申込を代わりに入れる」から入れられます。'
            : 'この状態の申込はありません。'}
        </p>
      ) : (
        <ul className="adm-rows">
          {shown.map((item) => (
            <li key={item.id}>
              <a className="adm-row" href={entryPaths.detail(item.id)}>
                <span className="adm-row-main">
                  <span className="adm-row-title">{item.fullName}</span>
                  <span className="adm-row-meta">
                    {[
                      item.division.name.ja,
                      `代表 ${item.repName}`,
                      registeredText(item.members, item.registered),
                      `${dateTimeText(item.submittedAt)} ${channelLabels[item.channel]}`,
                    ].join(' ・ ')}
                  </span>
                </span>
                <Chip tone={statusTones[item.status]}>{statusLabels[item.status]}</Chip>
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
